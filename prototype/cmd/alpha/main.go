// Alpha CLI - Define law (schemas, catalogs, rule tables, constraints)
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	workspaceDir string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "alpha",
		Short: "Partir Alpha - Define law (schemas, catalogs, rules)",
		Long:  `Alpha defines the "law" for Partir Core: schemas, catalogs, rule tables, and constraints.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&workspaceDir, "workspace", "w", "./rulebooks/alpha", "workspace directory")

	// Add commands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(resumeCmd())
	rootCmd.AddCommand(importCmd())
	rootCmd.AddCommand(lintCmd())
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(promoteCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(getCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
