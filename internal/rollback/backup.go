/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupOperations defines the interface for backup operations
type BackupOperations interface {
	// CreateBackup creates a backup of the specified compose file for a service
	CreateBackup(composeFilePath string, service string, imageTag string) (RollbackEntry, error)

	// GetBackupHistory returns a list of available backups for the specified service
	GetBackupHistory(service string) ([]RollbackEntry, error)

	// CleanupOldBackups removes older backups to respect the MaxHistory limit
	CleanupOldBackups(service string) error

	// RestoreFromBackup restores a service from a backup file
	RestoreFromBackup(entry RollbackEntry, targetComposeFile string) error

	// GetServices returns a list of all services that have backups
	GetServices() ([]string, error)
}

// Ensure BackupManager implements BackupOperations
var _ BackupOperations = (*BackupManager)(nil)

// BackupManager handles creation and management of Docker Compose file backups
type BackupManager struct {
	// BackupDir is the directory where backup files are stored
	BackupDir string

	// MaxHistory is the maximum number of backup files to keep per service
	MaxHistory int

	// ComposeFilePattern is the standard pattern for Docker Compose files
	ComposeFilePattern string
}

// NewBackupManager creates a new backup manager for Docker Compose files
func NewBackupManager(backupDir string, maxHistory int, composeFilePattern string) (*BackupManager, error) {
	// Validate and create the backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Use default pattern if not provided
	if composeFilePattern == "" {
		composeFilePattern = "docker-compose.yml"
	}

	return &BackupManager{
		BackupDir:          backupDir,
		MaxHistory:         maxHistory,
		ComposeFilePattern: composeFilePattern,
	}, nil
}

// FindComposeFile tries to locate the Docker Compose file in the given directory
// It only supports the standard patterns: docker-compose.yml, docker-compose.yaml, compose.yml, compose.yaml
func FindComposeFile(directory string) (string, error) {
	// Standard file patterns to check in order of preference
	standardPatterns := []string{
		"docker-compose.yaml",
		"docker-compose.yml",
		"compose.yaml",
		"compose.yml",
	}

	// Check each standard pattern
	for _, pattern := range standardPatterns {
		path := filepath.Join(directory, pattern)
		if fileExists(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("no standard docker compose file found in %s", directory)
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CreateBackup creates a backup of the specified Docker Compose file for a service
func (bm *BackupManager) CreateBackup(composeFilePath string, service string, imageTag string) (RollbackEntry, error) {
	// Generate a timestamped backup file name
	timestamp := time.Now()
	backupFileName := fmt.Sprintf("%s-%s-%s.yml", service, imageTag, timestamp.Format("20060102-150405"))
	backupPath := filepath.Join(bm.BackupDir, backupFileName)

	// Create a new backup file
	if err := copyFile(composeFilePath, backupPath); err != nil {
		return RollbackEntry{}, fmt.Errorf("failed to create backup: %w", err)
	}

	// Create and return a rollback entry with the backup details
	entry := RollbackEntry{
		ServiceName: service,
		ImageTag:    imageTag,
		Timestamp:   timestamp,
		ComposeFile: backupPath,
	}

	// Clean up old backups if we're exceeding the max history
	if err := bm.CleanupOldBackups(service); err != nil {
		// Log but don't fail the operation
		fmt.Printf("Warning: Failed to clean up old backups: %v\n", err)
	}

	return entry, nil
}

// GetBackupHistory returns a list of available backups for the specified service
func (bm *BackupManager) GetBackupHistory(service string) ([]RollbackEntry, error) {
	// Read all files in the backup directory
	files, err := os.ReadDir(bm.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter for files that match the service
	var entries []RollbackEntry
	prefix := service + "-"

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) {
			// Parse the file name to extract information
			parts := strings.Split(file.Name(), "-")
			if len(parts) < 3 {
				continue // Skip files with invalid format
			}

			// Extract the image tag (could be more complex in real implementation)
			imageTag := parts[1]

			// Try to parse the timestamp from the filename
			var timestamp time.Time
			if len(parts) >= 4 {
				// Handle format like service-tag-20060102-150405.yml
				dateStr := parts[2] + "-" + strings.TrimSuffix(parts[3], filepath.Ext(parts[3]))
				if t, err := time.Parse("20060102-150405", dateStr); err == nil {
					timestamp = t
				} else {
					// Fall back to file modification time if we can't parse the timestamp
					if fileInfo, err := file.Info(); err == nil {
						timestamp = fileInfo.ModTime()
					}
				}
			}

			entries = append(entries, RollbackEntry{
				ServiceName: service,
				ImageTag:    imageTag,
				Timestamp:   timestamp,
				ComposeFile: filepath.Join(bm.BackupDir, file.Name()),
			})
		}
	}

	// Sort entries by timestamp, most recent first
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

// CleanupOldBackups removes older backups to respect the MaxHistory limit
func (bm *BackupManager) CleanupOldBackups(service string) error {
	// Get all backups for the service
	entries, err := bm.GetBackupHistory(service)
	if err != nil {
		return fmt.Errorf("failed to get backup history: %w", err)
	}

	// Keep only the MaxHistory most recent backups
	if len(entries) > bm.MaxHistory {
		// The entries are already sorted with most recent first,
		// so we delete from MaxHistory to the end
		for i := bm.MaxHistory; i < len(entries); i++ {
			if err := os.Remove(entries[i].ComposeFile); err != nil {
				return fmt.Errorf("failed to remove old backup %s: %w", entries[i].ComposeFile, err)
			}
		}
	}

	return nil
}

// RestoreFromBackup restores a service from a backup file
func (bm *BackupManager) RestoreFromBackup(entry RollbackEntry, targetComposeFile string) error {
	// Ensure the backup file exists
	if _, err := os.Stat(entry.ComposeFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", entry.ComposeFile)
	}

	// Create a backup of the current file before restoration
	backupBeforeRestore := filepath.Join(
		filepath.Dir(targetComposeFile),
		fmt.Sprintf("%s.pre-rollback.%s",
			filepath.Base(targetComposeFile),
			time.Now().Format("20060102-150405"),
		),
	)

	if err := copyFile(targetComposeFile, backupBeforeRestore); err != nil {
		return fmt.Errorf("failed to create pre-rollback backup: %w", err)
	}

	// Restore the backup file
	if err := copyFile(entry.ComposeFile, targetComposeFile); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure the file is written to disk
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// GetServices returns a list of all services that have backups
func (bm *BackupManager) GetServices() ([]string, error) {
	// Create a set to deduplicate service names
	serviceSet := make(map[string]struct{})

	// Read all backup files
	files, err := os.ReadDir(bm.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Extract service names from backup filenames
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse the filename to extract the service name
		parts := strings.Split(file.Name(), "-")
		if len(parts) < 2 {
			continue
		}

		// The service name is the first part
		serviceName := parts[0]
		serviceSet[serviceName] = struct{}{}
	}

	// Convert set to slice
	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}

	return services, nil
}
