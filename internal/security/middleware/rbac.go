package middleware

import (
	"net/http"

	"github.com/partir/core/internal/security/auth"
)

// RequireRole creates a middleware that enforces a minimum role
func RequireRole(requiredRole auth.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !hasRole(claims.Role, requiredRole) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hasRole checks if userRole meets requiredRole.
// Hierarchy: Admin > Editor > Viewer
func hasRole(userRole, requiredRole auth.UserRole) bool {
	if userRole == auth.RoleAdmin {
		return true
	}
	if requiredRole == auth.RoleAdmin {
		return false
	}

	if userRole == auth.RoleEditor {
		return true
	}
	if requiredRole == auth.RoleEditor {
		return false
	}

	return true // Both are Viewer (or user is Viewer and required is Viewer)
}
