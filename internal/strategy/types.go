/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"dosync/internal/health"
	"dosync/internal/replica"
)

// UpdateStrategy defines the interface for all deployment strategies.
// Each strategy is responsible for executing a deployment update based on its algorithm.
type UpdateStrategy interface {
	// Execute performs the deployment update for the specified service using the new image tag.
	// It returns an error if the update fails.
	Execute(service string, newImageTag string) error

	// Configure sets up the strategy with the provided configuration.
	// It returns an error if the configuration is invalid.
	Configure(config StrategyConfig) error

	// Name returns the name of the strategy
	Name() string
}

// StrategyType represents the supported deployment strategy types
type StrategyType string

// Supported deployment strategy types
const (
	OneAtATimeStrategy StrategyType = "one-at-a-time"
	PercentageStrategy StrategyType = "percentage"
	BlueGreenStrategy  StrategyType = "blue-green"
	CanaryStrategy     StrategyType = "canary"
)

// ValidStrategyTypes returns a slice of all valid strategy types
func ValidStrategyTypes() []StrategyType {
	return []StrategyType{
		OneAtATimeStrategy,
		PercentageStrategy,
		BlueGreenStrategy,
		CanaryStrategy,
	}
}

// IsValidStrategyType checks if a given strategy type is valid
func IsValidStrategyType(strategyType string) bool {
	for _, validType := range ValidStrategyTypes() {
		if string(validType) == strategyType {
			return true
		}
	}
	return false
}

// ReplicaManager defines the interface required by deployment strategies for managing replicas.
type ReplicaManager interface {
	GetServiceReplicas(service string) ([]replica.Replica, error)
	UpdateReplica(r *replica.Replica, tag string) error
	RollbackReplica(r *replica.Replica) error
	RegisterDetector(replicaType replica.ReplicaType, detector replica.ReplicaDetector)
	HasDetector(replicaType replica.ReplicaType) bool
	GetDetector(replicaType replica.ReplicaType) replica.ReplicaDetector
	UnregisterDetector(replicaType replica.ReplicaType) bool
	GetAllReplicas() (map[string][]replica.Replica, error)
	RefreshReplicas() error
}

// BaseStrategy provides common functionality for all strategy implementations.
type BaseStrategy struct {
	Config         StrategyConfig
	ReplicaManager ReplicaManager
	HealthChecker  health.HealthChecker
	StrategyName   string
}

// Name returns the name of the strategy
func (b *BaseStrategy) Name() string {
	return b.StrategyName
}

// Configure sets up the base strategy with the provided configuration.
func (b *BaseStrategy) Configure(config StrategyConfig) error {
	// Store the configuration
	b.Config = config
	return nil
}
