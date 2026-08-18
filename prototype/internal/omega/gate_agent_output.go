package omega

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/partir/core/pkg/plugin"
)

// AgentOutputGate validates the AgentResult from LLM plugins
type AgentOutputGate struct{}

// NewAgentOutputGate creates a new agent output validation gate
func NewAgentOutputGate() *AgentOutputGate {
	return &AgentOutputGate{}
}

// ID implements Gate
func (g *AgentOutputGate) ID() string {
	return "agent_output"
}

// Run validates the AgentResult against schema and policy requirements
func (g *AgentOutputGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// Check if agent result is present
	if req.AgentResult == nil {
		defects = append(defects, Defect{
			ID:       "agent_output_missing",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "AgentResult is missing from request",
			Field:    "agent_result",
		})
		return defects
	}

	result := req.AgentResult

	// 1. Validate required fields
	if result.TicketID == "" {
		defects = append(defects, Defect{
			ID:       "missing_ticket_id",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "ticket_id is required",
			Field:    "ticket_id",
		})
	}

	if result.RunID == "" {
		defects = append(defects, Defect{
			ID:       "missing_run_id",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "run_id is required",
			Field:    "run_id",
		})
	}

	// 2. Validate LLM telemetry
	defects = append(defects, g.validateTelemetry(&result.Telemetry)...)

	// 3. Validate artifacts if present
	for i, artifact := range result.Artifacts {
		defects = append(defects, g.validateArtifact(i, &artifact)...)
	}

	// 4. Validate patch if present
	if result.Patch != nil {
		defects = append(defects, g.validatePatch(result.Patch)...)
	}

	// 5. Validate defects are properly structured
	for i, defect := range result.Defects {
		if defect.Class == "" {
			defects = append(defects, Defect{
				ID:       fmt.Sprintf("defect_%d_missing_class", i),
				Class:    plugin.DefectClassSchema,
				Severity: plugin.DefectSeverityWarning,
				Message:  fmt.Sprintf("Defect at index %d missing class", i),
				Field:    fmt.Sprintf("defects[%d].class", i),
			})
		}
	}

	return defects
}

func (g *AgentOutputGate) validateTelemetry(t *plugin.LLMTelemetry) []Defect {
	var defects []Defect

	if t.Provider == "" {
		defects = append(defects, Defect{
			ID:       "missing_provider",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "llm_telemetry.provider is required",
			Field:    "llm_telemetry.provider",
		})
	}

	if t.Model == "" {
		defects = append(defects, Defect{
			ID:       "missing_model",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "llm_telemetry.model is required",
			Field:    "llm_telemetry.model",
		})
	}

	// Tokens should be non-negative
	if t.TokensIn < 0 || t.TokensOut < 0 {
		defects = append(defects, Defect{
			ID:       "invalid_token_count",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "Token counts cannot be negative",
			Field:    "llm_telemetry.tokens_in/out",
		})
	}

	// Cost should be non-negative
	if t.CostUSD < 0 {
		defects = append(defects, Defect{
			ID:       "invalid_cost",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  "cost_usd cannot be negative",
			Field:    "llm_telemetry.cost_usd",
		})
	}

	return defects
}

func (g *AgentOutputGate) validateArtifact(idx int, a *plugin.AgentArtifact) []Defect {
	var defects []Defect

	if a.Path == "" {
		defects = append(defects, Defect{
			ID:       fmt.Sprintf("artifact_%d_missing_path", idx),
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityError,
			Message:  fmt.Sprintf("Artifact at index %d missing path", idx),
			Field:    fmt.Sprintf("artifacts[%d].path", idx),
		})
	}

	if a.Type == "" {
		defects = append(defects, Defect{
			ID:       fmt.Sprintf("artifact_%d_missing_type", idx),
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityWarning,
			Message:  fmt.Sprintf("Artifact at index %d missing type", idx),
			Field:    fmt.Sprintf("artifacts[%d].type", idx),
		})
	}

	// Hash is recommended for determinism checks
	if a.Hash == "" {
		defects = append(defects, Defect{
			ID:       fmt.Sprintf("artifact_%d_missing_hash", idx),
			Class:    plugin.DefectClassDeterminism,
			Severity: plugin.DefectSeverityInfo,
			Message:  fmt.Sprintf("Artifact at index %d missing hash (recommended for determinism)", idx),
			Field:    fmt.Sprintf("artifacts[%d].hash", idx),
		})
	}

	return defects
}

func (g *AgentOutputGate) validatePatch(p *plugin.AgentPatch) []Defect {
	var defects []Defect

	if p.Format == "" {
		defects = append(defects, Defect{
			ID:       "patch_missing_format",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityWarning,
			Message:  "patch.format is recommended (unified, git)",
			Field:    "patch.format",
		})
	}

	if p.Content == "" {
		defects = append(defects, Defect{
			ID:       "patch_empty_content",
			Class:    plugin.DefectClassSchema,
			Severity: plugin.DefectSeverityWarning,
			Message:  "patch.content is empty",
			Field:    "patch.content",
		})
	}

	return defects
}

// Ensure AgentOutputGate implements Gate
var _ Gate = (*AgentOutputGate)(nil)

// ParseAgentResult parses JSON into AgentResult
func ParseAgentResult(data []byte) (*plugin.AgentResult, error) {
	var result plugin.AgentResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse agent result: %w", err)
	}
	return &result, nil
}
