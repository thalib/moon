package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/thalib/moon/cmd/moon/internal/auth"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
)

// APIKeysHandler handles API key management endpoints (admin only).
type APIKeysHandler struct {
	db           database.Driver
	apiKeyRepo   *auth.APIKeyRepository
	tokenService *auth.TokenService
}

// NewAPIKeysHandler creates a new API keys handler.
func NewAPIKeysHandler(db database.Driver, jwtSecret string, accessExpiry, refreshExpiry int) *APIKeysHandler {
	return &APIKeysHandler{
		db:           db,
		apiKeyRepo:   auth.NewAPIKeyRepository(db),
		tokenService: auth.NewTokenService(jwtSecret, accessExpiry, refreshExpiry),
	}
}

// Error codes for API key management.
const (
	ErrCodeInvalidKeyName   = "INVALID_KEY_NAME"
	ErrCodeInvalidAction    = "INVALID_ACTION"
	ErrCodeAPIKeyNotFound   = "APIKEY_NOT_FOUND"
	ErrCodeAPIKeyNameExists = "APIKEY_NAME_EXISTS"
)

// API key name validation constants.
const (
	MinKeyNameLength     = 3
	MaxKeyNameLength     = 100
	MaxDescriptionLength = 500
)

// APIKeyListRequest represents a request to list API keys.
type APIKeyListRequest struct {
	Limit int    `json:"limit,omitempty"`
	After string `json:"after,omitempty"`
}

// APIKeyListResponse represents a response with API key list.
type APIKeyListResponse struct {
	APIKeys    []APIKeyPublicInfo `json:"apikeys"`
	NextCursor *string            `json:"next_cursor"`
	Limit      int                `json:"limit"`
}

// APIKeyPublicInfo represents public API key information (no actual key).
type APIKeyPublicInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Role        string  `json:"role"`
	CanWrite    bool    `json:"can_write"`
	CreatedAt   string  `json:"created_at"`
	LastUsedAt  *string `json:"last_used_at,omitempty"`
}

// CreateAPIKeyRequest represents a request to create an API key.
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Role        string `json:"role"`
	CanWrite    *bool  `json:"can_write,omitempty"`
}

// CreateAPIKeyResponse represents a response after creating an API key.
type CreateAPIKeyResponse struct {
	Message string           `json:"message"`
	Warning string           `json:"warning"`
	APIKey  APIKeyPublicInfo `json:"apikey"`
	Key     string           `json:"key"`
}

// UpdateAPIKeyRequest represents a request to update an API key.
type UpdateAPIKeyRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	CanWrite    *bool   `json:"can_write,omitempty"`
	Action      string  `json:"action,omitempty"`
}

// UpdateAPIKeyResponse represents a response after updating an API key.
type UpdateAPIKeyResponse struct {
	Message string           `json:"message"`
	Warning string           `json:"warning,omitempty"`
	APIKey  APIKeyPublicInfo `json:"apikey"`
	Key     string           `json:"key,omitempty"`
}

// DeleteAPIKeyResponse represents a response after deleting an API key.
type DeleteAPIKeyResponse struct {
	Message string `json:"message"`
}

// ValidAPIKeyRoles returns the valid roles for API keys.
func ValidAPIKeyRoles() []string {
	return []string{"admin", "user"}
}

// IsValidAPIKeyRole checks if a role is valid for API keys.
func IsValidAPIKeyRole(role string) bool {
	for _, r := range ValidAPIKeyRoles() {
		if r == role {
			return true
		}
	}
	return false
}

// List handles GET /apikeys:list
func (h *APIKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, "admin access required", ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	limitStr := r.URL.Query().Get(constants.QueryParamLimit)
	after := r.URL.Query().Get("after")

	limit := constants.DefaultPaginationLimit
	if limitStr != "" {
		if l := parseIntWithDefault(limitStr, constants.DefaultPaginationLimit); l > 0 && l <= constants.MaxPaginationLimit {
			limit = l
		}
	}

	keys, err := h.apiKeyRepo.ListPaginated(ctx, auth.APIKeyListOptions{
		Limit:     limit + 1,
		AfterULID: after,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}

	var nextCursor *string
	if len(keys) > limit {
		keys = keys[:limit]
		cursor := keys[len(keys)-1].ULID
		nextCursor = &cursor
	}

	publicKeys := make([]APIKeyPublicInfo, len(keys))
	for i, key := range keys {
		publicKeys[i] = apiKeyToPublicInfo(key)
	}

	h.logAdminAction("apikey_list", claims.UserID, "")

	writeJSON(w, http.StatusOK, APIKeyListResponse{
		APIKeys:    publicKeys,
		NextCursor: nextCursor,
		Limit:      limit,
	})
}

// Get handles GET /apikeys:get?id={ulid}
func (h *APIKeysHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	_, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, "admin access required", ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	keyID := r.URL.Query().Get("id")
	if keyID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	apiKey, err := h.apiKeyRepo.GetByULID(ctx, keyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get API key")
		return
	}

	if apiKey == nil {
		writeErrorWithCode(w, http.StatusNotFound, "API key not found", ErrCodeAPIKeyNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"apikey": apiKeyToPublicInfo(apiKey),
	})
}

// Create handles POST /apikeys:create
func (h *APIKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, "admin access required", ErrCodeAdminRequired)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()

	// Validate required fields
	if req.Name == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "name is required", ErrCodeMissingRequiredField)
		return
	}
	if req.Role == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "role is required", ErrCodeMissingRequiredField)
		return
	}

	// Validate name length
	if len(req.Name) < MinKeyNameLength || len(req.Name) > MaxKeyNameLength {
		writeErrorWithCode(w, http.StatusBadRequest, "name must be between 3 and 100 characters", ErrCodeInvalidKeyName)
		return
	}

	// Validate description length
	if len(req.Description) > MaxDescriptionLength {
		writeErrorWithCode(w, http.StatusBadRequest, "description must not exceed 500 characters", ErrCodeInvalidFieldValue)
		return
	}

	// Validate role
	if !IsValidAPIKeyRole(req.Role) {
		writeErrorWithCode(w, http.StatusBadRequest, "role must be 'admin' or 'user'", ErrCodeInvalidRole)
		return
	}

	// Check if name exists
	exists, err := h.apiKeyRepo.NameExists(ctx, req.Name, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check name")
		return
	}
	if exists {
		writeErrorWithCode(w, http.StatusConflict, "API key name already exists", ErrCodeAPIKeyNameExists)
		return
	}

	// Generate API key
	rawKey, keyHash, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	// Determine can_write (default false)
	canWrite := false
	if req.CanWrite != nil {
		canWrite = *req.CanWrite
	}

	apiKey := &auth.APIKey{
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		Role:        req.Role,
		CanWrite:    canWrite,
	}

	if err := h.apiKeyRepo.Create(ctx, apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	h.logAdminAction("apikey_created", claims.UserID, apiKey.ULID)

	writeJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		Message: "API key created successfully",
		Warning: "Store this key securely. It will not be shown again.",
		APIKey:  apiKeyToPublicInfo(apiKey),
		Key:     rawKey,
	})
}

// Update handles POST /apikeys:update?id={ulid}
func (h *APIKeysHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, "admin access required", ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	keyID := r.URL.Query().Get("id")
	if keyID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	var req UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	apiKey, err := h.apiKeyRepo.GetByULID(ctx, keyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get API key")
		return
	}

	if apiKey == nil {
		writeErrorWithCode(w, http.StatusNotFound, "API key not found", ErrCodeAPIKeyNotFound)
		return
	}

	// Handle rotate action
	if req.Action == "rotate" {
		rawKey, keyHash, err := auth.GenerateAPIKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate new API key")
			return
		}

		if err := h.apiKeyRepo.UpdateKeyHash(ctx, apiKey.ID, keyHash); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to rotate API key")
			return
		}

		h.logAdminAction("apikey_rotated", claims.UserID, apiKey.ULID)

		writeJSON(w, http.StatusOK, UpdateAPIKeyResponse{
			Message: "API key rotated successfully",
			Warning: "Store this key securely. It will not be shown again.",
			APIKey:  apiKeyToPublicInfo(apiKey),
			Key:     rawKey,
		})
		return
	}

	// Handle invalid action
	if req.Action != "" {
		writeErrorWithCode(w, http.StatusBadRequest, "invalid action", ErrCodeInvalidAction)
		return
	}

	// Normal update: name, description, can_write
	updated := false

	if req.Name != nil {
		// Validate name length
		if len(*req.Name) < MinKeyNameLength || len(*req.Name) > MaxKeyNameLength {
			writeErrorWithCode(w, http.StatusBadRequest, "name must be between 3 and 100 characters", ErrCodeInvalidKeyName)
			return
		}

		// Check if name exists for another key
		exists, err := h.apiKeyRepo.NameExists(ctx, *req.Name, apiKey.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check name")
			return
		}
		if exists {
			writeErrorWithCode(w, http.StatusConflict, "API key name already exists", ErrCodeAPIKeyNameExists)
			return
		}

		apiKey.Name = *req.Name
		updated = true
	}

	if req.Description != nil {
		// Validate description length
		if len(*req.Description) > MaxDescriptionLength {
			writeErrorWithCode(w, http.StatusBadRequest, "description must not exceed 500 characters", ErrCodeInvalidFieldValue)
			return
		}
		apiKey.Description = *req.Description
		updated = true
	}

	if req.CanWrite != nil {
		apiKey.CanWrite = *req.CanWrite
		updated = true
	}

	if !updated {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	if err := h.apiKeyRepo.UpdateMetadata(ctx, apiKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update API key")
		return
	}

	h.logAdminAction("apikey_updated", claims.UserID, apiKey.ULID)

	writeJSON(w, http.StatusOK, UpdateAPIKeyResponse{
		Message: "API key updated successfully",
		APIKey:  apiKeyToPublicInfo(apiKey),
	})
}

// Destroy handles POST /apikeys:destroy?id={ulid}
func (h *APIKeysHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, err := h.validateAdminAccess(r)
	if err != nil {
		writeErrorWithCode(w, http.StatusForbidden, "admin access required", ErrCodeAdminRequired)
		return
	}

	ctx := r.Context()

	keyID := r.URL.Query().Get("id")
	if keyID == "" {
		writeErrorWithCode(w, http.StatusBadRequest, "id is required", ErrCodeMissingRequiredField)
		return
	}

	apiKey, err := h.apiKeyRepo.GetByULID(ctx, keyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get API key")
		return
	}

	if apiKey == nil {
		writeErrorWithCode(w, http.StatusNotFound, "API key not found", ErrCodeAPIKeyNotFound)
		return
	}

	if err := h.apiKeyRepo.Delete(ctx, apiKey.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	h.logAdminAction("apikey_deleted", claims.UserID, keyID)

	writeJSON(w, http.StatusOK, DeleteAPIKeyResponse{
		Message: "API key deleted successfully",
	})
}

// validateAdminAccess validates that the request is from an admin user.
func (h *APIKeysHandler) validateAdminAccess(r *http.Request) (*auth.Claims, error) {
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
func (h *APIKeysHandler) logAdminAction(action, adminULID, targetULID string) {
	if targetULID != "" {
		log.Printf("INFO: ADMIN_ACTION %s by=%s key_id=%s", action, adminULID, targetULID)
	} else {
		log.Printf("INFO: ADMIN_ACTION %s by=%s", action, adminULID)
	}
}

// apiKeyToPublicInfo converts an APIKey to public info.
func apiKeyToPublicInfo(apiKey *auth.APIKey) APIKeyPublicInfo {
	info := APIKeyPublicInfo{
		ID:          apiKey.ULID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		Role:        apiKey.Role,
		CanWrite:    apiKey.CanWrite,
		CreatedAt:   apiKey.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if apiKey.LastUsedAt != nil {
		lastUsed := apiKey.LastUsedAt.Format("2006-01-02T15:04:05Z")
		info.LastUsedAt = &lastUsed
	}

	return info
}
