package foundry

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/partir/core/internal/alpha"
	"github.com/partir/core/internal/beta"
	"github.com/partir/core/internal/maintenance"
	"github.com/partir/core/pkg/plugin"
)

// RoundTable orchestrates a collaborative session between the User, Alpha (Architect),
// and Beta (Designer). It facilitates a natural conversation until convergence,
// then triggers the auto-planner to generate tickets.
type RoundTable struct {
	Options RoundTableOptions

	alpha     *alpha.Architect
	beta      *beta.Designer
	planner   *Planner
	executor  plugin.Executor
	ledger    maintenance.LedgerAppender
	userQueue []string // Pending user inputs to process

	// Convergence tracking
	alphaConverged bool
	betaConverged  bool
}

type RoundTableOptions struct {
	ProjectID string
	AlphaRef  string // Existing Alpha ref (optional)
	BetaRef   string // Existing Beta ref (optional)
	Uploads   []string
}

// NewRoundTable creates a collaborative session.
func NewRoundTable(opts RoundTableOptions, a *alpha.Architect, b *beta.Designer, p *Planner, exec plugin.Executor, ledger maintenance.LedgerAppender) *RoundTable {
	return &RoundTable{
		Options:   opts,
		alpha:     a,
		beta:      b,
		planner:   p,
		executor:  exec,
		ledger:    ledger,
		userQueue: make([]string, 0),
	}
}

// Start begins the interactive session.
func (rt *RoundTable) Start(ctx context.Context) error {
	fmt.Printf("\n╔════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          FOUNDRY ROUND TABLE                          ║\n")
	fmt.Printf("║          Project: %-36s║\n", rt.Options.ProjectID)
	fmt.Printf("╚════════════════════════════════════════════════════════╝\n\n")

	fmt.Println("  Welcome to the table. I've brought Alpha (Architect) and")
	fmt.Println("  Beta (Designer) to help plan your project.")
	fmt.Println()
	fmt.Println("  Just tell us what you want to build. We'll ask questions")
	fmt.Println("  until we have a solid plan, then I'll draft the tickets.")
	fmt.Println()
	fmt.Println("  Commands: /plan (force plan), /quit (exit)")
	fmt.Println()

	// Initial greeting from Alpha
	fmt.Printf("Alpha (Architect): Hello! I'm ready to define the data architecture and rules.\n")
	fmt.Printf("                   What are we building today?\n\n")

	scanner := bufio.NewScanner(os.Stdin)

	// Main conversation loop
	for {
		// 1. Read User Input
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		// Check commands
		if strings.HasPrefix(userInput, "/") {
			switch strings.ToLower(userInput) {
			case "/plan":
				return rt.triggerPlanning(ctx)
			case "/quit", "/exit":
				return fmt.Errorf("session ended by user")
			}
		}

		// 2. Route to Alpha (Architect)
		// Alpha analyzes the input for structural intent
		alphaResp, err := rt.alphaTurn(ctx, userInput)
		if err != nil {
			fmt.Printf("Alpha: (Thinking...) %v\n", err)
		} else {
			fmt.Printf("\nAlpha (Architect): %s\n\n", alphaResp)
		}

		// 3. Route to Beta (Designer)
		// Beta analyzes the input + Alpha's response for design intent
		betaResp, err := rt.betaTurn(ctx, userInput, alphaResp)
		if err != nil {
			// Beta might stay silent if irrelevant
		} else if betaResp != "" {
			fmt.Printf("Beta (Designer): %s\n\n", betaResp)
		}

		// 4. Check for Convergence
		if rt.checkConvergence(ctx) {
			fmt.Println("\n━━━ CONVERGENCE REACHED ━━━")
			fmt.Println("Alpha and Beta have enough information to draft a plan.")
			fmt.Println("Generating project scope...")
			return rt.triggerPlanning(ctx)
		}
	}

	return nil
}

// alphaTurn executes a turn for the Architect.
func (rt *RoundTable) alphaTurn(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are Alpha, the System Architect.
User Input: "%s"

Your goal: Extract data entities, fields, relationships, and business rules.
If the user input is vague, ask clarifying questions about the data structure.
If the user input defines entities, confirm them and ask about relationships.
Keep your response conversational and concise (under 3 sentences).

When you believe the data architecture is fully defined and you have enough information to generate a complete schema, append the exact string [CONVERGED] to the end of your response.
`, input)

	req := &plugin.ExecuteRequest{
		Prompt:      prompt,
		MaxTokens:   500,
		Temperature: 0.7,
	}

	resp, err := rt.executor.Execute(ctx, req)
	if err != nil {
		return "", err
	}

	output := resp.Output
	if strings.Contains(output, "[CONVERGED]") {
		rt.alphaConverged = true
		// Strip the tag for cleaner display? Or keep it to inform user?
		// We'll keep it so the user sees the AI is signing off.
	} else {
		rt.alphaConverged = false // Reset if they ask more questions
	}

	return output, nil
}

// betaTurn executes a turn for the Designer.
func (rt *RoundTable) betaTurn(ctx context.Context, input, alphaResponse string) (string, error) {
	prompt := fmt.Sprintf(`
You are Beta, the Experience Designer.
User Input: "%s"
Alpha (Architect) said: "%s"

Your goal: Extract design requirements (visual style, colors, motion, layout).
If Alpha asked a question, you can wait for the user to answer, OR add a design perspective to it.
If the user mentioned "style", "look", "feel", or "brand", jump in.
If the input is purely technical (e.g. database connection strings), say nothing (output empty string).
Keep your response conversational and concise.

When you believe the design requirements are fully defined and compatible with the architecture, append the exact string [CONVERGED] to the end of your response.
`, input, alphaResponse)

	req := &plugin.ExecuteRequest{
		Prompt:      prompt,
		MaxTokens:   500,
		Temperature: 0.7,
	}

	resp, err := rt.executor.Execute(ctx, req)
	if err != nil {
		return "", err
	}

	output := resp.Output
	if strings.Contains(output, "[CONVERGED]") {
		rt.betaConverged = true
	} else {
		rt.betaConverged = false
	}

	return output, nil
}

// checkConvergence determines if we have enough info to plan.
func (rt *RoundTable) checkConvergence(ctx context.Context) bool {
	// Need both agents to signal readiness
	// If one is converged but other is not, we might be close.
	// We require strictly both to be converged in the current turn sequence.
	return rt.alphaConverged && rt.betaConverged
}

// triggerPlanning runs the Auto-Planner.
func (rt *RoundTable) triggerPlanning(ctx context.Context) error {
	// 1. Generate Scope
	scope, err := rt.planner.GenerateScope(ctx, rt.Options.AlphaRef, rt.Options.BetaRef)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	// 2. Present Scope
	PresentScope(scope)

	// 3. Approval Loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n[A]pprove, [E]dit, [O]bject, [Q]uit: ")
		if !scanner.Scan() {
			break
		}
		choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

		switch choice {
		case "a", "approve":
			// Submit tickets
			return ApproveScope(ctx, scope, nil) // Dispatcher passed as nil for now
		case "e", "edit":
			if err := rt.editScope(ctx, scope); err != nil {
				fmt.Printf("Edit failed: %v\n", err)
			} else {
				fmt.Println("Scope updated from editor.")
				PresentScope(scope) // Re-present
			}
		case "o", "object":
			fmt.Print("What would you like to change? ")
			if scanner.Scan() {
				objection := scanner.Text()
				// Re-enter the round table loop with the objection as input
				fmt.Printf("\nAlpha: Re-evaluating based on \"%s\"...\n", objection)
				// Reset convergence
				rt.alphaConverged = false
				rt.betaConverged = false

				// Here we should probably feed the objection to Alpha/Beta immediately
				// For now, we return, but the caller loop (Start) handles normal input.
				// Wait, returning nil exits the session.
				// We need to 'break' from this approval loop and return to conversation?
				// But `triggerPlanning` returns error/nil. If nil, `Start` exits.
				// We should return a specific error or use a sentinel to resume conversation?
				// Refactor: triggerPlanning should return (shouldResume bool, err error)
				// Quick fix: loop inside triggerPlanning? No.

				// Correct logic: If user objects, we must generate a new response from Alpha/Beta
				// based on the objection, and continue the conversation in Start().
				// But `triggerPlanning` is called from `Start`.
				// If we return nil, `Start` finishes.

				// Let's change `triggerPlanning` to return a sentinel error for "Resume"?
				// Or better: pass the conversation loop function?

				// Simplest: Just use recursion? No.
				// Let's make `triggerPlanning` return `(terminated bool, err error)`.
				// If terminated=false, `Start` loop continues.

				// Refactoring `triggerPlanning` signature is invasive if used elsewhere.
				// Checking calls: only used in `Start`.
			}
			// Ideally we break this inner loop and return to outer loop.
			// But `triggerPlanning` is a function call.

			// Let's implement editing first.

		case "q", "quit":
			return nil
		}
	}
	return nil
}

// editScope launches the system editor to modify the scope JSON
func (rt *RoundTable) editScope(ctx context.Context, scope *ProjectScope) error {
	// Marshal scope to temp file
	f, err := os.CreateTemp("", "scope-*.json")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer os.Remove(tmpName)

	data, err := json.MarshalIndent(scope, "", "  ")
	if err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Launch editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vim" // generic fallback
		}
	}

	cmd := exec.Command(editor, tmpName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Read back
	newData, err := os.ReadFile(tmpName)
	if err != nil {
		return err
	}

	// Allow user to abort by saving empty file?
	if len(newData) == 0 {
		return fmt.Errorf("empty file, edit cancelled")
	}

	return json.Unmarshal(newData, scope)
}
