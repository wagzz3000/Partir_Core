package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/partir/core/pkg/plugin"
	"github.com/spf13/cobra"
)

var executorsCmd = &cobra.Command{
	Use:   "executors",
	Short: "Manage executor plugins",
	Long:  `Manage HTTP executor plugins for LLM execution`,
}

var executorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered executors",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := plugin.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.Executors) == 0 {
			fmt.Println("No executors registered.")
			fmt.Println("\nTo add an executor:")
			fmt.Println("  partir executors add --name agent_ollama --url http://localhost:8081")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tURL")
		fmt.Fprintln(w, "----\t----\t---")
		for name, exec := range cfg.Executors {
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, exec.Type, exec.URL)
		}
		w.Flush()
	},
}

var executorsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an executor plugin",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		url, _ := cmd.Flags().GetString("url")

		if name == "" || url == "" {
			fmt.Fprintln(os.Stderr, "Error: --name and --url are required")
			os.Exit(1)
		}

		cfg, err := plugin.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		cfg.AddExecutor(name, url)

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Added executor %q at %s\n", name, url)
		fmt.Println("\nTest connectivity with:")
		fmt.Printf("  partir executors ping --name %s\n", name)
	},
}

var executorsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an executor plugin",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		cfg, err := plugin.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if _, ok := cfg.GetExecutor(name); !ok {
			fmt.Fprintf(os.Stderr, "Error: executor %q not found\n", name)
			os.Exit(1)
		}

		cfg.RemoveExecutor(name)

		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Removed executor %q\n", name)
	},
}

var executorsPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test connectivity to an executor",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		cfg, err := plugin.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		execCfg, ok := cfg.GetExecutor(name)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: executor %q not found\n", name)
			os.Exit(1)
		}

		client, err := plugin.NewHTTPClientFromConfig(name, execCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Pinging %s at %s...\n", name, execCfg.URL)

		// Health check
		health, err := client.Health(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Health: %s\n", health.Status)

		// Capabilities
		caps, err := client.Capabilities(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Capabilities check failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Capabilities:\n")
		fmt.Printf("  ID:            %s\n", caps.ID)
		fmt.Printf("  Name:          %s\n", caps.Name)
		fmt.Printf("  Version:       %s\n", caps.Version)
		fmt.Printf("  Models:        %v\n", caps.Models)
		fmt.Printf("  Default Model: %s\n", caps.DefaultModel)
		fmt.Printf("  Max Context:   %d tokens\n", caps.MaxContextLen)
		fmt.Printf("  Supports Plan: %t\n", caps.SupportsPlan)
		fmt.Printf("  Supports Patch: %t\n", caps.SupportsPatch)
	},
}

func init() {
	executorsAddCmd.Flags().String("name", "", "Name for the executor (required)")
	executorsAddCmd.Flags().String("url", "", "URL for the executor (required)")

	executorsRemoveCmd.Flags().String("name", "", "Name of the executor to remove (required)")

	executorsPingCmd.Flags().String("name", "", "Name of the executor to ping (required)")

	executorsCmd.AddCommand(executorsListCmd)
	executorsCmd.AddCommand(executorsAddCmd)
	executorsCmd.AddCommand(executorsRemoveCmd)
	executorsCmd.AddCommand(executorsPingCmd)
}
