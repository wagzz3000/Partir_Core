package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// PresentScope displays the project scope to the user in a human-readable format.
func PresentScope(scope *ProjectScope) {
	fmt.Printf("\n╔════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          PROJECT SCOPE                                ║\n")
	fmt.Printf("╚════════════════════════════════════════════════════════╝\n")
	fmt.Printf("Summary: %s\n", scope.Summary)
	fmt.Printf("Tickets: %d  |  Groups: %d  |  Est. Cost: $%.2f\n\n",
		scope.TotalTickets, len(scope.TicketGroups), scope.EstCostUSD)

	for _, g := range scope.TicketGroups {
		fmt.Printf("Phase %d: %s\n", g.Phase, g.Name)
		fmt.Printf("  %s\n", g.Description)
		for _, t := range g.Tickets {
			fmt.Printf("  - [%s] %s (%s)\n", t.TicketID, t.Title, t.JobType)
		}
		fmt.Println()
	}
}

// SaveScope persists the scope to a JSON file.
func SaveScope(scope *ProjectScope, path string) error {
	data, err := json.MarshalIndent(scope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadScope loads a saved scope from a JSON file.
func LoadScope(path string) (*ProjectScope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scope ProjectScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return nil, err
	}
	return &scope, nil
}

// ApproveScope submits all tickets in the scope to the dispatcher.
func ApproveScope(ctx context.Context, scope *ProjectScope, d *Dispatcher) error {
	fmt.Println("\n🚀 Approving scope... Submitting tickets to Foundry.")

	count := 0
	for _, g := range scope.TicketGroups {
		for _, t := range g.Tickets {
			if d != nil {
				if err := d.Submit(ctx, &t); err != nil {
					return fmt.Errorf("failed to submit ticket %s: %w", t.TicketID, err)
				}
			} else {
				// Dry run / simulation if dispatcher is nil (for mocked tests)
				fmt.Printf("  [DRY RUN] Submitted ticket: %s\n", t.TicketID)
			}
			count++
		}
	}

	fmt.Printf("\n✅ Successfully submitted %d tickets.\nProduction will begin shortly.\n", count)
	return nil
}
