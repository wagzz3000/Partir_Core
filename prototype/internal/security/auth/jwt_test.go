package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSecretProvider for testing
type MockSecretProvider struct {
	mock.Mock
}

func (m *MockSecretProvider) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func TestJWTAuthenticator(t *testing.T) {
	mockSec := new(MockSecretProvider)
	mockSec.On("Get", mock.Anything, "PARTIR_JWT_SECRET").Return("my-secret-key", nil)

	auth, err := NewJWTAuthenticator(mockSec)
	assert.NoError(t, err)

	ctx := context.Background()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UserID:   "user-1",
		TenantID: "tenant-a",
		Role:     RoleEditor,
	}

	// Generate
	token, err := auth.GenerateToken(ctx, claims)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Authenticate
	parsedClaims, err := auth.Authenticate(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", parsedClaims.UserID)
	assert.Equal(t, RoleEditor, parsedClaims.Role)
}

func TestJWTAuthenticator_Expired(t *testing.T) {
	mockSec := new(MockSecretProvider)
	mockSec.On("Get", mock.Anything, "PARTIR_JWT_SECRET").Return("my-secret-key", nil)

	auth, _ := NewJWTAuthenticator(mockSec)

	// Generate expired token
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
		UserID: "user-1",
	}
	token, _ := auth.GenerateToken(context.Background(), claims)

	// Authenticate
	_, err := auth.Authenticate(context.Background(), token)
	assert.Error(t, err) // Should be error, possibly "token has invalid claims: token is expired"
}
