package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// RepoAdapter commits artifacts to the artifact store
// and publishes confirmation messages via NATS.
// Currently abstract — could be Git, MinIO, or any store.
type RepoAdapter struct {
	nc     *nats.Conn
	ledger LedgerAppender
}

// NewRepoAdapter creates a new repo adapter.
func NewRepoAdapter(nc *nats.Conn, ledger LedgerAppender) *RepoAdapter {
	return &RepoAdapter{
		nc:     nc,
		ledger: ledger,
	}
}

// Confirm publishes a repo confirmation after successful artifact storage.
// This is called by the Foundry Orchestrator after artifacts pass all gates.
func (r *RepoAdapter) Confirm(ctx context.Context, ticketID, runID, artifactID, repoRef string) error {
	confirm := &RepoConfirmation{
		TicketID:   ticketID,
		RunID:      runID,
		ArtifactID: artifactID,
		RepoRef:    repoRef,
		Timestamp:  time.Now(),
	}

	data, err := json.Marshal(confirm)
	if err != nil {
		return fmt.Errorf("marshal confirmation: %w", err)
	}

	if err := r.nc.Publish(TopicConfirm, data); err != nil {
		return fmt.Errorf("publish confirmation to %s: %w", TopicConfirm, err)
	}

	// Log to ledger
	if r.ledger != nil {
		r.ledger.Append(ctx, "repo_commit", map[string]interface{}{
			"ticket_id":   ticketID,
			"run_id":      runID,
			"artifact_id": artifactID,
			"repo_ref":    repoRef,
		})
	}

	log.Printf("[collab-repo] Confirmed: ticket=%s artifact=%s ref=%s",
		ticketID, artifactID, repoRef)

	return nil
}
