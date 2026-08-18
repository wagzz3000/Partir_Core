package omega

import (
	"context"
	"fmt"
)

// UniquenessGate validates artifact uniqueness
type UniquenessGate struct {
	existsFunc func(ctx context.Context, hash string) (bool, error)
}

// NewUniquenessGate creates a new uniqueness gate
func NewUniquenessGate(existsFunc func(ctx context.Context, hash string) (bool, error)) *UniquenessGate {
	return &UniquenessGate{existsFunc: existsFunc}
}

func (g *UniquenessGate) ID() string   { return "uniqueness" }
func (g *UniquenessGate) Name() string { return "Uniqueness Check" }

func (g *UniquenessGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// Check batch uniqueness (no duplicates in the same batch)
	seen := make(map[string]bool)
	for _, artifact := range req.Artifacts {
		if seen[artifact.Hash] {
			defects = append(defects, *NewDefect(g.ID(), DefectClassUniqueness,
				fmt.Sprintf("duplicate artifact in batch: %s", artifact.Hash)))
		}
		seen[artifact.Hash] = true
	}

	// Check catalog uniqueness (not already in registry)
	for _, artifact := range req.Artifacts {
		exists, err := g.existsFunc(ctx, artifact.Hash)
		if err != nil {
			defects = append(defects, *NewDefect(g.ID(), DefectClassUniqueness,
				fmt.Sprintf("failed to check uniqueness: %v", err)))
			continue
		}
		if exists {
			d := NewDefect(g.ID(), DefectClassUniqueness,
				fmt.Sprintf("artifact already exists: %s", artifact.Hash))
			d.SuggestedFix = "Modify the input or seed to generate a unique variant, or accept the existing artifact."
			d.WithOffendingFields(map[string]string{"hash": artifact.Hash})
			defects = append(defects, *d)
		}
	}

	return defects
}
