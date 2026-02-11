package ux

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSpinner(t *testing.T) {
	s := NewSpinner("Loading data")
	s.Start()
	time.Sleep(200 * time.Millisecond)
	s.Stop(true)
	// Just verify it doesn't panic
}

func TestProgressBar(t *testing.T) {
	pb := NewProgressBar("Processing", 5, 20)

	for i := 1; i <= 5; i++ {
		pb.Update(i)
	}
	// Just verify it doesn't panic
}

func TestSuggestedFix(t *testing.T) {
	// Just verify it doesn't panic
	SuggestedFix("Schema validation failed for field 'title'", "Add a 'title' field to your input JSON")
}

func TestContextError(t *testing.T) {
	// Just verify it doesn't panic
	ContextError(
		assert.AnError,
		"foundry dispatch → plugin.Execute()",
		"Check plugin logs for detailed error output",
	)
}

func TestStageMarkers(t *testing.T) {
	StageStart("Running preflight checks")
	StageDone("Running preflight checks", 150*time.Millisecond)
	StageFail("Loading plugin", assert.AnError)
}
