package omega

import (
	"context"
	"fmt"
)

// DeterminismGate validates deterministic outputs
type DeterminismGate struct{}

func NewDeterminismGate() *DeterminismGate { return &DeterminismGate{} }

func (g *DeterminismGate) ID() string   { return "determinism" }
func (g *DeterminismGate) Name() string { return "Determinism Check" }

func (g *DeterminismGate) Run(ctx context.Context, req *GateRequest) []Defect {
	// This gate requires multiple runs with same inputs to validate
	// For now, we just ensure artifacts have hashes
	var defects []Defect

	for _, artifact := range req.Artifacts {
		if artifact.Hash == "" {
			d := NewDefect(g.ID(), DefectClassDeterminism,
				fmt.Sprintf("artifact missing hash: %s", artifact.ArtifactID))
			d.SuggestedFix = "Ensure the plugin computes and returns a SHA-256 hash for all artifacts."
			d.WithOffendingFields(map[string]string{"artifact_id": artifact.ArtifactID})
			defects = append(defects, *d)
		}
	}

	return defects
}
