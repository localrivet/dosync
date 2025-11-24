/*
Copyright © 2025 LocalRivet <github.com/localrivet>
*/
package manager

import (
	"dosync/internal/health"
	"dosync/internal/metrics"
	"dosync/internal/notification"
	"dosync/internal/rollback"
	"dosync/internal/strategy"
	"fmt"
	"time"
)

// ManagerConfig holds configuration options for test fixtures
type ManagerConfig struct {
	HealthCheck     health.HealthCheckConfig
	RollbackConfig  rollback.RollbackConfig
	RetentionConfig metrics.RetentionConfig
	Strategy        string
}

// CustomComponents allows injecting custom implementations of dependencies for testing
type CustomComponents struct {
	ReplicaManager     interface{}
	HealthChecker      interface{}
	RollbackController interface{}
	DependencyManager  interface{}
	Notifier           interface{}
	Logger             interface{}
}

// SlackConfig holds configuration for Slack notifications
type SlackConfig struct {
	Enabled    bool
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
}

// NotificationsConfig contains configuration for various notification types
type NotificationsConfig struct {
	SlackConfig *SlackConfig
	// Could add email, webhook, etc. configurations here
}

// RollingUpdateConfig contains configuration for the rolling update manager
type RollingUpdateConfig struct {
	// ComposeFilePath is the path to the docker-compose.yml file
	ComposeFilePath string

	// HealthCheckTimeout is the maximum time to wait for a service to become healthy
	HealthCheckTimeout time.Duration

	// HealthCheckRetries is the number of health check retries before declaring failure
	HealthCheckRetries int

	// UpdateStrategy defines which update strategy to use (e.g., "one-at-a-time", "percentage")
	UpdateStrategy string

	// RollbackOnFailure determines whether to automatically roll back on failure
	RollbackOnFailure bool

	// RollbackConfig contains configuration for rollback operations
	RollbackConfig rollback.RollbackConfig

	// NotificationsConfig contains configuration for notifications
	NotificationsConfig *NotificationsConfig

	// MetricsDB is the path to the metrics database file
	MetricsDB string
}

// NotificationConfigItem represents a single notification provider configuration
type NotificationConfigItem struct {
	// Type is the notification provider type (slack, email, webhook)
	Type string

	// Config is the provider-specific configuration
	Config map[string]interface{}
}

// Validate checks if the config is valid
func (c *RollingUpdateConfig) Validate() error {
	if c.ComposeFilePath == "" {
		return ErrMissingComposeFile
	}
	return nil
}

// ApplyDefaults sets default values for unspecified fields
func (c *RollingUpdateConfig) ApplyDefaults() {
	if c.HealthCheckTimeout == 0 {
		c.HealthCheckTimeout = 30 * time.Second
	}
	if c.HealthCheckRetries == 0 {
		c.HealthCheckRetries = 3
	}
	if c.UpdateStrategy == "" {
		c.UpdateStrategy = "one-at-a-time"
	}
}

// CreateStrategy creates the appropriate UpdateStrategy based on config.
// Following ONE WAY OF DOING THINGS - uses strategy package directly.
func CreateStrategy(replicaManager strategy.ReplicaManager, healthChecker health.HealthChecker, config strategy.StrategyConfig) (UpdateStrategy, error) {
	return strategy.NewUpdateStrategy(config, replicaManager, healthChecker)
}

// CreateSlackNotifier creates a Slack notifier from SlackConfig.
// Following ONE WAY OF DOING THINGS - uses notification package directly.
func CreateSlackNotifier(cfg *SlackConfig) (Notifier, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("slack notification not enabled")
	}

	notificationConfig := notification.NotificationConfig{
		Type:      "slack",
		Endpoint:  cfg.WebhookURL,
		Token:     "", // WebhookURL is used directly for incoming webhooks
		Channel:   cfg.Channel,
		OnSuccess: true,
		OnFailure: true,
		OnRollback: true,
	}

	notifier := &notification.SlackNotifier{}
	if err := notifier.Configure(notificationConfig); err != nil {
		return nil, fmt.Errorf("failed to configure slack notifier: %w", err)
	}

	return notifier, nil
}
