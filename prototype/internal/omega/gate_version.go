package omega

import (
	"context"
	"fmt"
)

// VersionCompatGate validates version compatibility
type VersionCompatGate struct {
	coreVersion string
}

// NewVersionCompatGate creates a new version compatibility gate
func NewVersionCompatGate(coreVersion string) *VersionCompatGate {
	return &VersionCompatGate{coreVersion: coreVersion}
}

func (g *VersionCompatGate) ID() string   { return "version_compat" }
func (g *VersionCompatGate) Name() string { return "Version Compatibility" }

func (g *VersionCompatGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// Check core version
	if req.CoreVersion != "" && req.CoreVersion != g.coreVersion {
		d := NewDefect(g.ID(), DefectClassVersion,
			fmt.Sprintf("core version mismatch: expected %s, got %s", g.coreVersion, req.CoreVersion))
		d.SuggestedFix = fmt.Sprintf("Upgrade the plugin to support Partir Core %s, or downgrade the runtime.", g.coreVersion)
		defects = append(defects, *d)
	}

	return defects
}
