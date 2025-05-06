/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"testing"
	"time"

	"dosync/internal/replica"
)

// TestHealthCheckTypes ensures that the health check type constants are defined correctly
func TestHealthCheckTypes(t *testing.T) {
	testCases := []struct {
		typeName HealthCheckType
		expected string
	}{
		{DockerHealthCheck, "docker"},
		{HTTPHealthCheck, "http"},
		{TCPHealthCheck, "tcp"},
		{CommandHealthCheck, "command"},
	}

	for _, tc := range testCases {
		if string(tc.typeName) != tc.expected {
			t.Errorf("Health check type %v should be %q but got %q", tc.typeName, tc.expected, string(tc.typeName))
		}
	}
}

// TestHealthCheckResult tests the HealthCheckResult struct
func TestHealthCheckResult(t *testing.T) {
	// Create a sample result
	now := time.Now()
	result := HealthCheckResult{
		Healthy:   true,
		Message:   "Service is healthy",
		Timestamp: now,
	}

	// Verify the fields
	if !result.Healthy {
		t.Error("HealthCheckResult Healthy field should be true")
	}
	if result.Message != "Service is healthy" {
		t.Errorf("HealthCheckResult Message field should be 'Service is healthy', got %q", result.Message)
	}
	if !result.Timestamp.Equal(now) {
		t.Errorf("HealthCheckResult Timestamp field should equal the time it was created")
	}
}

// TestHealthCheckConfig tests the HealthCheckConfig struct
func TestHealthCheckConfig(t *testing.T) {
	// Create a sample configuration
	config := HealthCheckConfig{
		Type:             HTTPHealthCheck,
		Endpoint:         "/health",
		Port:             8080,
		Command:          "curl localhost:8080/health",
		Timeout:          time.Second * 10,
		RetryInterval:    time.Second * 2,
		SuccessThreshold: 2,
		FailureThreshold: 3,
	}

	// Verify the fields
	if config.Type != HTTPHealthCheck {
		t.Errorf("HealthCheckConfig Type field should be HTTPHealthCheck, got %v", config.Type)
	}
	if config.Endpoint != "/health" {
		t.Errorf("HealthCheckConfig Endpoint field should be '/health', got %q", config.Endpoint)
	}
	if config.Port != 8080 {
		t.Errorf("HealthCheckConfig Port field should be 8080, got %d", config.Port)
	}
	if config.Command != "curl localhost:8080/health" {
		t.Errorf("HealthCheckConfig Command field should be 'curl localhost:8080/health', got %q", config.Command)
	}
	if config.Timeout != time.Second*10 {
		t.Errorf("HealthCheckConfig Timeout field should be 10 seconds, got %v", config.Timeout)
	}
	if config.RetryInterval != time.Second*2 {
		t.Errorf("HealthCheckConfig RetryInterval field should be 2 seconds, got %v", config.RetryInterval)
	}
	if config.SuccessThreshold != 2 {
		t.Errorf("HealthCheckConfig SuccessThreshold field should be 2, got %d", config.SuccessThreshold)
	}
	if config.FailureThreshold != 3 {
		t.Errorf("HealthCheckConfig FailureThreshold field should be 3, got %d", config.FailureThreshold)
	}
}

// MockHealthChecker implements the HealthChecker interface for testing
type MockHealthChecker struct {
	IsHealthy       bool
	ConfigureResult error
}

func (m *MockHealthChecker) Check(replica replica.Replica) (bool, error) {
	return m.IsHealthy, nil
}

func (m *MockHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	result := HealthCheckResult{
		Healthy:   m.IsHealthy,
		Message:   "Mock health check",
		Timestamp: time.Now(),
	}
	return result, nil
}

func (m *MockHealthChecker) Configure(config HealthCheckConfig) error {
	return m.ConfigureResult
}

func (m *MockHealthChecker) GetType() HealthCheckType {
	return DockerHealthCheck
}

// TestHealthCheckerInterface ensures that the HealthChecker interface can be implemented
func TestHealthCheckerInterface(t *testing.T) {
	// Create a mock that implements HealthChecker interface
	mockChecker := &MockHealthChecker{
		IsHealthy:       true,
		ConfigureResult: nil,
	}

	// Verify that mockChecker implements HealthChecker
	var _ HealthChecker = mockChecker

	// Test Check method
	testReplica := replica.Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "container123",
		Status:      "running",
	}

	healthy, err := mockChecker.Check(testReplica)
	if err != nil {
		t.Errorf("MockHealthChecker.Check returned unexpected error: %v", err)
	}
	if !healthy {
		t.Error("MockHealthChecker.Check should return true for IsHealthy=true")
	}

	// Test CheckWithDetails method
	result, err := mockChecker.CheckWithDetails(testReplica)
	if err != nil {
		t.Errorf("MockHealthChecker.CheckWithDetails returned unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Error("MockHealthChecker.CheckWithDetails should return Healthy=true for IsHealthy=true")
	}
	if result.Message != "Mock health check" {
		t.Errorf("MockHealthChecker.CheckWithDetails returned unexpected message: %q", result.Message)
	}

	// Test Configure method
	config := HealthCheckConfig{
		Type:             HTTPHealthCheck,
		Timeout:          time.Second * 5,
		SuccessThreshold: 1,
	}
	err = mockChecker.Configure(config)
	if err != nil {
		t.Errorf("MockHealthChecker.Configure returned unexpected error: %v", err)
	}

	// Test GetType method
	if mockChecker.GetType() != DockerHealthCheck {
		t.Errorf("MockHealthChecker.GetType returned unexpected type: %v", mockChecker.GetType())
	}
}
