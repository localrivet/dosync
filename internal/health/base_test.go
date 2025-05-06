/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"errors"
	"testing"
	"time"

	"dosync/internal/replica"
)

// TestBaseCheckerImplementsInterface ensures BaseChecker implements required methods
func TestBaseCheckerImplementsInterface(t *testing.T) {
	// This test will fail at compile time if BaseChecker doesn't implement HealthChecker
	var _ HealthChecker = &testChecker{}
}

// testChecker embeds BaseChecker and implements the Check method
type testChecker struct {
	BaseChecker
	shouldBeHealthy bool
	checkError      error
}

// newTestChecker creates a new test checker
func newTestChecker(checkType HealthCheckType, healthy bool) (*testChecker, error) {
	config := HealthCheckConfig{
		Type:             checkType,
		Timeout:          time.Second * 5,
		RetryInterval:    time.Second,
		SuccessThreshold: 2,
		FailureThreshold: 3,
	}

	// Add required config fields based on the type
	switch checkType {
	case HTTPHealthCheck:
		config.Endpoint = "/health"
	case TCPHealthCheck:
		config.Port = 8080
	case CommandHealthCheck:
		config.Command = "test command"
	}

	base, err := NewBaseChecker(checkType, config)
	if err != nil {
		return nil, err
	}

	return &testChecker{
		BaseChecker:     *base,
		shouldBeHealthy: healthy,
		checkError:      nil,
	}, nil
}

// Check implements the Check method for the testChecker
func (tc *testChecker) Check(r replica.Replica) (bool, error) {
	if tc.checkError != nil {
		return false, tc.checkError
	}

	// Update the status based on the shouldBeHealthy flag
	message := "Service is healthy"
	if !tc.shouldBeHealthy {
		message = "Service is unhealthy"
	}

	tc.UpdateStatus(tc.shouldBeHealthy, message)
	return tc.shouldBeHealthy, nil
}

// CheckWithDetails overrides the BaseChecker's CheckWithDetails method
func (tc *testChecker) CheckWithDetails(r replica.Replica) (HealthCheckResult, error) {
	if tc.checkError != nil {
		return HealthCheckResult{
			Healthy:   false,
			Message:   tc.checkError.Error(),
			Timestamp: time.Now(),
		}, tc.checkError
	}

	// Call Check to update the status
	tc.Check(r)

	// Return the result
	return tc.CreateHealthCheckResult(), nil
}

// TestBaseCheckerConfiguration tests the configuration of BaseChecker
func TestBaseCheckerConfiguration(t *testing.T) {
	// Create a valid configuration
	config := HealthCheckConfig{
		Type:             HTTPHealthCheck,
		Endpoint:         "/health",
		Timeout:          time.Second * 10,
		RetryInterval:    time.Second * 2,
		SuccessThreshold: 2,
		FailureThreshold: 3,
	}

	// Create a base checker
	checker, err := NewBaseChecker(HTTPHealthCheck, config)
	if err != nil {
		t.Fatalf("Failed to create BaseChecker: %v", err)
	}

	// Verify configuration
	if checker.Config.Type != HTTPHealthCheck {
		t.Errorf("BaseChecker has wrong type: expected %v, got %v", HTTPHealthCheck, checker.Config.Type)
	}
	if checker.Config.Endpoint != "/health" {
		t.Errorf("BaseChecker has wrong endpoint: expected /health, got %s", checker.Config.Endpoint)
	}
	if checker.Config.Timeout != time.Second*10 {
		t.Errorf("BaseChecker has wrong timeout: expected 10s, got %v", checker.Config.Timeout)
	}

	// Test GetType
	if checker.GetType() != HTTPHealthCheck {
		t.Errorf("GetType returned wrong type: expected %v, got %v", HTTPHealthCheck, checker.GetType())
	}

	// Test reconfiguration
	newConfig := HealthCheckConfig{
		Type:             TCPHealthCheck,
		Port:             8080,
		Timeout:          time.Second * 5,
		RetryInterval:    time.Second,
		SuccessThreshold: 1,
		FailureThreshold: 2,
	}

	err = checker.Configure(newConfig)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	// Verify new configuration
	if checker.Config.Type != TCPHealthCheck {
		t.Errorf("Reconfigured BaseChecker has wrong type: expected %v, got %v", TCPHealthCheck, checker.Config.Type)
	}
	if checker.Config.Port != 8080 {
		t.Errorf("Reconfigured BaseChecker has wrong port: expected 8080, got %d", checker.Config.Port)
	}
}

// TestBaseCheckerStatusTracking tests the threshold logic for health status tracking
func TestBaseCheckerStatusTracking(t *testing.T) {
	checker, err := newTestChecker(HTTPHealthCheck, true)
	if err != nil {
		t.Fatalf("Failed to create test checker: %v", err)
	}

	// Set thresholds for testing
	checker.Config.SuccessThreshold = 2
	checker.Config.FailureThreshold = 2

	// Create a test replica
	testReplica := replica.Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "container123",
		Status:      "running",
	}

	// Initially should not be healthy
	healthy, message, _ := checker.GetStatus()
	if healthy {
		t.Error("Checker should initially not be healthy")
	}
	if message != "" {
		t.Errorf("Initial message should be empty, got %q", message)
	}

	// First successful check should not make it healthy yet (threshold is 2)
	_, err = checker.Check(testReplica)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	healthy, message, _ = checker.GetStatus()
	if healthy {
		t.Error("Checker should not be healthy after only one successful check")
	}
	if message != "Service is healthy" {
		t.Errorf("Unexpected message: %q", message)
	}

	// Second successful check should make it healthy
	_, err = checker.Check(testReplica)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	healthy, message, _ = checker.GetStatus()
	if !healthy {
		t.Error("Checker should be healthy after two successful checks")
	}

	// Now make it unhealthy and check failure thresholds
	checker.shouldBeHealthy = false

	// First failure should not make it unhealthy yet
	_, err = checker.Check(testReplica)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	healthy, message, _ = checker.GetStatus()
	if !healthy {
		t.Error("Checker should still be healthy after only one failure")
	}

	// Second failure should make it unhealthy
	_, err = checker.Check(testReplica)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	healthy, message, _ = checker.GetStatus()
	if healthy {
		t.Error("Checker should be unhealthy after two failures")
	}
	if message != "Service is unhealthy" {
		t.Errorf("Unexpected message: %q", message)
	}
}

// TestBaseCheckerErrorHandling tests how the base checker handles errors
func TestBaseCheckerErrorHandling(t *testing.T) {
	checker, err := newTestChecker(DockerHealthCheck, true)
	if err != nil {
		t.Fatalf("Failed to create test checker: %v", err)
	}

	// Set an error to be returned from Check
	testError := errors.New("test health check error")
	checker.checkError = testError

	// Create a test replica
	testReplica := replica.Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "container123",
		Status:      "running",
	}

	// Check should return the error
	_, err = checker.Check(testReplica)
	if err != testError {
		t.Errorf("Expected Check to return test error, got: %v", err)
	}

	// CheckWithDetails should also return the error and create an unhealthy result
	result, err := checker.CheckWithDetails(testReplica)
	if err != testError {
		t.Errorf("Expected CheckWithDetails to return test error, got: %v", err)
	}
	if result.Healthy {
		t.Error("Result should be unhealthy when there's an error")
	}
	if result.Message != testError.Error() {
		t.Errorf("Result message should be the error message, got: %q", result.Message)
	}
}

// TestBaseCheckerShouldCheck tests the ShouldCheck method timing logic
func TestBaseCheckerShouldCheck(t *testing.T) {
	// Create a checker with a short retry interval for testing
	config := HealthCheckConfig{
		Type:          DockerHealthCheck,
		RetryInterval: 100 * time.Millisecond,
	}

	checker, err := NewBaseChecker(DockerHealthCheck, config)
	if err != nil {
		t.Fatalf("Failed to create BaseChecker: %v", err)
	}

	// Initially should return true (no check has been performed)
	if !checker.ShouldCheck() {
		t.Error("ShouldCheck should return true initially")
	}

	// Update status to simulate a check
	checker.UpdateStatus(true, "Test")

	// Should return false immediately after a check
	if checker.ShouldCheck() {
		t.Error("ShouldCheck should return false immediately after a check")
	}

	// After waiting, should return true
	time.Sleep(150 * time.Millisecond)
	if !checker.ShouldCheck() {
		t.Error("ShouldCheck should return true after waiting for retry interval")
	}
}
