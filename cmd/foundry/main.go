// Foundry CLI - The production system
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/partir/core/internal/alpha"
	"github.com/partir/core/internal/beta"
	"github.com/partir/core/internal/factory"
	"github.com/partir/core/internal/foundry"
	"github.com/partir/core/internal/interview"
	"github.com/partir/core/internal/omega"
	"github.com/partir/core/internal/security/secrets"
	"github.com/partir/core/internal/session"
	"github.com/partir/core/internal/storage"
	"github.com/partir/core/pkg/plugin"
	"github.com/spf13/cobra"
)

var (
	configPath string
	dbURL      string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "foundry",
		Short: "Partir Foundry - Production system for executing tickets",
		Long:  `Foundry routes tickets to plugins, runs jobs, and stores artifacts.`,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&dbURL, "db", "postgres://partir:partir@localhost:5432/partir?sslmode=disable", "database URL")

	// Initialize stack for commands that need it
	rootCmd.AddCommand(submitCmd())
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(artifactsCmd())
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(scopeCmd())
	rootCmd.AddCommand(resumeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// FoundryStack holds all the initialized components
type FoundryStack struct {
	Dispatcher *foundry.Dispatcher
	Alpha      *alpha.Architect
	Beta       *beta.Designer
	Planner    *foundry.Planner
	Executor   plugin.Executor
	Store      *storage.Storage
}

func initStack(projectID string) (*FoundryStack, error) {
	// 1. Initialize Secrets
	sec := secrets.NewEnvProvider()
	ctx := context.Background()

	// 1.5 Initialize Storage (handles DB and MinIO connections)
	// We use env vars for MinIO config for now
	minioURL := os.Getenv("PARTIR_MINIO_ENDPOINT")
	if minioURL == "" {
		minioURL = "localhost:9000"
	}

	minioPass, err := sec.Get(ctx, "PARTIR_MINIO_SECRET_KEY")
	if err != nil {
		minioPass = "partir" // Default dev password
	}

	minioUser, err := sec.Get(ctx, "PARTIR_MINIO_ACCESS_KEY")
	if err != nil {
		minioUser = "partir"
	}

	cfg := storage.Config{
		PostgresURL: dbURL,
		MinioURL:    minioURL,
		MinioBucket: "partir-artifacts",
		MinioUser:   minioUser,
		MinioPass:   minioPass,
	}

	store, err := storage.New(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	// 2. We don't need raw DB open anymore as storage handles it,
	// checking health via store.Ping() implicitly happens in New()

	// 3. Initialize Plugins and Executor
	plugins := plugin.NewRegistry()
	// TODO: Load plugins from config/DB properly

	// Create the execution strategy for Foundry (LocalExecutor)
	localExec := foundry.NewLocalExecutor(plugins)

	// Get the LLM Executor (for Alpha/Beta/Planner)
	llmExec := getExecutor()
	// Note: llmExec might be nil if not configured, which is handled by callers

	// 4. Initialize Factory
	fact := factory.NewRegistry(store.Pool)

	// 5. Initialize Omega
	om := omega.NewEngine(store.Pool)

	// 6. Initialize Dispatcher
	dispatcher := foundry.NewDispatcher(store, plugins, fact, om)
	dispatcher.SetExecutor(localExec)

	// 7. Initialize Alpha & Beta (needed for Round Table)
	// We need a dummy assembly for this simple CLI usage
	assembly := &interview.PairAssembly{
		ProjectID: projectID,
		Alpha:     [4]string{"A1", "A2", "A3", "A4"},
		Beta:      [4]string{"B1", "B2", "B3", "B4"},
		Omega:     [4]string{"O1", "O2", "O3", "O4"},
	}

	// Mock workspaces for now - in real usage these come from config
	wsBase, _ := os.Getwd()
	alphaWS := alpha.NewWorkspace(wsBase)
	betaWS := beta.NewWorkspace(wsBase)

	// Create Alpha Architect
	var architect *alpha.Architect
	if llmExec != nil {
		architect = alpha.NewArchitect(projectID, alpha.ArchitectConfig{
			Workspace: alphaWS,
			Executor:  llmExec,
			Assembly:  assembly,
			GraphName: projectID + "_alpha_A1",
		})
	}

	// Create Beta Designer
	var designer *beta.Designer
	if llmExec != nil {
		designer = beta.NewDesigner(projectID, beta.DesignerConfig{
			Workspace: betaWS,
			Executor:  llmExec,
			Assembly:  assembly,
			GraphName: projectID + "_beta_B1",
		})
	}

	// Create Planner
	var planner *foundry.Planner
	if llmExec != nil {
		planner = foundry.NewPlanner(llmExec, wsBase)
	}

	return &FoundryStack{
		Dispatcher: dispatcher,
		Alpha:      architect,
		Beta:       designer,
		Planner:    planner,
		Executor:   llmExec,
		Store:      store,
	}, nil
}

// getExecutor uses the same logic as alpha/beta CLI interactive.go
func getExecutor() plugin.Executor {
	// Logic to load configured executor
	exec, err := plugin.NewLLMClient("") // Use default executor
	if err != nil {
		// Log warning but allow proceeding (some commands don't need LLM)
		// Or return nil, effectively disabling LLM features
		// fmt.Printf("Warning: Failed to load LLM executor: %v\n", err)
		return nil
	}
	return exec
}

func submitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "submit <ticket.json>",
		Short: "Submit a new ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := initStack("any") // Project ID irrelevant for submit
			if err != nil {
				return err
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read ticket: %w", err)
			}

			var ticket foundry.Ticket
			if err := json.Unmarshal(data, &ticket); err != nil {
				return fmt.Errorf("failed to parse ticket: %w", err)
			}

			ctx := context.Background()
			if err := stack.Dispatcher.Submit(ctx, &ticket); err != nil {
				return fmt.Errorf("failed to submit ticket: %w", err)
			}

			fmt.Printf("✓ Ticket submitted: %s\n", ticket.TicketID)
			return nil
		},
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <ticket_id>",
		Short: "Execute a submitted ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := initStack("any")
			if err != nil {
				return err
			}

			ticketID := args[0]
			fmt.Printf("Running ticket: %s...\n", ticketID)

			ctx := context.Background()
			result, err := stack.Dispatcher.Run(ctx, ticketID)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			if result.Success {
				fmt.Println("✓ Execution successful")
				for _, a := range result.Artifacts {
					fmt.Printf("  - Artifact: %s (%s)\n", a.ArtifactID, a.ArtifactType)
				}
			} else {
				fmt.Println("✗ Execution failed")
				if result.OmegaResult != nil {
					for _, d := range result.OmegaResult.Defects {
						fmt.Printf("  - Defect: %s\n", d.Message)
					}
				}
				if result.Error != "" {
					fmt.Printf("  - Error: %s\n", result.Error)
				}
			}

			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <ticket_id>",
		Short: "Get ticket status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := initStack("any")
			if err != nil {
				return err
			}
			defer stack.Store.Close()

			ticketID := args[0]
			ctx := context.Background()

			ticket, err := stack.Store.Tickets.GetByTicketID(ctx, ticketID)
			if err != nil {
				return fmt.Errorf("failed to get ticket: %w", err)
			}

			fmt.Printf("Ticket: %s\n", ticket.TicketID)
			fmt.Printf("Title:  %s\n", ticket.Title)
			fmt.Printf("State:  %s\n", ticket.State)
			fmt.Printf("Plugin: %s\n", ticket.PluginID)

			runs, err := stack.Store.Runs.ListByTicket(ctx, ticketID)
			if err == nil && len(runs) > 0 {
				fmt.Println("\nExecution Runs:")
				for _, r := range runs {
					fmt.Printf("  - Run %s (Attempt %d): %s\n", r.RunID, r.Attempt, r.Status)
				}
			}

			return nil
		},
	}
}

func artifactsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "artifacts <ticket_id>",
		Short: "List artifacts for a ticket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := initStack("any")
			if err != nil {
				return err
			}
			defer stack.Store.Close()

			ticketID := args[0]
			ctx := context.Background()

			artifacts, err := stack.Store.Artifacts.ListByTicket(ctx, ticketID)
			if err != nil {
				return fmt.Errorf("failed to list artifacts: %w", err)
			}

			if len(artifacts) == 0 {
				fmt.Println("No artifacts found.")
				return nil
			}

			fmt.Printf("Artifacts for ticket %s:\n", ticketID)
			for _, a := range artifacts {
				fmt.Printf("  - %s (%s) [%s]\n", a.ArtifactID, a.ArtifactType, a.Hash[:8])
			}

			return nil
		},
	}
}

func planCmd() *cobra.Command {
	var alphaRef, betaRef string
	var uploads []string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Start a collaborative planning session (Round Table)",
		RunE: func(cmd *cobra.Command, args []string) error {
			project := "default"
			if len(args) > 0 {
				project = args[0]
			}

			stack, err := initStack(project)
			if err != nil {
				return fmt.Errorf("failed to init stack: %w", err)
			}

			opts := foundry.RoundTableOptions{
				ProjectID: project,
				AlphaRef:  alphaRef,
				BetaRef:   betaRef,
				Uploads:   uploads,
			}

			if stack.Executor == nil {
				return fmt.Errorf("LLM executor required for planning. Run 'partir executors add' first")
			}

			// Create and persist session before starting
			ses := session.NewSession(project)
			ses.CurrentStep = "round-table"
			ses.Context["alpha_ref"] = alphaRef
			ses.Context["beta_ref"] = betaRef
			if err := ses.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save session: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Session saved: %s (resume with 'foundry resume')\n", ses.ID)
			}

			rt := foundry.NewRoundTable(opts, stack.Alpha, stack.Beta, stack.Planner, stack.Executor, nil)
			rtErr := rt.Start(context.Background())

			// Mark session complete (or paused on error)
			if rtErr != nil {
				ses.State = "paused"
				ses.Context["error"] = rtErr.Error()
			} else {
				ses.State = "completed"
			}
			_ = ses.Save()

			return rtErr
		},
	}

	cmd.Flags().StringVar(&alphaRef, "alpha", "", "Alpha rulebook reference")
	cmd.Flags().StringVar(&betaRef, "beta", "", "Beta rulebook reference")
	cmd.Flags().StringSliceVar(&uploads, "upload", nil, "Files to upload")

	return cmd
}

func scopeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scope",
		Short: "Manage project scopes",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "load <file>",
		Short: "Load and view a saved scope",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := foundry.LoadScope(args[0])
			if err != nil {
				return err
			}
			foundry.PresentScope(scope)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "approve <file>",
		Short: "Approve a saved scope and submit tickets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stack, err := initStack("any")
			if err != nil {
				return err
			}

			scope, err := foundry.LoadScope(args[0])
			if err != nil {
				return err
			}

			return foundry.ApproveScope(context.Background(), scope, stack.Dispatcher)
		},
	})

	return cmd
}

func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume a previously saved Round Table session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ses, err := session.Load()
			if err != nil {
				return fmt.Errorf("failed to load session: %w", err)
			}
			if ses == nil {
				fmt.Println("No saved session found. Start a new one with 'foundry plan'.")
				return nil
			}

			if ses.State != "active" && ses.State != "paused" {
				fmt.Printf("Session %s is %s — nothing to resume.\n", ses.ID, ses.State)
				return nil
			}

			fmt.Printf("Resuming session %s (ticket: %s, attempt: %d, step: %s)\n",
				ses.ID, ses.TicketID, ses.Attempt, ses.CurrentStep)

			stack, err := initStack("default")
			if err != nil {
				return err
			}

			if stack.Executor == nil {
				return fmt.Errorf("LLM executor required. Run 'partir executors add' first")
			}

			opts := foundry.RoundTableOptions{
				ProjectID: "default",
			}

			rt := foundry.NewRoundTable(opts, stack.Alpha, stack.Beta, stack.Planner, stack.Executor, nil)
			return rt.Start(context.Background())
		},
	}
}
