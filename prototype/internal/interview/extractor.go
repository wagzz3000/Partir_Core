package interview

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractedBlock represents a structured output block extracted from LLM output.
// LLM is instructed to wrap generated artifacts in tagged code blocks like:
//
//	```alpha:schema:product
//	{ "title": "Product", ... }
//	```
type ExtractedBlock struct {
	// Module is the source module: "alpha" or "beta"
	Module string `json:"module"`

	// Category is the artifact type: "schema", "intent", "palette", "effects", etc.
	Category string `json:"category"`

	// Name is the artifact name (optional, may be empty for singleton types)
	Name string `json:"name"`

	// Content is the raw JSON/YAML content of the block
	Content string `json:"content"`
}

// tagPattern matches tagged code blocks in LLM output.
// Format: ```module:category:name or ```module:category
//
// Examples:
//
//	```alpha:schema:product
//	```beta:palette
//	```alpha:rules
var tagPattern = regexp.MustCompile("(?s)```(alpha|beta):([a-z_]+)(?::([a-z_]+))?\\s*\\n(.*?)```")

// ExtractTaggedBlocks finds all tagged code blocks in the LLM output.
func ExtractTaggedBlocks(output string) []ExtractedBlock {
	if output == "" {
		return nil
	}

	matches := tagPattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}

	blocks := make([]ExtractedBlock, 0, len(matches))
	for _, match := range matches {
		block := ExtractedBlock{
			Module:   match[1],
			Category: match[2],
			Name:     match[3], // may be empty
			Content:  strings.TrimSpace(match[4]),
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// WriteExtractedBlock writes an extracted block to the appropriate file in the workspace.
// File placement follows the workspace convention:
//
//	alpha:schema:product   → schemas/product.json
//	alpha:rules            → rules/rules.json
//	alpha:intent           → intent.json
//	alpha:connections      → connections.json
//	beta:palette           → render_vocab_palettes.json
//	beta:effects           → render_vocab_effects.json
//	beta:animations        → render_vocab_animations.json
//	beta:style_rules       → style_rules.json
//	beta:brief             → design_brief.json
func WriteExtractedBlock(projectDir string, block ExtractedBlock) error {
	var relPath string

	switch block.Module {
	case "alpha":
		switch block.Category {
		case "schema":
			name := block.Name
			if name == "" {
				name = "unnamed"
			}
			relPath = filepath.Join("schemas", name+".json")
		case "rules":
			relPath = filepath.Join("rules", "rules.json")
		case "intent":
			relPath = "intent.json"
		case "connections":
			relPath = "connections.json"
		case "entities":
			relPath = "entities.json"
		case "rulebook":
			relPath = "rulebook.json"
		default:
			relPath = block.Category + ".json"
		}
	case "beta":
		switch block.Category {
		case "palette":
			relPath = "render_vocab_palettes.json"
		case "effects":
			relPath = "render_vocab_effects.json"
		case "animations":
			relPath = "render_vocab_animations.json"
		case "style_rules":
			relPath = "style_rules.json"
		case "brief":
			relPath = "design_brief.json"
		default:
			relPath = block.Category + ".json"
		}
	default:
		return fmt.Errorf("unknown module: %s", block.Module)
	}

	fullPath := filepath.Join(projectDir, relPath)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(block.Content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}

	return nil
}
