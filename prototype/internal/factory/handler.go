package factory

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/partir/core/internal/security/auth"
)

type Handler struct {
	Registry      *Registry
	Authenticator auth.Authenticator
}

func NewHandler(r *Registry, a auth.Authenticator) *Handler {
	return &Handler{
		Registry:      r,
		Authenticator: a,
	}
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := h.Registry.GetFactoryStatus(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Add derived metrics here if needed (e.g. Drain Time T)
	json.NewEncoder(w).Encode(status)
}

func (h *Handler) GetWorkers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workers, err := h.Registry.ListWorkers(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(workers)
}

func (h *Handler) GetDMAIC(w http.ResponseWriter, r *http.Request) {
	// Mock DMAIC stats for now, pending real implementation in Registry
	stats := map[string]interface{}{
		"define":  12,   // Active Scopes
		"measure": 85.5, // Throughput
		"analyze": 3,    // Top Defects
		"improve": 4,    // Fix-it tickets
		"control": 98.2, // Gate Pass Rate
	}
	json.NewEncoder(w).Encode(stats)
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

func (h *Handler) PostChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Connect this to the actual LLM/Factory Manager
	// For V0 Wiring, we echo back or give a standard response
	resp := ChatResponse{
		Response: "Factory Manager received: " + req.Message,
	}

	json.NewEncoder(w).Encode(resp)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Bootstrap credentials from ENV
	adminUser := os.Getenv("PARTIR_ADMIN_USER")
	adminPass := os.Getenv("PARTIR_ADMIN_PASS")

	// Fallback to defaults if not set (for dev/easy start)
	if adminUser == "" {
		adminUser = "admin"
	}
	if adminPass == "" {
		adminPass = "admin"
	}

	if req.Username != adminUser || req.Password != adminPass {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	claims := auth.Claims{
		UserID:   req.Username,
		TenantID: "default",
		Role:     auth.RoleAdmin,
	}

	token, err := h.Authenticator.GenerateToken(r.Context(), claims)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
