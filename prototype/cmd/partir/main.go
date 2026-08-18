// partir - Main CLI entry point for Partir Core
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/partir/core/internal/factory"
	"github.com/partir/core/internal/security/secrets"
)

var rootCmd = &cobra.Command{
	Use:   "partir",
	Short: "Partir Core - AI Content Factory",
	Long: `Partir Core is an AI-powered content generation factory.

It provides:
  - Alpha: Data schema and catalog management
  - Beta: Render rules and style management
  - Foundry: Ticket-based content generation
  - Omega: Quality gates and validation
  - Executors: LLM plugin management`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("partir v0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(executorsCmd)
}

func main() {
	// Initialize Secrets
	sec := secrets.NewEnvProvider()
	ctx := context.Background()

	// Initialize factory components if NATS URL is present
	natsURL, _ := sec.Get(ctx, "NATS_URL") // treat as optional, or check err if mandatory? Original code implies optional (if != "").

	if natsURL != "" {
		dbURL, err := sec.Get(ctx, "PARTIR_DB_URL")
		if err != nil {
			dbURL = "postgres://partir:partir@localhost:5432/partir?sslmode=disable"
		}

		ctx := context.Background()
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to DB for factory telemetry: %v\n", err)
		} else {
			if err := pool.Ping(ctx); err != nil {
				fmt.Printf("Warning: Failed to ping DB: %v\n", err)
			} else {
				registry := factory.NewRegistry(pool)
				collector := factory.NewCollector(registry)

				// Initialize metrics
				if err := factory.InitMetrics(); err != nil {
					fmt.Printf("Warning: Failed to init factory metrics: %v\n", err)
				}

				listener, err := factory.NewHeartbeatListener(registry, collector, natsURL)
				if err != nil {
					fmt.Printf("Warning: Failed to start heartbeat listener: %v\n", err)
				} else {
					if err := listener.Start(ctx); err != nil {
						fmt.Printf("Warning: Failed to subscribe to heartbeats: %v\n", err)
					} else {
						// Ensure stations exist
						go func() {
							if err := registry.EnsureStationsExist(ctx); err != nil {
								fmt.Printf("Warning: Failed to ensure stations: %v\n", err)
							}
						}()
						// We don't defer listener.Stop() here because main() runs Execute() below blocks?
						// Cobra execute blocks? No. Cobra Execute runs the command.
						// BUT main() usually exits after Execute().
						// The original code had listener logic inside main BEFORE Execute?
						// Ah, it launches things?
						// The heartbeat listener has a Start method. It likely runs in background.
						// But if main exits, it stops.
						// The original code had `defer listener.Stop()`.
						// It assumes Execute() blocks if it runs a long-running command?
						// Or maybe this logic is ensuring factory services run alongside the CLI tool?
						// "partir" is the CLI tool. Why does it start a heartbeat listener?
						// To act as a factory supervisor perhaps?
						// If user runs `partir factory monitor`, maybe.
						// But for `partir version`, we don't need this.
						// Anyway, preserving logic, just updating DB.
						defer listener.Stop()
					}
				}
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
