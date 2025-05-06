/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dosync/internal/health"
	"dosync/internal/replica"
)

// TestNewUpdateStrategy tests the strategy factory function
func TestNewUpdateStrategy(t *testing.T) {
	// Create a valid health check config for testing
	validHealthCheck := health.HealthCheckConfig{
		Type:    health.TCPHealthCheck,
		Port:    8080,
		Timeout: 5 * time.Second,
	}

	// Create a stub health checker for testing
	stubHealthChecker := health.NewStubTCPHealthChecker(true)

	tests := []struct {
		name                 string
		config               StrategyConfig
		expectStrategy       bool   // true if we expect a strategy to be returned
		expectedStrategyName string // name of expected strategy
		expectError          bool   // true if we expect an error
		errorMsg             string // expected error message
	}{
		{
			name: "One-at-a-time strategy returns implementation",
			config: StrategyConfig{
				Type:        string(OneAtATimeStrategy),
				HealthCheck: validHealthCheck,
				Timeout:     5 * time.Minute,
			},
			expectStrategy:       true,
			expectedStrategyName: OneAtATimeStrategyName,
			expectError:          false,
		},
		{
			name: "Percentage strategy returns implementation",
			config: StrategyConfig{
				Type:        string(PercentageStrategy),
				HealthCheck: validHealthCheck,
				Percentage:  20,
				Timeout:     5 * time.Minute,
			},
			expectStrategy:       true,
			expectedStrategyName: PercentageStrategyName,
			expectError:          false,
		},
		{
			name: "Blue-green strategy returns implementation",
			config: StrategyConfig{
				Type:        string(BlueGreenStrategy),
				HealthCheck: validHealthCheck,
				Timeout:     5 * time.Minute,
			},
			expectStrategy:       true,
			expectedStrategyName: BlueGreenStrategyName,
			expectError:          false,
		},
		{
			name: "Canary strategy returns implementation",
			config: StrategyConfig{
				Type:        string(CanaryStrategy),
				HealthCheck: validHealthCheck,
				Percentage:  10,
				Timeout:     5 * time.Minute,
			},
			expectStrategy:       true,
			expectedStrategyName: CanaryStrategyName,
			expectError:          false,
		},
		{
			name: "Invalid strategy type",
			config: StrategyConfig{
				Type:        "invalid-type",
				HealthCheck: validHealthCheck,
				Timeout:     5 * time.Minute,
			},
			expectStrategy: false,
			expectError:    true,
			errorMsg:       "invalid strategy type",
		},
		{
			name: "Invalid config (missing health check type)",
			config: StrategyConfig{
				Type:    string(OneAtATimeStrategy),
				Timeout: 5 * time.Minute,
				HealthCheck: health.HealthCheckConfig{
					// Missing Type field
					Port:    8080,
					Timeout: 5 * time.Second,
				},
			},
			expectStrategy: false,
			expectError:    true,
			errorMsg:       "invalid health check configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a stub replica manager - nil is fine since we're just testing factory creation
			var replicaManager *replica.ReplicaManager = nil

			strategy, err := NewUpdateStrategy(tt.config, replicaManager, stubHealthChecker)

			if tt.expectError {
				require.Error(t, err, "Expected an error but got nil")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error message doesn't match expected")
				assert.Nil(t, strategy, "Expected nil strategy but got non-nil")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
				require.NotNil(t, strategy, "Expected non-nil strategy but got nil")
				assert.Equal(t, tt.expectedStrategyName, strategy.Name(), "Strategy name does not match expected")
			}
		})
	}
}
