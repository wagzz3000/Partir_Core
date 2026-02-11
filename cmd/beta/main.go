// Beta CLI - Define expression (visualization, style, render vocab)
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/partir/core/internal/beta"
	"github.com/spf13/cobra"
)

var workspaceDir string

func main() {
	rootCmd := &cobra.Command{
		Use:   "beta",
		Short: "Partir Beta - Define expression (render vocab, styles)",
		Long:  `Beta defines the "expression" for Partir Core: visualization, style rules, and render vocabulary.`,
	}

	rootCmd.PersistentFlags().StringVarP(&workspaceDir, "workspace", "w", "./rulebooks/beta", "workspace directory")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(resumeCmd())
	rootCmd.AddCommand(lintCmd())
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(promoteCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(exportFigmaCmd())
	rootCmd.AddCommand(importFigmaCmd())
	rootCmd.AddCommand(syncFigmaCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Initialize a new Beta rulebook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			if ws.Exists(name) {
				return fmt.Errorf("rulebook %q already exists", name)
			}

			rb, err := ws.Init(name)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Created Beta rulebook: %s\n", name)
			fmt.Printf("  ID: %s\n", rb.Manifest.ID)
			fmt.Printf("  Effects: %d\n", len(rb.RenderVocab.Effects))
			fmt.Printf("  Palettes: %d\n", len(rb.RenderVocab.Palettes))
			return nil
		},
	}
}

func lintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint <name>",
		Short: "Validate a Beta rulebook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			rb, err := ws.Load(name)
			if err != nil {
				return err
			}

			// Basic validation
			errors := []string{}

			if rb.Manifest.Name == "" {
				errors = append(errors, "manifest.name is required")
			}

			// Check effects have IDs
			for i, e := range rb.RenderVocab.Effects {
				if e.ID == "" {
					errors = append(errors, fmt.Sprintf("effects[%d].id is required", i))
				}
			}

			// Check palettes have colors
			for i, p := range rb.RenderVocab.Palettes {
				if len(p.Colors) == 0 {
					errors = append(errors, fmt.Sprintf("palettes[%d].colors is empty", i))
				}
			}

			if len(errors) > 0 {
				fmt.Println("Errors:")
				for _, e := range errors {
					fmt.Printf("  ✗ %s\n", e)
				}
				return fmt.Errorf("linting failed")
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
		Short: "Compile rulebook to canonical JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			data, err := ws.Build(name)
			if err != nil {
				return err
			}

			if outputFile != "" {
				return os.WriteFile(outputFile, data, 0644)
			}
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file")
	return cmd
}

func promoteCmd() *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "promote <name>",
		Short: "Promote rulebook to immutable version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			rb, err := ws.Load(name)
			if err != nil {
				return err
			}

			if err := rb.Promote(version); err != nil {
				return err
			}

			if err := ws.Save(name, rb); err != nil {
				return err
			}

			fmt.Printf("✓ Promoted %s to version %s\n", name, version)
			fmt.Printf("  Hash: %s\n", rb.Manifest.Hash)
			return nil
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "version (required)")
	cmd.MarkFlagRequired("version")
	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Beta rulebooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := os.ReadDir(workspaceDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No rulebooks found.")
					return nil
				}
				return err
			}

			ws := beta.NewWorkspace(workspaceDir)
			for _, entry := range entries {
				if !entry.IsDir() || !ws.Exists(entry.Name()) {
					continue
				}
				rb, _ := ws.Load(entry.Name())
				version := rb.Manifest.Version
				if version == "" {
					version = "(draft)"
				}
				fmt.Printf("  %s@%s\n", entry.Name(), version)
			}
			return nil
		},
	}
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get rulebook details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := beta.NewWorkspace(workspaceDir)
			rb, err := ws.Load(args[0])
			if err != nil {
				return err
			}

			data, _ := json.MarshalIndent(rb.Manifest, "", "  ")
			fmt.Println(string(data))
			fmt.Printf("\nEffects: %d, Palettes: %d, Animations: %d\n",
				len(rb.RenderVocab.Effects),
				len(rb.RenderVocab.Palettes),
				len(rb.RenderVocab.Animations))
			return nil
		},
	}
}
