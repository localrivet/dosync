/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHealthChecker tests the factory function that creates health checkers.
func TestNewHealthChecker(t *testing.T) {
	tests := []struct {
		name          string
		config        HealthCheckConfig
		expectedType  HealthCheckType
		expectedError bool
	}{
		{
			name: "Docker health checker",
			config: HealthCheckConfig{
				Type: DockerHealthCheck,
			},
			expectedType:  DockerHealthCheck,
			expectedError: false,
		},
		{
			name: "HTTP health checker",
			config: HealthCheckConfig{
				Type:     HTTPHealthCheck,
				Endpoint: "/health",
				Port:     8080,
			},
			expectedType:  HTTPHealthCheck,
			expectedError: false,
		},
		{
			name: "TCP health checker",
			config: HealthCheckConfig{
				Type: TCPHealthCheck,
				Port: 8080,
			},
			expectedType:  TCPHealthCheck,
			expectedError: false,
		},
		{
			name: "Command health checker",
			config: HealthCheckConfig{
				Type:    CommandHealthCheck,
				Command: "echo hello",
			},
			expectedType:  CommandHealthCheck,
			expectedError: false,
		},
		{
			name:   "Empty type",
			config: HealthCheckConfig{
				// Type not specified
			},
			expectedError: true,
		},
		{
			name: "Unsupported type",
			config: HealthCheckConfig{
				Type: "unsupported",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, err := NewHealthChecker(tt.config)

			if tt.expectedError {
				assert.Error(t, err, "Expected an error but got nil")
				assert.Nil(t, checker, "Expected nil checker when error occurs")
				return
			}

			require.NoError(t, err, "Expected no error but got: %v", err)
			require.NotNil(t, checker, "Expected non-nil checker")
			assert.Equal(t, tt.expectedType, checker.GetType(), "Checker type mismatch")

			// Test that returned checker implements all required methods
			assert.Implements(t, (*HealthChecker)(nil), checker, "Returned object should implement HealthChecker interface")
		})
	}
}
