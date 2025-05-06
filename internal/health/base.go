/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"sync"
	"time"

	"dosync/internal/replica"
)

// BaseChecker provides common functionality that can be embedded in specific health checker implementations
type BaseChecker struct {
	// Config holds the health check configuration
	Config HealthCheckConfig

	// checkerType is the type of health checker
	checkerType HealthCheckType

	// successCount tracks consecutive successful checks
	successCount int

	// failureCount tracks consecutive failed checks
	failureCount int

	// healthStatus is the current health status
	healthStatus bool

	// lastCheckTime is when the last check was performed
	lastCheckTime time.Time

	// lastMessage is the message from the last check
	lastMessage string

	// mu protects concurrent access to the checker's state
	mu sync.RWMutex
}

// NewBaseChecker creates a new base health checker with the given configuration
func NewBaseChecker(checkerType HealthCheckType, config HealthCheckConfig) (*BaseChecker, error) {
	// Create a copy of the config to avoid modification of the original
	configCopy := config

	// Validate and apply defaults to the configuration
	if err := ValidateConfig(&configCopy); err != nil {
		return nil, err
	}

	return &BaseChecker{
		Config:       configCopy,
		checkerType:  checkerType,
		healthStatus: false, // Initially not healthy until checked
	}, nil
}

// GetType returns the type of health checker
func (b *BaseChecker) GetType() HealthCheckType {
	return b.checkerType
}

// Configure updates the health checker's configuration
func (b *BaseChecker) Configure(config HealthCheckConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create a copy of the config to avoid modification of the original
	configCopy := config

	// Validate the new configuration
	if err := ValidateConfig(&configCopy); err != nil {
		return err
	}

	// Update the configuration
	b.Config = configCopy

	// Reset state since configuration has changed
	b.successCount = 0
	b.failureCount = 0

	return nil
}

// UpdateStatus updates the health status based on the check result
// This should be called by specific health checker implementations after each health check
func (b *BaseChecker) UpdateStatus(healthy bool, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastCheckTime = time.Now()
	b.lastMessage = message

	if healthy {
		b.successCount++
		b.failureCount = 0

		// Health status changes to healthy when success threshold is reached
		if b.successCount >= b.Config.SuccessThreshold {
			b.healthStatus = true
		}
	} else {
		b.failureCount++
		b.successCount = 0

		// Health status changes to unhealthy when failure threshold is reached
		if b.failureCount >= b.Config.FailureThreshold {
			b.healthStatus = false
		}
	}
}

// GetStatus returns the current health status and metadata
// This can be used by specific health checker implementations in their Check methods
func (b *BaseChecker) GetStatus() (bool, string, time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.healthStatus, b.lastMessage, b.lastCheckTime
}

// CreateHealthCheckResult creates a HealthCheckResult from the current status
func (b *BaseChecker) CreateHealthCheckResult() HealthCheckResult {
	healthy, message, timestamp := b.GetStatus()

	return HealthCheckResult{
		Healthy:   healthy,
		Message:   message,
		Timestamp: timestamp,
	}
}

// ShouldCheck determines if it's time to perform another health check
// based on the retry interval and the last check time
func (b *BaseChecker) ShouldCheck() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// If no check has been performed yet, or if it's been long enough since the last check
	return b.lastCheckTime.IsZero() || time.Since(b.lastCheckTime) >= b.Config.RetryInterval
}

// These are placeholder methods that specific health checkers will override

// Check performs a health check on the specified replica
// This is a placeholder that specific health checkers must override
func (b *BaseChecker) Check(replica replica.Replica) (bool, error) {
	// This should be overridden by specific implementations
	panic("BaseChecker.Check must be overridden by a specific health checker implementation")
}

// CheckWithDetails performs a health check and returns detailed information
// This will typically be overridden by specific health checkers for more detailed results
func (b *BaseChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	// Default implementation calls Check and wraps the result in a HealthCheckResult
	// Specific implementations can override for more detailed information
	_, err := b.Check(replica)
	if err != nil {
		return HealthCheckResult{
			Healthy:   false,
			Message:   err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return b.CreateHealthCheckResult(), nil
}
