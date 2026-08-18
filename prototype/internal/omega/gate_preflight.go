package omega

import (
	"context"
	"fmt"
	"os"
	"runtime"
)

// PreflightGate runs checks before Foundry dispatch (Factory 0)
type PreflightGate struct {
	requiredEnvVars []string
	greenTagFile    string
}

// NewPreflightGate creates a new preflight gate
func NewPreflightGate() *PreflightGate {
	return &PreflightGate{
		requiredEnvVars: []string{"PARTIR_CORE_VERSION"},
		greenTagFile:    ".green_tag",
	}
}

func (g *PreflightGate) ID() string   { return "preflight" }
func (g *PreflightGate) Name() string { return "Preflight Checks" }

func (g *PreflightGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// 1. Environment check (Green Tag)
	// In production, check for .green_tag file or env var
	if os.Getenv("PARTIR_GREEN_TAG") == "" {
		// This is a soft check - just warn in dev mode
		// In production mode (PARTIR_ENV=production), this would be a hard fail
		if os.Getenv("PARTIR_ENV") == "production" {
			d := NewDefect(g.ID(), DefectClassPreflight,
				"Green Tag required for production execution")
			d.SuggestedFix = "Ensure the PARTIR_GREEN_TAG environment variable is set to a valid Postgres version (e.g., 16.4)."
			defects = append(defects, *d)
		}
	}

	// 2. OS/shell check
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		d := NewDefect(g.ID(), DefectClassPreflight,
			fmt.Sprintf("unsupported OS: %s", runtime.GOOS))
		d.SuggestedFix = "Run Partir Core on Linux, Darwin, or Windows."
		defects = append(defects, *d)
	}

	// 3. Ticket completeness
	if req.AlphaRef == "" {
		d := NewDefect(g.ID(), DefectClassPreflight, "alpha_ref is required")
		d.SuggestedFix = "Provide an 'alpha_ref' in the ticket (e.g., 'project-a@0.1.0')."
		defects = append(defects, *d)
	}

	if req.PluginID == "" {
		d := NewDefect(g.ID(), DefectClassPreflight, "plugin_id is required")
		d.SuggestedFix = "Provide a 'plugin_id' in the ticket matching a registered plugin."
		defects = append(defects, *d)
	}

	// 4. Input validation (schema check against plugin's input schema)
	// This would be expanded with actual schema validation
	if len(req.Inputs) == 0 || string(req.Inputs) == "null" {
		d := NewDefect(g.ID(), DefectClassPreflight, "inputs are required")
		d.WithSuggestedFix("provide inputs JSON in ticket")
		defects = append(defects, *d)
	}

	return defects
}

// CompatibilityGate checks plugin ↔ rulebook compatibility
type CompatibilityGate struct {
	pluginVersions map[string]string // pluginID -> requiredAlphaVersion
}

// NewCompatibilityGate creates a compatibility gate
func NewCompatibilityGate() *CompatibilityGate {
	return &CompatibilityGate{
		pluginVersions: make(map[string]string),
	}
}

func (g *CompatibilityGate) ID() string   { return "compatibility" }
func (g *CompatibilityGate) Name() string { return "Compatibility Check" }

// RegisterPlugin registers a plugin's compatibility requirements
func (g *CompatibilityGate) RegisterPlugin(pluginID, requiredAlphaVersion string) {
	g.pluginVersions[pluginID] = requiredAlphaVersion
}

func (g *CompatibilityGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// Check if plugin has compatibility requirements
	if requiredVersion, ok := g.pluginVersions[req.PluginID]; ok {
		// Parse alpha_ref to extract version
		// Format: "name@version"
		// For now, just check it's provided
		if req.AlphaRef == "" {
			d := NewDefect(g.ID(), DefectClassCompat,
				fmt.Sprintf("plugin %s requires alpha rulebook version %s", req.PluginID, requiredVersion))
			d.SuggestedFix = fmt.Sprintf("Use 'alpha promote' to create version %s, or update the plugin requirements.", requiredVersion)
			defects = append(defects, *d)
		}
	}

	return defects
}
