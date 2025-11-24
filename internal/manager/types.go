package manager

import (
	"dosync/internal/notification"
	"dosync/internal/strategy"
)

// UpdateStrategy is an alias to strategy.UpdateStrategy
// Following the ONE WAY OF DOING THINGS rule, we use the strategy package's interface
// which is the complete, production implementation.
type UpdateStrategy = strategy.UpdateStrategy

// Notifier is an alias to notification.Notifier
// Following the ONE WAY OF DOING THINGS rule, we use the notification package's interface
// which has the complete implementation (Slack, Email, Webhook).
type Notifier = notification.Notifier
