package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/partir/core/internal/alpha"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Initialize a new Alpha rulebook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			if ws.Exists(name) {
				return fmt.Errorf("rulebook %q already exists", name)
			}

			rb, err := ws.Init(name)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Created Alpha rulebook: %s\n", name)
			fmt.Printf("  ID: %s\n", rb.Manifest.ID)
			fmt.Printf("  Path: %s/%s\n", workspaceDir, name)
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Add schemas to schemas/")
			fmt.Println("  2. Add catalogs to catalogs/")
			fmt.Println("  3. Add rules to rules/rules.json")
			fmt.Println("  4. Run: alpha lint", name)
			fmt.Println("  5. Run: alpha promote", name, "--version 0.1.0")
			return nil
		},
	}
}

func lintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint <name>",
		Short: "Validate an Alpha rulebook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			rb, err := ws.Load(name)
			if err != nil {
				return err
			}

			linter := alpha.NewLinter()
			result := linter.Lint(rb)

			if len(result.Warnings) > 0 {
				fmt.Println("Warnings:")
				for _, w := range result.Warnings {
					fmt.Printf("  ⚠ %s: %s\n", w.Path, w.Message)
				}
				fmt.Println()
			}

			if !result.Valid {
				fmt.Println("Errors:")
				for _, e := range result.Errors {
					fmt.Printf("  ✗ %s: %s\n", e.Path, e.Message)
				}
				return fmt.Errorf("linting failed with %d errors", len(result.Errors))
			}

			fmt.Printf("✓ Rulebook %q is valid\n", name)
			return nil
		},
	}
}

func buildCmd() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "build <name>",
		Short: "Compile rulebook sources into canonical JSON bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			data, err := ws.Build(name)
			if err != nil {
				return err
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return err
				}
				fmt.Printf("✓ Built rulebook to: %s\n", outputFile)
			} else {
				fmt.Println(string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
	return cmd
}

func promoteCmd() *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "promote <name>",
		Short: "Promote rulebook to a new immutable version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			rb, err := ws.Load(name)
			if err != nil {
				return err
			}

			// Lint first
			linter := alpha.NewLinter()
			result := linter.Lint(rb)
			if !result.Valid {
				return fmt.Errorf("rulebook failed linting, fix errors before promoting")
			}

			// Promote
			if err := rb.Promote(version); err != nil {
				return err
			}

			// Update manifest
			if err := ws.WriteManifest(name, rb); err != nil {
				return err
			}

			// Archive version
			if err := ws.ArchiveVersion(name, rb); err != nil {
				return fmt.Errorf("failed to archive version: %w", err)
			}

			fmt.Printf("✓ Promoted %s to version %s\n", name, version)
			fmt.Printf("  Hash: %s\n", rb.Manifest.Hash)
			fmt.Printf("  Ref: %s@%s\n", name, version)
			return nil
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "version to promote to (required)")
	cmd.MarkFlagRequired("version")
	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Alpha rulebooks in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := os.ReadDir(workspaceDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No rulebooks found.")
					return nil
				}
				return err
			}

			ws := alpha.NewWorkspace(workspaceDir)
			found := false

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !ws.Exists(name) {
					continue
				}

				rb, err := ws.Load(name)
				if err != nil {
					fmt.Printf("  %s (error loading: %v)\n", name, err)
					continue
				}

				found = true
				version := rb.Manifest.Version
				if version == "" {
					version = "(draft)"
				}
				fmt.Printf("  %s@%s\n", name, version)
			}

			if !found {
				fmt.Println("No rulebooks found.")
			}

			return nil
		},
	}
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>[@version]",
		Short: "Get rulebook details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var version string
			if idx := strings.LastIndex(name, "@"); idx != -1 {
				version = name[idx+1:]
				name = name[:idx]
			}

			ws := alpha.NewWorkspace(workspaceDir)

			rb, err := ws.LoadVersion(name, version)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(rb.Manifest, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))

			fmt.Printf("\nSchemas: %d\n", len(rb.Schemas))
			fmt.Printf("Catalogs: %d\n", len(rb.Catalogs))
			fmt.Printf("Rules: %d combinations, %d cost tables, %d invariants\n",
				len(rb.Rules.Combinations),
				len(rb.Rules.CostTables),
				len(rb.Rules.Invariants))

			return nil
		},
	}
}
