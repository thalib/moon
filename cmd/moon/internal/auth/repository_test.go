package auth

import (
	"context"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/database"
)

func setupTestDB(t *testing.T) database.Driver {
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

	// Initialize schema
	if err := Bootstrap(ctx, db, nil); err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID == 0 {
		t.Error("Create() did not set user ID")
	}
	if user.ULID == "" {
		t.Error("Create() did not set user ULID")
	}
	if user.CreatedAt.IsZero() {
		t.Error("Create() did not set CreatedAt")
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Test GetByUsername
	found, err := repo.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByUsername() returned nil")
	}
	if found.Username != "testuser" {
		t.Errorf("GetByUsername() username = %q, want %q", found.Username, "testuser")
	}

	// Test not found
	notFound, err := repo.GetByUsername(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if notFound != nil {
		t.Error("GetByUsername() should return nil for nonexistent user")
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByEmail() returned nil")
	}
	if found.Email != "test@example.com" {
		t.Errorf("GetByEmail() email = %q, want %q", found.Email, "test@example.com")
	}
}

func TestUserRepository_GetByULID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.GetByULID(ctx, user.ULID)
	if err != nil {
		t.Fatalf("GetByULID() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByULID() returned nil")
	}
	if found.ULID != user.ULID {
		t.Errorf("GetByULID() ulid = %q, want %q", found.ULID, user.ULID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update email
	user.Email = "updated@example.com"
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(ctx, user.ID)
	if found.Email != "updated@example.com" {
		t.Errorf("Update() email = %q, want %q", found.Email, "updated@example.com")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	found, _ := repo.GetByID(ctx, user.ID)
	if found != nil {
		t.Error("Delete() did not remove user")
	}
}

func TestUserRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Initially zero
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Create a user
	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}

func TestUserRepository_Exists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		username string
		email    string
		want     bool
	}{
		{"testuser", "other@example.com", true},
		{"other", "test@example.com", true},
		{"testuser", "test@example.com", true},
		{"other", "other@example.com", false},
	}

	for _, tt := range tests {
		exists, err := repo.Exists(ctx, tt.username, tt.email)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists != tt.want {
			t.Errorf("Exists(%q, %q) = %v, want %v", tt.username, tt.email, exists, tt.want)
		}
	}
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	token := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("refresh-token-123"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if token.ID == 0 {
		t.Error("Create() did not set token ID")
	}
}

func TestRefreshTokenRepository_GetByHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	tokenHash := HashToken("refresh-token-123")
	token := &RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := tokenRepo.Create(ctx, token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetByHash() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByHash() returned nil")
	}
	if found.TokenHash != tokenHash {
		t.Errorf("GetByHash() hash mismatch")
	}
}

func TestRefreshTokenRepository_DeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Role:         "admin",
		CanWrite:     true,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	// Create multiple tokens
	for i := 0; i < 3; i++ {
		token := &RefreshToken{
			UserID:    user.ID,
			TokenHash: HashToken("token-" + string(rune('0'+i))),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, token); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := tokenRepo.DeleteByUserID(ctx, user.ID); err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}

	// Verify all tokens deleted
	for i := 0; i < 3; i++ {
		found, _ := tokenRepo.GetByHash(ctx, HashToken("token-"+string(rune('0'+i))))
		if found != nil {
			t.Error("DeleteByUserID() did not delete all tokens")
		}
	}
}

func TestAPIKeyRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:        "Test Key",
		Description: "A test API key",
		KeyHash:     keyHash,
		Role:        "admin",
		CanWrite:    true,
	}

	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if apiKey.ID == 0 {
		t.Error("Create() did not set ID")
	}
	if apiKey.ULID == "" {
		t.Error("Create() did not set ULID")
	}
}

func TestAPIKeyRepository_GetByHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	rawKey, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:        "Test Key",
		Description: "A test API key",
		KeyHash:     keyHash,
		Role:        "admin",
		CanWrite:    true,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Test GetByHash
	found, err := repo.GetByHash(ctx, HashAPIKey(rawKey))
	if err != nil {
		t.Fatalf("GetByHash() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByHash() returned nil")
	}
	if found.Name != "Test Key" {
		t.Errorf("GetByHash() name = %q, want %q", found.Name, "Test Key")
	}
}

func TestAPIKeyRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	// Create a few API keys
	for i := 0; i < 3; i++ {
		_, keyHash, _ := GenerateAPIKey()
		apiKey := &APIKey{
			Name:     "Test Key " + string(rune('A'+i)),
			KeyHash:  keyHash,
			Role:     "user",
			CanWrite: true,
		}
		if err := repo.Create(ctx, apiKey); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	keys, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("List() returned %d keys, want 3", len(keys))
	}
}

func TestAPIKeyRepository_ListPaginated(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	// Create 5 API keys
	var createdKeys []*APIKey
	for i := 0; i < 5; i++ {
		_, keyHash, _ := GenerateAPIKey()
		apiKey := &APIKey{
			Name:     "Test Key " + string(rune('A'+i)),
			KeyHash:  keyHash,
			Role:     "user",
			CanWrite: true,
		}
		if err := repo.Create(ctx, apiKey); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		createdKeys = append(createdKeys, apiKey)
	}

	// Test pagination with limit
	keys, err := repo.ListPaginated(ctx, APIKeyListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("ListPaginated() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListPaginated() with limit 2 returned %d keys, want 2", len(keys))
	}

	// Test pagination with after cursor
	keys, err = repo.ListPaginated(ctx, APIKeyListOptions{Limit: 2, AfterULID: keys[1].ULID})
	if err != nil {
		t.Fatalf("ListPaginated() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListPaginated() after cursor returned %d keys, want 2", len(keys))
	}
}

func TestAPIKeyRepository_NameExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:     "Unique Name",
		KeyHash:  keyHash,
		Role:     "user",
		CanWrite: true,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Test existing name
	exists, err := repo.NameExists(ctx, "Unique Name", 0)
	if err != nil {
		t.Fatalf("NameExists() error = %v", err)
	}
	if !exists {
		t.Error("NameExists() should return true for existing name")
	}

	// Test non-existing name
	exists, err = repo.NameExists(ctx, "Non Existing", 0)
	if err != nil {
		t.Fatalf("NameExists() error = %v", err)
	}
	if exists {
		t.Error("NameExists() should return false for non-existing name")
	}

	// Test excluding own ID
	exists, err = repo.NameExists(ctx, "Unique Name", apiKey.ID)
	if err != nil {
		t.Fatalf("NameExists() error = %v", err)
	}
	if exists {
		t.Error("NameExists() should return false when excluding own ID")
	}
}

func TestAPIKeyRepository_UpdateKeyHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:     "Rotate Test Key",
		KeyHash:  keyHash,
		Role:     "user",
		CanWrite: true,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Generate new key hash
	_, newKeyHash, _ := GenerateAPIKey()

	// Update the hash
	if err := repo.UpdateKeyHash(ctx, apiKey.ID, newKeyHash); err != nil {
		t.Fatalf("UpdateKeyHash() error = %v", err)
	}

	// Verify old hash doesn't work
	found, _ := repo.GetByHash(ctx, keyHash)
	if found != nil {
		t.Error("UpdateKeyHash() old hash should not work")
	}

	// Verify new hash works
	found, err := repo.GetByHash(ctx, newKeyHash)
	if err != nil {
		t.Fatalf("GetByHash() error = %v", err)
	}
	if found == nil {
		t.Fatal("UpdateKeyHash() new hash should work")
	}
	if found.ID != apiKey.ID {
		t.Error("UpdateKeyHash() should return same key")
	}
}

func TestAPIKeyRepository_UpdateMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:        "Original Name",
		Description: "Original Description",
		KeyHash:     keyHash,
		Role:        "user",
		CanWrite:    false,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update metadata
	apiKey.Name = "Updated Name"
	apiKey.Description = "Updated Description"
	apiKey.CanWrite = true

	if err := repo.UpdateMetadata(ctx, apiKey); err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	// Verify updates
	found, err := repo.GetByID(ctx, apiKey.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByID() returned nil")
	}
	if found.Name != "Updated Name" {
		t.Errorf("UpdateMetadata() name = %q, want %q", found.Name, "Updated Name")
	}
	if found.Description != "Updated Description" {
		t.Errorf("UpdateMetadata() description = %q, want %q", found.Description, "Updated Description")
	}
	if !found.CanWrite {
		t.Error("UpdateMetadata() can_write should be true")
	}
}

func TestAPIKeyRepository_GetByULID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:     "ULID Test Key",
		KeyHash:  keyHash,
		Role:     "admin",
		CanWrite: true,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Test GetByULID
	found, err := repo.GetByULID(ctx, apiKey.ULID)
	if err != nil {
		t.Fatalf("GetByULID() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByULID() returned nil")
	}
	if found.ULID != apiKey.ULID {
		t.Errorf("GetByULID() ulid = %q, want %q", found.ULID, apiKey.ULID)
	}

	// Test not found
	notFound, err := repo.GetByULID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByULID() error = %v", err)
	}
	if notFound != nil {
		t.Error("GetByULID() should return nil for nonexistent ULID")
	}
}

func TestAPIKeyRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, keyHash, _ := GenerateAPIKey()
	apiKey := &APIKey{
		Name:     "Delete Test Key",
		KeyHash:  keyHash,
		Role:     "user",
		CanWrite: false,
	}
	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, apiKey.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	found, _ := repo.GetByID(ctx, apiKey.ID)
	if found != nil {
		t.Error("Delete() did not remove API key")
	}
}

func TestBootstrap_WithAdmin(t *testing.T) {
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

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	bootstrapCfg := &BootstrapConfig{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "adminpassword123",
	}

	if err := Bootstrap(ctx, db, bootstrapCfg); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Verify admin was created
	repo := NewUserRepository(db)
	user, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if user == nil {
		t.Fatal("Bootstrap admin was not created")
	}
	if user.Role != string(RoleAdmin) {
		t.Errorf("Bootstrap admin role = %q, want %q", user.Role, string(RoleAdmin))
	}

	// Verify password works
	if err := ComparePassword(user.PasswordHash, "adminpassword123"); err != nil {
		t.Error("Bootstrap admin password doesn't match")
	}
}

func TestBootstrap_SkipsIfUsersExist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a user first
	repo := NewUserRepository(db)
	user := &User{
		Username:     "existinguser",
		Email:        "existing@example.com",
		PasswordHash: "hash123",
		Role:         "user",
		CanWrite:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Try to bootstrap - should skip
	bootstrapCfg := &BootstrapConfig{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "adminpassword123",
	}

	if err := Bootstrap(ctx, db, bootstrapCfg); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Verify admin was NOT created
	admin, _ := repo.GetByUsername(ctx, "admin")
	if admin != nil {
		t.Error("Bootstrap should not create admin when users exist")
	}

	// Verify count is still 1
	count, _ := repo.Count(ctx)
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}
