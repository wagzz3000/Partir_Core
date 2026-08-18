package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/partir/core/internal/factory"
	"github.com/partir/core/internal/security/auth"
	"github.com/partir/core/internal/security/secrets"
	"github.com/partir/core/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Partir API Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")

		// 1. Initialize Secrets & Config (Chained: Docker Secrets -> Env)
		sec := secrets.NewMultiProvider(
			secrets.NewFileProvider("/run/secrets"),
			secrets.NewEnvProvider(),
		)
		ctx := context.Background()

		dbURL, err := sec.Get(ctx, "PARTIR_DB_URL")
		if err != nil {
			return fmt.Errorf("failed to get DB URL: %w", err)
		}

		// 2. Connect to Database (Non-fatal for bootstrap)
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			fmt.Printf("WARNING: failed to connect to database: %v. Continuing for non-DB endpoints...\n", err)
		} else {
			if err := pool.Ping(ctx); err != nil {
				fmt.Printf("WARNING: failed to ping database: %v. Continuing for non-DB endpoints...\n", err)
			} else {
				fmt.Println("Successfully connected to database.")
			}
		}
		defer func() {
			if pool != nil {
				pool.Close()
			}
		}()

		// 3. Initialize Factory Registry
		registry := factory.NewRegistry(pool)

		// 4. Initialize Authenticator
		authenticator, err := auth.NewJWTAuthenticator(sec)
		if err != nil {
			return fmt.Errorf("failed to initialize authenticator: %w", err)
		}

		// 5. Start Server
		srv := server.NewServer(port, registry, authenticator)

		go func() {
			if err := srv.Start(); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}
		}()

		// 5. Wait for shutdown signal
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Stop(shutdownCtx)
	},
}

func init() {
	serverCmd.Flags().Int("port", 8080, "Port to listen on")
	rootCmd.AddCommand(serverCmd)
}
