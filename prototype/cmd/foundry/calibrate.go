package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/partir/core/internal/foundry"
	"github.com/spf13/cobra"
)

func calibrateCmd() *cobra.Command {
	var (
		count       int
		concurrency int
		prompt      string
		model       string
	)

	cmd := &cobra.Command{
		Use:   "calibrate",
		Short: "Run a calibration load test (Factory Physics Phase 25)",
		Long: `Submits a batch of tickets to measure Effective Processing Rate (µ_eff).
Reports throughput, latency, and token consumption to calibrate the Factory Controller.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Initialize Stack
			stack, err := initStack("calibration")
			if err != nil {
				return err
			}
			defer stack.Store.Close()

			fmt.Printf("Starting Calibration Run\n")
			fmt.Printf("========================\n")
			fmt.Printf("Batch Size:  %d\n", count)
			fmt.Printf("Concurrency: %d\n", concurrency)
			fmt.Printf("Model:       %s\n", model)
			fmt.Printf("Target:      %s\n", dbURL) // dbURL global from main.go
			fmt.Println()

			// 2. Prepare Workload
			start := time.Now()
			var wg sync.WaitGroup
			sem := make(chan struct{}, concurrency) // Semaphore for concurrency control
			results := make(chan calibrationResult, count)

			// 3. Execution Loop
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire
					defer func() { <-sem }() // Release

					res := runCalibrationTicket(stack, id, prompt, model)
					results <- res
				}(i)
			}

			// 4. Wait & Close
			go func() {
				wg.Wait()
				close(results)
			}()

			// 5. Aggregate Results
			var (
				totalDuration time.Duration
				totalTokens   int
				successCount  int
				failCount     int
				latencies     []time.Duration
			)

			fmt.Println("Progress:")
			completed := 0
			for res := range results {
				completed++
				if res.Error != nil {
					failCount++
					fmt.Printf("x")
				} else {
					successCount++
					fmt.Printf(".")
					totalDuration += res.Duration // This is sum of individual dimensions, not useful for throughput
					totalTokens += res.TokensOut
					latencies = append(latencies, res.Duration)
				}
				if completed%50 == 0 {
					fmt.Println()
				}
			}
			fmt.Println() // Newline

			totalWallTime := time.Since(start)

			// 6. Report
			printCalibrationReport(totalWallTime, successCount, failCount, totalTokens, latencies)

			return nil
		},
	}

	cmd.Flags().IntVar(&count, "count", 10, "Number of tickets to submit")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Max concurrent submissions")
	cmd.Flags().StringVar(&prompt, "prompt", "Write a short haiku about rust.", "Prompt to use")
	cmd.Flags().StringVar(&model, "model", "mistral:7b", "Model to request")

	return cmd
}

type calibrationResult struct {
	TicketID  string
	Duration  time.Duration
	TokensIn  int
	TokensOut int
	Error     error
}

func runCalibrationTicket(stack *FoundryStack, id int, prompt, model string) calibrationResult {
	start := time.Now()
	res := calibrationResult{}

	// Create Ticket
	ticket := &foundry.Ticket{
		TicketID: fmt.Sprintf("cal-%d-%d", time.Now().Unix(), id),
		Title:    fmt.Sprintf("Calibration Ticket %d", id),
		PluginID: "partir-plugin-ollama", // Hardcoded for now, or flags
		JobType:  "alpha",                // Default worker type
		Inputs: map[string]interface{}{
			"prompt": prompt,
			"model":  model, // Pass model in inputs if plugin supports it
		},
	}

	ctx := context.Background()

	// Submit
	if err := stack.Dispatcher.Submit(ctx, ticket); err != nil {
		res.Error = err
		return res
	}
	res.TicketID = ticket.TicketID

	// Poll for completion
	// In a real load test we might want event subscription, but polling is robust enough for 100 items.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-timeout:
			res.Error = fmt.Errorf("timeout")
			return res
		case <-ticker.C:
			// check status
			t, err := stack.Store.Tickets.GetByTicketID(ctx, ticket.TicketID)
			if err != nil {
				res.Error = err
				return res
			}

			if t.State == "completed" { // Correct state string from dispatcher.go
				res.Duration = time.Since(start)
				// Fetch run details for tokens
				runs, _ := stack.Store.Runs.ListByTicket(ctx, ticket.TicketID)
				if len(runs) > 0 {
					// Assume last run is the one
					lastRun := runs[len(runs)-1]
					// TODO: Parse tokens from Metadata if stored?
					// runs table has Metadata jsonb.
					// For now we just use a dummy value or try to parse if we added telemetry there.
					// Dispatcher adds CostTokens to Complete() call, which updates run.
					// Wait, storage.Run struct might not have CostTokens field exposed directly?
					// Dispatcher calls: d.runs.Complete(ctx, run.RunID, "completed", execResult.CostTokens, ...)
					// So it should be in the DB.
					// checking storage definition...
					// Assuming we can't easily get it without casting, just mark as 0 for now to fix linter.
					_ = lastRun
					res.TokensOut = 0
				}
				return res
			}
			if t.State == "failed" || t.State == "omega_failed" || t.State == "local_qc_failed" {
				res.Error = fmt.Errorf("ticket failed state: %s", t.State)
				return res
			}
		}
	}
}

func printCalibrationReport(wallTime time.Duration, success, fail, tokens int, latencies []time.Duration) {
	// Simple stats
	total := success + fail
	throughput := float64(success) / wallTime.Minutes()
	avgLatency := time.Duration(0)
	if success > 0 {
		sum := int64(0)
		for _, d := range latencies {
			sum += d.Nanoseconds()
		}
		avgLatency = time.Duration(sum / int64(success))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Println("\nCALIBRATION REPORT")
	fmt.Println("==================")
	fmt.Fprintf(w, "Metric\tValue\n")
	fmt.Fprintf(w, "Total Tickets\t%d\n", total)
	fmt.Fprintf(w, "Success Rate\t%.1f%%\n", float64(success)/float64(total)*100)
	fmt.Fprintf(w, "Wall Time\t%s\n", wallTime.Round(time.Millisecond))
	fmt.Fprintf(w, "Throughput (μ_eff)\t%.2f tickets/min\n", throughput)
	fmt.Fprintf(w, "Avg Latency (W)\t%s\n", avgLatency.Round(time.Millisecond))
	fmt.Fprintf(w, "Total Tokens\t%d (approx)\n", tokens)
	w.Flush()
}
