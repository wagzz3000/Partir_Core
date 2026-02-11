package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Severity levels for alerts
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Alert represents a system alert
type Alert struct {
	Severity    Severity               `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookConfig defines a webhook destination
type WebhookConfig struct {
	Name    string            `json:"name"` // "slack", "pagerduty", "custom"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// AlertManager manages alert routing and delivery
type AlertManager struct {
	mu       sync.RWMutex
	webhooks []WebhookConfig
	client   *http.Client
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		webhooks: make([]WebhookConfig, 0),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AddWebhook registers a webhook destination
func (am *AlertManager) AddWebhook(config WebhookConfig) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.webhooks = append(am.webhooks, config)
}

// Fire sends an alert to all registered webhooks
func (am *AlertManager) Fire(ctx context.Context, alert Alert) error {
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	am.mu.RLock()
	hooks := make([]WebhookConfig, len(am.webhooks))
	copy(hooks, am.webhooks)
	am.mu.RUnlock()

	var lastErr error
	for _, hook := range hooks {
		if err := am.send(ctx, hook, alert); err != nil {
			lastErr = fmt.Errorf("webhook %q failed: %w", hook.Name, err)
		}
	}
	return lastErr
}

func (am *AlertManager) send(ctx context.Context, hook WebhookConfig, alert Alert) error {
	body, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hook.Headers {
		req.Header.Set(k, v)
	}

	resp, err := am.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
