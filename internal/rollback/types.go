/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"time"
)

// RollbackController defines the interface for managing service rollbacks
type RollbackController interface {
	// PrepareRollback creates a backup of the current service state that can be used for rollback
	PrepareRollback(service string) error

	// Rollback returns the service to its previous state
	Rollback(service string) error

	// RollbackToVersion rolls back a service to a specific previous version
	RollbackToVersion(service string, version string) error

	// GetRollbackHistory returns the history of rollback entries for a service
	GetRollbackHistory(service string) ([]RollbackEntry, error)

	// ShouldRollback determines if a service should be rolled back based on health status
	ShouldRollback(service string, healthStatus bool, rollbackOnFailure bool) bool

	// CleanupOldBackups removes old backup files based on retention policy
	CleanupOldBackups() error
}

// RollbackEntry represents a single rollback point in the history
type RollbackEntry struct {
	// ServiceName is the name of the service this entry belongs to
	ServiceName string

	// ImageTag is the Docker image tag used in this version
	ImageTag string

	// Timestamp is when this entry was created
	Timestamp time.Time

	// ComposeFile is the path to the backup docker-compose file
	ComposeFile string
}
