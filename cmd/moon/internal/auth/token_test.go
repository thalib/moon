package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewTokenService(t *testing.T) {
	svc := NewTokenService("secret", 3600, 604800)

	if svc == nil {
		t.Fatal("NewTokenService returned nil")
	}

	if svc.AccessExpiry() != time.Hour {
		t.Errorf("AccessExpiry() = %v, want %v", svc.AccessExpiry(), time.Hour)
	}

	if svc.RefreshExpiry() != 7*24*time.Hour {
		t.Errorf("RefreshExpiry() = %v, want %v", svc.RefreshExpiry(), 7*24*time.Hour)
	}
}

func TestTokenService_GenerateTokenPair(t *testing.T) {
	svc := NewTokenService("test-secret-key-123", 3600, 604800)

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	pair, refreshToken, err := svc.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}

	if pair.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}

	if refreshToken == "" {
		t.Error("raw refreshToken is empty")
	}

	if pair.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", pair.TokenType, "Bearer")
	}

	if pair.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt is in the past")
	}

	// Verify tokens are different
	if pair.AccessToken == pair.RefreshToken {
		t.Error("AccessToken equals RefreshToken")
	}
}

func TestTokenService_ValidateAccessToken(t *testing.T) {
	svc := NewTokenService("test-secret-key-123", 3600, 604800)

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	pair, _, err := svc.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", claims.UserID, user.ID)
	}

	if claims.Username != user.Username {
		t.Errorf("Username = %q, want %q", claims.Username, user.Username)
	}

	if claims.Role != user.Role {
		t.Errorf("Role = %q, want %q", claims.Role, user.Role)
	}

	if claims.CanWrite != user.CanWrite {
		t.Errorf("CanWrite = %v, want %v", claims.CanWrite, user.CanWrite)
	}
}

func TestTokenService_ValidateAccessToken_InvalidToken(t *testing.T) {
	svc := NewTokenService("test-secret-key-123", 3600, 604800)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "not.a.token"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.Rq8IjqbeYcOoK1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateAccessToken(tt.token)
			if err == nil {
				t.Error("ValidateAccessToken() expected error, got nil")
			}
		})
	}
}

func TestTokenService_ValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewTokenService("secret-1", 3600, 604800)
	svc2 := NewTokenService("secret-2", 3600, 604800)

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	pair, _, err := svc1.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	_, err = svc2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Error("ValidateAccessToken() expected error for wrong secret")
	}
}

func TestTokenService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a service with very short expiry
	svc := NewTokenService("test-secret", -1, 604800) // negative expiry = already expired

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	pair, _, err := svc.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	_, err = svc.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Error("ValidateAccessToken() expected error for expired token")
	}
}

func TestTokenService_GenerateRefreshToken_Uniqueness(t *testing.T) {
	svc := NewTokenService("secret", 3600, 604800)

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pair, _, err := svc.GenerateTokenPair(user)
		if err != nil {
			t.Fatalf("GenerateTokenPair() error = %v", err)
		}

		if tokens[pair.RefreshToken] {
			t.Error("Generated duplicate refresh token")
		}
		tokens[pair.RefreshToken] = true
	}
}

func TestClaims_RegisteredClaims(t *testing.T) {
	svc := NewTokenService("test-secret", 3600, 604800)

	user := &User{
		ID:       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Username: "testuser",
		Role:     "admin",
		CanWrite: true,
	}

	pair, _, err := svc.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	// Check subject is set to user ID
	if claims.Subject != user.ID {
		t.Errorf("Subject = %q, want %q", claims.Subject, user.ID)
	}

	// Check issued at is set
	if claims.IssuedAt == nil {
		t.Error("IssuedAt is nil")
	}

	// Check expires at is set
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt is nil")
	}
}

func TestTokenPair_Structure(t *testing.T) {
	pair := &TokenPair{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		TokenType:    "Bearer",
	}

	if pair.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if pair.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", pair.TokenType, "Bearer")
	}
}

func TestTokenService_generateRefreshToken(t *testing.T) {
	svc := NewTokenService("secret", 3600, 604800)

	token, err := svc.generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken() error = %v", err)
	}

	// Token should be base64 URL encoded
	if strings.ContainsAny(token, "+/") {
		t.Error("Token contains non-URL-safe characters")
	}

	// Token should have reasonable length
	if len(token) < 32 {
		t.Errorf("Token length = %d, want >= 32", len(token))
	}
}
