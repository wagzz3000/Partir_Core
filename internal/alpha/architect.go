// Package alpha - architect.go provides the interactive Alpha Architect.
//
// The Architect runs a 6-phase interview to convert a user's idea into a
// complete Alpha Rulebook. All schemas, catalogs, and rules are generated
// by Alpha — the user never writes JSON by hand.
//
// Phases:
//  1. Project Intent — scope, scale, industry, compliance, budget
//  2. Upload & Discover — process uploaded docs/diagrams/code
//  3. Data Modeling — generate JSON Schema from conversation
//  4. Connections — relationships between data models
//  5. Business Rules — invariants, combinations, constraints
//  6. Review & Validate — present final rulebook, validate tech stability
package alpha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/partir/core/internal/interview"
	"github.com/partir/core/pkg/plugin"
)

// Architect orchestrates the interactive architecture interview.
type Architect struct {
	engine    *interview.Engine
	workspace *Workspace
	intake    *interview.IntakeProcessor
	project   string
}

// ArchitectConfig holds configuration for creating a new Architect.
type ArchitectConfig struct {
	Workspace *Workspace
	Executor  plugin.Executor
	Assembly  *interview.PairAssembly
	GraphName string
	Graph     interview.GraphClient
	Ledger    interview.LedgerAppender
}

// NewArchitect creates a new Alpha Architect.
func NewArchitect(project string, cfg ArchitectConfig) *Architect {
	projectDir := filepath.Join(cfg.Workspace.BaseDir, project)

	engine := interview.New(interview.Config{
		Executor:   cfg.Executor,
		Assembly:   cfg.Assembly,
		ActivePair: cfg.Assembly.ActiveAlpha(),
		GraphName:  cfg.GraphName,
		Graph:      cfg.Graph,
		Ledger:     cfg.Ledger,
		ProjectDir: projectDir,
	})

	return &Architect{
		engine:    engine,
		workspace: cfg.Workspace,
		intake:    interview.NewIntakeProcessor(cfg.Executor),
		project:   project,
	}
}

// Start runs the full interactive architecture session.
// It initializes the workspace, runs all 6 phases, and produces a Rulebook.
func (a *Architect) Start(ctx context.Context) (*Rulebook, error) {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║          ALPHA — Architecture Architect               ║")
	fmt.Printf("║          Project: %-36s ║\n", a.project)
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  I'm Alpha. I'll help you design the architecture for")
	fmt.Println("  your project. No technical knowledge required — just")
	fmt.Println("  tell me about your idea.")
	fmt.Println()
	fmt.Println("  Commands: /done (next phase), /quit (exit), /history, /decisions")
	fmt.Println()

	// Initialize workspace if it doesn't exist
	if !a.workspace.Exists(a.project) {
		if _, err := a.workspace.Init(a.project); err != nil {
			return nil, fmt.Errorf("init workspace: %w", err)
		}
	}

	// Run all phases
	phases := interview.AlphaPhases()
	for _, phase := range phases {
		result, err := a.engine.RunPhase(ctx, phase)
		if err != nil {
			return nil, fmt.Errorf("phase %s failed: %w", phase.ID, err)
		}

		if result != nil && len(result.Extracted) > 0 {
			log.Printf("[alpha] Phase %s produced %d artifacts", phase.ID, len(result.Extracted))
		}
	}

	// Load the generated workspace
	rb, err := a.workspace.Load(a.project)
	if err != nil {
		log.Printf("[alpha] Warning: could not load final rulebook: %v", err)
		// Return a blank rulebook — the files were written to disk
		return NewRulebook(a.project), nil
	}

	fmt.Println()
	fmt.Println("━━━ Architecture Complete ━━━")
	fmt.Printf("  Rulebook: %s\n", a.project)
	fmt.Printf("  Schemas:  %d\n", len(rb.Schemas))
	fmt.Printf("  Rules:    %d invariants, %d combinations\n",
		len(rb.Rules.Invariants), len(rb.Rules.Combinations))
	fmt.Println()

	return rb, nil
}

// StartWithUpload runs the session with an initial file upload.
func (a *Architect) StartWithUpload(ctx context.Context, uploadPath string) (*Rulebook, error) {
	// Process the uploaded file first
	result, err := a.intake.ProcessFile(ctx, uploadPath)
	if err != nil {
		return nil, fmt.Errorf("process upload %s: %w", uploadPath, err)
	}

	fmt.Printf("📄 Processed: %s (%s)\n", filepath.Base(uploadPath), result.FileType)
	fmt.Printf("   Summary: %s\n\n", result.Summary)

	// Save the intake result for the session
	intakePath := filepath.Join(a.workspace.BaseDir, a.project, "intake_"+filepath.Base(uploadPath)+".json")
	if err := writeJSON(intakePath, result); err != nil {
		log.Printf("[alpha] Warning: could not save intake result: %v", err)
	}

	return a.Start(ctx)
}

// Resume re-enters a previous session using graph memory.
func (a *Architect) Resume(ctx context.Context) (*Rulebook, error) {
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║          ALPHA — Resuming Session                     ║")
	fmt.Printf("║          Project: %-36s ║\n", a.project)
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	if err := a.engine.Resume(ctx); err != nil {
		return nil, fmt.Errorf("resume session: %w", err)
	}

	return a.Start(ctx)
}

// writeJSON writes data as indented JSON to a file.
func writeJSON(path string, data interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
