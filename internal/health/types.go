/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"time"

	"dosync/internal/replica"
)

// HealthCheckType defines the type of health check to perform
type HealthCheckType string

const (
	// DockerHealthCheck uses Docker's built-in health check mechanism
	DockerHealthCheck HealthCheckType = "docker"

	// HTTPHealthCheck makes HTTP requests to specified endpoints
	HTTPHealthCheck HealthCheckType = "http"

	// TCPHealthCheck attempts to establish TCP connections
	TCPHealthCheck HealthCheckType = "tcp"

	// CommandHealthCheck executes commands inside containers
	CommandHealthCheck HealthCheckType = "command"
)

// HealthCheckResult represents the outcome of a health check
type HealthCheckResult struct {
	// Healthy indicates whether the check passed
	Healthy bool

	// Message provides additional details about the health check result
	Message string

	// Timestamp records when the check was performed
	Timestamp time.Time
}

// HealthCheckConfig defines the configuration for health checks
type HealthCheckConfig struct {
	// Type defines which health checker to use
	Type HealthCheckType

	// Endpoint is the URL path for HTTP checks
	Endpoint string

	// Port is the port number for TCP checks
	Port int

	// Command is the command to execute for custom checks
	Command string

	// Timeout is the maximum duration to wait for a health check to complete
	Timeout time.Duration

	// RetryInterval is the time to wait between retries
	RetryInterval time.Duration

	// SuccessThreshold is the number of consecutive successful checks required
	SuccessThreshold int

	// FailureThreshold is the number of consecutive failed checks required
	FailureThreshold int
}

// HealthChecker defines the interface for all health check implementations
type HealthChecker interface {
	// Check performs a health check on the specified replica
	// Returns true if healthy, false otherwise, along with any error encountered
	Check(replica replica.Replica) (bool, error)

	// CheckWithDetails performs a health check and returns detailed result information
	CheckWithDetails(replica replica.Replica) (HealthCheckResult, error)

	// Configure sets up the health checker with the provided configuration
	Configure(config HealthCheckConfig) error

	// GetType returns the type of this health checker
	GetType() HealthCheckType
}
