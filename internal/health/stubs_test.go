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

// TestStubHealthChecker tests the generic stub health checker
func TestStubHealthChecker(t *testing.T) {
	// Create a stub that reports healthy
	healthyStub := NewStubHealthChecker(HTTPHealthCheck, true)

	// Create a test replica
	testReplica := replica.Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "container123",
		Status:      "running",
	}

	// Test Check method
	healthy, err := healthyStub.Check(testReplica)
	if err != nil {
		t.Errorf("StubHealthChecker.Check returned unexpected error: %v", err)
	}
	if !healthy {
		t.Error("StubHealthChecker.Check should return true for IsHealthy=true")
	}

	// Test CheckWithDetails method
	result, err := healthyStub.CheckWithDetails(testReplica)
	if err != nil {
		t.Errorf("StubHealthChecker.CheckWithDetails returned unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Error("StubHealthChecker.CheckWithDetails should return Healthy=true for IsHealthy=true")
	}
	if result.Message != "Service is healthy" {
		t.Errorf("StubHealthChecker.CheckWithDetails returned unexpected message: %q", result.Message)
	}

	// Now test an unhealthy stub
	unhealthyStub := NewStubHealthChecker(TCPHealthCheck, false)

	// Test Check method for unhealthy
	healthy, err = unhealthyStub.Check(testReplica)
	if err != nil {
		t.Errorf("StubHealthChecker.Check returned unexpected error: %v", err)
	}
	if healthy {
		t.Error("StubHealthChecker.Check should return false for IsHealthy=false")
	}

	// Test CheckWithDetails method for unhealthy
	result, err = unhealthyStub.CheckWithDetails(testReplica)
	if err != nil {
		t.Errorf("StubHealthChecker.CheckWithDetails returned unexpected error: %v", err)
	}
	if result.Healthy {
		t.Error("StubHealthChecker.CheckWithDetails should return Healthy=false for IsHealthy=false")
	}
	if result.Message != "Service is unhealthy" {
		t.Errorf("StubHealthChecker.CheckWithDetails returned unexpected message: %q", result.Message)
	}
}

// TestStubHealthCheckerWithError tests the stub health checker with error configuration
func TestStubHealthCheckerWithError(t *testing.T) {
	// Create a stub that returns an error
	testError := errors.New("test health check error")
	stub := NewStubHealthChecker(DockerHealthCheck, true)
	stub.ErrorToReturn = testError

	// Create a test replica
	testReplica := replica.Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "container123",
		Status:      "running",
	}

	// Test Check method
	_, err := stub.Check(testReplica)
	if err != testError {
		t.Errorf("StubHealthChecker.Check should return the configured error")
	}

	// Test CheckWithDetails method
	_, err = stub.CheckWithDetails(testReplica)
	if err != testError {
		t.Errorf("StubHealthChecker.CheckWithDetails should return the configured error")
	}
}

// TestStubHealthCheckerConfigure tests the stub health checker's Configure method
func TestStubHealthCheckerConfigure(t *testing.T) {
	stub := NewStubHealthChecker(HTTPHealthCheck, true)

	// Create a new configuration
	newConfig := HealthCheckConfig{
		Type:             HTTPHealthCheck,
		Endpoint:         "/status",
		Port:             9090,
		Timeout:          time.Second * 30,
		RetryInterval:    time.Second * 5,
		SuccessThreshold: 5,
		FailureThreshold: 2,
	}

	// Configure the stub
	err := stub.Configure(newConfig)
	if err != nil {
		t.Errorf("StubHealthChecker.Configure returned unexpected error: %v", err)
	}

	// Verify the configuration was updated
	if stub.Config.Endpoint != "/status" {
		t.Errorf("StubHealthChecker.Configure did not update Endpoint, got %q", stub.Config.Endpoint)
	}
	if stub.Config.Port != 9090 {
		t.Errorf("StubHealthChecker.Configure did not update Port, got %d", stub.Config.Port)
	}
	if stub.Config.Timeout != time.Second*30 {
		t.Errorf("StubHealthChecker.Configure did not update Timeout, got %v", stub.Config.Timeout)
	}
	if stub.Config.RetryInterval != time.Second*5 {
		t.Errorf("StubHealthChecker.Configure did not update RetryInterval, got %v", stub.Config.RetryInterval)
	}
	if stub.Config.SuccessThreshold != 5 {
		t.Errorf("StubHealthChecker.Configure did not update SuccessThreshold, got %d", stub.Config.SuccessThreshold)
	}
	if stub.Config.FailureThreshold != 2 {
		t.Errorf("StubHealthChecker.Configure did not update FailureThreshold, got %d", stub.Config.FailureThreshold)
	}
}

// TestSpecificHealthCheckerStubs tests the specific health checker stub constructors
func TestSpecificHealthCheckerStubs(t *testing.T) {
	// Test all the specific health checker stub constructors
	testCases := []struct {
		name         string
		constructor  func(bool) *StubHealthChecker
		expectedType HealthCheckType
	}{
		{"Docker", NewStubDockerHealthChecker, DockerHealthCheck},
		{"HTTP", NewStubHTTPHealthChecker, HTTPHealthCheck},
		{"TCP", NewStubTCPHealthChecker, TCPHealthCheck},
		{"Command", NewStubCommandHealthChecker, CommandHealthCheck},
	}

	for _, tc := range testCases {
		t.Run(string(tc.expectedType), func(t *testing.T) {
			// Test with healthy=true
			healthyStub := tc.constructor(true)
			if healthyStub.GetType() != tc.expectedType {
				t.Errorf("%s stub has wrong type: expected %v, got %v", tc.name, tc.expectedType, healthyStub.GetType())
			}
			if !healthyStub.IsHealthy {
				t.Errorf("%s stub with healthy=true should have IsHealthy=true", tc.name)
			}

			// Test with healthy=false
			unhealthyStub := tc.constructor(false)
			if unhealthyStub.GetType() != tc.expectedType {
				t.Errorf("%s stub has wrong type: expected %v, got %v", tc.name, tc.expectedType, unhealthyStub.GetType())
			}
			if unhealthyStub.IsHealthy {
				t.Errorf("%s stub with healthy=false should have IsHealthy=false", tc.name)
			}
		})
	}
}
