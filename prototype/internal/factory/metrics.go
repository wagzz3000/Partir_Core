package factory

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("partir.factory")

	// Station metrics
	stationThroughput metric.Int64Counter
	stationDefects    metric.Int64Counter
	stationLatency    metric.Float64Histogram
	stationTokens     metric.Int64Counter

	// Worker metrics
	workerStatus     metric.Int64Gauge
	workerUptime     metric.Float64Gauge
	workerCrashes    metric.Int64Counter
	workerRestarts   metric.Int64Counter
	workerExecutions metric.Int64Counter

	metricsOnce sync.Once
)

// InitMetrics initializes the factory metrics
func InitMetrics() error {
	var initErr error
	metricsOnce.Do(func() {
		var err error

		// Station throughput counter
		stationThroughput, err = meter.Int64Counter(
			"partir.station.throughput.total",
			metric.WithDescription("Total tickets completed at this station"),
			metric.WithUnit("{tickets}"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Station defects counter
		stationDefects, err = meter.Int64Counter(
			"partir.station.defects.total",
			metric.WithDescription("Total gate failures at this station"),
			metric.WithUnit("{defects}"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Station latency histogram
		stationLatency, err = meter.Float64Histogram(
			"partir.station.latency.seconds",
			metric.WithDescription("Execution latency at this station"),
			metric.WithUnit("s"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Station tokens counter
		stationTokens, err = meter.Int64Counter(
			"partir.station.tokens.total",
			metric.WithDescription("Total LLM tokens consumed at this station"),
			metric.WithUnit("{tokens}"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Worker status gauge (1=active, 0=inactive)
		workerStatus, err = meter.Int64Gauge(
			"partir.worker.status",
			metric.WithDescription("Worker status (1=active, 0=sleeping/crashed)"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Worker uptime gauge
		workerUptime, err = meter.Float64Gauge(
			"partir.worker.uptime.seconds",
			metric.WithDescription("Worker uptime in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Worker crashes counter
		workerCrashes, err = meter.Int64Counter(
			"partir.worker.crashes.total",
			metric.WithDescription("Total worker crashes"),
			metric.WithUnit("{crashes}"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Worker restarts counter
		workerRestarts, err = meter.Int64Counter(
			"partir.worker.restarts.total",
			metric.WithDescription("Total worker restarts"),
			metric.WithUnit("{restarts}"),
		)
		if err != nil {
			initErr = err
			return
		}

		// Worker executions counter
		workerExecutions, err = meter.Int64Counter(
			"partir.worker.executions.total",
			metric.WithDescription("Total executions by this worker"),
			metric.WithUnit("{executions}"),
		)
		if err != nil {
			initErr = err
			return
		}
	})
	return initErr
}

// Collector collects and records factory metrics
type Collector struct {
	registry   *Registry
	workerInfo map[string]workerMeta // Track per-worker metadata
	mu         sync.RWMutex
}

type workerMeta struct {
	startTime time.Time
	crashes   int64
	restarts  int64
}

// NewCollector creates a new metrics collector
func NewCollector(registry *Registry) *Collector {
	return &Collector{
		registry:   registry,
		workerInfo: make(map[string]workerMeta),
	}
}

// RecordExecution records a completed execution at a station
func (c *Collector) RecordExecution(ctx context.Context, stationID StationID, workerID string, latency time.Duration, tokens int64, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("station", string(stationID)),
		attribute.String("worker_id", workerID),
	}

	// Record throughput
	stationThroughput.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record latency
	stationLatency.Record(ctx, latency.Seconds(), metric.WithAttributes(attrs...))

	// Record tokens
	stationTokens.Add(ctx, tokens, metric.WithAttributes(attrs...))

	// Record worker execution
	workerExecutions.Add(ctx, 1, metric.WithAttributes(
		attribute.String("worker_id", workerID),
	))

	// Record defect if not successful
	if !success {
		stationDefects.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordWorkerStatus updates the worker status metric
func (c *Collector) RecordWorkerStatus(ctx context.Context, workerID string, workerType string, stationID string, status WorkerStatus) {
	attrs := []attribute.KeyValue{
		attribute.String("worker_id", workerID),
		attribute.String("worker_type", workerType),
		attribute.String("station", stationID),
	}

	var statusValue int64 = 0
	if status == WorkerStatusActive {
		statusValue = 1
	}

	workerStatus.Record(ctx, statusValue, metric.WithAttributes(attrs...))

	// Update uptime
	c.mu.RLock()
	meta, exists := c.workerInfo[workerID]
	c.mu.RUnlock()

	if exists {
		uptime := time.Since(meta.startTime).Seconds()
		workerUptime.Record(ctx, uptime, metric.WithAttributes(
			attribute.String("worker_id", workerID),
		))
	}
}

// RecordWorkerCrash records a worker crash
func (c *Collector) RecordWorkerCrash(ctx context.Context, workerID string) {
	c.mu.Lock()
	meta := c.workerInfo[workerID]
	meta.crashes++
	c.workerInfo[workerID] = meta
	c.mu.Unlock()

	workerCrashes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("worker_id", workerID),
	))
}

// RecordWorkerRestart records a worker restart
func (c *Collector) RecordWorkerRestart(ctx context.Context, workerID string) {
	c.mu.Lock()
	meta := c.workerInfo[workerID]
	meta.restarts++
	meta.startTime = time.Now()
	c.workerInfo[workerID] = meta
	c.mu.Unlock()

	workerRestarts.Add(ctx, 1, metric.WithAttributes(
		attribute.String("worker_id", workerID),
	))
}

// RegisterWorkerStart records when a worker starts
func (c *Collector) RegisterWorkerStart(workerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workerInfo[workerID] = workerMeta{
		startTime: time.Now(),
	}
}

// CollectAll collects current metrics for all workers and stations
func (c *Collector) CollectAll(ctx context.Context) error {
	status, err := c.registry.GetFactoryStatus(ctx)
	if err != nil {
		return err
	}

	// Record worker statuses
	for _, w := range status.Workers {
		c.RecordWorkerStatus(ctx, w.ID, w.Type, w.AssignedStation, w.Status)
	}

	return nil
}
