package main

import (
	"context"
	"fmt"

	"github.com/partir/core/internal/alpha"
	"github.com/partir/core/internal/interview"
	"github.com/partir/core/pkg/plugin"
	"github.com/spf13/cobra"
)

// startCmd runs the interactive Alpha Architect session.
func startCmd() *cobra.Command {
	var uploadFile string

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start an interactive architecture session",
		Long:  `Start an AI-powered interview that generates schemas, catalogs, and rules from conversation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			exec := getExecutor()
			if exec == nil {
				return fmt.Errorf("no executor plugin configured. Register one with: partir plugin register")
			}

			assembly := &interview.PairAssembly{
				ProjectID: name,
				Alpha:     [4]string{"A1", "A2", "A3", "A4"},
				Beta:      [4]string{"B1", "B2", "B3", "B4"},
				Omega:     [4]string{"O1", "O2", "O3", "O4"},
			}

			architect := alpha.NewArchitect(name, alpha.ArchitectConfig{
				Workspace: ws,
				Executor:  exec,
				Assembly:  assembly,
				GraphName: name + "_alpha_A1",
			})

			ctx := context.Background()

			if uploadFile != "" {
				_, err := architect.StartWithUpload(ctx, uploadFile)
				return err
			}

			_, err := architect.Start(ctx)
			return err
		},
	}

	cmd.Flags().StringVar(&uploadFile, "upload", "", "file to upload (document, diagram, spreadsheet)")
	return cmd
}

// resumeCmd re-enters a previous architecture session.
func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a previous architecture session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)

			exec := getExecutor()
			if exec == nil {
				return fmt.Errorf("no executor plugin configured")
			}

			assembly := &interview.PairAssembly{
				ProjectID: name,
				Alpha:     [4]string{"A1", "A2", "A3", "A4"},
				Beta:      [4]string{"B1", "B2", "B3", "B4"},
				Omega:     [4]string{"O1", "O2", "O3", "O4"},
			}

			architect := alpha.NewArchitect(name, alpha.ArchitectConfig{
				Workspace: ws,
				Executor:  exec,
				Assembly:  assembly,
				GraphName: name + "_alpha_A1",
			})

			_, err := architect.Resume(context.Background())
			return err
		},
	}
}

// importCmd provides sub-commands for importing from external sources.
func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import architecture from external sources",
	}

	cmd.AddCommand(importDBCmd())
	cmd.AddCommand(importOpenAPICmd())
	cmd.AddCommand(importCatalogCmd())
	return cmd
}

func importDBCmd() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "db <name>",
		Short: "Import from existing database (Postgres, MySQL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)
			importer := alpha.NewImporter(ws, getExecutor())

			rb, err := importer.ImportFromDB(context.Background(), name, dsn)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Imported %d schemas from database\n", len(rb.Schemas))
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "database connection string (required)")
	cmd.MarkFlagRequired("dsn")
	return cmd
}

func importOpenAPICmd() *cobra.Command {
	var specPath string

	cmd := &cobra.Command{
		Use:   "openapi <name>",
		Short: "Import from OpenAPI/Swagger specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)
			importer := alpha.NewImporter(ws, getExecutor())

			_, err := importer.ImportFromOpenAPI(context.Background(), name, specPath)
			if err != nil {
				return err
			}

			fmt.Println("✓ Imported schemas from OpenAPI spec")
			return nil
		},
	}

	cmd.Flags().StringVar(&specPath, "spec", "", "path to OpenAPI spec file (required)")
	cmd.MarkFlagRequired("spec")
	return cmd
}

func importCatalogCmd() *cobra.Command {
	var csvFile, linkURL string

	cmd := &cobra.Command{
		Use:   "catalog <name>",
		Short: "Import catalog from CSV file or link remote source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := alpha.NewWorkspace(workspaceDir)
			importer := alpha.NewImporter(ws, getExecutor())

			if csvFile != "" {
				err := importer.ImportCatalogFromCSV(context.Background(), name, csvFile, name)
				if err != nil {
					return err
				}
				fmt.Println("✓ Catalog imported from CSV")
				return nil
			}

			if linkURL != "" {
				err := importer.ImportCatalogFromURL(context.Background(), name, linkURL, name)
				if err != nil {
					return err
				}
				fmt.Println("✓ External catalog source linked")
				return nil
			}

			return fmt.Errorf("provide --csv or --link")
		},
	}

	cmd.Flags().StringVar(&csvFile, "csv", "", "CSV file to import")
	cmd.Flags().StringVar(&linkURL, "link", "", "URL to link as external source")
	return cmd
}

// getExecutor tries to load the configured executor plugin.
// Returns nil if no executor is configured.
func getExecutor() plugin.Executor {
	// In production, this reads from the plugin registry.
	// For now, attempt to connect to the default Ollama plugin.
	// The real implementation will use plugin.LoadFromRegistry().
	return nil
}
