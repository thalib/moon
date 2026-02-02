package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/auth"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/middleware"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	db               database.Driver
	userRepo         *auth.UserRepository
	tokenRepo        *auth.RefreshTokenRepository
	tokenService     *auth.TokenService
	tokenBlacklist   *auth.TokenBlacklist
	loginRateLimiter *middleware.LoginRateLimiter
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(db database.Driver, jwtSecret string, accessExpiry, refreshExpiry int) *AuthHandler {
	return &AuthHandler{
		db:             db,
		userRepo:       auth.NewUserRepository(db),
		tokenRepo:      auth.NewRefreshTokenRepository(db),
		tokenService:   auth.NewTokenService(jwtSecret, accessExpiry, refreshExpiry),
		tokenBlacklist: auth.NewTokenBlacklist(db),
		loginRateLimiter: middleware.NewLoginRateLimiter(middleware.LoginRateLimiterConfig{
			MaxAttempts:   5,   // 5 failed attempts
			WindowSeconds: 900, // 15 minutes
		}),
	}
}

// NewAuthHandlerWithRateLimiter creates a new auth handler with custom rate limiter config.
func NewAuthHandlerWithRateLimiter(db database.Driver, jwtSecret string, accessExpiry, refreshExpiry, maxAttempts, windowSeconds int) *AuthHandler {
	return &AuthHandler{
		db:             db,
		userRepo:       auth.NewUserRepository(db),
		tokenRepo:      auth.NewRefreshTokenRepository(db),
		tokenService:   auth.NewTokenService(jwtSecret, accessExpiry, refreshExpiry),
		tokenBlacklist: auth.NewTokenBlacklist(db),
		loginRateLimiter: middleware.NewLoginRateLimiter(middleware.LoginRateLimiterConfig{
			MaxAttempts:   maxAttempts,
			WindowSeconds: windowSeconds,
		}),
	}
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	User         UserInfo  `json:"user"`
}

// UserInfo represents public user information.
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	CanWrite bool   `json:"can_write"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UpdateMeRequest represents a request to update the current user.
type UpdateMeRequest struct {
	Email       string `json:"email,omitempty"`
	Password    string `json:"password,omitempty"`
	OldPassword string `json:"old_password,omitempty"`
}

// Login handles POST /auth:login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Get client IP for rate limiting
	clientIP := getClientIP(r)

	// Check login rate limit before processing
	blocked, resetAt := h.loginRateLimiter.IsBlocked(clientIP, req.Username)
	if blocked {
		middleware.LogLoginRateLimitExceeded(clientIP, req.Username, r.URL.Path)
		middleware.WriteLoginRateLimitError(w, resetAt)
		return
	}

	ctx := r.Context()

	// Get user by username
	user, err := h.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	if user == nil {
		// Record failed attempt
		h.loginRateLimiter.CheckAndRecord(clientIP, req.Username)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Verify password
	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		// Record failed attempt
		h.loginRateLimiter.CheckAndRecord(clientIP, req.Username)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Successful login - reset rate limit counter
	h.loginRateLimiter.ResetForUser(clientIP, req.Username)

	// Generate token pair
	tokenPair, rawRefreshToken, err := h.tokenService.GenerateTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Store refresh token hash
	refreshToken := &auth.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashToken(rawRefreshToken),
		ExpiresAt: time.Now().Add(h.tokenService.RefreshExpiry()),
	}

	if err := h.tokenRepo.Create(ctx, refreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Update last login
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Non-fatal error, log but continue
	}

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
		User: UserInfo{
			ID:       user.ULID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			CanWrite: user.CanWrite,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// Logout handles POST /auth:logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	ctx := r.Context()

	// Delete the refresh token
	tokenHash := auth.HashToken(req.RefreshToken)
	if err := h.tokenRepo.DeleteByHash(ctx, tokenHash); err != nil {
		// Non-fatal, token might already be deleted
	}

	// Blacklist the current access token to invalidate it immediately
	accessToken, err := h.extractAccessToken(r)
	if err == nil && accessToken != "" {
		// Parse the token to get expiration and user ID
		claims, err := h.tokenService.ValidateAccessToken(accessToken)
		if err == nil {
			// Get user by ULID from claims
			user, err := h.userRepo.GetByULID(ctx, claims.UserID)
			if err == nil && user != nil {
				expiresAt := claims.ExpiresAt.Time
				_ = h.tokenBlacklist.Add(ctx, accessToken, user.ID, expiresAt)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// extractAccessToken extracts the access token from the Authorization header
func (h *AuthHandler) extractAccessToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != strings.ToLower(constants.AuthSchemeBearer) {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// Refresh handles POST /auth:refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	ctx := r.Context()

	// Validate refresh token
	tokenHash := auth.HashToken(req.RefreshToken)
	refreshToken, err := h.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate token")
		return
	}

	if refreshToken == nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if refreshToken.IsExpired() {
		// Delete expired token
		h.tokenRepo.Delete(ctx, refreshToken.ID)
		writeError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Generate new token pair
	tokenPair, newRawRefreshToken, err := h.tokenService.GenerateTokenPair(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	// Delete old refresh token
	h.tokenRepo.Delete(ctx, refreshToken.ID)

	// Store new refresh token
	newRefreshToken := &auth.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashToken(newRawRefreshToken),
		ExpiresAt: time.Now().Add(h.tokenService.RefreshExpiry()),
	}

	if err := h.tokenRepo.Create(ctx, newRefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
		User: UserInfo{
			ID:       user.ULID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			CanWrite: user.CanWrite,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// Me handles GET /auth:me (get current user) and POST /auth:me (update current user)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetMe(w, r)
	case http.MethodPost:
		h.UpdateMe(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GetMe handles GET /auth:me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token in Authorization header
	userID, err := h.extractUserIDFromToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx := r.Context()

	user, err := h.userRepo.GetByULID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	response := UserInfo{
		ID:       user.ULID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		CanWrite: user.CanWrite,
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": response})
}

// UpdateMe handles POST /auth:me
func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token
	userID, err := h.extractUserIDFromToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()

	user, err := h.userRepo.GetByULID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// Update email if provided
	if req.Email != "" {
		user.Email = req.Email
	}

	// Update password if provided
	if req.Password != "" {
		// Require old password for password change
		if req.OldPassword == "" {
			writeError(w, http.StatusBadRequest, "old_password is required to change password")
			return
		}

		// Verify old password
		if err := auth.ComparePassword(user.PasswordHash, req.OldPassword); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid old password")
			return
		}

		// Hash new password
		newHash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update password")
			return
		}
		user.PasswordHash = newHash

		// Password changed - revoke all refresh tokens to force re-login
		if err := h.tokenRepo.DeleteAllByUserID(ctx, user.ID); err != nil {
			// Log but don't fail - password update is more important
			writeError(w, http.StatusInternalServerError, "failed to revoke sessions")
			return
		}

		// Blacklist the current access token
		accessToken, err := h.extractAccessToken(r)
		if err == nil && accessToken != "" {
			claims, err := h.tokenService.ValidateAccessToken(accessToken)
			if err == nil {
				expiresAt := claims.ExpiresAt.Time
				_ = h.tokenBlacklist.Add(ctx, accessToken, user.ID, expiresAt)
			}
		}
	}

	// Save changes
	if err := h.userRepo.Update(ctx, user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	response := UserInfo{
		ID:       user.ULID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		CanWrite: user.CanWrite,
	}

	message := "user updated successfully"
	if req.Password != "" {
		message = "password updated successfully, please login again"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"user":    response,
	})
}

// extractUserIDFromToken extracts the user ID from the Authorization header.
func (h *AuthHandler) extractUserIDFromToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if authHeader == "" {
		return "", http.ErrNoCookie // Use as a generic "not found" error
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != strings.ToLower(constants.AuthSchemeBearer) {
		return "", http.ErrNoCookie
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", http.ErrNoCookie
	}

	claims, err := h.tokenService.ValidateAccessToken(token)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

// getClientIP extracts the client IP from the request.
// It checks X-Forwarded-For and X-Real-IP headers first (for reverse proxy scenarios),
// then falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common in reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs; the first one is the client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header (used by nginx)
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in the form "IP:port", we need to extract just the IP
	addr := r.RemoteAddr
	if colonIndex := strings.LastIndex(addr, ":"); colonIndex != -1 {
		return addr[:colonIndex]
	}
	return addr
}
