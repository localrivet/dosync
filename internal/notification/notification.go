// Package notification provides interfaces and implementations for sending notifications
// about deployment events such as deployment starts, successful completions, failures, and rollbacks.
package notification

import (
	"errors"
	"fmt"
	"time"
)

// NotificationType represents the type of notification provider
type NotificationType string

const (
	// SlackNotification represents a Slack notification
	SlackNotification NotificationType = "slack"
	// EmailNotification represents an email notification
	EmailNotification NotificationType = "email"
	// WebhookNotification represents a webhook notification
	WebhookNotification NotificationType = "webhook"
)

// NotificationConfig represents the configuration for a notification service
type NotificationConfig struct {
	// Type is the notification type: "slack", "email", or "webhook"
	Type string `json:"type" yaml:"type"`
	// Endpoint is the URL or address of the notification service
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	// Token is the authentication token (for Slack)
	Token string `json:"token,omitempty" yaml:"token,omitempty"`
	// Channel is the channel name (for Slack)
	Channel string `json:"channel,omitempty" yaml:"channel,omitempty"`
	// Recipients is the list of email addresses (for Email)
	Recipients []string `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	// OnSuccess determines whether to send notifications on successful deployments
	OnSuccess bool `json:"onSuccess" yaml:"onSuccess"`
	// OnFailure determines whether to send notifications on failed deployments
	OnFailure bool `json:"onFailure" yaml:"onFailure"`
	// OnRollback determines whether to send notifications on rollbacks
	OnRollback bool `json:"onRollback" yaml:"onRollback"`
}

// Validate checks the notification configuration for correctness
func (c NotificationConfig) Validate() error {
	if c.Type == "" {
		return errors.New("notification type cannot be empty")
	}

	if c.Endpoint == "" {
		return errors.New("notification endpoint cannot be empty")
	}

	// Type-specific validation
	switch NotificationType(c.Type) {
	case SlackNotification:
		if c.Token == "" {
			return errors.New("Slack token is required")
		}
		if c.Channel == "" {
			return errors.New("Slack channel is required")
		}
	case EmailNotification:
		if len(c.Recipients) == 0 {
			return errors.New("email recipients list cannot be empty")
		}
	case WebhookNotification:
		// No additional requirements for webhook
	default:
		return fmt.Errorf("unsupported notification type: %s", c.Type)
	}

	return nil
}

// Notifier is the interface that all notification implementations must satisfy
type Notifier interface {
	// Configure sets up the notifier with the provided configuration
	Configure(config NotificationConfig) error

	// SendDeploymentStarted sends a notification when a deployment starts
	SendDeploymentStarted(service string, version string) error

	// SendDeploymentSuccess sends a notification when a deployment succeeds
	SendDeploymentSuccess(service string, version string, duration time.Duration) error

	// SendDeploymentFailure sends a notification when a deployment fails
	SendDeploymentFailure(service string, version string, errorMessage string) error

	// SendRollback sends a notification when a rollback occurs
	SendRollback(service string, fromVersion string, toVersion string) error

	// Helper methods to check notification settings
	ShouldNotifyOnSuccess() bool
	ShouldNotifyOnFailure() bool
	ShouldNotifyOnRollback() bool
}

// BaseNotifier provides common functionality for notifier implementations
type BaseNotifier struct {
	Config NotificationConfig
}

// Configure sets up the base notifier with the provided configuration
func (n *BaseNotifier) Configure(config NotificationConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	n.Config = config
	return nil
}

// ShouldNotifyOnSuccess returns whether to notify on successful deployments
func (n *BaseNotifier) ShouldNotifyOnSuccess() bool {
	return n.Config.OnSuccess
}

// ShouldNotifyOnFailure returns whether to notify on failed deployments
func (n *BaseNotifier) ShouldNotifyOnFailure() bool {
	return n.Config.OnFailure
}

// ShouldNotifyOnRollback returns whether to notify on rollbacks
func (n *BaseNotifier) ShouldNotifyOnRollback() bool {
	return n.Config.OnRollback
}
