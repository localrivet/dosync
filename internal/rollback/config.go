/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"fmt"
	"path/filepath"
)

// RollbackConfig contains configuration options for the rollback controller
type RollbackConfig struct {
	// ComposeFilePath is the path to the main docker-compose.yml file
	ComposeFilePath string

	// ComposeFilePattern is the standard pattern for Docker Compose files
	// Example: "docker-compose.yml" or "compose.yaml"
	ComposeFilePattern string

	// BackupDir is the directory where backup files will be stored
	BackupDir string

	// MaxHistory is the maximum number of backup entries to keep per service
	MaxHistory int

	// DefaultRollbackOnFailure determines whether to automatically roll back failed deployments
	DefaultRollbackOnFailure bool
}

// Validate checks if the rollback configuration is valid
func (c *RollbackConfig) Validate() error {
	if c.ComposeFilePath == "" {
		return fmt.Errorf("compose file path is required")
	}

	if c.MaxHistory <= 0 {
		return fmt.Errorf("max history must be greater than 0")
	}

	return nil
}

// ApplyDefaults sets default values for unspecified configuration options
func (c *RollbackConfig) ApplyDefaults() {
	if c.BackupDir == "" {
		// By default, store backups in a subdirectory of the compose file location
		c.BackupDir = filepath.Join(filepath.Dir(c.ComposeFilePath), "backups")
	}

	if c.MaxHistory <= 0 {
		c.MaxHistory = 10
	}

	// Set default compose file pattern if not specified
	// We only support docker-compose.yml/yaml or compose.yml/yaml as standards
	if c.ComposeFilePattern == "" {
		c.ComposeFilePattern = "docker-compose.yml"
	} else if c.ComposeFilePattern != "docker-compose.yml" &&
		c.ComposeFilePattern != "docker-compose.yaml" &&
		c.ComposeFilePattern != "compose.yml" &&
		c.ComposeFilePattern != "compose.yaml" {
		// If the pattern doesn't match our standards, default to docker-compose.yml
		c.ComposeFilePattern = "docker-compose.yml"
	}

	// Default to true only for new configs
	// We don't need to check the zero value since for booleans it's false
	// and if it's explicitly set to false, we want to keep it that way
}
