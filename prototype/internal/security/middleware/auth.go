package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/partir/core/internal/security/auth"
)

type contextKey string

const (
	UserContextKey contextKey = "user_claims"
)

// AuthMiddleware creates an HTTP middleware for JWT authentication
func AuthMiddleware(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Printf("[Auth] Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			claims, err := authenticator.Authenticate(r.Context(), tokenString)
			if err != nil {
				log.Printf("[Auth] Token authentication failed: %v", err)
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			log.Printf("[Auth] Authenticated user: %s (Role: %s)", claims.UserID, claims.Role)

			// Add claims to context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves user claims from context
func GetClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	return claims, ok
}
