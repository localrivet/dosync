/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package manager

import (
	"dosync/internal/health"
	"dosync/internal/metrics"
	"dosync/internal/replica"
	"dosync/internal/rollback"
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

// StrategyAdapter is a placeholder for strategy implementation
type StrategyAdapter struct {
	strategy string
	logger   Logger
}

// NewStrategyAdapter creates a new strategy adapter
func NewStrategyAdapter(strategy string, logger Logger) (*StrategyAdapter, error) {
	return &StrategyAdapter{
		strategy: strategy,
		logger:   logger,
	}, nil
}

// NotifierAdapter implements the Notifier interface
type NotifierAdapter struct {
	config *SlackConfig
	logger Logger
}

// SlackConfig holds configuration for Slack notifications
type SlackConfig struct {
	Enabled    bool
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
}

// NewNotifierAdapter creates a new notifier adapter
func NewNotifierAdapter(config *SlackConfig, logger Logger) (*NotifierAdapter, error) {
	return &NotifierAdapter{
		config: config,
		logger: logger,
	}, nil
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

// Implement UpdateStrategy for StrategyAdapter
func (s *StrategyAdapter) Execute(replicas []replica.Replica, imageTag string, healthCheck func(replica.Replica) bool) error {
	s.logger.Info("Executing strategy '%s' for %d replicas with imageTag '%s' (stub)", s.strategy, len(replicas), imageTag)
	return nil
}

// Implement Notifier interface for NotifierAdapter
func (n *NotifierAdapter) ShouldNotifyOnStart() bool    { return true }
func (n *NotifierAdapter) ShouldNotifyOnSuccess() bool  { return true }
func (n *NotifierAdapter) ShouldNotifyOnFailure() bool  { return true }
func (n *NotifierAdapter) ShouldNotifyOnRollback() bool { return true }
func (n *NotifierAdapter) SendDeploymentStart(service, version string) error {
	n.logger.Info("NotifierAdapter: Deployment start for %s:%s", service, version)
	return nil
}
func (n *NotifierAdapter) SendDeploymentSuccess(service, version string, duration time.Duration) error {
	n.logger.Info("NotifierAdapter: Deployment success for %s:%s in %v", service, version, duration)
	return nil
}
func (n *NotifierAdapter) SendDeploymentFailure(service, version, reason string) error {
	n.logger.Info("NotifierAdapter: Deployment failure for %s:%s, reason: %s", service, version, reason)
	return nil
}
func (n *NotifierAdapter) SendRollback(service, fromVersion, toVersion string) error {
	n.logger.Info("NotifierAdapter: Rollback for %s from %s to %s", service, fromVersion, toVersion)
	return nil
}
