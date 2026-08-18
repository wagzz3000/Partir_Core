package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/partir/core/internal/security/auth"
	"github.com/stretchr/testify/assert"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		userRole       auth.UserRole
		requiredRole   auth.UserRole
		expectedStatus int
	}{
		{"Admin accesses Admin", auth.RoleAdmin, auth.RoleAdmin, http.StatusOK},
		{"Admin accesses Editor", auth.RoleAdmin, auth.RoleEditor, http.StatusOK},
		{"Admin accesses Viewer", auth.RoleAdmin, auth.RoleViewer, http.StatusOK},

		{"Editor accesses Admin", auth.RoleEditor, auth.RoleAdmin, http.StatusForbidden},
		{"Editor accesses Editor", auth.RoleEditor, auth.RoleEditor, http.StatusOK},
		{"Editor accesses Viewer", auth.RoleEditor, auth.RoleViewer, http.StatusOK},

		{"Viewer accesses Admin", auth.RoleViewer, auth.RoleAdmin, http.StatusForbidden},
		{"Viewer accesses Editor", auth.RoleViewer, auth.RoleEditor, http.StatusForbidden},
		{"Viewer accesses Viewer", auth.RoleViewer, auth.RoleViewer, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Middleware
			handler := RequireRole(tt.requiredRole)(nextHandler)

			// Request with claims
			req := httptest.NewRequest("GET", "/", nil)
			claims := &auth.Claims{Role: tt.userRole}
			ctx := context.WithValue(req.Context(), UserContextKey, claims)
			req = req.WithContext(ctx)

			// Recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestRequireRole_NoClaims(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireRole(auth.RoleViewer)(nextHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
