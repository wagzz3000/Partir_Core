package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	HeartbeatSubject = "partir.worker.heartbeat"
	HeartbeatTimeout = 30 * time.Second
)

// HeartbeatListener listens for worker heartbeats via NATS
type HeartbeatListener struct {
	registry  *Registry
	collector *Collector
	nc        *nats.Conn
	sub       *nats.Subscription
}

// NewHeartbeatListener creates a new heartbeat listener
func NewHeartbeatListener(registry *Registry, collector *Collector, natsURL string) (*HeartbeatListener, error) {
	nc, err := nats.Connect(natsURL,
		nats.Name("partir-core-heartbeat"),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &HeartbeatListener{
		registry:  registry,
		collector: collector,
		nc:        nc,
	}, nil
}

// Start begins listening for heartbeats
func (h *HeartbeatListener) Start(ctx context.Context) error {
	var err error
	h.sub, err = h.nc.Subscribe(HeartbeatSubject, func(msg *nats.Msg) {
		h.handleHeartbeat(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to heartbeats: %w", err)
	}

	// Start crash detector
	go h.runCrashDetector(ctx)

	log.Printf("Heartbeat listener started on %s", HeartbeatSubject)
	return nil
}

// Stop stops the heartbeat listener
func (h *HeartbeatListener) Stop() error {
	if h.sub != nil {
		h.sub.Unsubscribe()
	}
	if h.nc != nil {
		h.nc.Close()
	}
	return nil
}

func (h *HeartbeatListener) handleHeartbeat(ctx context.Context, msg *nats.Msg) {
	var hb WorkerHeartbeat
	if err := json.Unmarshal(msg.Data, &hb); err != nil {
		log.Printf("Failed to parse heartbeat: %v", err)
		return
	}

	// Register or update worker
	now := time.Now()
	worker := &Worker{
		ID:               hb.WorkerID,
		Type:             hb.WorkerType,
		Endpoint:         hb.Endpoint,
		Status:           hb.Status,
		AssignedStation:  hb.AssignedStation,
		AssignedSlot:     hb.AssignedSlot,
		ModelFingerprint: hb.ModelFingerprint,
		LastHeartbeat:    &now,
	}

	if hb.MemoryBindingID != "" {
		worker.MemoryBindingID = &hb.MemoryBindingID
	}
	if hb.MemoryNamespace != "" {
		worker.MemoryNamespace = &hb.MemoryNamespace
	}

	err := h.registry.RegisterWorker(ctx, worker)
	if err != nil {
		log.Printf("Failed to register worker %s: %v", hb.WorkerID, err)
		return
	}

	// Update metrics
	if h.collector != nil {
		// Use authoritative station assignment from DB (populated by RegisterWorker)
		station := worker.AssignedStation
		h.collector.RecordWorkerStatus(ctx, hb.WorkerID, hb.WorkerType, station, hb.Status)
	}

	log.Printf("Heartbeat from %s (%s): %s [Slot: %s]", hb.WorkerID, hb.WorkerType, hb.Status, hb.AssignedSlot)
}

func (h *HeartbeatListener) runCrashDetector(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			crashed, err := h.registry.DetectCrashedWorkers(ctx, HeartbeatTimeout)
			if err != nil {
				log.Printf("Failed to detect crashed workers: %v", err)
				continue
			}

			for _, w := range crashed {
				log.Printf("Worker %s (%s) detected as crashed", w.ID, w.Type)
				if h.collector != nil {
					h.collector.RecordWorkerCrash(ctx, w.ID)
				}
			}
		}
	}
}
