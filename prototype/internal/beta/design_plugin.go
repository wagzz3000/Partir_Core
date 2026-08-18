package beta

import "context"

// DesignPlugin defines the interface for design tool integrations.
// Figma is the first implementation, but the architecture supports any
// design tool (Sketch, Penpot, Adobe XD, etc.).
//
// Plugins handle converting between Beta's RenderVocab (palettes, effects,
// animations) and the external tool's native format.
type DesignPlugin interface {
	// ID returns the plugin identifier (e.g., "figma", "sketch", "penpot")
	ID() string

	// Export creates a design file from the Beta rulebook.
	// Returns a reference to the created file (e.g., Figma file key, URL).
	Export(ctx context.Context, rb *Rulebook) (string, error)

	// Import reads a design file and extracts design tokens.
	// ref is the design tool-specific reference (e.g., Figma file key).
	Import(ctx context.Context, ref string) (*RenderVocab, error)

	// Sync re-reads the design file after user edits and returns what changed.
	// Beta can then ask the user about the changes:
	//   "I see you changed the primary color from blue to purple.
	//    Should I update all dependent palettes?"
	Sync(ctx context.Context, ref string, rb *Rulebook) (*SyncDiff, error)
}

// SyncDiff describes what changed in the external design tool
// since the last export or sync.
type SyncDiff struct {
	// ChangedPalettes lists palette IDs that were modified
	ChangedPalettes []string `json:"changed_palettes,omitempty"`

	// ChangedEffects lists effect IDs that were modified
	ChangedEffects []string `json:"changed_effects,omitempty"`

	// ChangedAnimations lists animation IDs that were modified
	ChangedAnimations []string `json:"changed_animations,omitempty"`

	// AddedTokens lists new design tokens found in the external tool
	AddedTokens []string `json:"added_tokens,omitempty"`

	// RemovedTokens lists tokens removed from the external tool
	RemovedTokens []string `json:"removed_tokens,omitempty"`

	// Summary is an LLM-generated summary of the changes
	Summary string `json:"summary"`
}
