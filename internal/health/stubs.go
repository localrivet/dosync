/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"time"

	"dosync/internal/replica"
)

// StubHealthChecker provides a basic implementation of the HealthChecker interface for testing purposes
type StubHealthChecker struct {
	CheckerType   HealthCheckType
	IsHealthy     bool
	ErrorToReturn error
	Config        HealthCheckConfig
}

// NewStubHealthChecker creates a new stub health checker with default values
func NewStubHealthChecker(checkerType HealthCheckType, healthy bool) *StubHealthChecker {
	return &StubHealthChecker{
		CheckerType:   checkerType,
		IsHealthy:     healthy,
		ErrorToReturn: nil,
		Config: HealthCheckConfig{
			Type:             checkerType,
			Timeout:          time.Second * 5,
			RetryInterval:    time.Second,
			SuccessThreshold: 2,
			FailureThreshold: 3,
		},
	}
}

// Check always returns the preconfigured health status
func (s *StubHealthChecker) Check(replica replica.Replica) (bool, error) {
	return s.IsHealthy, s.ErrorToReturn
}

// CheckWithDetails returns a detailed health check result
func (s *StubHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	message := "Service is healthy"
	if !s.IsHealthy {
		message = "Service is unhealthy"
	}

	result := HealthCheckResult{
		Healthy:   s.IsHealthy,
		Message:   message,
		Timestamp: time.Now(),
	}

	return result, s.ErrorToReturn
}

// Configure updates the stub's configuration
func (s *StubHealthChecker) Configure(config HealthCheckConfig) error {
	s.Config = config
	return nil
}

// GetType returns the health checker type
func (s *StubHealthChecker) GetType() HealthCheckType {
	return s.CheckerType
}

// StubDockerHealthChecker provides a specific stub for Docker health checks
func NewStubDockerHealthChecker(healthy bool) *StubHealthChecker {
	return NewStubHealthChecker(DockerHealthCheck, healthy)
}

// StubHTTPHealthChecker provides a specific stub for HTTP health checks
func NewStubHTTPHealthChecker(healthy bool) *StubHealthChecker {
	return NewStubHealthChecker(HTTPHealthCheck, healthy)
}

// StubTCPHealthChecker provides a specific stub for TCP health checks
func NewStubTCPHealthChecker(healthy bool) *StubHealthChecker {
	return NewStubHealthChecker(TCPHealthCheck, healthy)
}

// StubCommandHealthChecker provides a specific stub for command-based health checks
func NewStubCommandHealthChecker(healthy bool) *StubHealthChecker {
	return NewStubHealthChecker(CommandHealthCheck, healthy)
}
