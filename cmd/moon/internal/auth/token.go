package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/thalib/moon/cmd/moon/internal/constants"
)

// TokenPair contains access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Claims represents the JWT claims for access tokens.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	CanWrite bool   `json:"can_write"`
	jwt.RegisteredClaims
}

// TokenService handles JWT token generation and validation.
type TokenService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewTokenService creates a new token service.
func NewTokenService(secret string, accessExpirySec, refreshExpirySec int) *TokenService {
	return &TokenService{
		secret:        []byte(secret),
		accessExpiry:  time.Duration(accessExpirySec) * time.Second,
		refreshExpiry: time.Duration(refreshExpirySec) * time.Second,
	}
}

// GenerateTokenPair generates both access and refresh tokens for a user.
func (s *TokenService) GenerateTokenPair(user *User) (*TokenPair, string, error) {
	accessToken, expiresAt, err := s.generateAccessToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, refreshToken, nil
}

// generateAccessToken generates a new JWT access token.
func (s *TokenService) generateAccessToken(user *User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessExpiry)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		CanWrite: user.CanWrite,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-constants.JWTClockSkew)),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// generateRefreshToken generates a random refresh token.
func (s *TokenService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateAccessToken validates an access token and returns its claims.
func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshExpiry returns the refresh token expiry duration.
func (s *TokenService) RefreshExpiry() time.Duration {
	return s.refreshExpiry
}

// AccessExpiry returns the access token expiry duration.
func (s *TokenService) AccessExpiry() time.Duration {
	return s.accessExpiry
}
