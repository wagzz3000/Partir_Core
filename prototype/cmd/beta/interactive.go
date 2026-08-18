package main

import (
	"context"
	"fmt"

	"github.com/partir/core/internal/beta"
	"github.com/partir/core/internal/interview"
	"github.com/partir/core/pkg/plugin"
	"github.com/spf13/cobra"
)

// startCmd runs the interactive Beta Designer session.
func startCmd() *cobra.Command {
	var uploadFile, alphaRef string

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start an interactive design session",
		Long:  `Start an AI-powered interview that generates render vocabulary, palettes, effects, and style rules.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			exec := getBetaExecutor()
			if exec == nil {
				return fmt.Errorf("no executor plugin configured. Register one with: partir plugin register")
			}

			assembly := &interview.PairAssembly{
				ProjectID: name,
				Alpha:     [4]string{"A1", "A2", "A3", "A4"},
				Beta:      [4]string{"B1", "B2", "B3", "B4"},
				Omega:     [4]string{"O1", "O2", "O3", "O4"},
			}

			var designPlugin beta.DesignPlugin
			// Auto-detect Figma plugin if FIGMA_TOKEN is set
			figmaToken := ""
			if figmaToken != "" {
				designPlugin = beta.NewFigmaPlugin(figmaToken, "")
			}

			designer := beta.NewDesigner(name, beta.DesignerConfig{
				Workspace:    ws,
				Executor:     exec,
				Assembly:     assembly,
				GraphName:    name + "_beta_B1",
				AlphaRef:     alphaRef,
				DesignPlugin: designPlugin,
			})

			ctx := context.Background()

			if uploadFile != "" {
				_, err := designer.StartWithUpload(ctx, uploadFile)
				return err
			}

			_, err := designer.Start(ctx)
			return err
		},
	}

	cmd.Flags().StringVar(&uploadFile, "upload", "", "file to upload (sketch, Figma export, image)")
	cmd.Flags().StringVar(&alphaRef, "alpha-ref", "", "Alpha rulebook reference for contextual questions (e.g., my_game@0.1.0)")
	return cmd
}

// resumeCmd re-enters a previous design session.
func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a previous design session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			exec := getBetaExecutor()
			if exec == nil {
				return fmt.Errorf("no executor plugin configured")
			}

			assembly := &interview.PairAssembly{
				ProjectID: name,
				Alpha:     [4]string{"A1", "A2", "A3", "A4"},
				Beta:      [4]string{"B1", "B2", "B3", "B4"},
				Omega:     [4]string{"O1", "O2", "O3", "O4"},
			}

			designer := beta.NewDesigner(name, beta.DesignerConfig{
				Workspace: ws,
				Executor:  exec,
				Assembly:  assembly,
				GraphName: name + "_beta_B1",
			})

			_, err := designer.Resume(context.Background())
			return err
		},
	}
}

// exportFigmaCmd exports rulebook to Figma.
func exportFigmaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export figma <name>",
		Short: "Export design rulebook to Figma",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			exec := getBetaExecutor()
			figmaPlugin := beta.NewFigmaPlugin("", "")

			assembly := &interview.PairAssembly{ProjectID: name}
			designer := beta.NewDesigner(name, beta.DesignerConfig{
				Workspace:    ws,
				Executor:     exec,
				Assembly:     assembly,
				DesignPlugin: figmaPlugin,
			})

			_, err := designer.ExportDesign(context.Background())
			return err
		},
	}
}

// importFigmaCmd imports design tokens from Figma.
func importFigmaCmd() *cobra.Command {
	var fileKey string

	cmd := &cobra.Command{
		Use:   "import figma <name>",
		Short: "Import design tokens from Figma file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			figmaPlugin := beta.NewFigmaPlugin("", "")

			assembly := &interview.PairAssembly{ProjectID: name}
			designer := beta.NewDesigner(name, beta.DesignerConfig{
				Workspace:    ws,
				Executor:     getBetaExecutor(),
				Assembly:     assembly,
				DesignPlugin: figmaPlugin,
			})

			return designer.ImportDesign(context.Background(), fileKey)
		},
	}

	cmd.Flags().StringVar(&fileKey, "file-key", "", "Figma file key (required)")
	cmd.MarkFlagRequired("file-key")
	return cmd
}

// syncFigmaCmd syncs changes from Figma back to rulebook.
func syncFigmaCmd() *cobra.Command {
	var fileKey string

	cmd := &cobra.Command{
		Use:   "sync figma <name>",
		Short: "Sync design changes from Figma",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ws := beta.NewWorkspace(workspaceDir)

			figmaPlugin := beta.NewFigmaPlugin("", "")

			assembly := &interview.PairAssembly{ProjectID: name}
			designer := beta.NewDesigner(name, beta.DesignerConfig{
				Workspace:    ws,
				Executor:     getBetaExecutor(),
				Assembly:     assembly,
				DesignPlugin: figmaPlugin,
			})

			return designer.SyncDesign(context.Background(), fileKey)
		},
	}

	cmd.Flags().StringVar(&fileKey, "file-key", "", "Figma file key (required)")
	cmd.MarkFlagRequired("file-key")
	return cmd
}

// getBetaExecutor tries to load the configured executor plugin.
func getBetaExecutor() plugin.Executor {
	return nil
}
