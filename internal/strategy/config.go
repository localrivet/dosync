/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"fmt"
	"time"

	"dosync/internal/health"
)

// StrategyConfig represents the configuration for a deployment strategy.
// It contains all the parameters needed for any type of deployment strategy.
type StrategyConfig struct {
	// Type specifies the strategy type to use
	Type string

	// HealthCheck defines how to check the health of updated replicas
	HealthCheck health.HealthCheckConfig

	// DelayBetweenUpdates is the minimum time to wait between updating replicas
	DelayBetweenUpdates time.Duration

	// Percentage defines the percentage of replicas to update at once (for percentage-based strategy)
	Percentage int

	// PreUpdateCommand is an optional command to execute before updating a replica
	PreUpdateCommand string

	// PostUpdateCommand is an optional command to execute after updating a replica
	PostUpdateCommand string

	// Timeout is the maximum duration to wait for a deployment to complete
	Timeout time.Duration

	// RollbackOnFailure determines whether to rollback to the previous version if an update fails
	RollbackOnFailure bool

	// VerificationPeriod is the grace period after switching traffic in blue/green deployments (optional)
	VerificationPeriod time.Duration
}

// Validate checks if the configuration is valid for the specified strategy type.
// It returns an error if the configuration is invalid.
func (c *StrategyConfig) Validate() error {
	// Check if the strategy type is valid
	if !IsValidStrategyType(c.Type) {
		return fmt.Errorf("invalid strategy type: %s", c.Type)
	}

	// Check timeout
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	// Type-specific validations
	switch StrategyType(c.Type) {
	case PercentageStrategy:
		if c.Percentage <= 0 || c.Percentage > 100 {
			return fmt.Errorf("percentage must be between 1 and 100, got %d", c.Percentage)
		}
	case CanaryStrategy:
		if c.Percentage <= 0 || c.Percentage > 50 {
			return fmt.Errorf("canary percentage should typically be between 1 and 50, got %d", c.Percentage)
		}
	}

	// Validate health check configuration
	if _, err := health.NewHealthChecker(c.HealthCheck); err != nil {
		return fmt.Errorf("invalid health check configuration: %w", err)
	}

	return nil
}

// ApplyDefaults sets default values for configuration fields that are not set.
func (c *StrategyConfig) ApplyDefaults() {
	// Set default timeout if not specified
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Minute
	}

	// Set default delay between updates if not specified
	if c.DelayBetweenUpdates <= 0 {
		c.DelayBetweenUpdates = 1 * time.Second
	}

	// Set default percentage for percentage-based strategies
	if (StrategyType(c.Type) == PercentageStrategy || StrategyType(c.Type) == CanaryStrategy) && c.Percentage <= 0 {
		// Default to 20% for percentage strategy
		if StrategyType(c.Type) == PercentageStrategy {
			c.Percentage = 20
		}
		// Default to 10% for canary strategy
		if StrategyType(c.Type) == CanaryStrategy {
			c.Percentage = 10
		}
	}

	// RollbackOnFailure is a boolean, so it defaults to false in Go
	// We only set it to true for new configs, but we'll keep any explicitly false values
	// as false (which means explicitly telling the system NOT to roll back on failure)

	// Set default verification period for blue/green if not specified
	if StrategyType(c.Type) == BlueGreenStrategy && c.VerificationPeriod <= 0 {
		c.VerificationPeriod = 30 * time.Second
	}
}
