/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dosync/internal/replica"
)

// Helper function to create a minimal valid config for TCP health checks
func createValidTCPConfig(port int) HealthCheckConfig {
	return HealthCheckConfig{
		Type:    TCPHealthCheck,
		Port:    port,
		Timeout: 1 * time.Second, // Short timeout for tests
	}
}

// TestNewTCPHealthChecker tests the creation of a new TCPHealthChecker
func TestNewTCPHealthChecker(t *testing.T) {
	config := createValidTCPConfig(8080)

	checker, err := NewTCPHealthChecker(config)
	require.NoError(t, err, "NewTCPHealthChecker should not return an error with valid config")
	require.NotNil(t, checker, "NewTCPHealthChecker should return a non-nil checker")
	assert.Equal(t, TCPHealthCheck, checker.GetType(), "Checker type should be TCPHealthCheck")

	// Test with invalid config type
	invalidConfig := createValidTCPConfig(8080)
	invalidConfig.Type = HTTPHealthCheck
	_, err = NewTCPHealthChecker(invalidConfig)
	require.Error(t, err, "NewTCPHealthChecker should return an error with invalid config type")

	// Test with invalid port (0)
	invalidPortConfig := createValidTCPConfig(0)
	_, err = NewTCPHealthChecker(invalidPortConfig)
	require.Error(t, err, "NewTCPHealthChecker should return an error with invalid port")
	assert.Contains(t, err.Error(), "TCP health check requires a valid port")
}

// TestTCPHealthChecker_Check tests the Check and CheckWithDetails methods
func TestTCPHealthChecker_Check(t *testing.T) {
	rep := replica.Replica{ContainerID: "tcp-test-container", ServiceName: "test-service"}

	// Start a test server to listen on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to start test TCP listener")
	defer listener.Close()

	// Extract the port that was assigned
	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err, "Failed to extract port from listener address")
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "Failed to convert port string to integer")

	// Start accepting connections in a separate goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Listener closed, exit the goroutine
				return
			}
			// Just close the connection immediately for this test
			conn.Close()
		}
	}()

	tests := []struct {
		name            string
		configPort      int
		expectedHealthy bool
		expectedErr     bool
		expectedMsgPart string
	}{
		{
			name:            "Healthy endpoint (port is open)",
			configPort:      port, // Use the port from our test listener
			expectedHealthy: true,
			expectedErr:     false,
			expectedMsgPart: "TCP connection successful",
		},
		{
			name:            "Unhealthy endpoint (connection refused)",
			configPort:      port + 1, // Use a port where nothing is listening
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsgPart: "TCP connection failed",
		},
		{
			name:            "Invalid port (0)",
			configPort:      0, // Invalid port
			expectedHealthy: false,
			expectedErr:     true,
			expectedMsgPart: "Invalid port configured for TCP health check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createValidTCPConfig(tt.configPort)
			checker, err := NewTCPHealthChecker(config)
			if tt.configPort == 0 {
				// For the invalid port test, we expect checker creation to fail
				// So we'll skip the rest of the test
				assert.Error(t, err)
				return
			}
			require.NoError(t, err, "Failed to create TCP health checker")

			// Perform the check using CheckWithDetails first
			result, checkDetailsErr := checker.CheckWithDetails(rep)

			// Assert detailed results
			assert.Equal(t, tt.expectedHealthy, result.Healthy, "Detailed check: Health status mismatch")
			assert.Contains(t, result.Message, tt.expectedMsgPart, "Detailed check: Message mismatch")

			if tt.expectedErr {
				assert.Error(t, checkDetailsErr, "Detailed check: Expected an error, but got nil")
			} else {
				assert.NoError(t, checkDetailsErr, "Detailed check: Expected no error, but got: %v", checkDetailsErr)
			}

			// Perform the check using Check
			healthy, checkErr := checker.Check(rep)

			// Assert simple check results
			assert.Equal(t, tt.expectedHealthy, healthy, "Simple check: Health status mismatch")
			if tt.expectedErr {
				assert.Error(t, checkDetailsErr, "CheckWithDetails should have returned an error for this case")
			} else {
				assert.NoError(t, checkErr, "Simple check: Expected no error, but got: %v", checkErr)
			}

			// Check status tracking (ensure UpdateStatus was called)
			finalHealthyStatus, finalMsg, _ := checker.GetStatus()
			assert.Equal(t, tt.expectedHealthy, finalHealthyStatus, "Stored health status mismatch")
			assert.Contains(t, finalMsg, tt.expectedMsgPart, "Stored message mismatch")
		})
	}
}

// TestTCPHealthChecker_ShouldCheck tests the rate limiting behavior
func TestTCPHealthChecker_ShouldCheck(t *testing.T) {
	// Create a test listener on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to start test TCP listener")
	defer listener.Close()

	// Extract the port that was assigned
	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err, "Failed to extract port from listener address")
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "Failed to convert port string to integer")

	// Start accepting connections in a separate goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Listener closed, exit the goroutine
				return
			}
			conn.Close()
		}
	}()

	// Create the TCP health checker
	config := createValidTCPConfig(port)
	// Use a slightly higher retry interval for reliability in tests
	config.RetryInterval = 150 * time.Millisecond
	checker, err := NewTCPHealthChecker(config)
	require.NoError(t, err, "Failed to create TCP health checker")

	rep := replica.Replica{ContainerID: "tcp-test-container"}

	// First check should always proceed
	assert.True(t, checker.ShouldCheck(), "First check should proceed")
	_, err = checker.Check(rep)
	require.NoError(t, err)

	// Second check immediately after should be skipped
	assert.False(t, checker.ShouldCheck(), "Second check immediately after should be skipped")
	_, err = checker.Check(rep)
	require.NoError(t, err) // Check shouldn't return error, just the cached status

	// Wait for longer than the retry interval
	time.Sleep(config.RetryInterval + 20*time.Millisecond)

	// Third check after interval should proceed
	assert.True(t, checker.ShouldCheck(), "Third check after interval should proceed")
	_, err = checker.Check(rep)
	require.NoError(t, err)
}
