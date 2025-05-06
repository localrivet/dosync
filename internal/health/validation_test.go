/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"testing"
	"time"
)

// TestValidateConfig_ValidConfigs tests that valid configurations pass validation
func TestValidateConfig_ValidConfigs(t *testing.T) {
	testCases := []struct {
		name   string
		config HealthCheckConfig
	}{
		{
			name: "Docker health check - minimal",
			config: HealthCheckConfig{
				Type: DockerHealthCheck,
			},
		},
		{
			name: "HTTP health check - complete",
			config: HealthCheckConfig{
				Type:             HTTPHealthCheck,
				Endpoint:         "/health",
				Timeout:          10 * time.Second,
				RetryInterval:    2 * time.Second,
				SuccessThreshold: 2,
				FailureThreshold: 3,
			},
		},
		{
			name: "HTTP health check - endpoint without slash",
			config: HealthCheckConfig{
				Type:     HTTPHealthCheck,
				Endpoint: "health",
			},
		},
		{
			name: "TCP health check",
			config: HealthCheckConfig{
				Type: TCPHealthCheck,
				Port: 8080,
			},
		},
		{
			name: "Command health check",
			config: HealthCheckConfig{
				Type:    CommandHealthCheck,
				Command: "curl localhost:8080/health",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			err := ValidateConfig(&config)
			if err != nil {
				t.Errorf("Expected config to be valid, got error: %v", err)
			}

			// Check that defaults are applied appropriately
			if config.Timeout <= 0 {
				t.Error("Expected default timeout to be applied")
			}
			if config.RetryInterval <= 0 {
				t.Error("Expected default retry interval to be applied")
			}
			if config.SuccessThreshold <= 0 {
				t.Error("Expected default success threshold to be applied")
			}
			if config.FailureThreshold <= 0 {
				t.Error("Expected default failure threshold to be applied")
			}

			// Check that endpoints are normalized
			if config.Type == HTTPHealthCheck && config.Endpoint[0] != '/' {
				t.Error("Expected HTTP endpoint to start with /")
			}
		})
	}
}

// TestValidateConfig_InvalidConfigs tests that invalid configurations fail validation
func TestValidateConfig_InvalidConfigs(t *testing.T) {
	testCases := []struct {
		name   string
		config HealthCheckConfig
	}{
		{
			name: "Invalid health check type",
			config: HealthCheckConfig{
				Type: "invalid",
			},
		},
		{
			name: "HTTP health check without endpoint",
			config: HealthCheckConfig{
				Type: HTTPHealthCheck,
				// Endpoint is missing
			},
		},
		{
			name: "TCP health check with invalid port (too low)",
			config: HealthCheckConfig{
				Type: TCPHealthCheck,
				Port: 0,
			},
		},
		{
			name: "TCP health check with invalid port (too high)",
			config: HealthCheckConfig{
				Type: TCPHealthCheck,
				Port: 70000,
			},
		},
		{
			name: "Command health check without command",
			config: HealthCheckConfig{
				Type: CommandHealthCheck,
				// Command is missing
			},
		},
		{
			name: "Timeout too low",
			config: HealthCheckConfig{
				Type:    DockerHealthCheck,
				Timeout: 10 * time.Millisecond,
			},
		},
		{
			name: "Timeout too high",
			config: HealthCheckConfig{
				Type:    DockerHealthCheck,
				Timeout: 10 * time.Minute,
			},
		},
		{
			name: "Retry interval too low",
			config: HealthCheckConfig{
				Type:          DockerHealthCheck,
				RetryInterval: 10 * time.Millisecond,
			},
		},
		{
			name: "Success threshold too high",
			config: HealthCheckConfig{
				Type:             DockerHealthCheck,
				SuccessThreshold: 20,
			},
		},
		{
			name: "Failure threshold too high",
			config: HealthCheckConfig{
				Type:             DockerHealthCheck,
				FailureThreshold: 20,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			err := ValidateConfig(&config)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestValidateConfig_DefaultValues tests that default values are applied correctly
func TestValidateConfig_DefaultValues(t *testing.T) {
	// Create a minimal valid config
	config := HealthCheckConfig{
		Type: DockerHealthCheck,
	}

	err := ValidateConfig(&config)
	if err != nil {
		t.Fatalf("Expected config to be valid, got error: %v", err)
	}

	// Check default values
	if config.Timeout != DefaultTimeout {
		t.Errorf("Expected default timeout %v, got %v", DefaultTimeout, config.Timeout)
	}
	if config.RetryInterval != DefaultRetryInterval {
		t.Errorf("Expected default retry interval %v, got %v", DefaultRetryInterval, config.RetryInterval)
	}
	if config.SuccessThreshold != DefaultSuccessThreshold {
		t.Errorf("Expected default success threshold %d, got %d", DefaultSuccessThreshold, config.SuccessThreshold)
	}
	if config.FailureThreshold != DefaultFailureThreshold {
		t.Errorf("Expected default failure threshold %d, got %d", DefaultFailureThreshold, config.FailureThreshold)
	}
}
