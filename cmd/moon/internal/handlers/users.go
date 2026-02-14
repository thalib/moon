package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/auth"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
)

// UsersHandler handles user management endpoints (admin only).
type UsersHandler struct {
	db             database.Driver
	userRepo       *auth.UserRepository
	tokenRepo      *auth.RefreshTokenRepository
	tokenService   *auth.TokenService
	passwordPolicy *auth.PasswordPolicy
}

// NewUsersHandler creates a new users handler.
func NewUsersHandler(db database.Driver, jwtSecret string, accessExpiry, refreshExpiry int) *UsersHandler {
	return &UsersHandler{
		db:             db,
		userRepo:       auth.NewUserRepository(db),
		tokenRepo:      auth.NewRefreshTokenRepository(db),
		tokenService:   auth.NewTokenService(jwtSecret, accessExpiry, refreshExpiry),
		passwordPolicy: auth.DefaultPasswordPolicy(),
	}
}

// Error codes for user management.
const (
	ErrCodeMissingRequiredField  = "MISSING_REQUIRED_FIELD"
	ErrCodeInvalidFieldValue     = "INVALID_FIELD_VALUE"
	ErrCodeWeakPassword          = "WEAK_PASSWORD"
	ErrCodeInvalidEmailFormat    = "INVALID_EMAIL_FORMAT"
	ErrCodeInvalidRole           = "INVALID_ROLE"
	ErrCodeAdminRequired         = "ADMIN_REQUIRED"
	ErrCodeCannotModifySelf      = "CANNOT_MODIFY_SELF"
	ErrCodeCannotDeleteLastAdmin = "CANNOT_DELETE_LAST_ADMIN"
	ErrCodeUserNotFound          = "USER_NOT_FOUND"
	ErrCodeUsernameExists        = "USERNAME_EXISTS"
	ErrCodeEmailExists           = "EMAIL_EXISTS"
)

// UserListRequest represents a request to list users.
type UserListRequest struct {
	Limit int    `json:"limit,omitempty"`
	After string `json:"after,omitempty"`
	Role  string `json:"role,omitempty"`
}

// UserListResponse represents a response with user list.
type UserListResponse struct {
	Users      []UserPublicInfo `json:"users"`
	NextCursor *string          `json:"next_cursor"`
	Limit      int              `json:"limit"`
}

// UserPublicInfo represents public user information for admin APIs.
type UserPublicInfo struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	CanWrite    bool    `json:"can_write"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
}

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	CanWrite *bool  `json:"can_write,omitempty"`
}

// CreateUserResponse represents a response after creating a user.
type CreateUserResponse struct {
	Message string         `json:"message"`
	User    UserPublicInfo `json:"user"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Email       *string `json:"email,omitempty"`
	Role        *string `json:"role,omitempty"`
	CanWrite    *bool   `json:"can_write,omitempty"`
	Action      string  `json:"action,omitempty"`
	NewPassword string  `json:"new_password,omitempty"`
}

// UpdateUserResponse represents a response after updating a user.
type UpdateUserResponse struct {
	Message string         `json:"message"`
	User    UserPublicInfo `json:"user"`
}

// DeleteUserResponse represents a response after deleting a user.
type DeleteUserResponse struct {
	Message string `json:"message"`
}

// emailRegex validates email format.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// List handles GET /users:list
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate admin access
	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, err.Error(), ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	// Parse query parameters
	limitStr := r.URL.Query().Get(constants.QueryParamLimit)
	after := r.URL.Query().Get("after")
	roleFilter := r.URL.Query().Get("role")

	limit := constants.DefaultPaginationLimit
	if limitStr != "" {
		if l := parseIntWithDefault(limitStr, constants.DefaultPaginationLimit); l > 0 && l <= constants.MaxPaginationLimit {
			limit = l
		}
	}

	// Validate role filter if provided
	if roleFilter != "" && !auth.IsValidRole(roleFilter) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid role filter", ErrCodeInvalidRole)
		return
	}

	// List users
	users, err := h.userRepo.List(ctx, auth.ListOptions{
		Limit:      limit + 1, // Fetch one extra to determine if there are more
		AfterID:    after,
		RoleFilter: roleFilter,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	// Determine if there are more results
	var nextCursor *string
	if len(users) > limit {
		users = users[:limit]
		cursor := users[len(users)-1].ID
		nextCursor = &cursor
	}

	// Convert to public info
	publicUsers := make([]UserPublicInfo, len(users))
	for i, user := range users {
		publicUsers[i] = userToPublicInfo(user)
	}

	h.logAdminAction("user_list", claims.UserID, "")

	writeJSON(w, http.StatusOK, UserListResponse{
		Users:      publicUsers,
		NextCursor: nextCursor,
		Limit:      limit,
	})
}

// Get handles GET /users:get?id={ulid}
func (h *UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate admin access
	_, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, err.Error(), ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	// Get user ID from query
	userID := r.URL.Query().Get("id")
	if userID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user == nil {
		writeErrorWithCode(w, http.StatusNotFound, "user not found", ErrCodeUserNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": userToPublicInfo(user),
	})
}

// Create handles POST /users:create
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate admin access
	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, err.Error(), ErrCodeAdminRequired)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()

	// Validate required fields
	if req.Username == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "username is required", ErrCodeMissingRequiredField)
		return
	}
	if req.Email == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "email is required", ErrCodeMissingRequiredField)
		return
	}
	if req.Password == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "password is required", ErrCodeMissingRequiredField)
		return
	}
	if req.Role == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "role is required", ErrCodeMissingRequiredField)
		return
	}

	// Validate email format
	if !emailRegex.MatchString(req.Email) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid email format", ErrCodeInvalidEmailFormat)
		return
	}

	// Validate role
	if !auth.IsValidRole(req.Role) {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid role", ErrCodeInvalidRole)
		return
	}

	// Validate password
	if err := h.passwordPolicy.Validate(req.Password); err != nil {
		writeErrorWithCode(w, http.StatusBadRequest, err.Error(), ErrCodeWeakPassword)
		return
	}

	// Check if username exists
	exists, err := h.userRepo.UsernameExists(ctx, req.Username, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check username")
		return
	}
	if exists {
		writeErrorWithCode(w, http.StatusConflict, "username already exists", ErrCodeUsernameExists)
		return
	}

	// Check if email exists
	exists, err = h.userRepo.EmailExists(ctx, req.Email, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check email")
		return
	}
	if exists {
		writeErrorWithCode(w, http.StatusConflict, "email already exists", ErrCodeEmailExists)
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Determine can_write (default based on role)
	canWrite := true
	if req.CanWrite != nil {
		canWrite = *req.CanWrite
	} else if req.Role == string(auth.RoleReadOnly) {
		canWrite = false
	}

	// Create user
	user := &auth.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		CanWrite:     canWrite,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	h.logAdminAction("user_created", claims.UserID, user.ID)

	writeJSON(w, http.StatusCreated, CreateUserResponse{
		Message: "user created successfully",
		User:    userToPublicInfo(user),
	})
}

// Update handles POST /users:update?id={ulid}
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate admin access
	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, err.Error(), ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	// Get user ID from query
	userID := r.URL.Query().Get("id")
	if userID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	// Check if admin is trying to modify themselves
	if claims.UserID == userID {
		writeErrorWithCode(w, http.StatusForbidden, "cannot modify own account via user management endpoints", ErrCodeCannotModifySelf)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user == nil {
		writeErrorWithCode(w, http.StatusNotFound, "user not found", ErrCodeUserNotFound)
		return
	}

	// Handle action-based updates
	switch req.Action {
	case "reset_password":
		if req.NewPassword == "" {
			writeErrorWithCode(w, http.StatusBadRequest, "new_password is required for password reset", ErrCodeMissingRequiredField)
			return
		}

		// Validate password
		if err := h.passwordPolicy.Validate(req.NewPassword); err != nil {
			writeErrorWithCode(w, http.StatusBadRequest, err.Error(), ErrCodeWeakPassword)
			return
		}

		// Hash new password
		passwordHash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reset password")
			return
		}
		user.PasswordHash = passwordHash

		if err := h.userRepo.Update(ctx, user); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update user")
			return
		}

		h.logAdminAction("password_reset", claims.UserID, user.ID)

		writeJSON(w, http.StatusOK, UpdateUserResponse{
			Message: "password reset successfully",
			User:    userToPublicInfo(user),
		})
		return

	case "revoke_sessions":
		// Delete all refresh tokens for this user
		if err := h.tokenRepo.DeleteByUserID(ctx, user.PKID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to revoke sessions")
			return
		}

		h.logAdminAction("sessions_revoked", claims.UserID, user.ID)

		writeJSON(w, http.StatusOK, UpdateUserResponse{
			Message: "all sessions revoked successfully",
			User:    userToPublicInfo(user),
		})
		return

	case "":
		// Normal update, continue below
	default:
		writeErrorWithCode(w, http.StatusBadRequest, "invalid action", ErrCodeInvalidFieldValue)
		return
	}

	// Normal update: role, can_write, email
	updated := false

	if req.Email != nil {
		// Validate email format
		if !emailRegex.MatchString(*req.Email) {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid email format", ErrCodeInvalidEmailFormat)
			return
		}

		// Check if email exists for another user
		exists, err := h.userRepo.EmailExists(ctx, *req.Email, user.PKID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check email")
			return
		}
		if exists {
			writeErrorWithCode(w, http.StatusConflict, "email already exists", ErrCodeEmailExists)
			return
		}

		user.Email = *req.Email
		updated = true
	}

	if req.Role != nil {
		// Validate role
		if !auth.IsValidRole(*req.Role) {
			writeErrorWithCode(w, http.StatusBadRequest, "invalid role", ErrCodeInvalidRole)
			return
		}

		// Check if downgrading the last admin
		if user.Role == string(auth.RoleAdmin) && *req.Role != string(auth.RoleAdmin) {
			adminCount, err := h.userRepo.CountByRole(ctx, string(auth.RoleAdmin))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to check admin count")
				return
			}
			if adminCount <= 1 {
				writeErrorWithCode(w, http.StatusForbidden, "cannot downgrade the last admin user", ErrCodeCannotDeleteLastAdmin)
				return
			}
		}

		user.Role = *req.Role
		updated = true
	}

	if req.CanWrite != nil {
		user.CanWrite = *req.CanWrite
		updated = true
	}

	if !updated {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	if err := h.userRepo.Update(ctx, user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	h.logAdminAction("user_updated", claims.UserID, user.ID)

	writeJSON(w, http.StatusOK, UpdateUserResponse{
		Message: "user updated successfully",
		User:    userToPublicInfo(user),
	})
}

// Destroy handles POST /users:destroy?id={ulid}
func (h *UsersHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate admin access
	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, err.Error(), ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	// Get user ID from query
	userID := r.URL.Query().Get("id")
	if userID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	// Check if admin is trying to delete themselves
	if claims.UserID == userID {
		writeErrorWithCode(w, http.StatusForbidden, "cannot delete own account via user management endpoints", ErrCodeCannotModifySelf)
		return
	}

	// Get user to check role
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user == nil {
		writeErrorWithCode(w, http.StatusNotFound, "user not found", ErrCodeUserNotFound)
		return
	}

	// Check if deleting the last admin
	if user.Role == string(auth.RoleAdmin) {
		adminCount, err := h.userRepo.CountByRole(ctx, string(auth.RoleAdmin))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check admin count")
			return
		}
		if adminCount <= 1 {
			writeErrorWithCode(w, http.StatusForbidden, "cannot delete the last admin user", ErrCodeCannotDeleteLastAdmin)
			return
		}
	}

	// Delete user's refresh tokens first (cascade)
	if err := h.tokenRepo.DeleteByUserID(ctx, user.PKID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user sessions")
		return
	}

	// Delete user
	if err := h.userRepo.Delete(ctx, user.PKID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	h.logAdminAction("user_deleted", claims.UserID, userID)

	writeJSON(w, http.StatusOK, DeleteUserResponse{
		Message: "user deleted successfully",
	})
}

// validateAdminAccess validates that the request is from an admin user.
func (h *UsersHandler) validateAdminAccess(r *http.Request) (*auth.Claims, error) {
	authHeader := r.Header.Get(constants.HeaderAuthorization)
	if authHeader == "" {
		return nil, http.ErrNoCookie
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != strings.ToLower(constants.AuthSchemeBearer) {
		return nil, http.ErrNoCookie
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return nil, http.ErrNoCookie
	}

	claims, err := h.tokenService.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}

	if claims.Role != string(auth.RoleAdmin) {
		return nil, http.ErrNoCookie
	}

	return claims, nil
}

// logAdminAction logs an admin action for audit purposes.
func (h *UsersHandler) logAdminAction(action, adminULID, targetULID string) {
	if targetULID != "" {
		log.Printf("INFO: ADMIN_ACTION %s by=%s target=%s", action, adminULID, targetULID)
	} else {
		log.Printf("INFO: ADMIN_ACTION %s by=%s", action, adminULID)
	}
}

// userToPublicInfo converts a User to public info.
func userToPublicInfo(user *auth.User) UserPublicInfo {
	info := UserPublicInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CanWrite:  user.CanWrite,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if user.LastLoginAt != nil {
		lastLogin := user.LastLoginAt.Format("2006-01-02T15:04:05Z")
		info.LastLoginAt = &lastLogin
	}

	return info
}

// parseIntWithDefault parses an int string with a default value.
func parseIntWithDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return defaultVal
		}
		result = result*10 + int(c-'0')
	}
	return result
}

// writeErrorWithCode writes a JSON error response with an error code.
func writeErrorWithCode(w http.ResponseWriter, statusCode int, message, code string) {
	writeJSON(w, statusCode, map[string]any{
		"error":      message,
		"error_code": code,
		"code":       statusCode,
	})
}
