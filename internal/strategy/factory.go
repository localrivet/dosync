/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"fmt"

	"dosync/internal/health"
)

// NewUpdateStrategy creates a new deployment strategy based on the provided configuration.
// It returns an appropriate implementation of the UpdateStrategy interface.
func NewUpdateStrategy(
	config StrategyConfig,
	replicaManager ReplicaManager,
	healthChecker health.HealthChecker,
) (UpdateStrategy, error) {
	// Apply default values to config
	config.ApplyDefaults()

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid strategy configuration: %w", err)
	}

	// Create the appropriate strategy based on the type
	switch StrategyType(config.Type) {
	case OneAtATimeStrategy:
		return NewOneAtATimeStrategy(replicaManager, healthChecker, config), nil
	case PercentageStrategy:
		return NewPercentageStrategy(replicaManager, healthChecker, config), nil
	case BlueGreenStrategy:
		return NewBlueGreenStrategy(replicaManager, healthChecker, config), nil
	case CanaryStrategy:
		return NewCanaryStrategy(replicaManager, healthChecker, config), nil
	default:
		return nil, fmt.Errorf("unsupported strategy type: %s", config.Type)
	}
}
