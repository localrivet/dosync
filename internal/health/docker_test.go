/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dosync/internal/replica"
)

// Helper function to create a minimal valid config for Docker health checks
func createValidDockerConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Type: DockerHealthCheck,
	}
}

// MockDockerClient provides a mock implementation of the Docker client for testing
type MockDockerClient struct {
	client.APIClient // Embed the interface
	InspectFunc      func(ctx context.Context, containerID string) (container.InspectResponse, error)
	CloseFunc        func() error
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	if m.InspectFunc != nil {
		return m.InspectFunc(ctx, containerID)
	}
	return container.InspectResponse{}, fmt.Errorf("InspectFunc not implemented")
}

func (m *MockDockerClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestNewDockerHealthChecker tests the creation of a new DockerHealthChecker
func TestNewDockerHealthChecker(t *testing.T) {
	config := createValidDockerConfig()

	checker, err := NewDockerHealthChecker(config)
	require.NoError(t, err, "NewDockerHealthChecker should not return an error with valid config")
	require.NotNil(t, checker, "NewDockerHealthChecker should return a non-nil checker")
	assert.Equal(t, DockerHealthCheck, checker.GetType(), "Checker type should be DockerHealthCheck")
	assert.NotNil(t, checker.dockerClient, "Docker client should be initialized")

	// Test with invalid config type
	invalidConfig := createValidDockerConfig()
	invalidConfig.Type = HTTPHealthCheck
	_, err = NewDockerHealthChecker(invalidConfig)
	require.Error(t, err, "NewDockerHealthChecker should return an error with invalid config type")
}

// TestDockerHealthChecker_Check tests the Check method
func TestDockerHealthChecker_Check(t *testing.T) {
	// Mock replica for testing
	rep := replica.Replica{ContainerID: "test-container"}

	tests := []struct {
		name            string
		containerID     string // Use specific container ID for test case if needed
		inspectRespBase *container.ContainerJSONBase
		inspectErr      error
		expectedHealthy bool
		expectedErr     bool
		expectedMsg     string
	}{
		{
			name: "Healthy container",
			inspectRespBase: &container.ContainerJSONBase{
				State: &container.State{Health: &container.Health{Status: container.Healthy}},
			},
			expectedHealthy: true,
			expectedMsg:     fmt.Sprintf("Container %s is healthy", rep.ContainerID),
		},
		{
			name: "Unhealthy container",
			inspectRespBase: &container.ContainerJSONBase{
				State: &container.State{Health: &container.Health{Status: container.Unhealthy}},
			},
			expectedHealthy: false,
			expectedMsg:     fmt.Sprintf("Container %s is unhealthy", rep.ContainerID),
		},
		{
			name: "Starting container",
			inspectRespBase: &container.ContainerJSONBase{
				State: &container.State{Health: &container.Health{Status: container.Starting}},
			},
			expectedHealthy: false,
			expectedMsg:     fmt.Sprintf("Container %s is starting", rep.ContainerID),
		},
		{
			name: "Container without health check",
			inspectRespBase: &container.ContainerJSONBase{
				State: &container.State{Health: nil}, // No health check configured
			},
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsg:     fmt.Sprintf("Container %s does not have a health check configured", rep.ContainerID),
		},
		{
			name:            "Container inspect error",
			inspectRespBase: nil,
			inspectErr:      fmt.Errorf("docker inspect error"),
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsg:     fmt.Sprintf("Failed to inspect container %s: docker inspect error", rep.ContainerID),
		},
		{
			name:            "Empty container ID",
			containerID:     "",  // Explicitly set empty container ID for this test
			inspectRespBase: nil, // Inspect won't be called
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsg:     "Container ID is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			mockClient := &MockDockerClient{
				InspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
					// Construct the full InspectResponse using the base
					resp := container.InspectResponse{}
					if tt.inspectRespBase != nil {
						resp.ContainerJSONBase = tt.inspectRespBase
					}
					return resp, tt.inspectErr
				},
			}

			// Create checker with mock client
			config := createValidDockerConfig()
			checker, err := NewDockerHealthChecker(config)
			require.NoError(t, err)
			checker.dockerClient = mockClient // Inject mock client

			// Determine the replica to use for the test
			currentRep := rep
			if tt.containerID != "" || tt.name == "Empty container ID" { // Use specific ID if provided or for the empty ID test
				currentRep.ContainerID = tt.containerID
			}

			// Perform the check
			result, err := checker.CheckWithDetails(currentRep)

			// Assert results
			assert.Equal(t, tt.expectedHealthy, result.Healthy, "Health status mismatch")
			assert.Contains(t, result.Message, tt.expectedMsg, "Message mismatch") // Use Contains for flexibility

			if tt.expectedErr {
				assert.Error(t, err, "Expected an error, but got nil")
			} else {
				assert.NoError(t, err, "Expected no error, but got: %v", err)
			}

			// Check status tracking (ensure UpdateStatus was called)
			healthyStatus, msg, _ := checker.GetStatus()
			assert.Equal(t, tt.expectedHealthy, healthyStatus, "Stored health status mismatch")
			assert.Contains(t, msg, tt.expectedMsg, "Stored message mismatch")
		})
	}
}

// TestDockerHealthChecker_ShouldCheck tests the rate limiting behaviour
func TestDockerHealthChecker_ShouldCheck(t *testing.T) {
	mockClient := &MockDockerClient{
		InspectFunc: func(ctx context.Context, containerID string) (container.InspectResponse, error) {
			// Correctly construct the response using ContainerJSONBase
			return container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{Health: &container.Health{Status: container.Healthy}},
				},
			}, nil
		},
	}

	config := createValidDockerConfig()
	// Use a valid retry interval (>= MinRetryInterval which is 100ms)
	config.RetryInterval = 100 * time.Millisecond
	checker, err := NewDockerHealthChecker(config)
	require.NoError(t, err, "Failed to create checker: %v", err) // Add error message for clarity
	checker.dockerClient = mockClient

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
