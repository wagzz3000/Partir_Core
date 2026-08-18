// Package interview provides the shared conversation engine used by
// Alpha (architect) and Beta (designer) interactive sessions.
//
// The engine runs multi-turn LLM conversations backed by:
//   - Chrono-graph memory: each conversation turn stored in the active pair's FalkorDB graph
//   - Org Ledger: every decision logged for audit and cross-pair awareness
//   - PairAssembly: 12 memory pairs per project (4A + 4B + 4O)
//
// On project init, 1 pair is active and 3 are on hot standby.
// Sleep and maintenance states emerge organically as the project matures.
package interview

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/partir/core/pkg/plugin"
)

// LedgerAppender is the interface for writing audit events to the Factory Ledger.
type LedgerAppender interface {
	Append(ctx context.Context, eventType string, payload map[string]interface{}) error
}

// GraphClient provides chrono-graph (FalkorDB) operations for conversation memory.
// Each pair has its own graph namespace, ensuring project-level data isolation.
type GraphClient interface {
	// StoreMessage persists a conversation turn to the pair's graph
	StoreMessage(ctx context.Context, graphName string, msg Message) error

	// QueryHistory retrieves recent conversation history from the pair's graph
	QueryHistory(ctx context.Context, graphName string, limit int) ([]Message, error)

	// QueryDecisions retrieves past decisions from the pair's graph
	QueryDecisions(ctx context.Context, graphName string) ([]Decision, error)

	// StoreDecision persists an architectural/design decision to the graph
	StoreDecision(ctx context.Context, graphName string, dec Decision) error

	// QueryPeerGraph queries another pair's graph for expertise (within same project only)
	QueryPeerGraph(ctx context.Context, peerGraphName, cypherQuery string) ([]interface{}, error)
}

// Message represents a single conversation turn.
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "system", "user", "assistant"
	Content   string    `json:"content"`
	Phase     string    `json:"phase"` // which interview phase this belongs to
	Timestamp time.Time `json:"timestamp"`
}

// Decision represents an architectural or design decision made during the interview.
type Decision struct {
	ID        string                 `json:"id"`
	Phase     string                 `json:"phase"`
	Summary   string                 `json:"summary"`
	Rationale string                 `json:"rationale"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// PairAssembly represents the 12 memory pairs assigned to a project.
// On init: 1 active + 3 hot standby per role. Sleep/maintenance emerge over time.
type PairAssembly struct {
	ProjectID string    `json:"project_id"`
	Alpha     [4]string `json:"alpha"` // A1 (active), A2 (hot), A3 (hot), A4 (hot)
	Beta      [4]string `json:"beta"`  // B1 (active), B2 (hot), B3 (hot), B4 (hot)
	Omega     [4]string `json:"omega"` // O1 (active), O2 (hot), O3 (hot), O4 (hot)
}

// ActiveAlpha returns the active Alpha pair ID.
func (pa *PairAssembly) ActiveAlpha() string { return pa.Alpha[0] }

// ActiveBeta returns the active Beta pair ID.
func (pa *PairAssembly) ActiveBeta() string { return pa.Beta[0] }

// ActiveOmega returns the active Omega pair ID.
func (pa *PairAssembly) ActiveOmega() string { return pa.Omega[0] }

// AllPairs returns all 12 pair IDs in the assembly.
func (pa *PairAssembly) AllPairs() []string {
	pairs := make([]string, 0, 12)
	for _, p := range pa.Alpha {
		pairs = append(pairs, p)
	}
	for _, p := range pa.Beta {
		pairs = append(pairs, p)
	}
	for _, p := range pa.Omega {
		pairs = append(pairs, p)
	}
	return pairs
}

// Engine is the multi-turn LLM conversation loop shared by Alpha and Beta.
// It manages conversation state, graph memory, ledger audit, and phase progression.
type Engine struct {
	executor   plugin.Executor
	assembly   *PairAssembly
	activePair string // currently active pair ID (e.g., "A1" or "B1")
	graphName  string // FalkorDB graph name scoped to the active pair
	graph      GraphClient
	ledger     LedgerAppender
	projectDir string // workspace directory for generated files

	// Session state
	systemPrompt string
	history      []Message
	currentPhase string
}

// Config holds configuration for creating a new Engine.
type Config struct {
	Executor   plugin.Executor
	Assembly   *PairAssembly
	ActivePair string
	GraphName  string
	Graph      GraphClient
	Ledger     LedgerAppender
	ProjectDir string
}

// New creates a new conversation engine.
func New(cfg Config) *Engine {
	return &Engine{
		executor:   cfg.Executor,
		assembly:   cfg.Assembly,
		activePair: cfg.ActivePair,
		graphName:  cfg.GraphName,
		graph:      cfg.Graph,
		ledger:     cfg.Ledger,
		projectDir: cfg.ProjectDir,
		history:    make([]Message, 0),
	}
}

// RunPhase executes a single interview phase interactively.
// It sets the system prompt, asks starter questions, and loops on user input
// until the phase produces a result or the user moves on.
func (e *Engine) RunPhase(ctx context.Context, phase Phase) (*PhaseResult, error) {
	e.currentPhase = phase.ID
	e.systemPrompt = phase.SystemPrompt

	// Log phase start to ledger
	if e.ledger != nil {
		e.ledger.Append(ctx, "phase_start", map[string]interface{}{
			"project_id": e.assembly.ProjectID,
			"pair_id":    e.activePair,
			"phase":      phase.ID,
			"phase_name": phase.Name,
		})
	}

	fmt.Printf("\n━━━ %s ━━━\n\n", phase.Name)

	// If resuming, load past decisions for context
	if e.graph != nil {
		decisions, err := e.graph.QueryDecisions(ctx, e.graphName)
		if err == nil && len(decisions) > 0 {
			fmt.Printf("📋 Resuming with %d past decisions in memory.\n\n", len(decisions))
		}
	}

	// Send starter question via LLM
	starterPrompt := phase.StarterPrompt
	if starterPrompt == "" && len(phase.Questions) > 0 {
		starterPrompt = phase.Questions[0]
	}

	// Get LLM's opening — pass system prompt + context
	opening, err := e.llmTurn(ctx, starterPrompt)
	if err != nil {
		return nil, fmt.Errorf("phase %s opening: %w", phase.ID, err)
	}
	fmt.Printf("🤖 %s\n\n", opening)

	// Interactive loop: read user input → send to LLM → print response
	scanner := bufio.NewScanner(os.Stdin)
	var extractedOutput string

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break // EOF or Ctrl+C
		}
		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		// Special commands
		switch strings.ToLower(userInput) {
		case "/done", "/next":
			fmt.Println("\n✅ Phase complete. Moving on...")
			goto phaseComplete
		case "/quit", "/exit":
			return nil, fmt.Errorf("session ended by user")
		case "/history":
			e.printHistory()
			continue
		case "/decisions":
			e.printDecisions(ctx)
			continue
		}

		// Store user message in graph
		userMsg := Message{
			Role:      "user",
			Content:   userInput,
			Phase:     phase.ID,
			Timestamp: time.Now(),
		}
		e.history = append(e.history, userMsg)
		if e.graph != nil {
			e.graph.StoreMessage(ctx, e.graphName, userMsg)
		}

		// Send to LLM
		response, err := e.llmTurn(ctx, userInput)
		if err != nil {
			log.Printf("[interview] LLM error: %v", err)
			fmt.Println("⚠️  I had trouble processing that. Could you rephrase?")
			continue
		}

		// Store assistant response
		assistantMsg := Message{
			Role:      "assistant",
			Content:   response,
			Phase:     phase.ID,
			Timestamp: time.Now(),
		}
		e.history = append(e.history, assistantMsg)
		if e.graph != nil {
			e.graph.StoreMessage(ctx, e.graphName, assistantMsg)
		}

		fmt.Printf("\n🤖 %s\n\n", response)

		// Try to extract structured output
		extracted := ExtractTaggedBlocks(response)
		if len(extracted) > 0 {
			for _, block := range extracted {
				err := WriteExtractedBlock(e.projectDir, block)
				if err != nil {
					log.Printf("[interview] Failed to write extracted block: %v", err)
				} else {
					fmt.Printf("📄 Generated: %s/%s\n", block.Category, block.Name)
				}
			}
			extractedOutput = response

			// Store decision
			dec := Decision{
				Phase:     phase.ID,
				Summary:   fmt.Sprintf("Generated %s artifacts", phase.OutputType),
				Rationale: "User-approved during interactive session",
				Timestamp: time.Now(),
			}
			if e.graph != nil {
				e.graph.StoreDecision(ctx, e.graphName, dec)
			}
		}
	}

phaseComplete:
	// Log phase completion to ledger
	if e.ledger != nil {
		e.ledger.Append(ctx, "phase_complete", map[string]interface{}{
			"project_id": e.assembly.ProjectID,
			"pair_id":    e.activePair,
			"phase":      phase.ID,
			"has_output": extractedOutput != "",
		})
	}

	return &PhaseResult{
		PhaseID:   phase.ID,
		Completed: true,
		RawOutput: extractedOutput,
		Extracted: ExtractTaggedBlocks(extractedOutput),
	}, nil
}

// Resume re-enters a session using graph memory to recall past context.
func (e *Engine) Resume(ctx context.Context) error {
	if e.graph == nil {
		return fmt.Errorf("no graph client configured — cannot resume")
	}

	// Load conversation history from graph
	history, err := e.graph.QueryHistory(ctx, e.graphName, 100)
	if err != nil {
		return fmt.Errorf("failed to load conversation history: %w", err)
	}
	e.history = history

	// Load past decisions
	decisions, err := e.graph.QueryDecisions(ctx, e.graphName)
	if err != nil {
		log.Printf("[interview] Warning: could not load past decisions: %v", err)
	}

	fmt.Printf("📋 Resumed session: %d messages, %d decisions in memory.\n",
		len(history), len(decisions))

	return nil
}

// QueryPeer allows one pair to query another pair's graph for expertise.
// This is restricted to pairs within the same project (data isolation enforced).
func (e *Engine) QueryPeer(ctx context.Context, peerGraphName, query string) ([]interface{}, error) {
	if e.graph == nil {
		return nil, fmt.Errorf("no graph client configured")
	}

	// Log cross-pair query to ledger
	if e.ledger != nil {
		e.ledger.Append(ctx, "peer_query", map[string]interface{}{
			"project_id": e.assembly.ProjectID,
			"from_pair":  e.activePair,
			"to_graph":   peerGraphName,
			"query":      query,
		})
	}

	return e.graph.QueryPeerGraph(ctx, peerGraphName, query)
}

// llmTurn sends a message to the LLM and returns the response.
func (e *Engine) llmTurn(ctx context.Context, userMessage string) (string, error) {
	// Build the full prompt with history context
	prompt := e.buildPrompt(userMessage)

	resp, err := e.executor.Execute(ctx, &plugin.ExecuteRequest{
		TicketID:     e.assembly.ProjectID,
		RunID:        fmt.Sprintf("%s-%s-%d", e.activePair, e.currentPhase, time.Now().Unix()),
		Prompt:       prompt,
		SystemPrompt: e.systemPrompt,
		MaxTokens:    4096,
		Temperature:  0.7,
	})
	if err != nil {
		return "", fmt.Errorf("executor error: %w", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("LLM error: %s", resp.Error)
	}

	return resp.Output, nil
}

// buildPrompt constructs the full prompt including recent conversation history.
func (e *Engine) buildPrompt(currentMessage string) string {
	var sb strings.Builder

	// Include recent history (last 20 turns for context window management)
	start := 0
	if len(e.history) > 20 {
		start = len(e.history) - 20
	}
	for _, msg := range e.history[start:] {
		switch msg.Role {
		case "user":
			sb.WriteString("User: ")
		case "assistant":
			sb.WriteString("Assistant: ")
		}
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("User: ")
	sb.WriteString(currentMessage)
	return sb.String()
}

// printHistory displays the conversation history to the user.
func (e *Engine) printHistory() {
	fmt.Println("\n── Conversation History ──")
	for _, msg := range e.history {
		switch msg.Role {
		case "user":
			fmt.Printf("  You: %s\n", msg.Content)
		case "assistant":
			fmt.Printf("  🤖: %s\n", msg.Content)
		}
	}
	fmt.Println("── End History ──")
}

// printDecisions displays past decisions from the graph.
func (e *Engine) printDecisions(ctx context.Context) {
	if e.graph == nil {
		fmt.Println("  (no graph configured)")
		return
	}

	decisions, err := e.graph.QueryDecisions(ctx, e.graphName)
	if err != nil {
		fmt.Printf("  Error loading decisions: %v\n", err)
		return
	}

	fmt.Println("\n── Past Decisions ──")
	for _, dec := range decisions {
		fmt.Printf("  [%s] %s — %s\n", dec.Phase, dec.Summary, dec.Rationale)
	}
	if len(decisions) == 0 {
		fmt.Println("  (no decisions recorded yet)")
	}
	fmt.Println("── End Decisions ──")
}

// PhaseResult contains the output of a completed interview phase.
type PhaseResult struct {
	PhaseID   string           `json:"phase_id"`
	Completed bool             `json:"completed"`
	RawOutput string           `json:"raw_output"`
	Extracted []ExtractedBlock `json:"extracted,omitempty"`
}

// ToJSON serializes the phase result.
func (pr *PhaseResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(pr, "", "  ")
}
