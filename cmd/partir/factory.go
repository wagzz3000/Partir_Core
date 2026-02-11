package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/partir/core/internal/factory"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

var factoryCmd = &cobra.Command{
	Use:   "factory",
	Short: "Manage the code factory (stations and workers)",
	Long:  `Commands for managing factory floor operations: stations, workers, and metrics.`,
}

var factoryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show factory status (slots and workers)",
	RunE:  runFactoryStatus,
}

var factoryWorkersCmd = &cobra.Command{
	Use:   "workers",
	Short: "List all registered workers",
	RunE:  runFactoryWorkers,
}

var factoryBindingsCmd = &cobra.Command{
	Use:   "bindings",
	Short: "Audit worker-to-memory binding relationships",
	RunE:  runFactoryBindings,
}

var factoryAssignCmd = &cobra.Command{
	Use:   "assign <station> <worker-id>",
	Short: "Assign a worker to a station (auto-assigns slot)",
	Args:  cobra.ExactArgs(2),
	RunE:  runFactoryAssign,
}

var factoryUnassignCmd = &cobra.Command{
	Use:   "unassign <worker-id>",
	Short: "Remove worker from their station assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runFactoryUnassign,
}

var factoryMetricsCmd = &cobra.Command{
	Use:   "metrics [station]",
	Short: "Show station metrics",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runFactoryMetrics,
}

var factoryInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize factory (create stations)",
	RunE:  runFactoryInit,
}

func init() {
	factoryCmd.AddCommand(factoryStatusCmd)
	factoryCmd.AddCommand(factoryWorkersCmd)
	factoryCmd.AddCommand(factoryBindingsCmd)
	factoryCmd.AddCommand(factoryAssignCmd)
	factoryCmd.AddCommand(factoryUnassignCmd)
	factoryCmd.AddCommand(factoryMetricsCmd)
	factoryCmd.AddCommand(factoryInitCmd)

	rootCmd.AddCommand(factoryCmd)
}

func getRegistry() (*factory.Registry, error) {
	dbURL := os.Getenv("PARTIR_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://partir:partir@localhost:5432/partir?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return factory.NewRegistry(pool), nil
}

func runFactoryStatus(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()
	status, err := registry.GetFactoryStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get factory status: %w", err)
	}

	// Print header
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      FACTORY FLOOR STATUS                      ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Total Workers: %-3d  Active: %-3d  Crashed: %-3d               ║\n",
		status.TotalWorkers, status.ActiveWorkers, status.CrashedWorkers)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")

	// Print stations
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, s := range status.Stations {
		fmt.Fprintf(w, "\nSTATION: %s (%s)\n", strings.ToUpper(string(s.ID)), strings.ToUpper(string(s.Status)))
		fmt.Fprintln(w, "SLOT\tWORKER ID\tSTATUS\tHEARTBEAT\t")
		fmt.Fprintln(w, "────\t─────────\t──────\t─────────\t")

		if len(s.Workers) == 0 {
			fmt.Fprintln(w, "-\t(empty)\t-\t-\t")
		} else {
			for _, worker := range s.Workers {
				heartbeat := "-"
				if worker.LastHeartbeat != nil {
					heartbeat = time.Since(*worker.LastHeartbeat).Round(time.Second).String() + " ago"
				}
				slot := worker.AssignedSlot
				if slot == "" {
					slot = "?"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", slot, worker.ID, worker.Status, heartbeat)
			}
		}
	}
	w.Flush()

	fmt.Println("\n╚═══════════════════════════════════════════════════════════════╝")

	return nil
}

func runFactoryWorkers(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()
	workers, err := registry.ListWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workers: %w", err)
	}

	if len(workers) == 0 {
		fmt.Println("No workers registered. Workers will auto-register when they connect.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tSTATION\tSLOT\tENDPOINT\tLAST HEARTBEAT")
	fmt.Fprintln(w, "──\t────\t──────\t───────\t────\t────────\t──────────────")

	for _, worker := range workers {
		station := "-"
		if worker.AssignedStation != "" {
			station = worker.AssignedStation
		}
		slot := "-"
		if worker.AssignedSlot != "" {
			slot = worker.AssignedSlot
		}
		heartbeat := "never"
		if worker.LastHeartbeat != nil {
			heartbeat = time.Since(*worker.LastHeartbeat).Round(time.Second).String() + " ago"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			worker.ID, worker.Type, worker.Status, station, slot, worker.Endpoint, heartbeat)
	}
	w.Flush()

	return nil
}

func runFactoryBindings(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()
	workers, err := registry.ListWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workers: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "WORKER ID\tTYPE\tMEMORY BINDING ID\tFINGERPRINT (SHA256)\tNAMESPACE")
	fmt.Fprintln(w, "─────────\t────\t─────────────────\t────────────────────\t─────────")

	for _, worker := range workers {
		bindingID := "stateless"
		if worker.MemoryBindingID != nil && *worker.MemoryBindingID != "" {
			bindingID = *worker.MemoryBindingID
		}

		fingerprint := worker.ModelFingerprint
		if len(fingerprint) > 16 {
			fingerprint = fingerprint[:16] + "..."
		} else if fingerprint == "" {
			fingerprint = "unknown"
		}

		namespace := "default"
		if worker.MemoryNamespace != nil && *worker.MemoryNamespace != "" {
			namespace = *worker.MemoryNamespace
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			worker.ID, worker.Type, bindingID, fingerprint, namespace)
	}
	w.Flush()
	return nil
}

func runFactoryAssign(cmd *cobra.Command, args []string) error {
	stationID := factory.StationID(args[0])
	workerID := args[1]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = registry.AssignWorkerToStation(ctx, stationID, workerID)
	if err != nil {
		return fmt.Errorf("failed to assign worker: %w", err)
	}

	fmt.Printf("✓ Assigned worker %s to station %s\n", workerID, stationID)
	return nil
}

func runFactoryUnassign(cmd *cobra.Command, args []string) error {
	workerID := args[0]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = registry.UnassignWorker(ctx, workerID)
	if err != nil {
		return fmt.Errorf("failed to unassign worker: %w", err)
	}

	fmt.Printf("✓ Unassigned worker %s\n", workerID)
	return nil
}

func runFactoryMetrics(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// If specific station requested
	if len(args) == 1 {
		stationID := factory.StationID(args[0])
		station, err := registry.GetStation(ctx, stationID)
		if err != nil {
			return fmt.Errorf("failed to get station: %w", err)
		}

		data, _ := json.MarshalIndent(station, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// All stations metrics
	status, err := registry.GetFactoryStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get factory status: %w", err)
	}

	fmt.Println("Factory Metrics Summary")
	fmt.Println("═══════════════════════")
	fmt.Printf("Total Workers:   %d\n", status.TotalWorkers)
	fmt.Printf("Active Workers:  %d\n", status.ActiveWorkers)
	fmt.Printf("Crashed Workers: %d\n", status.CrashedWorkers)
	fmt.Println()
	fmt.Println("(Detailed metrics available in Grafana at http://localhost:3000)")

	return nil
}

func runFactoryInit(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Ensure stations exist
	err = registry.EnsureStationsExist(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize stations: %w", err)
	}

	fmt.Println("✓ Factory initialized")
	fmt.Println("  - Alpha station ready")
	fmt.Println("  - Beta station ready")
	fmt.Println("  - Omega station ready")
	fmt.Println()
	fmt.Println("Workers will auto-register when they connect via NATS heartbeat.")

	return nil
}
