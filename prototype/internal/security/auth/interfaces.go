package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpired      = errors.New("token expired")
)

// UserRole defines the role of the user (viewer, editor, admin)
type UserRole string

const (
	RoleViewer UserRole = "viewer"
	RoleEditor UserRole = "editor"
	RoleAdmin  UserRole = "admin"
)

// Claims extends standard JWT claims with Partir specific fields
type Claims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Role     UserRole `json:"role"`
}

// Authenticator defines the interface for authentication
type Authenticator interface {
	// Authenticate validates a token and returns the claims
	Authenticate(ctx context.Context, tokenString string) (*Claims, error)

	// GenerateToken creates a new token for the given claims
	GenerateToken(ctx context.Context, claims Claims) (string, error)
}
