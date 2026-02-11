package secrets

import "context"

// SecretProvider defines the interface for retrieving secrets
type SecretProvider interface {
	// Get retrieves a secret by key
	Get(ctx context.Context, key string) (string, error)
}
