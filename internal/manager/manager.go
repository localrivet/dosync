package manager

import (
	"fmt"
	"time"

	"dosync/internal/dependency"
	"dosync/internal/health"
	"dosync/internal/logx"
	"dosync/internal/metrics"
	"dosync/internal/replica"
	"dosync/internal/rollback"
)

// RollingUpdateManager integrates all components for managing rolling updates
type RollingUpdateManager struct {
	Config             *RollingUpdateConfig
	ReplicaManager     *replica.ReplicaManager
	HealthChecker      health.HealthChecker
	Strategy           UpdateStrategy
	RollbackController *rollback.RollbackControllerImpl
	DependencyManager  dependency.DependencyManager
	Notifiers          []Notifier
	MetricsCollector   *metrics.Collector

	// Recovery and error handling
	recovery *RecoveryHandler
	logger   Logger
}

// NewRollingUpdateManager creates a new manager with the specified configuration file
func NewRollingUpdateManager(config *RollingUpdateConfig, logger Logger) (*RollingUpdateManager, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	if logger == nil {
		logger = logx.NewDefaultLogger()
	}

	manager := &RollingUpdateManager{
		Config: config,
		logger: logger,
	}

	// Initialize the recovery handler early so we can use it during setup
	manager.recovery = NewRecoveryHandler(manager, logger)

	// Initialize components with error wrapping for better context
	if err := manager.initComponents(); err != nil {
		return nil, WrapError(err, "initialization", "failed to initialize manager components", "", "", true, false)
	}

	logger.Info("RollingUpdateManager successfully initialized")
	return manager, nil
}

// initComponents initializes all the required components for the manager
func (rum *RollingUpdateManager) initComponents() error {
	var err error

	// Initialize ReplicaManager
	rum.logger.Info("Initializing ReplicaManager")
	rum.ReplicaManager, err = replica.NewReplicaManagerWithAllDetectors(rum.Config.ComposeFilePath)
	if err != nil {
		return WrapError(err, "replica", "failed to initialize ReplicaManager", "", "", true, false)
	}

	// Initialize HealthChecker based on config
	rum.logger.Info("Initializing HealthChecker")
	healthConfig := createHealthCheckConfig(rum.Config.HealthCheckTimeout, rum.Config.HealthCheckRetries)
	rum.HealthChecker, err = health.NewHealthChecker(healthConfig)
	if err != nil {
		return WrapError(err, "health", "failed to initialize HealthChecker", "", "", true, false)
	}

	// Initialize Strategy based on config
	rum.logger.Info("Initializing UpdateStrategy: %s", rum.Config.UpdateStrategy)
	strategyAdapter, err := NewStrategyAdapter(rum.Config.UpdateStrategy, rum.logger)
	if err != nil {
		return WrapError(err, "strategy", "failed to initialize Strategy", "", "", true, false)
	}
	rum.Strategy = strategyAdapter

	// Initialize RollbackController
	rum.logger.Info("Initializing RollbackController")
	rum.RollbackController, err = rollback.NewRollbackController(rum.Config.RollbackConfig)
	if err != nil {
		return WrapError(err, "rollback", "failed to initialize RollbackController", "", "", true, false)
	}

	// Initialize DependencyManager
	rum.logger.Info("Initializing DependencyManager")
	rum.DependencyManager, err = dependency.NewDependencyManager(rum.Config.ComposeFilePath)
	if err != nil {
		return WrapError(err, "dependency", "failed to initialize DependencyManager", "", "", true, false)
	}

	// Initialize Notifiers
	rum.logger.Info("Initializing Notifiers")
	notifierConfig := rum.Config.NotificationsConfig
	if notifierConfig != nil {
		// Initialize slack notifier if configured
		if notifierConfig.SlackConfig != nil && notifierConfig.SlackConfig.Enabled {
			notifier, err := NewNotifierAdapter(notifierConfig.SlackConfig, rum.logger)
			if err != nil {
				rum.logger.Error("Failed to initialize Slack notifier: %v", err)
				// Continue even if notification setup fails - it's not critical
			} else {
				rum.Notifiers = append(rum.Notifiers, notifier)
			}
		}

		// Add other notifiers here as they are implemented
	}

	// Initialize MetricsCollector
	rum.logger.Info("Initializing MetricsCollector")
	retention := createRetentionConfig()
	rum.MetricsCollector, err = metrics.NewCollector(rum.Config.MetricsDB, retention)
	if err != nil {
		rum.logger.Error("Failed to initialize MetricsCollector: %v", err)
		// Continue without metrics if it fails - not a critical component
	}

	return nil
}

// createHealthCheckConfig constructs a HealthCheckConfig for the manager
func createHealthCheckConfig(timeout time.Duration, retries int) health.HealthCheckConfig {
	return health.HealthCheckConfig{
		Type:             health.DockerHealthCheck,
		Timeout:          timeout,
		FailureThreshold: retries,
		// Other fields will be defaulted by health.ValidateConfig
	}
}

// createRetentionConfig returns the default metrics retention config
func createRetentionConfig() metrics.RetentionConfig {
	return metrics.DefaultRetentionConfig()
}

// Update performs a rolling update for the specified service with the new image tag
func (rum *RollingUpdateManager) Update(service string, newImageTag string) error {
	rum.logger.Info("Starting update for service %s to version %s", service, newImageTag)

	startTime := time.Now()

	// Record deployment start in metrics
	if rum.MetricsCollector != nil {
		if err := rum.MetricsCollector.RecordDeploymentStart(service, newImageTag); err != nil {
			// Log but continue - metrics are not critical
			rum.logger.Error("Error recording deployment start: %v", err)
		}
	}

	// Verify service exists
	services, err := rum.ReplicaManager.GetAllReplicas()
	if err != nil {
		wrappedErr := WrapError(err, "replica", "failed to get service replicas", service, newImageTag, true, false)
		rum.recovery.CleanupAfterFailure(service, newImageTag, wrappedErr)
		return wrappedErr
	}

	if _, exists := services[service]; !exists {
		wrappedErr := WrapError(ErrServiceNotFound, "validation", "service not found in compose file", service, newImageTag, true, false)
		rum.recovery.CleanupAfterFailure(service, newImageTag, wrappedErr)
		return wrappedErr
	}

	// Send start notifications
	for _, notifier := range rum.Notifiers {
		if notifier.ShouldNotifyOnStart() {
			if err := notifier.SendDeploymentStart(service, newImageTag); err != nil {
				rum.logger.Error("Error sending start notification: %v", err)
				// Continue despite notification error
			}
		}
	}

	// Get dependent services in the correct order
	updateOrder, err := rum.DependencyManager.GetUpdateOrder([]string{service})
	if err != nil {
		wrappedErr := WrapError(err, "dependency", "failed to determine update order", service, newImageTag, true, false)
		rum.recovery.CleanupAfterFailure(service, newImageTag, wrappedErr)
		return wrappedErr
	}

	rum.logger.Info("Update order determined: %v", updateOrder)

	// Execute the update strategy for each service in dependency order
	for _, svc := range updateOrder {
		rum.logger.Info("Updating service %s according to dependency order", svc)

		// Only apply new image tag to the requested service
		imageTag := ""
		if svc == service {
			imageTag = newImageTag
		}

		// Get replicas for the service
		replicas, err := rum.ReplicaManager.GetServiceReplicas(svc)
		if err != nil {
			wrappedErr := WrapError(err, "replica", "failed to get service replicas", svc, imageTag, true, true)
			// Try to recover from the error
			if recoveryErr := rum.recovery.HandleError(wrappedErr, svc, imageTag); recoveryErr != nil {
				rum.logger.Error("Failed to recover from error: %v", recoveryErr)
				rum.recovery.CleanupAfterFailure(svc, imageTag, recoveryErr)
				return recoveryErr
			}
		}

		rum.logger.Info("Found %d replicas for service %s", len(replicas), svc)

		// Execute the update strategy on the service
		if err := rum.Strategy.Execute(replicas, imageTag, func(replica replica.Replica) bool {
			// Health check callback for the strategy to use
			healthy, err := rum.HealthChecker.Check(replica)
			if err != nil {
				rum.logger.Error("Health check error for replica %s: %v", replica.ReplicaID, err)
				return false
			}
			return healthy
		}); err != nil {
			wrappedErr := WrapError(err, "strategy", "strategy execution failed", svc, imageTag, true, true)

			// Try to recover from the error
			if recoveryErr := rum.recovery.HandleError(wrappedErr, svc, imageTag); recoveryErr != nil {
				rum.logger.Error("Failed to recover from strategy error: %v", recoveryErr)
				rum.recovery.CleanupAfterFailure(svc, imageTag, recoveryErr)
				return recoveryErr
			}
		}

		// Verify all replicas are healthy after update
		updatedReplicas, err := rum.ReplicaManager.GetServiceReplicas(svc)
		if err != nil {
			wrappedErr := WrapError(err, "replica", "failed to get updated replicas", svc, imageTag, true, true)
			if recoveryErr := rum.recovery.HandleError(wrappedErr, svc, imageTag); recoveryErr != nil {
				rum.recovery.CleanupAfterFailure(svc, imageTag, recoveryErr)
				return recoveryErr
			}
		}

		for _, replica := range updatedReplicas {
			healthy, err := rum.HealthChecker.Check(replica)
			if err != nil || !healthy {
				wrappedErr := WrapError(ErrHealthCheckFailed, "health",
					fmt.Sprintf("health check failed for replica %s", replica.ReplicaID),
					svc, imageTag, true, true)

				if recoveryErr := rum.recovery.HandleError(wrappedErr, svc, imageTag); recoveryErr != nil {
					rum.recovery.CleanupAfterFailure(svc, imageTag, recoveryErr)
					return recoveryErr
				}
			}
		}

		rum.logger.Info("Successfully updated service %s", svc)
	}

	// Calculate deployment duration
	duration := time.Since(startTime)

	// Ensure the deployment completed successfully with final checks
	if err := rum.recovery.EnsureCompletion(service, newImageTag, duration); err != nil {
		return err
	}

	rum.logger.Info("Rolling update completed successfully for service %s to version %s (duration: %v)",
		service, newImageTag, duration)
	return nil
}

// Rollback performs a rollback for the specified service
func (rum *RollingUpdateManager) Rollback(service string) error {
	rum.logger.Info("Starting rollback for service %s", service)

	// Verify service exists
	services, err := rum.ReplicaManager.GetAllReplicas()
	if err != nil {
		return WrapError(err, "replica", "failed to get service list", service, "", true, false)
	}

	if _, exists := services[service]; !exists {
		return WrapError(ErrServiceNotFound, "validation", "service not found", service, "", true, false)
	}

	// Get rollback history
	entries, err := rum.RollbackController.GetRollbackHistory(service)
	if err != nil {
		return WrapError(err, "rollback", "failed to get rollback history", service, "", true, false)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no rollback history for service %s", service)
	}

	// Get previous version from the first (most recent) entry
	previousVersion := entries[0].ImageTag

	// Record rollback start in metrics
	currentVersion := "unknown" // Default in case we can't determine
	// Try to get the current version from the service replicas
	serviceReplicas, err := rum.ReplicaManager.GetServiceReplicas(service)
	if err == nil && len(serviceReplicas) > 0 {
		// Here we would actually need to extract version from the replica
		// For now, using a placeholder
		currentVersion = "current"
	}

	if rum.MetricsCollector != nil {
		if err := rum.MetricsCollector.RecordRollback(service, currentVersion, previousVersion); err != nil {
			rum.logger.Error("Failed to record rollback in metrics: %v", err)
			// Continue anyway - metrics are not critical
		}
	}

	// Send rollback notifications
	for _, notifier := range rum.Notifiers {
		if notifier.ShouldNotifyOnRollback() {
			if err := notifier.SendRollback(service, currentVersion, previousVersion); err != nil {
				rum.logger.Error("Failed to send rollback notification: %v", err)
				// Continue despite notification error
			}
		}
	}

	// Perform the rollback
	if err := rum.RollbackController.Rollback(service); err != nil {
		return WrapError(err, "rollback", "rollback execution failed", service, previousVersion, true, false)
	}

	// Verify health after rollback
	serviceReplicas, err = rum.ReplicaManager.GetServiceReplicas(service)
	if err != nil {
		return WrapError(err, "replica", "failed to get replicas after rollback", service, previousVersion, true, false)
	}

	for _, replica := range serviceReplicas {
		healthy, err := rum.HealthChecker.Check(replica)
		if err != nil || !healthy {
			return WrapError(ErrHealthCheckFailed, "health", "service unhealthy after rollback",
				service, previousVersion, true, false)
		}
	}

	rum.logger.Info("Successfully rolled back service %s to version %s", service, previousVersion)
	return nil
}

// RollbackToVersion rolls back a service to a specific version
func (rum *RollingUpdateManager) RollbackToVersion(service string, version string) error {
	rum.logger.Info("Starting targeted rollback for service %s to version %s", service, version)

	// Verify service exists
	services, err := rum.ReplicaManager.GetAllReplicas()
	if err != nil {
		return WrapError(err, "replica", "failed to get service list", service, version, true, false)
	}

	if _, exists := services[service]; !exists {
		return WrapError(ErrServiceNotFound, "validation", "service not found", service, version, true, false)
	}

	// Record rollback in metrics
	currentVersion := "unknown" // Default in case we can't determine
	// Try to get the current version from the service replicas
	serviceReplicas, err := rum.ReplicaManager.GetServiceReplicas(service)
	if err == nil && len(serviceReplicas) > 0 {
		// Here we would actually need to extract version from the replica
		// For now, using a placeholder
		currentVersion = "current"
	}

	if rum.MetricsCollector != nil {
		if err := rum.MetricsCollector.RecordRollback(service, currentVersion, version); err != nil {
			rum.logger.Error("Failed to record targeted rollback in metrics: %v", err)
			// Continue anyway - metrics are not critical
		}
	}

	// Send rollback notifications
	for _, notifier := range rum.Notifiers {
		if notifier.ShouldNotifyOnRollback() {
			if err := notifier.SendRollback(service, currentVersion, version); err != nil {
				rum.logger.Error("Failed to send rollback notification: %v", err)
				// Continue despite notification error
			}
		}
	}

	// Execute the rollback to the specific version
	if err := rum.RollbackController.RollbackToVersion(service, version); err != nil {
		return WrapError(err, "rollback", "targeted rollback execution failed", service, version, true, false)
	}

	// Verify health after rollback
	serviceReplicas, err = rum.ReplicaManager.GetServiceReplicas(service)
	if err != nil {
		return WrapError(err, "replica", "failed to get replicas after targeted rollback", service, version, true, false)
	}

	for _, replica := range serviceReplicas {
		healthy, err := rum.HealthChecker.Check(replica)
		if err != nil || !healthy {
			return WrapError(ErrHealthCheckFailed, "health", "service unhealthy after targeted rollback",
				service, version, true, false)
		}
	}

	rum.logger.Info("Successfully rolled back service %s to version %s", service, version)
	return nil
}
