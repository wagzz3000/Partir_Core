// Package metrics - HTTP server for exposing /metrics, /health, /ready endpoints
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ReadinessCheck is a function that returns an error if the component is not ready
type ReadinessCheck func() error

// Server provides an HTTP server for Prometheus metrics and health probes
type Server struct {
	server          *http.Server
	addr            string
	mu              sync.RWMutex
	readinessChecks map[string]ReadinessCheck
	startTime       time.Time
}

// NewServer creates a new metrics server
func NewServer(port int) *Server {
	s := &Server{
		addr:            fmt.Sprintf(":%d", port),
		readinessChecks: make(map[string]ReadinessCheck),
		startTime:       time.Now(),
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	return s
}

// RegisterReadinessCheck adds a named readiness check
func (s *Server) RegisterReadinessCheck(name string, check ReadinessCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readinessChecks[name] = check
}

// healthHandler returns liveness status (always OK if process is running)
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(s.startTime).String(),
	})
}

// readyHandler checks all registered readiness probes
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	checks := make(map[string]ReadinessCheck, len(s.readinessChecks))
	for k, v := range s.readinessChecks {
		checks[k] = v
	}
	s.mu.RUnlock()

	results := make(map[string]string)
	allReady := true

	for name, check := range checks {
		if err := check(); err != nil {
			results[name] = err.Error()
			allReady = false
		} else {
			results[name] = "ok"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allReady {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":  allReady,
		"checks": results,
	})
}

// Start begins serving metrics
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// StartAsync starts the server in a goroutine
func (s *Server) StartAsync() {
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()
	// Give server time to start
	time.Sleep(10 * time.Millisecond)
}
