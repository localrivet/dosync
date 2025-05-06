package manager

import (
	"dosync/internal/replica"
	"time"
)

// UpdateStrategy defines the interface for update strategies
type UpdateStrategy interface {
	// Execute performs the update strategy on the given replicas
	Execute(replicas []replica.Replica, imageTag string, healthCheck func(replica replica.Replica) bool) error
}

// Notifier defines the interface for notification providers
type Notifier interface {
	// ShouldNotifyOnStart returns true if this notifier should send notifications at the start of a deployment
	ShouldNotifyOnStart() bool

	// ShouldNotifyOnSuccess returns true if this notifier should send notifications on successful deployments
	ShouldNotifyOnSuccess() bool

	// ShouldNotifyOnFailure returns true if this notifier should send notifications on failed deployments
	ShouldNotifyOnFailure() bool

	// ShouldNotifyOnRollback returns true if this notifier should send notifications on rollbacks
	ShouldNotifyOnRollback() bool

	// SendDeploymentStart sends a notification at the start of a deployment
	SendDeploymentStart(service, version string) error

	// SendDeploymentSuccess sends a notification on successful deployment
	SendDeploymentSuccess(service, version string, duration time.Duration) error

	// SendDeploymentFailure sends a notification on failed deployment
	SendDeploymentFailure(service, version, reason string) error

	// SendRollback sends a notification on rollback
	SendRollback(service, fromVersion, toVersion string) error
}
