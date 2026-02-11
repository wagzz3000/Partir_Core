package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/partir/core/internal/security/secrets"
)

// JWTAuthenticator implements Authenticator using HMAC-SHA256
type JWTAuthenticator struct {
	secretKey []byte
}

// NewJWTAuthenticator creates a new JWT authenticator
func NewJWTAuthenticator(secretProvider secrets.SecretProvider) (*JWTAuthenticator, error) {
	key, err := secretProvider.Get(context.Background(), "PARTIR_JWT_SECRET")
	if err != nil {
		// Fallback for dev? Or strictly fail?
		// For security critical, we should fail or have strict default.
		// Let's assume dev default 'dev-secret' if missing in non-prod?
		// Better to error and let main handle fallback.
		return nil, fmt.Errorf("failed to get JWT secret: %w", err)
	}
	return &JWTAuthenticator{secretKey: []byte(key)}, nil
}

// Authenticate validates a token and returns the claims
func (a *JWTAuthenticator) Authenticate(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// GenerateToken creates a new token for the given claims
func (a *JWTAuthenticator) GenerateToken(ctx context.Context, claims Claims) (string, error) {
	// Set default expiration if not provided
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	}
	// Set issued at
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(time.Now())
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secretKey)
}
