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
)

// TestStrategyConfig_Validate tests the validation of strategy configurations
func TestStrategyConfig_Validate(t *testing.T) {
	// Create a valid health check config for testing
	validHealthCheck := health.HealthCheckConfig{
		Type:    health.TCPHealthCheck,
		Port:    8080,
		Timeout: 5 * time.Second,
	}

	tests := []struct {
		name        string
		config      StrategyConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid one-at-a-time strategy",
			config: StrategyConfig{
				Type:                string(OneAtATimeStrategy),
				HealthCheck:         validHealthCheck,
				DelayBetweenUpdates: 1 * time.Second,
				Timeout:             5 * time.Minute,
				RollbackOnFailure:   true,
			},
			expectError: false,
		},
		{
			name: "Valid percentage strategy",
			config: StrategyConfig{
				Type:                string(PercentageStrategy),
				HealthCheck:         validHealthCheck,
				DelayBetweenUpdates: 1 * time.Second,
				Percentage:          25,
				Timeout:             5 * time.Minute,
				RollbackOnFailure:   true,
			},
			expectError: false,
		},
		{
			name: "Valid canary strategy",
			config: StrategyConfig{
				Type:                string(CanaryStrategy),
				HealthCheck:         validHealthCheck,
				DelayBetweenUpdates: 1 * time.Second,
				Percentage:          10,
				Timeout:             5 * time.Minute,
				RollbackOnFailure:   true,
			},
			expectError: false,
		},
		{
			name: "Invalid strategy type",
			config: StrategyConfig{
				Type:        "invalid-type",
				Timeout:     5 * time.Minute,
				HealthCheck: validHealthCheck,
			},
			expectError: true,
			errorMsg:    "invalid strategy type",
		},
		{
			name: "Invalid timeout (zero)",
			config: StrategyConfig{
				Type:        string(OneAtATimeStrategy),
				Timeout:     0,
				HealthCheck: validHealthCheck,
			},
			expectError: true,
			errorMsg:    "timeout must be greater than zero",
		},
		{
			name: "Invalid percentage (too high)",
			config: StrategyConfig{
				Type:        string(PercentageStrategy),
				Percentage:  101,
				Timeout:     5 * time.Minute,
				HealthCheck: validHealthCheck,
			},
			expectError: true,
			errorMsg:    "percentage must be between 1 and 100",
		},
		{
			name: "Invalid percentage (negative)",
			config: StrategyConfig{
				Type:        string(PercentageStrategy),
				Percentage:  -10,
				Timeout:     5 * time.Minute,
				HealthCheck: validHealthCheck,
			},
			expectError: true,
			errorMsg:    "percentage must be between 1 and 100",
		},
		{
			name: "Invalid canary percentage (too high)",
			config: StrategyConfig{
				Type:        string(CanaryStrategy),
				Percentage:  75,
				Timeout:     5 * time.Minute,
				HealthCheck: validHealthCheck,
			},
			expectError: true,
			errorMsg:    "canary percentage should typically be between 1 and 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err, "Expected an error but got nil")
				assert.Contains(t, err.Error(), tt.errorMsg, "Error message doesn't match expected")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}

// TestStrategyConfig_ApplyDefaults tests default value application
func TestStrategyConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name               string
		config             StrategyConfig
		expectedTimeout    time.Duration
		expectedDelay      time.Duration
		expectedPercentage int
		expectedRollback   bool // The expected value after defaults are applied
	}{
		{
			name: "Apply defaults to empty config",
			config: StrategyConfig{
				Type: string(OneAtATimeStrategy),
				// RollbackOnFailure defaults to false unless set
			},
			expectedTimeout:    5 * time.Minute,
			expectedDelay:      1 * time.Second,
			expectedPercentage: 0,     // Not applicable for one-at-a-time
			expectedRollback:   false, // Defaults to false
		},
		{
			name: "Apply defaults to percentage strategy",
			config: StrategyConfig{
				Type: string(PercentageStrategy),
				// RollbackOnFailure defaults to false unless set
			},
			expectedTimeout:    5 * time.Minute,
			expectedDelay:      1 * time.Second,
			expectedPercentage: 20,
			expectedRollback:   false, // Defaults to false
		},
		{
			name: "Apply defaults to canary strategy",
			config: StrategyConfig{
				Type: string(CanaryStrategy),
				// RollbackOnFailure defaults to false unless set
			},
			expectedTimeout:    5 * time.Minute,
			expectedDelay:      1 * time.Second,
			expectedPercentage: 10,
			expectedRollback:   false, // Defaults to false
		},
		{
			name: "Respect explicitly set values",
			config: StrategyConfig{
				Type:                string(PercentageStrategy),
				Timeout:             10 * time.Minute,
				DelayBetweenUpdates: 5 * time.Second,
				Percentage:          30,
				RollbackOnFailure:   true, // Explicitly set to true
			},
			expectedTimeout:    10 * time.Minute,
			expectedDelay:      5 * time.Second,
			expectedPercentage: 30,
			expectedRollback:   true, // Should remain true as explicitly set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to apply defaults to
			config := tt.config
			config.ApplyDefaults()

			// Check that defaults were applied correctly
			assert.Equal(t, tt.expectedTimeout, config.Timeout)
			assert.Equal(t, tt.expectedDelay, config.DelayBetweenUpdates)
			assert.Equal(t, tt.expectedPercentage, config.Percentage)
			assert.Equal(t, tt.expectedRollback, config.RollbackOnFailure)
		})
	}
}
