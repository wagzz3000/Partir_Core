package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// AppConfig holds all externalized configuration
type AppConfig struct {
	// Database
	DatabaseURL string `json:"database_url" mapstructure:"database_url"`

	// NATS
	NatsURL string `json:"nats_url" mapstructure:"nats_url"`

	// MinIO
	MinioEndpoint string `json:"minio_endpoint" mapstructure:"minio_endpoint"`
	MinioBucket   string `json:"minio_bucket" mapstructure:"minio_bucket"`

	// Metrics
	MetricsPort int `json:"metrics_port" mapstructure:"metrics_port"`

	// Security
	JWTSecret string `json:"jwt_secret" mapstructure:"jwt_secret"`

	// Alerting
	SlackWebhookURL     string `json:"slack_webhook_url" mapstructure:"slack_webhook_url"`
	PagerDutyRoutingKey string `json:"pagerduty_routing_key" mapstructure:"pagerduty_routing_key"`

	// Resource Limits
	MaxConcurrentTickets int `json:"max_concurrent_tickets" mapstructure:"max_concurrent_tickets"`
	MaxTicketsPerHour    int `json:"max_tickets_per_hour" mapstructure:"max_tickets_per_hour"`
}

// LoadFromEnv loads configuration using Viper.
// Priority: env vars > config file > defaults.
func LoadFromEnv() *AppConfig {
	v := viper.New()

	// --- Defaults ---
	v.SetDefault("database_url", "postgres://partir:partir@localhost:5432/partir?sslmode=disable")
	v.SetDefault("nats_url", "nats://localhost:4222")
	v.SetDefault("minio_endpoint", "localhost:9000")
	v.SetDefault("minio_bucket", "partir-artifacts")
	v.SetDefault("metrics_port", 9090)
	v.SetDefault("max_concurrent_tickets", 10)
	v.SetDefault("max_tickets_per_hour", 100)

	// --- Config file (optional) ---
	v.SetConfigName("partir")         // partir.yaml, partir.json, partir.toml
	v.SetConfigType("yaml")           // default format
	v.AddConfigPath(".")              // current directory
	v.AddConfigPath("/etc/partir/")   // system config
	v.AddConfigPath("$HOME/.partir/") // user config

	// Silently ignore missing config file — env vars are sufficient
	_ = v.ReadInConfig()

	// --- Environment variables ---
	v.SetEnvPrefix("PARTIR")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Map env vars without the PARTIR_ prefix for legacy compatibility
	v.BindEnv("database_url", "PARTIR_DB_URL")
	v.BindEnv("nats_url", "NATS_URL")
	v.BindEnv("minio_endpoint", "PARTIR_MINIO_ENDPOINT")
	v.BindEnv("minio_bucket", "PARTIR_MINIO_BUCKET")
	v.BindEnv("metrics_port", "PARTIR_METRICS_PORT")
	v.BindEnv("jwt_secret", "PARTIR_JWT_SECRET")
	v.BindEnv("slack_webhook_url", "PARTIR_SLACK_WEBHOOK_URL")
	v.BindEnv("pagerduty_routing_key", "PARTIR_PAGERDUTY_KEY")
	v.BindEnv("max_concurrent_tickets", "PARTIR_MAX_CONCURRENT_TICKETS")
	v.BindEnv("max_tickets_per_hour", "PARTIR_MAX_TICKETS_PER_HOUR")

	cfg := &AppConfig{}
	v.Unmarshal(cfg)

	return cfg
}

// Validate checks that all required configuration is present and valid
func (c *AppConfig) Validate() error {
	var errs []string

	if c.DatabaseURL == "" {
		errs = append(errs, "PARTIR_DB_URL is required")
	}
	if c.MetricsPort < 1 || c.MetricsPort > 65535 {
		errs = append(errs, fmt.Sprintf("PARTIR_METRICS_PORT must be 1-65535, got %d", c.MetricsPort))
	}
	if c.MaxConcurrentTickets < 1 {
		errs = append(errs, "PARTIR_MAX_CONCURRENT_TICKETS must be >= 1")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
