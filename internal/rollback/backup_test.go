/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestNewBackupManager(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name               string
		backupDir          string
		maxHistory         int
		composeFilePattern string
		expectError        bool
		expectedPattern    string
	}{
		{
			name:               "Valid with defaults",
			backupDir:          tempDir,
			maxHistory:         5,
			composeFilePattern: "",
			expectError:        false,
			expectedPattern:    "docker-compose.yml",
		},
		{
			name:               "Valid with custom pattern",
			backupDir:          tempDir,
			maxHistory:         5,
			composeFilePattern: "compose.yaml",
			expectError:        false,
			expectedPattern:    "compose.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bm, err := NewBackupManager(tc.backupDir, tc.maxHistory, tc.composeFilePattern)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.backupDir, bm.BackupDir)
			assert.Equal(t, tc.maxHistory, bm.MaxHistory)
			assert.Equal(t, tc.expectedPattern, bm.ComposeFilePattern)

			// Check that the backup directory was created
			_, err = os.Stat(tc.backupDir)
			assert.NoError(t, err)
		})
	}
}

func TestFindComposeFile(t *testing.T) {
	tempDir := setupTestDir(t)

	tests := []struct {
		name         string
		setupFiles   map[string]string
		expectError  bool
		expectedFile string
	}{
		{
			name: "Find docker-compose.yaml",
			setupFiles: map[string]string{
				"docker-compose.yaml": "content",
			},
			expectError:  false,
			expectedFile: filepath.Join(tempDir, "docker-compose.yaml"),
		},
		{
			name: "Find docker-compose.yml",
			setupFiles: map[string]string{
				"docker-compose.yml": "content",
			},
			expectError:  false,
			expectedFile: filepath.Join(tempDir, "docker-compose.yml"),
		},
		{
			name: "Find compose.yaml",
			setupFiles: map[string]string{
				"compose.yaml": "content",
			},
			expectError:  false,
			expectedFile: filepath.Join(tempDir, "compose.yaml"),
		},
		{
			name: "Find compose.yml",
			setupFiles: map[string]string{
				"compose.yml": "content",
			},
			expectError:  false,
			expectedFile: filepath.Join(tempDir, "compose.yml"),
		},
		{
			name: "Prefer docker-compose.yaml over others",
			setupFiles: map[string]string{
				"docker-compose.yaml": "content1",
				"docker-compose.yml":  "content2",
				"compose.yaml":        "content3",
				"compose.yml":         "content4",
			},
			expectError:  false,
			expectedFile: filepath.Join(tempDir, "docker-compose.yaml"),
		},
		{
			name:        "Error when no standard file exists",
			setupFiles:  map[string]string{},
			expectError: true,
		},
		{
			name: "Ignore non-standard patterns",
			setupFiles: map[string]string{
				"compose.prod.yaml": "content",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up from previous test
			files, _ := os.ReadDir(tempDir)
			for _, file := range files {
				os.Remove(filepath.Join(tempDir, file.Name()))
			}

			// Create test files
			for name, content := range tc.setupFiles {
				createTestFile(t, tempDir, name, content)
			}

			// Test FindComposeFile
			file, err := FindComposeFile(tempDir)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedFile, file)
		})
	}
}

func TestBackupManager_CreateAndGetBackups(t *testing.T) {
	tempDir := setupTestDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	// Create test compose file
	composeFile := createTestFile(t, tempDir, "docker-compose.yml", "version: '3'\nservices:\n  web:\n    image: nginx:latest")

	// Create backup manager
	bm, err := NewBackupManager(backupDir, 3, "docker-compose.yml")
	require.NoError(t, err)

	// Test CreateBackup
	entry, err := bm.CreateBackup(composeFile, "web", "latest")
	require.NoError(t, err)

	// Verify entry
	assert.Equal(t, "web", entry.ServiceName)
	assert.Equal(t, "latest", entry.ImageTag)
	assert.NotEmpty(t, entry.ComposeFile)
	assert.NotZero(t, entry.Timestamp)

	// Test GetBackupHistory
	entries, err := bm.GetBackupHistory("web")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entry.ServiceName, entries[0].ServiceName)
	assert.Equal(t, entry.ImageTag, entries[0].ImageTag)
	assert.Equal(t, entry.ComposeFile, entries[0].ComposeFile)

	// Create multiple backups to test sorting and cleanup
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	bm.CreateBackup(composeFile, "web", "v2")
	time.Sleep(10 * time.Millisecond)
	bm.CreateBackup(composeFile, "web", "v3")
	time.Sleep(10 * time.Millisecond)
	bm.CreateBackup(composeFile, "web", "v4")

	// Test that GetBackupHistory returns entries and they are limited to MaxHistory
	entries, err = bm.GetBackupHistory("web")
	require.NoError(t, err)

	// Verify the right count
	assert.Len(t, entries, 3, "Should only have MaxHistory(3) entries")

	// Verify cleanup worked (only 3 entries should exist)
	files, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	assert.Len(t, files, 3, "Should only have MaxHistory(3) files")
}

func TestBackupManager_RestoreFromBackup(t *testing.T) {
	tempDir := setupTestDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	// Create original compose file
	originalContent := "version: '3'\nservices:\n  web:\n    image: nginx:latest"
	composeFile := createTestFile(t, tempDir, "docker-compose.yml", originalContent)

	// Create backup manager and backup
	bm, err := NewBackupManager(backupDir, 3, "docker-compose.yml")
	require.NoError(t, err)
	entry, err := bm.CreateBackup(composeFile, "web", "latest")
	require.NoError(t, err)

	// Modify the original file
	newContent := "version: '3'\nservices:\n  web:\n    image: nginx:alpine"
	err = os.WriteFile(composeFile, []byte(newContent), 0644)
	require.NoError(t, err)

	// Check the file was changed
	content, err := os.ReadFile(composeFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(content))

	// Restore from backup
	err = bm.RestoreFromBackup(entry, composeFile)
	require.NoError(t, err)

	// Verify the file was restored to original content
	content, err = os.ReadFile(composeFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(content))

	// Verify a pre-rollback backup was created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	preRollbackFound := false
	for _, file := range files {
		if strings.Contains(file.Name(), "pre-rollback") {
			preRollbackFound = true
			break
		}
	}
	assert.True(t, preRollbackFound, "Pre-rollback backup file should exist")
}
