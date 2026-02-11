package alpha

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// LintResult represents the result of linting a rulebook
type LintResult struct {
	Valid    bool
	Errors   []LintError
	Warnings []LintWarning
}

// LintError represents a lint error
type LintError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// LintWarning represents a lint warning
type LintWarning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Linter validates rulebook content
type Linter struct {
	compiler *jsonschema.Compiler
}

// NewLinter creates a new linter
func NewLinter() *Linter {
	return &Linter{
		compiler: jsonschema.NewCompiler(),
	}
}

// Lint validates a rulebook
func (l *Linter) Lint(rb *Rulebook) *LintResult {
	result := &LintResult{Valid: true}

	// Check manifest
	if rb.Manifest.Name == "" {
		result.Errors = append(result.Errors, LintError{
			Path:    "manifest.name",
			Message: "name is required",
		})
		result.Valid = false
	}

	if rb.Manifest.CoreVersion == "" {
		result.Errors = append(result.Errors, LintError{
			Path:    "manifest.core_version",
			Message: "core_version is required",
		})
		result.Valid = false
	}

	// Check compat
	if rb.Compat.MinCoreVersion == "" {
		result.Errors = append(result.Errors, LintError{
			Path:    "compat.min_core_version",
			Message: "min_core_version is required",
		})
		result.Valid = false
	}

	// Validate schemas are valid JSON Schema
	for name, schema := range rb.Schemas {
		if err := l.validateSchema(name, schema); err != nil {
			result.Errors = append(result.Errors, LintError{
				Path:    fmt.Sprintf("schemas.%s", name),
				Message: err.Error(),
			})
			result.Valid = false
		}
	}

	// Check catalogs have required fields
	for catalogName, entries := range rb.Catalogs {
		for i, entry := range entries {
			if entry.ID == "" {
				result.Errors = append(result.Errors, LintError{
					Path:    fmt.Sprintf("catalogs.%s[%d].id", catalogName, i),
					Message: "catalog entry id is required",
				})
				result.Valid = false
			}
			if entry.Type == "" {
				result.Errors = append(result.Errors, LintError{
					Path:    fmt.Sprintf("catalogs.%s[%d].type", catalogName, i),
					Message: "catalog entry type is required",
				})
				result.Valid = false
			}
		}
	}

	// Check rules for circular dependencies
	if err := l.checkRuleDependencies(rb.Rules); err != nil {
		result.Errors = append(result.Errors, LintError{
			Path:    "rules",
			Message: err.Error(),
		})
		result.Valid = false
	}

	// Warnings
	if len(rb.Schemas) == 0 {
		result.Warnings = append(result.Warnings, LintWarning{
			Path:    "schemas",
			Message: "no schemas defined",
		})
	}

	if len(rb.Catalogs) == 0 {
		result.Warnings = append(result.Warnings, LintWarning{
			Path:    "catalogs",
			Message: "no catalogs defined",
		})
	}

	return result
}

// validateSchema checks if a schema is valid JSON Schema
func (l *Linter) validateSchema(name string, schema interface{}) error {
	// For now, just check it's a valid object
	if schema == nil {
		return fmt.Errorf("schema is nil")
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must be an object")
	}

	// Check for $schema or type
	if _, hasSchema := schemaMap["$schema"]; !hasSchema {
		if _, hasType := schemaMap["type"]; !hasType {
			return fmt.Errorf("schema should have $schema or type property")
		}
	}

	return nil
}

// checkRuleDependencies checks for circular dependencies in rules
func (l *Linter) checkRuleDependencies(rules Rules) error {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, rule := range rules.Combinations {
		deps[rule.ID] = rule.Requires
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(id string) bool
	hasCycle = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		for _, dep := range deps[id] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for id := range deps {
		if !visited[id] {
			if hasCycle(id) {
				return fmt.Errorf("circular dependency detected involving rule: %s", id)
			}
		}
	}

	return nil
}
