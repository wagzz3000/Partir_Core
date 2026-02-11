package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var collabCmd = &cobra.Command{
	Use:   "collab",
	Short: "Collaboration API operations",
	Long:  `View active Andon signals, pending fixes, and manage ticket submissions.`,
}

var collabStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active signals and pending fixes",
	RunE:  runCollabStatus,
}

var collabSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a ticket via Collaboration API",
	RunE:  runCollabSubmit,
}

func init() {
	collabCmd.AddCommand(collabStatusCmd)
	collabCmd.AddCommand(collabSubmitCmd)
	rootCmd.AddCommand(collabCmd)
}

func runCollabStatus(cmd *cobra.Command, args []string) error {
	// The Collaboration API is stateless — status comes from the Factory Ledger.
	// For now, print a stub message. Full implementation will query the ledger
	// for recent collab events (andon, fix, alert, confirm).
	fmt.Println("Collaboration API Status")
	fmt.Println("========================")
	fmt.Println("The Collaboration API is stateless. Query the Factory Ledger for recent events:")
	fmt.Println("  Event types: andon_cord, fix_routed, human_alert, repo_confirm")
	fmt.Println("")
	fmt.Println("Set NATS_URL to enable live topic monitoring.")
	return nil
}

func runCollabSubmit(cmd *cobra.Command, args []string) error {
	fmt.Println("Ticket submission via Collaboration API requires NATS connection.")
	fmt.Println("Use the Control Surface or publish directly to collab.submit.")
	return nil
}
