/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dosync/internal/replica"
)

// Helper function to create a minimal valid config for Command health checks
func createValidCommandConfig(command string) HealthCheckConfig {
	return HealthCheckConfig{
		Type:    CommandHealthCheck,
		Command: command,
		Timeout: 1 * time.Second, // Short timeout for tests
	}
}

// TestNewCommandHealthChecker tests the creation of a new CommandHealthChecker
func TestNewCommandHealthChecker(t *testing.T) {
	config := createValidCommandConfig("echo hello")

	checker, err := NewCommandHealthChecker(config)
	require.NoError(t, err, "NewCommandHealthChecker should not return an error with valid config")
	require.NotNil(t, checker, "NewCommandHealthChecker should return a non-nil checker")
	assert.Equal(t, CommandHealthCheck, checker.GetType(), "Checker type should be CommandHealthCheck")

	// Test with invalid config type
	invalidConfig := createValidCommandConfig("echo hello")
	invalidConfig.Type = HTTPHealthCheck
	_, err = NewCommandHealthChecker(invalidConfig)
	require.Error(t, err, "NewCommandHealthChecker should return an error with invalid config type")
}

// TestCommandHealthChecker_Check_MockContainer mocks a container by using a test-specific environment variable
// This allows us to test without needing a real Docker installation
func TestCommandHealthChecker_Check_MockContainer(t *testing.T) {
	// Skip if we can't execute commands (e.g., in certain CI environments)
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("Skipping test that requires 'sh' command")
	}

	// Mock replica for testing
	rep := replica.Replica{ContainerID: "mock-container-for-testing"}

	tests := []struct {
		name            string
		containerID     string // Use specific container ID for test case if needed
		command         string // Command to execute
		mockExitCode    int    // Mock exit code for our mock-docker script
		expectedHealthy bool
		expectedErr     bool
		expectedMsg     string
	}{
		{
			name:            "Successful command",
			command:         "echo hello",
			mockExitCode:    0, // Success
			expectedHealthy: true,
			expectedMsg:     "executed successfully",
		},
		{
			name:            "Failed command",
			command:         "exit 1",
			mockExitCode:    1, // Failure
			expectedHealthy: false,
			expectedMsg:     "failed",
		},
		{
			name:            "Empty container ID",
			containerID:     "", // Explicitly set empty container ID
			command:         "echo hello",
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsg:     "Container ID is empty",
		},
		{
			name:            "Empty command",
			command:         "", // Empty command
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsg:     "No command specified for command health check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock docker exec command for testing
			// This temporarily modifies PATH to include our directory with a mock docker script
			origPath := os.Getenv("PATH")
			tmpDir := t.TempDir()

			// Create a mock docker script that outputs the exit code we want
			if tt.name != "Empty container ID" && tt.name != "Empty command" {
				mockDockerScript := fmt.Sprintf(`#!/bin/sh
echo "Mock docker exec for testing"
exit %d
`, tt.mockExitCode)

				err := os.WriteFile(tmpDir+"/docker", []byte(mockDockerScript), 0755)
				require.NoError(t, err, "Failed to write mock docker script")

				// Set PATH to include our directory with mock docker
				err = os.Setenv("PATH", tmpDir+":"+origPath)
				require.NoError(t, err, "Failed to set PATH")
				defer os.Setenv("PATH", origPath) // Restore original PATH
			}

			// Create checker
			config := createValidCommandConfig(tt.command)
			checker, err := NewCommandHealthChecker(config)

			// Skip the test if we couldn't create the checker (this would be for invalid configs)
			if err != nil {
				// If we expect this error (e.g., for empty command), that's fine
				if tt.command == "" {
					assert.Error(t, err)
					return
				}
				// Otherwise it's unexpected
				t.Fatalf("Failed to create checker: %v", err)
			}

			// Determine the replica to use for the test
			currentRep := rep
			if tt.containerID != "" || tt.name == "Empty container ID" {
				currentRep.ContainerID = tt.containerID
			}

			// Perform the check
			result, err := checker.CheckWithDetails(currentRep)

			// Assert results
			assert.Equal(t, tt.expectedHealthy, result.Healthy, "Health status mismatch")
			if tt.expectedMsg != "" {
				assert.Contains(t, result.Message, tt.expectedMsg, "Message mismatch")
			}

			if tt.expectedErr {
				assert.Error(t, err, "Expected an error, but got nil")
			} else if tt.name == "Failed command" {
				// Failed command will return an error, but it's an expected part of the test
				// so we don't assert no error in this case
			} else if tt.name != "Empty container ID" && tt.name != "Empty command" {
				assert.NoError(t, err, "Expected no error, but got: %v", err)
			}

			// Check status tracking (ensure UpdateStatus was called)
			healthyStatus, msg, _ := checker.GetStatus()
			assert.Equal(t, tt.expectedHealthy, healthyStatus, "Stored health status mismatch")
			if tt.expectedMsg != "" {
				assert.Contains(t, msg, tt.expectedMsg, "Stored message mismatch")
			}
		})
	}
}

// TestCommandHealthChecker_ShouldCheck tests the rate limiting behaviour
func TestCommandHealthChecker_ShouldCheck(t *testing.T) {
	// Skip if we can't execute commands (e.g., in certain CI environments)
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("Skipping test that requires 'sh' command")
	}

	// Create a mock docker exec command for testing
	origPath := os.Getenv("PATH")
	tmpDir := t.TempDir()

	// Create a mock docker script that always succeeds
	mockDockerScript := `#!/bin/sh
echo "Mock docker exec for testing"
exit 0
`

	err := os.WriteFile(tmpDir+"/docker", []byte(mockDockerScript), 0755)
	require.NoError(t, err, "Failed to write mock docker script")

	// Set PATH to include our directory with mock docker
	err = os.Setenv("PATH", tmpDir+":"+origPath)
	require.NoError(t, err, "Failed to set PATH")
	defer os.Setenv("PATH", origPath) // Restore original PATH

	config := createValidCommandConfig("echo hello")
	// Use a valid retry interval (>= MinRetryInterval which is 100ms)
	config.RetryInterval = 100 * time.Millisecond
	checker, err := NewCommandHealthChecker(config)
	require.NoError(t, err, "Failed to create checker: %v", err)

	rep := replica.Replica{ContainerID: "test-container"}

	// First check should always proceed
	assert.True(t, checker.ShouldCheck(), "First check should proceed")
	_, err = checker.Check(rep)
	require.NoError(t, err)

	// Second check immediately after should be skipped
	assert.False(t, checker.ShouldCheck(), "Second check immediately after should be skipped")
	_, err = checker.Check(rep)
	require.NoError(t, err) // Check shouldn't return error, just the cached status

	// Wait for longer than the retry interval
	time.Sleep(config.RetryInterval + 10*time.Millisecond)

	// Third check after interval should proceed
	assert.True(t, checker.ShouldCheck(), "Third check after interval should proceed")
	_, err = checker.Check(rep)
	require.NoError(t, err)
}
