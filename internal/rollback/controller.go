/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"fmt"
	"os/exec"
)

// Variables for testing
var execCommand = func(command string, args ...string) *exec.Cmd {
	return exec.Command(command, args...)
}

// RollbackControllerImpl is the main implementation of the RollbackController interface
type RollbackControllerImpl struct {
	// BackupManager handles backup file operations
	BackupManager BackupOperations

	// Config contains configuration settings
	Config RollbackConfig
}

// NewRollbackController creates a new rollback controller
func NewRollbackController(config RollbackConfig) (*RollbackControllerImpl, error) {
	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rollback configuration: %w", err)
	}

	// Apply default values to unspecified configuration options
	config.ApplyDefaults()

	// Create a backup manager
	backupManager, err := NewBackupManager(
		config.BackupDir,
		config.MaxHistory,
		config.ComposeFilePattern,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup manager: %w", err)
	}

	return &RollbackControllerImpl{
		BackupManager: backupManager,
		Config:        config,
	}, nil
}

// PrepareRollback creates a backup of the current service state that can be used for rollback
func (rc *RollbackControllerImpl) PrepareRollback(service string) error {
	// Create a backup of the current compose file
	_, err := rc.BackupManager.CreateBackup(
		rc.Config.ComposeFilePath,
		service,
		"latest", // We don't know the image tag at this point, will be updated later
	)
	if err != nil {
		return fmt.Errorf("failed to create rollback backup: %w", err)
	}

	return nil
}

// Rollback returns the service to its previous state (most recent backup)
func (rc *RollbackControllerImpl) Rollback(service string) error {
	// Get all available rollbacks for the service
	entries, err := rc.BackupManager.GetBackupHistory(service)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollback history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no rollback entries found for service %s", service)
	}

	// Use the most recent entry
	targetEntry := entries[0]

	// Restore the compose file
	if err := rc.BackupManager.RestoreFromBackup(targetEntry, rc.Config.ComposeFilePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Restart the service with docker-compose
	cmd := execCommand("docker-compose", "-f", rc.Config.ComposeFilePath, "up", "-d", service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service: %s, error: %w", string(output), err)
	}

	fmt.Printf("Successfully rolled back service %s to version %s\n", service, targetEntry.ImageTag)
	return nil
}

// RollbackToVersion rolls back a service to a specific previous version
func (rc *RollbackControllerImpl) RollbackToVersion(service string, version string) error {
	// Get all available rollbacks for the service
	entries, err := rc.BackupManager.GetBackupHistory(service)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollback history: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no rollback entries found for service %s", service)
	}

	// Find the entry with the matching image tag
	var targetEntry RollbackEntry
	found := false
	for _, entry := range entries {
		if entry.ImageTag == version {
			targetEntry = entry
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no rollback entry found for service %s with version %s", service, version)
	}

	// Restore the compose file
	if err := rc.BackupManager.RestoreFromBackup(targetEntry, rc.Config.ComposeFilePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Restart the service with docker-compose
	cmd := execCommand("docker-compose", "-f", rc.Config.ComposeFilePath, "up", "-d", service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service: %s, error: %w", string(output), err)
	}

	fmt.Printf("Successfully rolled back service %s to version %s\n", service, version)
	return nil
}

// GetRollbackHistory returns the history of rollback entries for a service
func (rc *RollbackControllerImpl) GetRollbackHistory(service string) ([]RollbackEntry, error) {
	return rc.BackupManager.GetBackupHistory(service)
}

// ShouldRollback determines if a service should be rolled back based on health status
func (rc *RollbackControllerImpl) ShouldRollback(service string, healthStatus bool, rollbackOnFailure bool) bool {
	// If the service is healthy, no need to rollback
	if healthStatus {
		return false
	}

	// Use the rollback setting from the parameter or the default from config
	return rollbackOnFailure || rc.Config.DefaultRollbackOnFailure
}

// CleanupOldBackups removes old backup files based on retention policy
func (rc *RollbackControllerImpl) CleanupOldBackups() error {
	// Get all services that have backups
	services, err := rc.BackupManager.GetServices()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	// Clean up old backups for each service
	for _, service := range services {
		if err := rc.BackupManager.CleanupOldBackups(service); err != nil {
			return fmt.Errorf("failed to clean up old backups for service %s: %w", service, err)
		}
	}

	return nil
}
