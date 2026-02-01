package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thalib/moon/cmd/moon/internal/auth"
	"github.com/thalib/moon/cmd/moon/internal/database"
)

func setupTestAPIKeysHandler(t *testing.T) (*APIKeysHandler, *auth.User, string, database.Driver) {
	t.Helper()

	cfg := database.Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     1,
		MaxIdleConns:     1,
	}

	db, err := database.NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create database driver: %v", err)
	}

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if err := auth.Bootstrap(ctx, db, nil); err != nil {
		t.Fatalf("failed to bootstrap auth: %v", err)
	}

	passwordHash, _ := auth.HashPassword("AdminPass123")
	userRepo := auth.NewUserRepository(db)
	adminUser := &auth.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleAdmin),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, adminUser); err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	handler := NewAPIKeysHandler(db, "test-secret-key", 3600, 604800)

	tokenService := auth.NewTokenService("test-secret-key", 3600, 604800)
	tokenPair, _, err := tokenService.GenerateTokenPair(adminUser)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	return handler, adminUser, tokenPair.AccessToken, db
}

func TestAPIKeysHandler_List_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/apikeys:list", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp APIKeyListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.APIKeys == nil {
		t.Error("List() should return apikeys array")
	}
}

func TestAPIKeysHandler_List_Unauthorized(t *testing.T) {
	handler, _, _, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/apikeys:list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("List() without auth status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAPIKeysHandler_List_NonAdminForbidden(t *testing.T) {
	handler, _, _, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	ctx := context.Background()

	passwordHash, _ := auth.HashPassword("UserPass123")
	userRepo := auth.NewUserRepository(db)
	regularUser := &auth.User{
		Username:     "regularuser",
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, regularUser); err != nil {
		t.Fatalf("failed to create regular user: %v", err)
	}

	tokenService := auth.NewTokenService("test-secret-key", 3600, 604800)
	tokenPair, _, err := tokenService.GenerateTokenPair(regularUser)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/apikeys:list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("List() with non-admin status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAPIKeysHandler_List_WithPagination(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/apikeys:list?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp APIKeyListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Limit != 10 {
		t.Errorf("List() limit = %d, want 10", resp.Limit)
	}
}

func TestAPIKeysHandler_Create_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Name:        "test-api-key",
		Description: "A test API key",
		Role:        "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CreateAPIKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.APIKey.Name != "test-api-key" {
		t.Errorf("Create() name = %v, want test-api-key", resp.APIKey.Name)
	}
	if resp.APIKey.Role != "user" {
		t.Errorf("Create() role = %v, want user", resp.APIKey.Role)
	}
	if resp.Key == "" {
		t.Error("Create() should return the key value")
	}
	if resp.Warning == "" {
		t.Error("Create() should return a warning")
	}
	if !hasPrefix(resp.Key, "moon_live_") {
		t.Errorf("Create() key should have moon_live_ prefix, got: %s", resp.Key)
	}
}

func TestAPIKeysHandler_Create_WithCanWrite(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	canWrite := true
	body := CreateAPIKeyRequest{
		Name:     "writable-key",
		Role:     "user",
		CanWrite: &canWrite,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CreateAPIKeyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.APIKey.CanWrite {
		t.Error("Create() can_write should be true")
	}
}

func TestAPIKeysHandler_Create_MissingName(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() without name status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeMissingRequiredField {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeMissingRequiredField)
	}
}

func TestAPIKeysHandler_Create_MissingRole(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Name: "test-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() without role status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeMissingRequiredField {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeMissingRequiredField)
	}
}

func TestAPIKeysHandler_Create_InvalidRole(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Name: "test-key",
		Role: "invalid",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() with invalid role status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeInvalidRole {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeInvalidRole)
	}
}

func TestAPIKeysHandler_Create_NameTooShort(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Name: "ab",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() with short name status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeInvalidKeyName {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeInvalidKeyName)
	}
}

func TestAPIKeysHandler_Create_DuplicateName(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create first key
	body := CreateAPIKeyRequest{
		Name: "duplicate-name",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("First Create() failed: %s", w.Body.String())
	}

	// Try to create second key with same name
	req = httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Create() with duplicate name status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeAPIKeyNameExists {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeAPIKeyNameExists)
	}
}

func TestAPIKeysHandler_Get_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	body := CreateAPIKeyRequest{
		Name: "test-get-key",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Now get it
	req = httptest.NewRequest(http.MethodGet, "/apikeys:get?id="+createResp.APIKey.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Get() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	apiKey := resp["apikey"].(map[string]any)

	if apiKey["name"] != "test-get-key" {
		t.Errorf("Get() name = %v, want test-get-key", apiKey["name"])
	}
}

func TestAPIKeysHandler_Get_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/apikeys:get?id=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Get() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeAPIKeyNotFound {
		t.Errorf("Get() error_code = %v, want %v", resp["error_code"], ErrCodeAPIKeyNotFound)
	}
}

func TestAPIKeysHandler_Get_MissingID(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/apikeys:get", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Get() without id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIKeysHandler_Update_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name: "original-name",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Update it
	newName := "updated-name"
	updateBody := UpdateAPIKeyRequest{
		Name: &newName,
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UpdateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.APIKey.Name != "updated-name" {
		t.Errorf("Update() name = %v, want updated-name", resp.APIKey.Name)
	}
	if resp.Key != "" {
		t.Error("Update() should not return key for non-rotate updates")
	}
}

func TestAPIKeysHandler_Update_Rotate(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name: "rotate-test-key",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	originalKey := createResp.Key

	// Rotate it
	updateBody := UpdateAPIKeyRequest{
		Action: "rotate",
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() with rotate status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UpdateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Key == "" {
		t.Error("Update() with rotate should return new key")
	}
	if resp.Key == originalKey {
		t.Error("Update() with rotate should return different key")
	}
	if resp.Warning == "" {
		t.Error("Update() with rotate should return warning")
	}
}

func TestAPIKeysHandler_Update_InvalidAction(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name: "invalid-action-test",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Try invalid action
	updateBody := UpdateAPIKeyRequest{
		Action: "invalid",
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() with invalid action status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeInvalidAction {
		t.Errorf("Update() error_code = %v, want %v", resp["error_code"], ErrCodeInvalidAction)
	}
}

func TestAPIKeysHandler_Update_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	newName := "new-name"
	updateBody := UpdateAPIKeyRequest{
		Name: &newName,
	}
	bodyBytes, _ := json.Marshal(updateBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:update?id=nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Update() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAPIKeysHandler_Update_NoFieldsToUpdate(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name: "no-update-test",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Try update with no fields
	updateBody := UpdateAPIKeyRequest{}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() with no fields status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIKeysHandler_Destroy_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name: "delete-test-key",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Delete it
	req = httptest.NewRequest(http.MethodPost, "/apikeys:destroy?id="+createResp.APIKey.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	handler.Destroy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Destroy() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify it's deleted
	req = httptest.NewRequest(http.MethodGet, "/apikeys:get?id="+createResp.APIKey.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Error("Destroy() key should be deleted")
	}
}

func TestAPIKeysHandler_Destroy_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/apikeys:destroy?id=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Destroy() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAPIKeysHandler_Destroy_MissingID(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/apikeys:destroy", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Destroy() without id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIKeysHandler_WrongMethods(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	tests := []struct {
		name    string
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"List with POST", http.MethodPost, "/apikeys:list", handler.List},
		{"Get with POST", http.MethodPost, "/apikeys:get?id=123", handler.Get},
		{"Create with GET", http.MethodGet, "/apikeys:create", handler.Create},
		{"Update with GET", http.MethodGet, "/apikeys:update?id=123", handler.Update},
		{"Destroy with GET", http.MethodGet, "/apikeys:destroy?id=123", handler.Destroy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s status = %d, want %d", tt.name, w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestNewAPIKeysHandler(t *testing.T) {
	cfg := database.Config{
		ConnectionString: "sqlite://:memory:",
		MaxOpenConns:     1,
		MaxIdleConns:     1,
	}

	db, err := database.NewDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create database driver: %v", err)
	}
	defer db.Close()

	handler := NewAPIKeysHandler(db, "secret", 3600, 604800)
	if handler == nil {
		t.Error("NewAPIKeysHandler() returned nil")
	}
	if handler.apiKeyRepo == nil {
		t.Error("NewAPIKeysHandler() apiKeyRepo is nil")
	}
	if handler.tokenService == nil {
		t.Error("NewAPIKeysHandler() tokenService is nil")
	}
}

func TestAPIKeyToPublicInfo(t *testing.T) {
	apiKey := &auth.APIKey{
		ULID:        "01H1234567890ABCDEFGHJKMNP",
		Name:        "test-key",
		Description: "Test description",
		Role:        "admin",
		CanWrite:    true,
	}

	info := apiKeyToPublicInfo(apiKey)

	if info.ID != apiKey.ULID {
		t.Errorf("apiKeyToPublicInfo() ID = %s, want %s", info.ID, apiKey.ULID)
	}
	if info.Name != apiKey.Name {
		t.Errorf("apiKeyToPublicInfo() Name = %s, want %s", info.Name, apiKey.Name)
	}
	if info.Description != apiKey.Description {
		t.Errorf("apiKeyToPublicInfo() Description = %s, want %s", info.Description, apiKey.Description)
	}
	if info.Role != apiKey.Role {
		t.Errorf("apiKeyToPublicInfo() Role = %s, want %s", info.Role, apiKey.Role)
	}
	if info.CanWrite != apiKey.CanWrite {
		t.Errorf("apiKeyToPublicInfo() CanWrite = %v, want %v", info.CanWrite, apiKey.CanWrite)
	}
}

func TestIsValidAPIKeyRole(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"admin", true},
		{"user", true},
		{"readonly", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsValidAPIKeyRole(tt.role)
		if result != tt.expected {
			t.Errorf("IsValidAPIKeyRole(%q) = %v, want %v", tt.role, result, tt.expected)
		}
	}
}

func TestAPIKeysHandler_Update_DuplicateName(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create first key
	body1 := CreateAPIKeyRequest{
		Name: "key-one",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(body1)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	// Create second key
	body2 := CreateAPIKeyRequest{
		Name: "key-two",
		Role: "user",
	}
	bodyBytes, _ = json.Marshal(body2)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Try to rename second key to first key's name
	newName := "key-one"
	updateBody := UpdateAPIKeyRequest{
		Name: &newName,
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Update() with duplicate name status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeAPIKeyNameExists {
		t.Errorf("Update() error_code = %v, want %v", resp["error_code"], ErrCodeAPIKeyNameExists)
	}
}

func TestAPIKeysHandler_Create_DescriptionTooLong(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a 501-character description
	longDesc := make([]byte, 501)
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	body := CreateAPIKeyRequest{
		Name:        "test-key",
		Description: string(longDesc),
		Role:        "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() with long description status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPIKeysHandler_Update_DescriptionUpdate(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first
	createBody := CreateAPIKeyRequest{
		Name:        "desc-update-test",
		Description: "original description",
		Role:        "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	// Update description
	newDesc := "updated description"
	updateBody := UpdateAPIKeyRequest{
		Description: &newDesc,
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UpdateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.APIKey.Description != "updated description" {
		t.Errorf("Update() description = %v, want 'updated description'", resp.APIKey.Description)
	}
}

func TestAPIKeysHandler_Update_CanWriteUpdate(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	// Create a key first (default can_write is false)
	createBody := CreateAPIKeyRequest{
		Name: "canwrite-update-test",
		Role: "user",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Create(w, req)

	var createResp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&createResp)

	if createResp.APIKey.CanWrite {
		t.Error("Create() default can_write should be false")
	}

	// Update can_write
	canWrite := true
	updateBody := UpdateAPIKeyRequest{
		CanWrite: &canWrite,
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest(http.MethodPost, "/apikeys:update?id="+createResp.APIKey.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UpdateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.APIKey.CanWrite {
		t.Error("Update() can_write should be true after update")
	}
}

func TestAPIKeysHandler_Create_AdminRole(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	body := CreateAPIKeyRequest{
		Name: "admin-key",
		Role: "admin",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create() with admin role status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.APIKey.Role != "admin" {
		t.Errorf("Create() role = %v, want admin", resp.APIKey.Role)
	}
}

func TestAPIKeysHandler_Update_MissingID(t *testing.T) {
	handler, _, adminToken, db := setupTestAPIKeysHandler(t)
	defer db.Close()

	newName := "new-name"
	updateBody := UpdateAPIKeyRequest{
		Name: &newName,
	}
	bodyBytes, _ := json.Marshal(updateBody)

	req := httptest.NewRequest(http.MethodPost, "/apikeys:update", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() without id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
