/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"fmt"
	"time"

	"dosync/internal/health"
	"dosync/internal/replica"
	"dosync/internal/strategy"
)

// DeploymentMonitor handles the detection of failed deployments and triggers automatic rollbacks
type DeploymentMonitor struct {
	// backupManager is used to access rollback history and perform backups
	backupManager *BackupManager

	// healthChecker is used to verify the service health
	healthChecker health.HealthChecker

	// config contains the rollback configuration options
	config RollbackConfig

	// CurrentDeployments tracks services that are currently being deployed
	CurrentDeployments map[string]*DeploymentState
}

// DeploymentState tracks the state of a deployment for a specific service
type DeploymentState struct {
	// ServiceName is the name of the service being deployed
	ServiceName string

	// NewImageTag is the image tag being deployed
	NewImageTag string

	// OldImageTag is the previous image tag (for rollback)
	OldImageTag string

	// StartTime is when the deployment began
	StartTime time.Time

	// HealthCheckAttempts tracks the number of health check attempts
	HealthCheckAttempts int

	// MaxHealthCheckAttempts is the maximum number of health checks before failing
	MaxHealthCheckAttempts int

	// RollbackOnFailure indicates if the service should be rolled back when deployment fails
	RollbackOnFailure bool
}

// NewDeploymentMonitor creates a new monitor for detecting failed deployments
func NewDeploymentMonitor(config RollbackConfig, healthChecker health.HealthChecker) (*DeploymentMonitor, error) {
	// Create the backup manager
	backupManager, err := NewBackupManager(
		config.BackupDir,
		config.MaxHistory,
		config.ComposeFilePattern,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup manager: %w", err)
	}

	return &DeploymentMonitor{
		backupManager:      backupManager,
		healthChecker:      healthChecker,
		config:             config,
		CurrentDeployments: make(map[string]*DeploymentState),
	}, nil
}

// StartMonitoring begins tracking a service deployment for potential rollback
func (dm *DeploymentMonitor) StartMonitoring(
	service string,
	newImageTag string,
	oldImageTag string,
	rollbackOnFailure bool,
	maxHealthCheckAttempts int,
) error {
	// Register the deployment for monitoring
	dm.CurrentDeployments[service] = &DeploymentState{
		ServiceName:            service,
		NewImageTag:            newImageTag,
		OldImageTag:            oldImageTag,
		StartTime:              time.Now(),
		HealthCheckAttempts:    0,
		MaxHealthCheckAttempts: maxHealthCheckAttempts,
		RollbackOnFailure:      rollbackOnFailure,
	}

	return nil
}

// StopMonitoring stops tracking a service deployment (typically when it succeeds)
func (dm *DeploymentMonitor) StopMonitoring(service string) {
	delete(dm.CurrentDeployments, service)
}

// CheckDeploymentHealth verifies the health of a monitored service
// Returns true if healthy, false if unhealthy (triggering rollback if configured)
func (dm *DeploymentMonitor) CheckDeploymentHealth(service string, replicaID string) (bool, error) {
	// Get the deployment state
	state, exists := dm.CurrentDeployments[service]
	if !exists {
		return false, fmt.Errorf("service %s is not being monitored", service)
	}

	// Create a replica object for health check
	replicaObj := createReplicaForHealthCheck(service, replicaID)

	// Perform health check
	healthy, err := dm.healthChecker.Check(replicaObj)
	if err != nil {
		return false, fmt.Errorf("health check failed: %w", err)
	}

	// Update attempts
	state.HealthCheckAttempts++

	// If the service is healthy, we're good
	if healthy {
		return true, nil
	}

	// If the service is unhealthy and we've exceeded our attempts, consider a rollback
	if state.HealthCheckAttempts >= state.MaxHealthCheckAttempts {
		if state.RollbackOnFailure {
			if err := dm.executeRollback(service, state.OldImageTag); err != nil {
				return false, fmt.Errorf("failed to rollback service %s: %w", service, err)
			}
			// Remove from monitoring after rollback
			dm.StopMonitoring(service)
		}
		return false, nil
	}

	// Still unhealthy but we haven't exceeded max attempts
	return false, nil
}

// ShouldRollback determines if a service should be rolled back based on health status
func (dm *DeploymentMonitor) ShouldRollback(
	service string,
	healthStatus bool,
	strategyConfig strategy.StrategyConfig,
) bool {
	// If the service is healthy, no need to rollback
	if healthStatus {
		return false
	}

	// Check if the service is being monitored
	state, exists := dm.CurrentDeployments[service]
	if !exists {
		// If service is not being monitored, use the default from config
		return strategyConfig.RollbackOnFailure || dm.config.DefaultRollbackOnFailure
	}

	// If the service is monitored, use its specific rollback setting
	return state.RollbackOnFailure
}

// executeRollback performs the actual rollback operation for a service
func (dm *DeploymentMonitor) executeRollback(service string, targetImageTag string) error {
	// Get all available rollbacks for the service
	entries, err := dm.backupManager.GetBackupHistory(service)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollback history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no rollback entries found for service %s", service)
	}

	// Find the entry for the target image tag or use the most recent if not specified
	var targetEntry RollbackEntry
	if targetImageTag == "" {
		// Use the most recent entry (entries are already sorted with most recent first)
		targetEntry = entries[0]
	} else {
		// Find the entry with the matching image tag
		found := false
		for _, entry := range entries {
			if entry.ImageTag == targetImageTag {
				targetEntry = entry
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no rollback entry found for service %s with image tag %s", service, targetImageTag)
		}
	}

	// Restore the target compose file
	if err := dm.backupManager.RestoreFromBackup(targetEntry, dm.config.ComposeFilePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// TODO: Restart the service with docker-compose up -d command
	// This would typically be handled by the controller that integrates with Docker Compose

	return nil
}

// Helper function to create a replica object for health checks
// In a real implementation, this would interact with the Docker API
func createReplicaForHealthCheck(service string, replicaID string) replica.Replica {
	// Return a simple object with the required fields
	return replica.Replica{
		ServiceName: service,
		ReplicaID:   replicaID,
		ContainerID: "placeholder", // This would be fetched from Docker in a real implementation
		Status:      "running",     // This would be fetched from Docker in a real implementation
	}
}
