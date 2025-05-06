/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockBackupManager is a mock implementation of the backup manager
type MockBackupManager struct {
	CreateBackupFn      func(composeFilePath, service, imageTag string) (RollbackEntry, error)
	GetBackupHistoryFn  func(service string) ([]RollbackEntry, error)
	CleanupOldBackupsFn func(service string) error
	RestoreFromBackupFn func(entry RollbackEntry, targetComposeFile string) error
	GetServicesFn       func() ([]string, error)
}

// Ensure MockBackupManager implements BackupOperations
var _ BackupOperations = (*MockBackupManager)(nil)

func (m *MockBackupManager) CreateBackup(composeFilePath, service, imageTag string) (RollbackEntry, error) {
	if m.CreateBackupFn != nil {
		return m.CreateBackupFn(composeFilePath, service, imageTag)
	}
	return RollbackEntry{}, nil
}

func (m *MockBackupManager) GetBackupHistory(service string) ([]RollbackEntry, error) {
	if m.GetBackupHistoryFn != nil {
		return m.GetBackupHistoryFn(service)
	}
	return []RollbackEntry{}, nil
}

func (m *MockBackupManager) CleanupOldBackups(service string) error {
	if m.CleanupOldBackupsFn != nil {
		return m.CleanupOldBackupsFn(service)
	}
	return nil
}

func (m *MockBackupManager) RestoreFromBackup(entry RollbackEntry, targetComposeFile string) error {
	if m.RestoreFromBackupFn != nil {
		return m.RestoreFromBackupFn(entry, targetComposeFile)
	}
	return nil
}

func (m *MockBackupManager) GetServices() ([]string, error) {
	if m.GetServicesFn != nil {
		return m.GetServicesFn()
	}
	return []string{}, nil
}

// TestExecCommand is used to mock exec.Command for testing
type TestExecCommand struct {
	mockOutput []byte
	mockError  error
}

// mockCommandFunc returns a replacement for exec.Command that returns a mocked command
func mockCommandFunc(mockOutput []byte, mockError error) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		// This is a special mock exec.Cmd that doesn't actually run anything
		// but will return the specified output and error
		return fakeExecCommand(mockOutput, mockError)
	}
}

// fakeExecCommand creates a command that doesn't execute anything, but returns our mock data
func fakeExecCommand(output []byte, err error) *exec.Cmd {
	// Create a minimal exec.Cmd
	cmd := &exec.Cmd{}

	// We'll use TestHelperProcess to simulate execution
	// This pattern is used in the os/exec tests in the Go standard library
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cmd.Path = os.Args[0]
	cmd.Args = append(cs, "echo", "mock output")

	// Pass our mock output through an environment variable
	cmd.Env = append(os.Environ(),
		"GO_TEST_MOCK_OUTPUT="+string(output),
		"GO_TEST_MOCK_ERROR="+boolToStr(err != nil),
	)

	return cmd
}

// boolToStr converts a boolean to a string "true" or "false"
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestHelperProcess isn't a real test, it's used to mock exec.Command
func TestHelperProcess(t *testing.T) {
	// If this isn't being run as a helper process, just return
	if os.Getenv("GO_TEST_MOCK_OUTPUT") == "" {
		return
	}

	// Get our mock parameters from environment
	output := os.Getenv("GO_TEST_MOCK_OUTPUT")
	mockError := os.Getenv("GO_TEST_MOCK_ERROR") == "true"

	// If we need to simulate an error, exit with non-zero status
	if mockError {
		os.Exit(1)
	}

	// Otherwise, print our mock output and exit successfully
	os.Stdout.WriteString(output)
	os.Exit(0)
}

// Original exec function to be restored
var originalExecCommand = execCommand

func TestNewRollbackController(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rollback-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a dummy compose file
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	err = os.WriteFile(composeFile, []byte("version: '3'"), 0644)
	assert.NoError(t, err)

	t.Run("Valid configuration", func(t *testing.T) {
		config := RollbackConfig{
			ComposeFilePath:    composeFile,
			BackupDir:          filepath.Join(tempDir, "backups"),
			MaxHistory:         5,
			ComposeFilePattern: "docker-compose.yml", // Explicitly set to match default
		}

		controller, err := NewRollbackController(config)
		assert.NoError(t, err)
		assert.NotNil(t, controller)
		assert.NotNil(t, controller.BackupManager)
		assert.Equal(t, config, controller.Config)
	})

	t.Run("Invalid configuration", func(t *testing.T) {
		config := RollbackConfig{
			// Missing ComposeFilePath
			BackupDir:  filepath.Join(tempDir, "backups"),
			MaxHistory: 5,
		}

		controller, err := NewRollbackController(config)
		assert.Error(t, err)
		assert.Nil(t, controller)
		assert.Contains(t, err.Error(), "invalid rollback configuration")
	})
}

func TestRollbackControllerImpl_PrepareRollback(t *testing.T) {
	// Setup mock backup manager
	mockEntry := RollbackEntry{
		ServiceName: "test-service",
		ImageTag:    "latest",
		Timestamp:   time.Now(),
		ComposeFile: "/path/to/backup.yml",
	}

	mockBackupManager := &MockBackupManager{
		CreateBackupFn: func(composeFilePath, service, imageTag string) (RollbackEntry, error) {
			assert.Equal(t, "test-service", service)
			assert.Equal(t, "latest", imageTag)
			return mockEntry, nil
		},
		GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
			assert.Equal(t, "test-service", service)
			return []RollbackEntry{mockEntry}, nil
		},
	}

	// Create controller with mock backup manager
	controller := &RollbackControllerImpl{
		BackupManager: mockBackupManager,
		Config: RollbackConfig{
			ComposeFilePath: "/path/to/compose.yml",
			MaxHistory:      5,
		},
	}

	// Test PrepareRollback
	err := controller.PrepareRollback("test-service")
	assert.NoError(t, err)

	// Verify the backup was created by checking GetRollbackHistory was called
	entries, err := controller.GetRollbackHistory("test-service")
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, mockEntry, entries[0])
}

func TestRollbackControllerImpl_Rollback(t *testing.T) {
	// Create a tempdir for testing
	tempDir, err := os.MkdirTemp("", "rollback-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Save the original exec function and restore it after the test
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	t.Run("Successful rollback", func(t *testing.T) {
		// Set up mock backup manager
		mockEntry := RollbackEntry{
			ServiceName: "test-service",
			ImageTag:    "v1.0.0",
			Timestamp:   time.Now(),
			ComposeFile: filepath.Join(tempDir, "backup.yml"),
		}

		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				assert.Equal(t, "test-service", service)
				return []RollbackEntry{mockEntry}, nil
			},
			RestoreFromBackupFn: func(entry RollbackEntry, targetComposeFile string) error {
				assert.Equal(t, mockEntry, entry)
				assert.Equal(t, "/path/to/compose.yml", targetComposeFile)
				return nil
			},
		}

		// Mock the exec command
		execCommand = mockCommandFunc([]byte("Container restarted successfully"), nil)

		// Create controller with mock backup manager
		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
				MaxHistory:      5,
			},
		}

		// Test the Rollback method
		err := controller.Rollback("test-service")
		assert.NoError(t, err)
	})

	t.Run("No rollback entries", func(t *testing.T) {
		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				return []RollbackEntry{}, nil
			},
		}

		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
			},
		}

		err := controller.Rollback("test-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rollback entries found")
	})

	t.Run("Error getting backup history", func(t *testing.T) {
		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				return nil, fmt.Errorf("backup history error")
			},
		}

		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
			},
		}

		err := controller.Rollback("test-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve rollback history")
	})

	t.Run("Error restoring backup", func(t *testing.T) {
		mockEntry := RollbackEntry{
			ServiceName: "test-service",
			ImageTag:    "v1.0.0",
			Timestamp:   time.Now(),
			ComposeFile: filepath.Join(tempDir, "backup.yml"),
		}

		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				return []RollbackEntry{mockEntry}, nil
			},
			RestoreFromBackupFn: func(entry RollbackEntry, targetComposeFile string) error {
				return fmt.Errorf("restore error")
			},
		}

		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
			},
		}

		err := controller.Rollback("test-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to restore from backup")
	})

	t.Run("Error restarting service", func(t *testing.T) {
		mockEntry := RollbackEntry{
			ServiceName: "test-service",
			ImageTag:    "v1.0.0",
			Timestamp:   time.Now(),
			ComposeFile: filepath.Join(tempDir, "backup.yml"),
		}

		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				return []RollbackEntry{mockEntry}, nil
			},
			RestoreFromBackupFn: func(entry RollbackEntry, targetComposeFile string) error {
				return nil
			},
		}

		// Mock the exec command to return an error
		execCommand = mockCommandFunc([]byte("ERROR: service not found"), fmt.Errorf("exec error"))

		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
			},
		}

		err := controller.Rollback("test-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to restart service")
	})
}

func TestRollbackControllerImpl_RollbackToVersion(t *testing.T) {
	// Create a tempdir for testing
	tempDir, err := os.MkdirTemp("", "rollback-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Save the original exec function and restore it after the test
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	t.Run("Successful version-specific rollback", func(t *testing.T) {
		entries := []RollbackEntry{
			{
				ServiceName: "test-service",
				ImageTag:    "v2.0.0",
				Timestamp:   time.Now(),
				ComposeFile: filepath.Join(tempDir, "backup-v2.yml"),
			},
			{
				ServiceName: "test-service",
				ImageTag:    "v1.5.0",
				Timestamp:   time.Now().Add(-time.Hour),
				ComposeFile: filepath.Join(tempDir, "backup-v1.5.yml"),
			},
			{
				ServiceName: "test-service",
				ImageTag:    "v1.0.0",
				Timestamp:   time.Now().Add(-2 * time.Hour),
				ComposeFile: filepath.Join(tempDir, "backup-v1.yml"),
			},
		}

		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				assert.Equal(t, "test-service", service)
				return entries, nil
			},
			RestoreFromBackupFn: func(entry RollbackEntry, targetComposeFile string) error {
				assert.Equal(t, "v1.5.0", entry.ImageTag)
				assert.Equal(t, "/path/to/compose.yml", targetComposeFile)
				return nil
			},
		}

		// Mock the exec command
		execCommand = mockCommandFunc([]byte("Container restarted successfully"), nil)

		// Create controller with mock backup manager
		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
				MaxHistory:      5,
			},
		}

		// Test the RollbackToVersion method
		err := controller.RollbackToVersion("test-service", "v1.5.0")
		assert.NoError(t, err)
	})

	t.Run("Version not found", func(t *testing.T) {
		entries := []RollbackEntry{
			{
				ServiceName: "test-service",
				ImageTag:    "v2.0.0",
				Timestamp:   time.Now(),
				ComposeFile: filepath.Join(tempDir, "backup-v2.yml"),
			},
			{
				ServiceName: "test-service",
				ImageTag:    "v1.0.0",
				Timestamp:   time.Now().Add(-2 * time.Hour),
				ComposeFile: filepath.Join(tempDir, "backup-v1.yml"),
			},
		}

		mockBackupManager := &MockBackupManager{
			GetBackupHistoryFn: func(service string) ([]RollbackEntry, error) {
				return entries, nil
			},
		}

		controller := &RollbackControllerImpl{
			BackupManager: mockBackupManager,
			Config: RollbackConfig{
				ComposeFilePath: "/path/to/compose.yml",
			},
		}

		err := controller.RollbackToVersion("test-service", "v1.5.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rollback entry found for service test-service with version v1.5.0")
	})
}

func TestRollbackControllerImpl_ShouldRollback(t *testing.T) {
	controller := &RollbackControllerImpl{
		Config: RollbackConfig{
			DefaultRollbackOnFailure: true,
		},
	}

	// Test healthy service should never rollback
	assert.False(t, controller.ShouldRollback("service", true, false))
	assert.False(t, controller.ShouldRollback("service", true, true))

	// Test unhealthy service with rollbackOnFailure true
	assert.True(t, controller.ShouldRollback("service", false, true))

	// Test unhealthy service with rollbackOnFailure false but DefaultRollbackOnFailure true
	assert.True(t, controller.ShouldRollback("service", false, false))

	// Test with both flags false
	controller.Config.DefaultRollbackOnFailure = false
	assert.False(t, controller.ShouldRollback("service", false, false))
}

func TestRollbackControllerImpl_CleanupOldBackups(t *testing.T) {
	// Setup test services
	services := []string{"service1", "service2"}

	// Track number of cleanup calls per service
	cleanupCalls := make(map[string]int)

	// Setup mock backup manager
	mockBackupManager := &MockBackupManager{
		GetServicesFn: func() ([]string, error) {
			return services, nil
		},
		CleanupOldBackupsFn: func(service string) error {
			cleanupCalls[service]++
			if service == "service2" {
				return errors.New("mock error")
			}
			return nil
		},
	}

	// Create controller with mock backup manager
	controller := &RollbackControllerImpl{
		BackupManager: mockBackupManager,
		Config: RollbackConfig{
			ComposeFilePath: "/path/to/compose.yml",
			MaxHistory:      2,
		},
	}

	// Test CleanupOldBackups
	err := controller.CleanupOldBackups()
	assert.Error(t, err) // Expect error due to service2
	assert.Contains(t, err.Error(), "failed to clean up old backups for service service2")

	// Verify that both services were processed
	assert.Equal(t, 1, cleanupCalls["service1"])
	assert.Equal(t, 1, cleanupCalls["service2"])

	// Test GetServices error
	mockBackupManager.GetServicesFn = func() ([]string, error) {
		return nil, errors.New("mock error")
	}
	err = controller.CleanupOldBackups()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get services")
}
