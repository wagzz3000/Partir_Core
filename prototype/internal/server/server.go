package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/partir/core/internal/factory"
	"github.com/partir/core/internal/security/auth"
	"github.com/partir/core/internal/security/middleware"
)

type Server struct {
	httpServer *http.Server
	registry   *factory.Registry
}

func NewServer(port int, registry *factory.Registry, authenticator auth.Authenticator) *Server {
	mux := http.NewServeMux()

	// 1. Public Routes
	h := factory.NewHandler(registry, authenticator)
	mux.HandleFunc("/api/login", h.Login)

	// 2. Factory API (Protected)
	factoryMux := http.NewServeMux()
	factoryMux.HandleFunc("/api/factory/status", h.GetStatus)
	factoryMux.HandleFunc("/api/factory/workers", h.GetWorkers)
	factoryMux.HandleFunc("/api/factory/dmaic", h.GetDMAIC)
	factoryMux.HandleFunc("/api/factory/chat", h.PostChat)

	// Wrap factory routes with AuthMiddleware
	authMW := middleware.AuthMiddleware(authenticator)
	mux.Handle("/api/factory/", authMW(factoryMux))

	// Health Checks
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ready"))
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		registry: registry,
	}
}

func (s *Server) Start() error {
	fmt.Printf("Partir API Server listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
