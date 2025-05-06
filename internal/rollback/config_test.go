/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRollbackConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RollbackConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: RollbackConfig{
				ComposeFilePath: "/path/to/docker-compose.yml",
				MaxHistory:      5,
			},
			wantErr: false,
		},
		{
			name: "Missing compose file path",
			config: RollbackConfig{
				MaxHistory: 5,
			},
			wantErr: true,
		},
		{
			name: "Invalid max history",
			config: RollbackConfig{
				ComposeFilePath: "/path/to/docker-compose.yml",
				MaxHistory:      0,
			},
			wantErr: true,
		},
		{
			name: "Negative max history",
			config: RollbackConfig{
				ComposeFilePath: "/path/to/docker-compose.yml",
				MaxHistory:      -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRollbackConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name         string
		initial      RollbackConfig
		expected     RollbackConfig
		checkSpecial func(t *testing.T, config RollbackConfig)
	}{
		{
			name: "Apply defaults to empty config",
			initial: RollbackConfig{
				ComposeFilePath: "/path/to/docker-compose.yml",
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                filepath.Join("/path/to", "backups"),
				MaxHistory:               10,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "docker-compose.yml",
			},
		},
		{
			name: "Respect explicitly set values",
			initial: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                "/custom/backup/dir",
				MaxHistory:               5,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "compose.yaml",
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                "/custom/backup/dir",
				MaxHistory:               5,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "compose.yaml",
			},
		},
		{
			name: "Fix invalid max history",
			initial: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				MaxHistory:               0,
				DefaultRollbackOnFailure: true,
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                filepath.Join("/path/to", "backups"),
				MaxHistory:               10,
				DefaultRollbackOnFailure: true,
				ComposeFilePattern:       "docker-compose.yml",
			},
		},
		{
			name: "Override non-standard ComposeFilePattern",
			initial: RollbackConfig{
				ComposeFilePath:    "/path/to/docker-compose.yml",
				MaxHistory:         5,
				ComposeFilePattern: "docker-compose.prod.yml",
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                filepath.Join("/path/to", "backups"),
				MaxHistory:               5,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "docker-compose.yml",
			},
		},
		{
			name: "Allow standard compose.yml pattern",
			initial: RollbackConfig{
				ComposeFilePath:    "/path/to/docker-compose.yml",
				MaxHistory:         5,
				ComposeFilePattern: "compose.yml",
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                filepath.Join("/path/to", "backups"),
				MaxHistory:               5,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "compose.yml",
			},
		},
		{
			name: "Allow standard compose.yaml pattern",
			initial: RollbackConfig{
				ComposeFilePath:    "/path/to/docker-compose.yml",
				MaxHistory:         5,
				ComposeFilePattern: "compose.yaml",
			},
			expected: RollbackConfig{
				ComposeFilePath:          "/path/to/docker-compose.yml",
				BackupDir:                filepath.Join("/path/to", "backups"),
				MaxHistory:               5,
				DefaultRollbackOnFailure: false,
				ComposeFilePattern:       "compose.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.initial
			config.ApplyDefaults()

			assert.Equal(t, tt.expected.ComposeFilePath, config.ComposeFilePath)
			assert.Equal(t, tt.expected.BackupDir, config.BackupDir)
			assert.Equal(t, tt.expected.MaxHistory, config.MaxHistory)
			assert.Equal(t, tt.expected.DefaultRollbackOnFailure, config.DefaultRollbackOnFailure,
				"DefaultRollbackOnFailure should be %v but got %v",
				tt.expected.DefaultRollbackOnFailure, config.DefaultRollbackOnFailure)
			assert.Equal(t, tt.expected.ComposeFilePattern, config.ComposeFilePattern,
				"ComposeFilePattern should be %v but got %v",
				tt.expected.ComposeFilePattern, config.ComposeFilePattern)

			if tt.checkSpecial != nil {
				tt.checkSpecial(t, config)
			}
		})
	}
}
