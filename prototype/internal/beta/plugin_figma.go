package beta

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// FigmaPlugin is the Figma implementation of the DesignPlugin interface.
// It uses the Figma REST API to create, read, and sync design files.
type FigmaPlugin struct {
	apiToken string
	teamID   string // Figma team ID for file creation
}

// NewFigmaPlugin creates a new Figma design plugin.
func NewFigmaPlugin(apiToken, teamID string) *FigmaPlugin {
	return &FigmaPlugin{
		apiToken: apiToken,
		teamID:   teamID,
	}
}

// ID returns the plugin identifier.
func (f *FigmaPlugin) ID() string { return "figma" }

// Export creates a Figma file from the Beta rulebook.
// It generates a component library with the defined palettes, effects, and typography.
//
// Flow:
//  1. Create a new Figma file via API
//  2. Add color styles from palettes
//  3. Add text styles from typography
//  4. Add effect styles (shadows, blurs, etc.)
//  5. Return the file key for user access
//
// NOTE: Full Figma API integration requires the Figma Plugin API or
// Variables API (for design tokens). This is a foundation implementation
// that will be expanded as Figma's API capabilities grow.
func (f *FigmaPlugin) Export(ctx context.Context, rb *Rulebook) (string, error) {
	if f.apiToken == "" {
		return "", fmt.Errorf("figma API token required for export")
	}

	// Build the export structure
	export := figmaExport{
		Name:       rb.Manifest.Name + " — Design System",
		Palettes:   rb.RenderVocab.Palettes,
		Effects:    rb.RenderVocab.Effects,
		Animations: rb.RenderVocab.Animations,
	}

	// For now, serialize the export for review
	// Full Figma REST API integration is Phase 11.2
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export: %w", err)
	}

	log.Printf("[figma] Export prepared: %d palettes, %d effects, %d animations",
		len(export.Palettes), len(export.Effects), len(export.Animations))

	// TODO: POST to Figma API when Variables API is available
	// For now, return the serialized export as a reference
	_ = data
	fileKey := fmt.Sprintf("figma_%s_export", rb.Manifest.Name)

	log.Printf("[figma] Export file key: %s (stub — full API integration pending)", fileKey)
	return fileKey, nil
}

// Import reads a Figma file and extracts design tokens.
// It pulls colors, typography, spacing, effects from an existing design.
//
// Flow:
//  1. GET /v1/files/:file_key via Figma API
//  2. Parse document.children for style definitions
//  3. Extract color fills → Palettes
//  4. Extract text styles → Typography
//  5. Extract effect styles → Effects
func (f *FigmaPlugin) Import(ctx context.Context, fileKey string) (*RenderVocab, error) {
	if f.apiToken == "" {
		return nil, fmt.Errorf("figma API token required for import")
	}

	log.Printf("[figma] Importing from file: %s", fileKey)

	// TODO: GET https://api.figma.com/v1/files/{fileKey}
	// Parse the response to extract design tokens

	// Stub: return empty vocab
	vocab := &RenderVocab{
		Palettes:   []Palette{},
		Effects:    []Effect{},
		Animations: []Animation{},
	}

	log.Printf("[figma] Import complete (stub — full API integration pending)")
	return vocab, nil
}

// Sync re-reads a Figma file after user edits and compares with the current rulebook.
// Returns a SyncDiff describing what the user changed in Figma.
//
// Flow:
//  1. Import current state from Figma
//  2. Compare with current rulebook
//  3. Build SyncDiff with:
//     - Changed palettes (color differences)
//     - Changed effects (parameter differences)
//     - Added/removed tokens
//  4. Generate LLM summary of changes
func (f *FigmaPlugin) Sync(ctx context.Context, fileKey string, rb *Rulebook) (*SyncDiff, error) {
	if f.apiToken == "" {
		return nil, fmt.Errorf("figma API token required for sync")
	}

	log.Printf("[figma] Syncing changes from file: %s", fileKey)

	// Import current Figma state
	currentVocab, err := f.Import(ctx, fileKey)
	if err != nil {
		return nil, fmt.Errorf("import for sync: %w", err)
	}

	diff := &SyncDiff{}

	// Compare palettes
	existingPalettes := make(map[string]Palette)
	for _, p := range rb.RenderVocab.Palettes {
		existingPalettes[p.ID] = p
	}
	for _, p := range currentVocab.Palettes {
		if _, exists := existingPalettes[p.ID]; !exists {
			diff.AddedTokens = append(diff.AddedTokens, "palette:"+p.ID)
		} else {
			// Mark as changed (deep comparison would go here)
			diff.ChangedPalettes = append(diff.ChangedPalettes, p.ID)
		}
	}

	// Compare effects
	existingEffects := make(map[string]Effect)
	for _, e := range rb.RenderVocab.Effects {
		existingEffects[e.ID] = e
	}
	for _, e := range currentVocab.Effects {
		if _, exists := existingEffects[e.ID]; !exists {
			diff.AddedTokens = append(diff.AddedTokens, "effect:"+e.ID)
		} else {
			diff.ChangedEffects = append(diff.ChangedEffects, e.ID)
		}
	}

	totalChanges := len(diff.ChangedPalettes) + len(diff.ChangedEffects) +
		len(diff.ChangedAnimations) + len(diff.AddedTokens) + len(diff.RemovedTokens)

	if totalChanges == 0 {
		diff.Summary = "No changes detected in Figma."
	} else {
		diff.Summary = fmt.Sprintf("%d design tokens changed since last export.", totalChanges)
	}

	log.Printf("[figma] Sync complete: %s", diff.Summary)
	return diff, nil
}

// figmaExport is the internal structure for Figma export.
type figmaExport struct {
	Name       string      `json:"name"`
	Palettes   []Palette   `json:"palettes"`
	Effects    []Effect    `json:"effects"`
	Animations []Animation `json:"animations"`
}
