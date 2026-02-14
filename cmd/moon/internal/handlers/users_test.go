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

func setupTestUsersHandler(t *testing.T) (*UsersHandler, *auth.User, string, database.Driver) {
	t.Helper()

	// Create in-memory SQLite database
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

	// Initialize auth schema
	if err := auth.Bootstrap(ctx, db, nil); err != nil {
		t.Fatalf("failed to bootstrap auth: %v", err)
	}

	// Create an admin user
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

	handler := NewUsersHandler(db, "test-secret-key", 3600, 604800)

	// Generate token for admin
	tokenService := auth.NewTokenService("test-secret-key", 3600, 604800)
	tokenPair, _, err := tokenService.GenerateTokenPair(adminUser)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	return handler, adminUser, tokenPair.AccessToken, db
}

func TestUsersHandler_List_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:list", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UserListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Users) == 0 {
		t.Error("List() should return at least one user")
	}
}

func TestUsersHandler_List_Unauthorized(t *testing.T) {
	handler, _, _, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("List() without auth status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUsersHandler_List_NonAdminForbidden(t *testing.T) {
	handler, _, _, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a regular user
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

	// Generate token for regular user
	tokenService := auth.NewTokenService("test-secret-key", 3600, 604800)
	tokenPair, _, err := tokenService.GenerateTokenPair(regularUser)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/users:list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("List() with non-admin status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUsersHandler_List_WithRoleFilter(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:list?role=admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UserListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, user := range resp.Users {
		if user.Role != "admin" {
			t.Errorf("List() with role filter returned user with role %s, want admin", user.Role)
		}
	}
}

func TestUsersHandler_Get_Success(t *testing.T) {
	handler, admin, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:get?id="+admin.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Get() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	userInfo := resp["user"].(map[string]any)
	if userInfo["username"] != "admin" {
		t.Errorf("Get() username = %v, want admin", userInfo["username"])
	}
}

func TestUsersHandler_Get_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:get?id=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Get() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUsersHandler_Get_MissingID(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:get", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Get() without id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUsersHandler_Create_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "NewUser123",
		Role:     "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CreateUserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.User.Username != "newuser" {
		t.Errorf("Create() username = %v, want newuser", resp.User.Username)
	}
	if resp.User.Role != "user" {
		t.Errorf("Create() role = %v, want user", resp.User.Role)
	}
}

func TestUsersHandler_Create_WeakPassword(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "weak",
		Role:     "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() with weak password status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeWeakPassword {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeWeakPassword)
	}
}

func TestUsersHandler_Create_InvalidEmail(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "newuser",
		Email:    "invalid-email",
		Password: "NewUser123",
		Role:     "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() with invalid email status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeInvalidEmailFormat {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeInvalidEmailFormat)
	}
}

func TestUsersHandler_Create_InvalidRole(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "NewUser123",
		Role:     "invalid",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
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

func TestUsersHandler_Create_DuplicateUsername(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "admin", // Already exists
		Email:    "new@example.com",
		Password: "NewUser123",
		Role:     "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Create() with duplicate username status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeUsernameExists {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeUsernameExists)
	}
}

func TestUsersHandler_Create_DuplicateEmail(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	body := CreateUserRequest{
		Username: "newuser",
		Email:    "admin@example.com", // Already exists
		Password: "NewUser123",
		Role:     "user",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Create() with duplicate email status = %d, want %d", w.Code, http.StatusConflict)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeEmailExists {
		t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeEmailExists)
	}
}

func TestUsersHandler_Update_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user to update
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	newEmail := "updated@example.com"
	body := UpdateUserRequest{
		Email: &newEmail,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp UpdateUserResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.User.Email != "updated@example.com" {
		t.Errorf("Update() email = %v, want updated@example.com", resp.User.Email)
	}
}

func TestUsersHandler_Update_ResetPassword(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	body := UpdateUserRequest{
		Action:      "reset_password",
		NewPassword: "NewPass456",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() with reset_password status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify new password works
	updatedUser, _ := userRepo.GetByID(ctx, testUser.ID)
	if err := auth.ComparePassword(updatedUser.PasswordHash, "NewPass456"); err != nil {
		t.Error("Update() password reset didn't work")
	}
}

func TestUsersHandler_Update_RevokeSessions(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	body := UpdateUserRequest{
		Action: "revoke_sessions",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() with revoke_sessions status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUsersHandler_Update_CannotModifySelf(t *testing.T) {
	handler, admin, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	newEmail := "newemail@example.com"
	body := UpdateUserRequest{
		Email: &newEmail,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+admin.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Update() on self status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeCannotModifySelf {
		t.Errorf("Update() error_code = %v, want %v", resp["error_code"], ErrCodeCannotModifySelf)
	}
}

func TestUsersHandler_Update_CannotDowngradeLastAdmin(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a second admin to test downgrade
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	secondAdmin := &auth.User{
		Username:     "admin2",
		Email:        "admin2@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleAdmin),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, secondAdmin); err != nil {
		t.Fatalf("failed to create second admin: %v", err)
	}

	// First, downgrade second admin (should succeed, since there's still one admin)
	userRole := "user"
	body := UpdateUserRequest{
		Role: &userRole,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+secondAdmin.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update() downgrade second admin status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUsersHandler_Destroy_Success(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user to delete
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/users:destroy?id="+testUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Destroy() status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user is deleted
	deletedUser, _ := userRepo.GetByID(ctx, testUser.ID)
	if deletedUser != nil {
		t.Error("Destroy() user should be deleted")
	}
}

func TestUsersHandler_Destroy_CannotDeleteSelf(t *testing.T) {
	handler, admin, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/users:destroy?id="+admin.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Destroy() on self status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error_code"] != ErrCodeCannotModifySelf {
		t.Errorf("Destroy() error_code = %v, want %v", resp["error_code"], ErrCodeCannotModifySelf)
	}
}

func TestUsersHandler_Destroy_CannotDeleteLastAdmin(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create another admin
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	secondAdmin := &auth.User{
		Username:     "admin2",
		Email:        "admin2@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleAdmin),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, secondAdmin); err != nil {
		t.Fatalf("failed to create second admin: %v", err)
	}

	// Delete second admin (should succeed)
	req := httptest.NewRequest(http.MethodPost, "/users:destroy?id="+secondAdmin.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Destroy() second admin status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUsersHandler_Destroy_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/users:destroy?id=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Destroy() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUsersHandler_Destroy_MissingID(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/users:destroy", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Destroy() without id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUsersHandler_Create_MissingFields(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	tests := []struct {
		name string
		body CreateUserRequest
	}{
		{"missing username", CreateUserRequest{Email: "a@b.com", Password: "Pass123!", Role: "user"}},
		{"missing email", CreateUserRequest{Username: "user", Password: "Pass123!", Role: "user"}},
		{"missing password", CreateUserRequest{Username: "user", Email: "a@b.com", Role: "user"}},
		{"missing role", CreateUserRequest{Username: "user", Email: "a@b.com", Password: "Pass123!"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/users:create", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Create() with %s status = %d, want %d", tt.name, w.Code, http.StatusBadRequest)
			}

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			if resp["error_code"] != ErrCodeMissingRequiredField {
				t.Errorf("Create() error_code = %v, want %v", resp["error_code"], ErrCodeMissingRequiredField)
			}
		})
	}
}

func TestNewUsersHandler(t *testing.T) {
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

	handler := NewUsersHandler(db, "secret", 3600, 604800)
	if handler == nil {
		t.Error("NewUsersHandler() returned nil")
	}
	if handler.userRepo == nil {
		t.Error("NewUsersHandler() userRepo is nil")
	}
	if handler.tokenRepo == nil {
		t.Error("NewUsersHandler() tokenRepo is nil")
	}
	if handler.tokenService == nil {
		t.Error("NewUsersHandler() tokenService is nil")
	}
	if handler.passwordPolicy == nil {
		t.Error("NewUsersHandler() passwordPolicy is nil")
	}
}

func TestUsersHandler_List_WrongMethod(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/users:list", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("List() with POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUsersHandler_Get_WrongMethod(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/users:get?id=123", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Get() with POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUsersHandler_Create_WrongMethod(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:create", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Create() with GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUsersHandler_Update_WrongMethod(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:update?id=123", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Update() with GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUsersHandler_Destroy_WrongMethod(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/users:destroy?id=123", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	handler.Destroy(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Destroy() with GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUsersHandler_Update_NotFound(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	newEmail := "test@example.com"
	body := UpdateUserRequest{
		Email: &newEmail,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id=nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Update() with nonexistent id status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUsersHandler_Update_InvalidAction(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	body := UpdateUserRequest{
		Action: "invalid_action",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() with invalid action status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUsersHandler_Update_ResetPasswordMissingPassword(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	body := UpdateUserRequest{
		Action: "reset_password",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() reset_password without password status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUsersHandler_Update_NoFieldsToUpdate(t *testing.T) {
	handler, _, adminToken, db := setupTestUsersHandler(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user
	passwordHash, _ := auth.HashPassword("TestPass123")
	userRepo := auth.NewUserRepository(db)
	testUser := &auth.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         string(auth.RoleUser),
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	body := UpdateUserRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users:update?id="+testUser.ID, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Update() with no fields status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserToPublicInfo(t *testing.T) {
	user := &auth.User{
		ID:       "01H1234567890ABCDEFGHJKMNP",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "admin",
		CanWrite: true,
	}

	info := userToPublicInfo(user)

	if info.ID != user.ID {
		t.Errorf("userToPublicInfo() ID = %s, want %s", info.ID, user.ID)
	}
	if info.Username != user.Username {
		t.Errorf("userToPublicInfo() Username = %s, want %s", info.Username, user.Username)
	}
	if info.Email != user.Email {
		t.Errorf("userToPublicInfo() Email = %s, want %s", info.Email, user.Email)
	}
	if info.Role != user.Role {
		t.Errorf("userToPublicInfo() Role = %s, want %s", info.Role, user.Role)
	}
	if info.CanWrite != user.CanWrite {
		t.Errorf("userToPublicInfo() CanWrite = %v, want %v", info.CanWrite, user.CanWrite)
	}
}

func TestParseIntWithDefault(t *testing.T) {
	tests := []struct {
		input    string
		defVal   int
		expected int
	}{
		{"10", 5, 10},
		{"", 5, 5},
		{"abc", 5, 5},
		{"100", 5, 100},
		{"0", 5, 0},
	}

	for _, tt := range tests {
		result := parseIntWithDefault(tt.input, tt.defVal)
		if result != tt.expected {
			t.Errorf("parseIntWithDefault(%q, %d) = %d, want %d", tt.input, tt.defVal, result, tt.expected)
		}
	}
}
