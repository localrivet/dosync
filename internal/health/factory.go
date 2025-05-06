/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"fmt"
)

// NewHealthChecker creates a new health checker based on the configuration type.
// This factory function returns the appropriate concrete implementation of the
// HealthChecker interface based on the type specified in the configuration.
func NewHealthChecker(config HealthCheckConfig) (HealthChecker, error) {
	// Validate the config type
	if config.Type == "" {
		return nil, fmt.Errorf("health check type must be specified")
	}

	// Create the appropriate health checker based on the type
	switch config.Type {
	case DockerHealthCheck:
		return NewDockerHealthChecker(config)
	case HTTPHealthCheck:
		return NewHTTPHealthChecker(config)
	case TCPHealthCheck:
		return NewTCPHealthChecker(config)
	case CommandHealthCheck:
		return NewCommandHealthChecker(config)
	default:
		return nil, fmt.Errorf("unsupported health check type: %s", config.Type)
	}
}
