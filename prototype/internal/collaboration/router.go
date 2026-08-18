package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// LedgerAppender is the interface the router uses to write audit events.
// Matches the FactoryLedger.Append signature from the worker plugin.
type LedgerAppender interface {
	Append(ctx context.Context, eventType string, payload map[string]interface{}) error
}

// Router is the stateless Collaboration API.
// It subscribes to NATS topics, triages violations, routes fix-it tickets,
// and publishes human alerts. All state is in the Factory Ledger.
type Router struct {
	nc     *nats.Conn
	ledger LedgerAppender
	subs   []*nats.Subscription
}

// NewRouter creates a Collaboration API router.
func NewRouter(nc *nats.Conn, ledger LedgerAppender) *Router {
	return &Router{
		nc:     nc,
		ledger: ledger,
	}
}

// Start subscribes to all collaboration topics and begins processing.
func (r *Router) Start() error {
	var err error

	// Subscribe to Andon Cord signals
	sub, err := r.nc.Subscribe(TopicAndon, r.handleAndon)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicAndon, err)
	}
	r.subs = append(r.subs, sub)

	// Subscribe to repo confirmations
	sub, err = r.nc.Subscribe(TopicConfirm, r.handleConfirm)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicConfirm, err)
	}
	r.subs = append(r.subs, sub)

	// Subscribe to ticket submissions from Control Surface
	sub, err = r.nc.Subscribe(TopicSubmit, r.handleSubmit)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicSubmit, err)
	}
	r.subs = append(r.subs, sub)

	log.Printf("[collab] Router started, subscribed to %s, %s, %s",
		TopicAndon, TopicConfirm, TopicSubmit)

	return nil
}

// Stop drains all subscriptions.
func (r *Router) Stop() {
	for _, sub := range r.subs {
		sub.Drain()
	}
	log.Printf("[collab] Router stopped")
}

// handleAndon processes Andon Cord halt signals.
// It classifies the violation and either routes a fix-it ticket (auto-fixable)
// or publishes a human alert (Alpha spec error).
func (r *Router) handleAndon(msg *nats.Msg) {
	var signal AndonSignal
	if err := json.Unmarshal(msg.Data, &signal); err != nil {
		log.Printf("[collab] Failed to unmarshal Andon signal: %v", err)
		return
	}

	ctx := context.Background()

	// Log the halt event to Factory Ledger
	if r.ledger != nil {
		r.ledger.Append(ctx, "andon_cord", signal.ToPayload())
	}

	log.Printf("[collab] Andon signal received: ticket=%s run=%s defects=%d",
		signal.TicketID, signal.RunID, len(signal.Defects))

	// Triage: classify the violation
	violation := ClassifyViolation(signal.Defects)

	switch violation {
	case ViolationBetaOutput, ViolationFoundryRuntime:
		// Auto-fixable → route fix-it ticket
		r.routeFixIt(ctx, &signal, violation)

	case ViolationAlphaSpec:
		// Not auto-fixable → alert human
		r.alertHuman(ctx, &signal, violation)

	default:
		// Unknown → alert human as safety fallback
		log.Printf("[collab] Unknown violation type for ticket=%s, escalating to human", signal.TicketID)
		r.alertHuman(ctx, &signal, violation)
	}
}

// routeFixIt creates a fix-it ticket and publishes it to collab.fix.
func (r *Router) routeFixIt(ctx context.Context, signal *AndonSignal, violation ViolationType) {
	targetAPI := "beta"
	if violation == ViolationFoundryRuntime {
		targetAPI = "foundry"
	}

	fix := &FixItTicket{
		FixID:         uuid.New().String(),
		OriginalRunID: signal.RunID,
		TicketID:      signal.TicketID,
		TargetAPI:     targetAPI,
		Violation:     violation,
		Defects:       signal.Defects,
		Timestamp:     time.Now(),
	}

	data, err := fix.Marshal()
	if err != nil {
		log.Printf("[collab] Failed to marshal fix-it ticket: %v", err)
		return
	}

	if err := r.nc.Publish(TopicFix, data); err != nil {
		log.Printf("[collab] Failed to publish fix-it to %s: %v", TopicFix, err)
		return
	}

	// Log fix-it to ledger
	if r.ledger != nil {
		r.ledger.Append(ctx, "fix_routed", map[string]interface{}{
			"fix_id":       fix.FixID,
			"ticket_id":    fix.TicketID,
			"target_api":   fix.TargetAPI,
			"violation":    string(fix.Violation),
			"defect_count": len(fix.Defects),
		})
	}

	log.Printf("[collab] Fix-it ticket routed: fix=%s target=%s violation=%s",
		fix.FixID, targetAPI, violation)
}

// alertHuman publishes a human escalation alert to collab.alert.
func (r *Router) alertHuman(ctx context.Context, signal *AndonSignal, violation ViolationType) {
	alert := &HumanAlert{
		AlertID:   uuid.New().String(),
		TicketID:  signal.TicketID,
		RunID:     signal.RunID,
		Violation: violation,
		Defects:   signal.Defects,
		Message:   ViolationDescription(violation),
		Timestamp: time.Now(),
	}

	data, err := alert.Marshal()
	if err != nil {
		log.Printf("[collab] Failed to marshal alert: %v", err)
		return
	}

	if err := r.nc.Publish(TopicAlert, data); err != nil {
		log.Printf("[collab] Failed to publish alert to %s: %v", TopicAlert, err)
		return
	}

	// Log alert to ledger
	if r.ledger != nil {
		r.ledger.Append(ctx, "human_alert", map[string]interface{}{
			"alert_id":  alert.AlertID,
			"ticket_id": alert.TicketID,
			"run_id":    alert.RunID,
			"violation": string(violation),
			"message":   alert.Message,
		})
	}

	log.Printf("[collab] ⚠️  Human alert published: alert=%s ticket=%s — %s",
		alert.AlertID, alert.TicketID, alert.Message)
}

// handleConfirm processes repo commit confirmations.
func (r *Router) handleConfirm(msg *nats.Msg) {
	var confirm RepoConfirmation
	if err := json.Unmarshal(msg.Data, &confirm); err != nil {
		log.Printf("[collab] Failed to unmarshal repo confirmation: %v", err)
		return
	}

	ctx := context.Background()

	// Log the confirmation
	if r.ledger != nil {
		r.ledger.Append(ctx, "repo_confirm", map[string]interface{}{
			"ticket_id":   confirm.TicketID,
			"run_id":      confirm.RunID,
			"artifact_id": confirm.ArtifactID,
			"repo_ref":    confirm.RepoRef,
		})
	}

	log.Printf("[collab] Repo confirmed: ticket=%s artifact=%s ref=%s",
		confirm.TicketID, confirm.ArtifactID, confirm.RepoRef)
}

// handleSubmit processes new ticket submissions from the Control Surface.
func (r *Router) handleSubmit(msg *nats.Msg) {
	ctx := context.Background()

	// Log the submission to ledger
	if r.ledger != nil {
		var payload map[string]interface{}
		json.Unmarshal(msg.Data, &payload)
		r.ledger.Append(ctx, "ticket_submitted", payload)
	}

	log.Printf("[collab] Ticket submitted via Control Surface (%d bytes)", len(msg.Data))
}
