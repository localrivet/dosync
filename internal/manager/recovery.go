package manager

import (
	"dosync/internal/logx"
	"fmt"
	"time"
)

// Logger interface abstracts the logging functionality
// Use logx.Logger directly
// type Logger = logx.Logger

type Logger = logx.Logger

// RecoveryHandler provides functions for handling and recovering from errors
type RecoveryHandler struct {
	logger     Logger
	manager    *RollingUpdateManager
	maxRetries int
}

// NewRecoveryHandler creates a new RecoveryHandler
func NewRecoveryHandler(manager *RollingUpdateManager, logger Logger) *RecoveryHandler {
	if logger == nil {
		logger = logx.NewDefaultLogger()
	}
	return &RecoveryHandler{
		logger:     logger,
		manager:    manager,
		maxRetries: 3,
	}
}

// HandleError processes an error and determines if recovery is possible
func (h *RecoveryHandler) HandleError(err error, service, version string) error {
	if err == nil {
		return nil
	}

	// Log the error with context
	h.logger.Error("Error occurred: %v", err)

	// Attempt to get additional context from the error
	component := GetErrorComponent(err)
	h.logger.Debug("Error component: %s", component)

	// Skip recovery for non-recoverable errors
	if !IsRecoverable(err) {
		h.logger.Error("Non-recoverable error: %v", err)
		return err
	}

	// Try to recover based on error type
	switch {
	case component == "strategy":
		return h.handleStrategyError(err, service, version)
	case component == "health":
		return h.handleHealthError(err, service, version)
	case component == "replica":
		return h.handleReplicaError(err, service, version)
	case component == "dependency":
		return h.handleDependencyError(err, service, version)
	default:
		// For unknown components, attempt rollback if critical
		if IsCritical(err) {
			h.logger.Warn("Critical error in unknown component: %v, attempting rollback", err)
			return h.performRollback(service, "Error in unknown component")
		}
		return err
	}
}

// handleStrategyError handles errors from the update strategy
func (h *RecoveryHandler) handleStrategyError(err error, service, version string) error {
	h.logger.Warn("Strategy error: %v", err)

	// Always rollback for strategy errors, they're likely to be serious
	return h.performRollback(service, fmt.Sprintf("Strategy error: %v", err))
}

// handleHealthError handles health check errors
func (h *RecoveryHandler) handleHealthError(err error, service, version string) error {
	h.logger.Warn("Health check error: %v", err)

	// Try waiting and rechecking health a few times before giving up
	for i := 0; i < h.maxRetries; i++ {
		h.logger.Info("Retrying health check for service %s (%d/%d)", service, i+1, h.maxRetries)
		time.Sleep(10 * time.Second * time.Duration(i+1)) // Exponential backoff

		// Get replicas and check health
		replicas, err := h.manager.ReplicaManager.GetServiceReplicas(service)
		if err != nil {
			h.logger.Error("Failed to get replicas for health retry: %v", err)
			continue
		}

		allHealthy := true
		for _, replica := range replicas {
			healthy, err := h.manager.HealthChecker.Check(replica)
			if err != nil || !healthy {
				allHealthy = false
				h.logger.Warn("Replica %s still unhealthy: %v", replica.ReplicaID, err)
				break
			}
		}

		if allHealthy {
			h.logger.Info("Health check now passing for service %s", service)
			return nil
		}
	}

	// After max retries, perform rollback
	h.logger.Error("Health check persistently failing for service %s, rolling back", service)
	return h.performRollback(service, "Persistent health check failures")
}

// handleReplicaError handles replica-related errors
func (h *RecoveryHandler) handleReplicaError(err error, service, version string) error {
	h.logger.Warn("Replica error: %v", err)

	// Try refreshing replica information first
	h.logger.Info("Attempting to refresh replica information for %s", service)
	if err := h.manager.ReplicaManager.RefreshReplicas(); err != nil {
		h.logger.Error("Failed to refresh replicas: %v", err)
		return h.performRollback(service, "Failed to refresh replicas")
	}

	// Check if replicas are now detectable
	replicas, err := h.manager.ReplicaManager.GetServiceReplicas(service)
	if err != nil || len(replicas) == 0 {
		h.logger.Error("Still unable to detect replicas for %s: %v", service, err)
		return h.performRollback(service, "Persistent replica detection failure")
	}

	h.logger.Info("Successfully refreshed replica information for %s", service)
	return nil
}

// handleDependencyError handles dependency-related errors
func (h *RecoveryHandler) handleDependencyError(err error, service, version string) error {
	h.logger.Warn("Dependency error: %v", err)

	// Circular dependency errors require manual intervention
	if err == ErrCircularDependencyDetected {
		h.logger.Error("Circular dependency detected: %v - requires manual intervention", err)
		return err
	}

	// For other dependency errors, try reevaluating
	deps, err := h.manager.DependencyManager.GetUpdateOrder([]string{service})
	if err != nil {
		h.logger.Error("Persistent dependency resolution error: %v", err)
		return err
	}

	h.logger.Info("Successfully resolved dependencies for %s: %v", service, deps)
	return nil
}

// performRollback executes a rollback for a service
func (h *RecoveryHandler) performRollback(service, reason string) error {
	h.logger.Info("Initiating rollback for service %s: %s", service, reason)

	if err := h.manager.Rollback(service); err != nil {
		h.logger.Error("Rollback failed for service %s: %v", service, err)
		return WrapError(err, "rollback", "automatic rollback failed", service, "", true, false)
	}

	h.logger.Info("Rollback successful for service %s", service)
	return nil
}

// CleanupAfterFailure performs cleanup operations after a failure
func (h *RecoveryHandler) CleanupAfterFailure(service, version string, err error) error {
	h.logger.Info("Performing cleanup after failure for service %s", service)

	// Log the cleanup operation in metrics
	if h.manager.MetricsCollector != nil {
		if err := h.manager.MetricsCollector.RecordDeploymentFailure(service, version, err.Error()); err != nil {
			h.logger.Error("Failed to record deployment failure in metrics: %v", err)
		}
	}

	// Send notifications about the failure and cleanup
	for _, notifier := range h.manager.Notifiers {
		if notifier.ShouldNotifyOnFailure() {
			if err := notifier.SendDeploymentFailure(service, version, err.Error()); err != nil {
				h.logger.Error("Failed to send failure notification: %v", err)
			}
		}
	}

	// Check if any resources need cleaning up
	h.logger.Info("Cleanup completed for service %s", service)
	return nil
}

// EnsureCompletion verifies that a deployment completed successfully
func (h *RecoveryHandler) EnsureCompletion(service, version string, duration time.Duration) error {
	h.logger.Info("Verifying deployment completion for service %s", service)

	// Perform final health checks
	replicas, err := h.manager.ReplicaManager.GetServiceReplicas(service)
	if err != nil {
		h.logger.Error("Failed to get replicas for completion verification: %v", err)
		return err
	}

	// Check each replica's health
	for _, replica := range replicas {
		healthy, err := h.manager.HealthChecker.Check(replica)
		if err != nil || !healthy {
			h.logger.Error("Replica %s is unhealthy after deployment: %v", replica.ReplicaID, err)
			return WrapError(ErrHealthCheckFailed, "completion", "health check failed after deployment", service, version, true, true)
		}
	}

	// Record successful completion in metrics
	if h.manager.MetricsCollector != nil {
		if err := h.manager.MetricsCollector.RecordDeploymentSuccess(service, version, duration); err != nil {
			h.logger.Error("Failed to record deployment success in metrics: %v", err)
		}
	}

	// Send success notifications
	for _, notifier := range h.manager.Notifiers {
		if notifier.ShouldNotifyOnSuccess() {
			if err := notifier.SendDeploymentSuccess(service, version, duration); err != nil {
				h.logger.Error("Failed to send success notification: %v", err)
			}
		}
	}

	h.logger.Info("Deployment successfully completed for service %s", service)
	return nil
}
