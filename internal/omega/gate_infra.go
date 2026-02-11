package omega

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InfraGate validates infrastructure state against green_tag_ref
type InfraGate struct {
	pool *pgxpool.Pool
}

// NewInfraGate creates a new infrastructure verification gate
func NewInfraGate(pool *pgxpool.Pool) *InfraGate {
	return &InfraGate{pool: pool}
}

func (g *InfraGate) ID() string   { return "infra" }
func (g *InfraGate) Name() string { return "Infrastructure Verification" }

func (g *InfraGate) Run(ctx context.Context, req *GateRequest) []Defect {
	var defects []Defect

	// 1. Get Green Tag Ref (future: from req.GreenTagRef)
	// greenTagRef := "default"
	// In a real scenario, this comes from the ticket via req
	// For now we assume req.GreenTagRef or similar, but GateRequest doesn't have it yet.
	// We'll update GateRequest or look it up.

	// Let's assume we query the ticket from storage if needed, or pass it in.
	// For now, let's just validte the DB version matches the "default" green tag
	// or whatever is considered "current" if no specific tag is requested.

	// 2. Get actual DB version
	var versionStr string
	err := g.pool.QueryRow(ctx, "SHOW server_version;").Scan(&versionStr)
	if err != nil {
		return append(defects, *NewDefect(g.ID(), DefectClassPreflight,
			fmt.Sprintf("failed to query DB version: %v", err)))
	}

	// Parse actual version (e.g. "16.4 (Debian 16.4-1.pgdg120+1)")
	// We just want "16.4"
	// Simple regex to grab major.minor
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return append(defects, *NewDefect(g.ID(), DefectClassPreflight,
			fmt.Sprintf("unrecognized DB version format: %s", versionStr)))
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	// 3. Compare against Green Tag Registry
	// If greenTagRef is provided, look it up.
	// For "Factory 0", we often just want to ensure we aren't drifting from *approved* versions.
	// Let's check if this version exists in `green_tags` and is active.

	var active bool
	var expectedMajor, expectedMinor int

	// Check if this specific version is allowed
	err = g.pool.QueryRow(ctx, `
		SELECT active, postgres_major, postgres_minor 
		FROM green_tags 
		WHERE postgres_major = $1 AND postgres_minor = $2 AND active = true
	`, major, minor).Scan(&active, &expectedMajor, &expectedMinor)

	if err != nil {
		// No matching active green tag found
		d := NewDefect(g.ID(), DefectClassPreflight,
			fmt.Sprintf("DB version %d.%d is not a registered/active green tag", major, minor))
		d.SuggestedFix = "Update the Postgres instance to a supported version or register this version in the 'green_tags' table."
		defects = append(defects, *d)
	}

	return defects
}
