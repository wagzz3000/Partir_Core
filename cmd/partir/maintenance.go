package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/partir/core/internal/maintenance"
	"github.com/spf13/cobra"
)

var maintenanceCmd = &cobra.Command{
	Use:   "maintenance",
	Short: "Maintenance operations for memory pairs",
	Long:  `Inspect, set phase, and release memory pairs that are in the "maintenance" state.`,
}

var maintenanceInspectCmd = &cobra.Command{
	Use:   "inspect <pair_id>",
	Short: "Inspect a maintenance pair",
	Args:  cobra.ExactArgs(1),
	RunE:  runMaintenanceInspect,
}

var maintenancePhaseCmd = &cobra.Command{
	Use:   "phase <pair_id> <phase>",
	Short: "Set maintenance phase (diagnose|triage|script|wot|cbt|surgery|rehab|test)",
	Args:  cobra.ExactArgs(2),
	RunE:  runMaintenancePhase,
}

var maintenanceReleaseCmd = &cobra.Command{
	Use:   "release <pair_id>",
	Short: "Release pair from maintenance → sleep",
	Args:  cobra.ExactArgs(1),
	RunE:  runMaintenanceRelease,
}

func init() {
	maintenanceCmd.AddCommand(maintenanceInspectCmd)
	maintenanceCmd.AddCommand(maintenancePhaseCmd)
	maintenanceCmd.AddCommand(maintenanceReleaseCmd)
	rootCmd.AddCommand(maintenanceCmd)
}

func getMaintenanceAPI() (*maintenance.API, error) {
	dbURL := os.Getenv("PARTIR_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://partir:partir@localhost:5432/partir?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Note: ledger is nil for CLI commands — no ledger tracking from CLI
	return maintenance.NewAPI(db, nil), nil
}

func runMaintenanceInspect(cmd *cobra.Command, args []string) error {
	api, err := getMaintenanceAPI()
	if err != nil {
		return err
	}

	result, err := api.Inspect(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("inspect failed: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return nil
}

func runMaintenancePhase(cmd *cobra.Command, args []string) error {
	api, err := getMaintenanceAPI()
	if err != nil {
		return err
	}

	if err := api.SetPhase(context.Background(), args[0], args[1]); err != nil {
		return fmt.Errorf("set phase failed: %w", err)
	}

	fmt.Printf("✅ Pair %s → phase %s\n", args[0], args[1])
	return nil
}

func runMaintenanceRelease(cmd *cobra.Command, args []string) error {
	api, err := getMaintenanceAPI()
	if err != nil {
		return err
	}

	if err := api.Release(context.Background(), args[0]); err != nil {
		return fmt.Errorf("release failed: %w", err)
	}

	fmt.Printf("✅ Pair %s released → sleep\n", args[0])
	return nil
}
