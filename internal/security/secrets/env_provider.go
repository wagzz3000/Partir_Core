package secrets

import (
	"context"
	"fmt"
	"os"
)

// EnvProvider retrieves secrets from environment variables
type EnvProvider struct{}

// NewEnvProvider creates a new environment variable secret provider
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Get supports retrieving secrets from env vars.
// It returns error if the env var is not set (empty string is considered not set for secrets context usually, or valid depending on policy).
// Here we treat missing as error to be explicit.
func (p *EnvProvider) Get(ctx context.Context, key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("secret not found: %s", key)
	}
	return val, nil
}
