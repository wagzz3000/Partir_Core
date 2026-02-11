package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/partir/core/pkg/plugin"
)

// Planner generates a project scope from finalized Alpha and Beta rulebooks.
type Planner struct {
	executor plugin.Executor
	baseDir  string // Root directory for rulebooks
}

// NewPlanner creates a new planner.
func NewPlanner(executor plugin.Executor, baseDir string) *Planner {
	// Default baseDir if empty
	if baseDir == "" {
		baseDir = "."
	}
	return &Planner{
		executor: executor,
		baseDir:  baseDir,
	}
}

// ProjectScope defines the full plan of execution.
type ProjectScope struct {
	Options      RoundTableOptions
	Summary      string        `json:"summary"`
	TicketGroups []TicketGroup `json:"ticket_groups"`
	TotalTickets int           `json:"total_tickets"`
	EstTokens    int64         `json:"est_tokens"`
	EstCostUSD   float64       `json:"est_cost_usd"`
}

// TicketGroup is a logical phase of work (e.g., "Schema Generation", "Content Population").
type TicketGroup struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Phase       int      `json:"phase"`
	Tickets     []Ticket `json:"tickets"`
}

// GenerateScope reads the rulebooks and asks the LLM to plan the work.
func (p *Planner) GenerateScope(ctx context.Context, alphaRef, betaRef string) (*ProjectScope, error) {
	// 1. Load Rulebooks (simulated for now by reading JSON files directly)
	// In a real implementation, we'd use the Alpha/Beta workspaces to load versioned rulebooks.
	alphaSummary := p.summarizeAlpha(alphaRef)
	betaSummary := p.summarizeBeta(betaRef)

	// 2. Construct Prompt
	prompt := fmt.Sprintf(`
You are the Foundry Planner.
Your job is to take an architecture (Alpha) and a design system (Beta) and break down the implementation into a series of Foundry tickets.

Project Context:
Alpha Architecture:
%s
Beta Design:
%s

Instructions:
1. Divide the work into logical phases (Ticket Groups).
2. Phase 1 must be foundational (e.g., database schema creation, core assets).
3. Phase 2 should be content generation (using the schemas).
4. Phase 3 should be validation/QA using Omega.
5. Create specific tickets for each task.

Output Format:
Return a JSON object matching this structure:
{
  "summary": "High level project summary...",
  "ticket_groups": [
    {
      "name": "Phase 1: Foundation",
      "description": "...",
      "phase": 1,
      "tickets": [
        {
          "title": "Create Product Table",
          "job_type": "schema_generation",
          "inputs": { ... }
        }
      ]
    }
  ]
}
Ensure valid JSON. Do not include markdown formatting.
`, alphaSummary, betaSummary)

	// 3. Execute LLM
	req := &plugin.ExecuteRequest{
		Prompt:      prompt,
		MaxTokens:   2000,
		Temperature: 0.2, // Lower temp for structured output
	}
	resp, err := p.executor.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// 4. Parse Response (handling potential markdown blocks)
	cleanResp := p.cleanJSON(resp.Output)
	var scope ProjectScope
	if err := json.Unmarshal([]byte(cleanResp), &scope); err != nil {
		return nil, fmt.Errorf("failed to parse scope JSON: %w\nResponse was: %s", err, cleanResp)
	}

	// 5. Post-process (calculate totals)
	totalTickets := 0
	// forx := 0 // Removed unused variable
	for _, g := range scope.TicketGroups {
		totalTickets += len(g.Tickets)
		for i := range g.Tickets {
			// Fill in missing refs
			g.Tickets[i].AlphaRef = alphaRef
			g.Tickets[i].BetaRef = betaRef

			// Generate IDs if missing
			if g.Tickets[i].TicketID == "" {
				g.Tickets[i].TicketID = fmt.Sprintf("ticket-%d-%d", g.Phase, i+1)
			}
		}
	}
	scope.TotalTickets = totalTickets
	scope.EstTokens = int64(totalTickets * 2000)                 // Rough estimate
	scope.EstCostUSD = float64(scope.EstTokens) / 1000000 * 0.20 // Approx. cost

	return &scope, nil
}

// cleanJSON strips markdown code fences from the output string
func (p *Planner) cleanJSON(input string) string {
	// Remove ```json and ``` or just ```
	lines := strings.Split(input, "\n")
	var result []string
	inBlock := false

	// Check if already clean (starts with {)
	if strings.HasPrefix(strings.TrimSpace(input), "{") {
		return input
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inBlock = !inBlock
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// summarizeAlpha reads the Alpha rulebook and returns a summary string.
func (p *Planner) summarizeAlpha(ref string) string {
	// Naive implementation: list file names in rulebooks/alpha/<ref>
	// Real implementation: parse manifest.json and schema files
	path := filepath.Join(p.baseDir, "rulebooks", "alpha", ref)
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Sprintf("Could not read Alpha rulebook at %s: %v", path, err)
	}
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Rulebook: %s\n", ref))
	for _, e := range entries {
		summary.WriteString(fmt.Sprintf("- %s\n", e.Name()))
	}
	return summary.String()
}

// summarizeBeta reads the Beta rulebook.
func (p *Planner) summarizeBeta(ref string) string {
	path := filepath.Join(p.baseDir, "rulebooks", "beta", ref)
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Sprintf("Could not read Beta rulebook at %s: %v", path, err)
	}
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Rulebook: %s\n", ref))
	for _, e := range entries {
		summary.WriteString(fmt.Sprintf("- %s\n", e.Name()))
	}
	return summary.String()
}
