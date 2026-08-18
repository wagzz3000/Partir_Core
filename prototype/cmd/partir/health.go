package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/partir/core/pkg/plugin"
	"github.com/spf13/cobra"
)

// HealthStatus represents the overall health of the system
type HealthStatus struct {
	Status     string                     `json:"status"` // "ok", "degraded", "error"
	Timestamp  string                     `json:"timestamp"`
	Version    string                     `json:"version"`
	Components map[string]ComponentHealth `json:"components"`
}

// ComponentHealth represents a single component's health
type ComponentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of Partir Core and connected services",
	Run: func(cmd *cobra.Command, args []string) {
		status := checkHealth()

		output, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(output))

		if status.Status != "ok" {
			os.Exit(1)
		}
	},
}

var healthzCmd = &cobra.Command{
	Use:   "healthz",
	Short: "Kubernetes-style health endpoint (returns 200 or 503)",
	Run: func(cmd *cobra.Command, args []string) {
		status := checkHealth()

		if status.Status == "ok" {
			fmt.Println("ok")
			os.Exit(0)
		} else {
			fmt.Printf("%s\n", status.Status)
			os.Exit(1)
		}
	},
}

var serveHealthCmd = &cobra.Command{
	Use:   "serve-health",
	Short: "Start HTTP health endpoint server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")

		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			status := checkHealth()
			w.Header().Set("Content-Type", "application/json")

			if status.Status != "ok" {
				w.WriteHeader(http.StatusServiceUnavailable)
			}

			json.NewEncoder(w).Encode(status)
		})

		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			status := checkHealth()

			if status.Status == "ok" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(status.Status))
			}
		})

		http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			// Readiness: can we serve traffic?
			status := checkHealth()

			if status.Status != "error" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ready"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("not ready"))
			}
		})

		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("Health server listening on %s\n", addr)
		fmt.Printf("  /health  - Full health status (JSON)\n")
		fmt.Printf("  /healthz - Liveness probe\n")
		fmt.Printf("  /ready   - Readiness probe\n")

		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting health server: %v\n", err)
			os.Exit(1)
		}
	},
}

func checkHealth() *HealthStatus {
	status := &HealthStatus{
		Status:     "ok",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Version:    "v0.1.0",
		Components: make(map[string]ComponentHealth),
	}

	// Check executors
	cfg, err := plugin.LoadConfig()
	if err != nil {
		status.Components["config"] = ComponentHealth{
			Status:  "error",
			Message: err.Error(),
		}
		status.Status = "degraded"
	} else {
		status.Components["config"] = ComponentHealth{Status: "ok"}

		// Check each executor
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for name, execCfg := range cfg.Executors {
			client, err := plugin.NewHTTPClientFromConfig(name, execCfg)
			if err != nil {
				status.Components["executor:"+name] = ComponentHealth{
					Status:  "error",
					Message: err.Error(),
				}
				status.Status = "degraded"
				continue
			}

			health, err := client.Health(ctx)
			if err != nil {
				status.Components["executor:"+name] = ComponentHealth{
					Status:  "error",
					Message: err.Error(),
				}
				status.Status = "degraded"
			} else {
				status.Components["executor:"+name] = ComponentHealth{
					Status:  health.Status,
					Message: health.Message,
				}
				if health.Status != "ok" {
					status.Status = "degraded"
				}
			}
		}
	}

	// If any component is in error state, mark overall as error
	for _, comp := range status.Components {
		if comp.Status == "error" {
			status.Status = "error"
			break
		}
	}

	return status
}

func init() {
	serveHealthCmd.Flags().Int("port", 8080, "Port to listen on")

	healthCmd.AddCommand(serveHealthCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(healthzCmd)
}
