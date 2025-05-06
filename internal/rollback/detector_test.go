/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dosync/internal/health"
	"dosync/internal/replica"
	"dosync/internal/strategy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHealthChecker implements the HealthChecker interface for testing
type MockHealthChecker struct {
	// ReturnHealth determines whether Check() should return healthy or not
	ReturnHealth bool
	// ReturnError is the error that will be returned by Check()
	ReturnError error
	// CheckCallCount tracks how many times Check() was called
	CheckCallCount int
}

// Check implements the HealthChecker interface for testing
func (m *MockHealthChecker) Check(replica replica.Replica) (bool, error) {
	m.CheckCallCount++
	return m.ReturnHealth, m.ReturnError
}

// CheckWithDetails implements the HealthChecker interface for testing
func (m *MockHealthChecker) CheckWithDetails(replica replica.Replica) (health.HealthCheckResult, error) {
	m.CheckCallCount++
	return health.HealthCheckResult{
		Healthy:   m.ReturnHealth,
		Message:   "Mock health check",
		Timestamp: time.Now(),
	}, m.ReturnError
}

// Configure implements the HealthChecker interface for testing
func (m *MockHealthChecker) Configure(config health.HealthCheckConfig) error {
	return nil
}

// GetType implements the HealthChecker interface for testing
func (m *MockHealthChecker) GetType() health.HealthCheckType {
	return health.HTTPHealthCheck
}

func setupTestDetectorDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "detector-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Create a test compose file
	composeContent := `
version: '3'
services:
  web:
    image: nginx:latest
`
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	err = os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Create backup directory
	backupDir := filepath.Join(tempDir, "backups")
	err = os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)

	return tempDir
}

func TestNewDeploymentMonitor(t *testing.T) {
	tempDir := setupTestDetectorDir(t)
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	backupDir := filepath.Join(tempDir, "backups")

	mockHealthChecker := &MockHealthChecker{
		ReturnHealth: true,
		ReturnError:  nil,
	}

	// Test with valid configuration
	config := RollbackConfig{
		ComposeFilePath: composePath,
		BackupDir:       backupDir,
		MaxHistory:      5,
	}
	config.ApplyDefaults()

	monitor, err := NewDeploymentMonitor(config, mockHealthChecker)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, config, monitor.config)
	assert.NotNil(t, monitor.backupManager)
	assert.NotNil(t, monitor.healthChecker)
	assert.Empty(t, monitor.CurrentDeployments)
}

func TestDeploymentMonitor_StartMonitoring(t *testing.T) {
	tempDir := setupTestDetectorDir(t)
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	backupDir := filepath.Join(tempDir, "backups")

	mockHealthChecker := &MockHealthChecker{
		ReturnHealth: true,
		ReturnError:  nil,
	}

	// Create the monitor
	config := RollbackConfig{
		ComposeFilePath: composePath,
		BackupDir:       backupDir,
		MaxHistory:      5,
	}
	config.ApplyDefaults()

	monitor, err := NewDeploymentMonitor(config, mockHealthChecker)
	require.NoError(t, err)

	// Start monitoring a service
	err = monitor.StartMonitoring("web", "v2", "v1", true, 3)
	require.NoError(t, err)

	// Verify the service is being monitored
	assert.Len(t, monitor.CurrentDeployments, 1)
	assert.Contains(t, monitor.CurrentDeployments, "web")

	// Verify the deployment state
	state := monitor.CurrentDeployments["web"]
	assert.Equal(t, "web", state.ServiceName)
	assert.Equal(t, "v2", state.NewImageTag)
	assert.Equal(t, "v1", state.OldImageTag)
	assert.True(t, state.RollbackOnFailure)
	assert.Equal(t, 3, state.MaxHealthCheckAttempts)
	assert.Equal(t, 0, state.HealthCheckAttempts)
	assert.False(t, state.StartTime.IsZero())
}

func TestDeploymentMonitor_StopMonitoring(t *testing.T) {
	tempDir := setupTestDetectorDir(t)
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	backupDir := filepath.Join(tempDir, "backups")

	mockHealthChecker := &MockHealthChecker{
		ReturnHealth: true,
		ReturnError:  nil,
	}

	// Create the monitor
	config := RollbackConfig{
		ComposeFilePath: composePath,
		BackupDir:       backupDir,
		MaxHistory:      5,
	}
	config.ApplyDefaults()

	monitor, err := NewDeploymentMonitor(config, mockHealthChecker)
	require.NoError(t, err)

	// Start monitoring a service
	err = monitor.StartMonitoring("web", "v2", "v1", true, 3)
	require.NoError(t, err)
	assert.Len(t, monitor.CurrentDeployments, 1)

	// Stop monitoring
	monitor.StopMonitoring("web")
	assert.Empty(t, monitor.CurrentDeployments)

	// Stopping a service that's not monitored should be a no-op
	monitor.StopMonitoring("unknown")
	assert.Empty(t, monitor.CurrentDeployments)
}

func TestDeploymentMonitor_ShouldRollback(t *testing.T) {
	tempDir := setupTestDetectorDir(t)
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	backupDir := filepath.Join(tempDir, "backups")

	mockHealthChecker := &MockHealthChecker{
		ReturnHealth: true,
		ReturnError:  nil,
	}

	// Create the monitor
	config := RollbackConfig{
		ComposeFilePath:          composePath,
		BackupDir:                backupDir,
		MaxHistory:               5,
		DefaultRollbackOnFailure: true,
	}
	config.ApplyDefaults()

	monitor, err := NewDeploymentMonitor(config, mockHealthChecker)
	require.NoError(t, err)

	tests := []struct {
		name           string
		service        string
		healthStatus   bool
		monitoring     bool
		rollbackFlag   bool
		strategyFlag   bool
		expectRollback bool
	}{
		{
			name:           "Healthy service should not rollback",
			service:        "web",
			healthStatus:   true,
			monitoring:     true,
			rollbackFlag:   true,
			strategyFlag:   true,
			expectRollback: false,
		},
		{
			name:           "Unhealthy monitored service with rollback flag true",
			service:        "web",
			healthStatus:   false,
			monitoring:     true,
			rollbackFlag:   true,
			strategyFlag:   false,
			expectRollback: true,
		},
		{
			name:           "Unhealthy monitored service with rollback flag false",
			service:        "web",
			healthStatus:   false,
			monitoring:     true,
			rollbackFlag:   false,
			strategyFlag:   true,
			expectRollback: false,
		},
		{
			name:           "Unhealthy unmonitored service with strategy flag true",
			service:        "db",
			healthStatus:   false,
			monitoring:     false,
			rollbackFlag:   false,
			strategyFlag:   true,
			expectRollback: true,
		},
		{
			name:           "Unhealthy unmonitored service with strategy flag false but default true",
			service:        "cache",
			healthStatus:   false,
			monitoring:     false,
			rollbackFlag:   false,
			strategyFlag:   false,
			expectRollback: true, // Because DefaultRollbackOnFailure is true
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear deployments
			monitor.CurrentDeployments = make(map[string]*DeploymentState)

			// Set up monitored service
			if tc.monitoring {
				err = monitor.StartMonitoring(tc.service, "v2", "v1", tc.rollbackFlag, 3)
				require.NoError(t, err)
			}

			// Create strategy config
			strategyConfig := strategy.StrategyConfig{
				RollbackOnFailure: tc.strategyFlag,
			}

			// Test ShouldRollback
			result := monitor.ShouldRollback(tc.service, tc.healthStatus, strategyConfig)
			assert.Equal(t, tc.expectRollback, result)
		})
	}
}

func TestDeploymentMonitor_CheckDeploymentHealth(t *testing.T) {
	tempDir := setupTestDetectorDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	tests := []struct {
		name               string
		healthStatus       bool
		attempts           int
		maxAttempts        int
		rollbackOnFailure  bool
		expectRollback     bool
		expectHealthStatus bool
	}{
		{
			name:               "Healthy service",
			healthStatus:       true,
			attempts:           0,
			maxAttempts:        3,
			rollbackOnFailure:  true,
			expectRollback:     false,
			expectHealthStatus: true,
		},
		{
			name:               "Unhealthy service but not enough attempts yet",
			healthStatus:       false,
			attempts:           1,
			maxAttempts:        3,
			rollbackOnFailure:  true,
			expectRollback:     false,
			expectHealthStatus: false,
		},
		{
			name:               "Unhealthy service, max attempts reached with rollback enabled",
			healthStatus:       false,
			attempts:           2,
			maxAttempts:        3,
			rollbackOnFailure:  true,
			expectRollback:     true,
			expectHealthStatus: false,
		},
		{
			name:               "Unhealthy service, max attempts reached but rollback disabled",
			healthStatus:       false,
			attempts:           2,
			maxAttempts:        3,
			rollbackOnFailure:  false,
			expectRollback:     false,
			expectHealthStatus: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a backup directory for each test case to avoid conflicts
			testBackupDir := filepath.Join(backupDir, tc.name)
			err := os.MkdirAll(testBackupDir, 0755)
			require.NoError(t, err)

			// Create a mock health checker
			mockHealthChecker := &MockHealthChecker{
				ReturnHealth: tc.healthStatus,
				ReturnError:  nil,
			}

			// Create a backup manager and add a backup for testing rollback
			bm, err := NewBackupManager(testBackupDir, 5, "docker-compose.yml")
			require.NoError(t, err)

			// Create a test backup file for the web service
			backupContent := `
version: '3'
services:
  web:
    image: nginx:v1
`
			backupPath := filepath.Join(testBackupDir, "web-v1-20230101-120000.yml")
			err = os.WriteFile(backupPath, []byte(backupContent), 0644)
			require.NoError(t, err)

			// Create a test compose file specific to this test case
			testComposePath := filepath.Join(tempDir, fmt.Sprintf("docker-compose-%s.yml", tc.name))
			err = os.WriteFile(testComposePath, []byte("version: '3'\nservices:\n  web:\n    image: nginx:v2"), 0644)
			require.NoError(t, err)

			// Create the monitor
			config := RollbackConfig{
				ComposeFilePath: testComposePath,
				BackupDir:       testBackupDir,
				MaxHistory:      5,
			}
			config.ApplyDefaults()

			monitor := &DeploymentMonitor{
				backupManager:      bm,
				healthChecker:      mockHealthChecker,
				config:             config,
				CurrentDeployments: make(map[string]*DeploymentState),
			}

			// Set up the deployment state for testing
			monitor.CurrentDeployments["web"] = &DeploymentState{
				ServiceName:            "web",
				NewImageTag:            "v2",
				OldImageTag:            "v1",
				StartTime:              time.Now(),
				HealthCheckAttempts:    tc.attempts,
				MaxHealthCheckAttempts: tc.maxAttempts,
				RollbackOnFailure:      tc.rollbackOnFailure,
			}

			// Store the initial state for later comparison
			initialDeploymentCount := len(monitor.CurrentDeployments)

			// Call CheckDeploymentHealth
			healthy, err := monitor.CheckDeploymentHealth("web", "1")
			require.NoError(t, err)
			assert.Equal(t, tc.expectHealthStatus, healthy)

			// Verify health check was called
			assert.Equal(t, 1, mockHealthChecker.CheckCallCount)

			// Verify rollback behavior
			if tc.expectRollback {
				// If rollback was expected, the service should no longer be monitored
				assert.Equal(t, initialDeploymentCount-1, len(monitor.CurrentDeployments),
					"Service should no longer be monitored after rollback")

				// Verify the compose file was restored with the backup content
				restoredContent, err := os.ReadFile(testComposePath)
				require.NoError(t, err)
				assert.Contains(t, string(restoredContent), "nginx:v1")
			} else {
				// If not rolled back, the service should still be monitored
				assert.Equal(t, initialDeploymentCount, len(monitor.CurrentDeployments),
					"Number of monitored services should remain the same")

				// Check that attempt counter was incremented
				state, exists := monitor.CurrentDeployments["web"]
				assert.True(t, exists, "Service should still be monitored")
				if exists {
					assert.Equal(t, tc.attempts+1, state.HealthCheckAttempts,
						"Health check attempts should be incremented")
				}
			}
		})
	}
}
