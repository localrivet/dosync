/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"fmt"
	"time"
)

// Default configuration values
const (
	DefaultTimeout          = 5 * time.Second
	DefaultRetryInterval    = 1 * time.Second
	DefaultSuccessThreshold = 1
	DefaultFailureThreshold = 3

	MinTimeout          = 1 * time.Second
	MinRetryInterval    = 100 * time.Millisecond
	MinSuccessThreshold = 1
	MinFailureThreshold = 1

	MaxTimeout          = 5 * time.Minute
	MaxSuccessThreshold = 10
	MaxFailureThreshold = 10
)

// ValidateConfig checks that a HealthCheckConfig is valid for the specified checker type
// and applies default values where appropriate
func ValidateConfig(config *HealthCheckConfig) error {
	// Check that the type is valid
	switch config.Type {
	case DockerHealthCheck, HTTPHealthCheck, TCPHealthCheck, CommandHealthCheck:
		// Valid type
	default:
		return fmt.Errorf("invalid health check type: %s", config.Type)
	}

	// Apply default values if not set
	if config.Timeout <= 0 {
		config.Timeout = DefaultTimeout
	} else if config.Timeout < MinTimeout {
		return fmt.Errorf("timeout %v is less than minimum %v", config.Timeout, MinTimeout)
	} else if config.Timeout > MaxTimeout {
		return fmt.Errorf("timeout %v exceeds maximum %v", config.Timeout, MaxTimeout)
	}

	if config.RetryInterval <= 0 {
		config.RetryInterval = DefaultRetryInterval
	} else if config.RetryInterval < MinRetryInterval {
		return fmt.Errorf("retry interval %v is less than minimum %v", config.RetryInterval, MinRetryInterval)
	}

	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = DefaultSuccessThreshold
	} else if config.SuccessThreshold < MinSuccessThreshold {
		return fmt.Errorf("success threshold %d is less than minimum %d", config.SuccessThreshold, MinSuccessThreshold)
	} else if config.SuccessThreshold > MaxSuccessThreshold {
		return fmt.Errorf("success threshold %d exceeds maximum %d", config.SuccessThreshold, MaxSuccessThreshold)
	}

	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultFailureThreshold
	} else if config.FailureThreshold < MinFailureThreshold {
		return fmt.Errorf("failure threshold %d is less than minimum %d", config.FailureThreshold, MinFailureThreshold)
	} else if config.FailureThreshold > MaxFailureThreshold {
		return fmt.Errorf("failure threshold %d exceeds maximum %d", config.FailureThreshold, MaxFailureThreshold)
	}

	// Type-specific validations
	switch config.Type {
	case HTTPHealthCheck:
		if config.Endpoint == "" {
			return fmt.Errorf("HTTP health check requires an endpoint")
		}
		// Ensure endpoint starts with /
		if config.Endpoint[0] != '/' {
			config.Endpoint = "/" + config.Endpoint
		}
	case TCPHealthCheck:
		return validateTCPConfig(config)
	case CommandHealthCheck:
		if config.Command == "" {
			return fmt.Errorf("command health check requires a command")
		}
	}

	return nil
}

// validateTCPConfig validates the configuration for a TCP health checker.
func validateTCPConfig(config *HealthCheckConfig) error {
	if config.Port <= 0 {
		return fmt.Errorf("TCP health check requires a valid port (> 0)")
	}
	// Add upper bound check for port number
	if config.Port > 65535 {
		return fmt.Errorf("TCP health check port must be <= 65535, got %d", config.Port)
	}
	return nil
}
