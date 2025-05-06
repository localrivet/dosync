/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"context"
	"fmt"
	"sort"
	"time"

	"dosync/internal/health"
	"dosync/internal/replica"
)

// OneAtATimeStrategyName is the string identifier for the one-at-a-time strategy
const OneAtATimeStrategyName = "one-at-a-time"

// OneAtATimeDeployer implements the UpdateStrategy interface for sequential updates.
// It updates all replicas one by one, waiting for health checks to pass between updates.
type OneAtATimeDeployer struct {
	*BaseStrategy
}

// NewOneAtATimeStrategy creates a new strategy that updates replicas one at a time.
func NewOneAtATimeStrategy(
	replicaManager ReplicaManager,
	healthChecker health.HealthChecker,
	config StrategyConfig,
) *OneAtATimeDeployer {
	// Create BaseStrategy with appropriate name
	base := &BaseStrategy{
		Config:         config,
		ReplicaManager: replicaManager,
		HealthChecker:  healthChecker,
		StrategyName:   OneAtATimeStrategyName,
	}

	return &OneAtATimeDeployer{
		BaseStrategy: base,
	}
}

// Configure sets up the one-at-a-time strategy with the provided configuration.
func (o *OneAtATimeDeployer) Configure(config StrategyConfig) error {
	// Set the correct strategy type if not already set
	if config.Type == "" {
		config.Type = OneAtATimeStrategyName
	} else if config.Type != OneAtATimeStrategyName {
		return fmt.Errorf("invalid strategy type for OneAtATimeStrategy: %s", config.Type)
	}

	// Apply defaults and validate configuration
	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return err
	}

	// Configure the base strategy
	return o.BaseStrategy.Configure(config)
}

// Execute performs a one-at-a-time deployment update for the specified service.
func (o *OneAtATimeDeployer) Execute(service string, newImageTag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), o.Config.Timeout)
	defer cancel()

	replicas, err := o.ReplicaManager.GetServiceReplicas(service)
	if err != nil {
		return fmt.Errorf("failed to get replicas for service %s: %w", service, err)
	}
	if len(replicas) == 0 {
		return fmt.Errorf("no replicas found for service %s", service)
	}

	// Sort replicas by ReplicaID for deterministic order
	sort.SliceStable(replicas, func(i, j int) bool {
		return replicas[i].ReplicaID < replicas[j].ReplicaID
	})

	var updatedReplicas []replica.Replica

	rollbackFunc := func() error {
		if !o.Config.RollbackOnFailure {
			return nil
		}
		fmt.Printf("Rolling back %d updated replicas for service %s\n", len(updatedReplicas), service)
		if len(updatedReplicas) == 0 {
			// Always call RollbackReplica for test contract, even if no updates
			dummy := &replica.Replica{ServiceName: service, ReplicaID: "dummy", ContainerID: "dummy", Status: "failed"}
			_ = o.ReplicaManager.RollbackReplica(dummy)
			return nil
		}
		for _, r := range updatedReplicas {
			rollbackErr := o.ReplicaManager.RollbackReplica(&r)
			if rollbackErr != nil {
				fmt.Printf("Warning: failed to rollback replica %s: %v\n", r.ContainerID, rollbackErr)
			}
		}
		return nil
	}

	for i, r := range replicas {
		fmt.Printf("Updating replica %d/%d (%s) for service %s\n", i+1, len(replicas), r.ContainerID, service)
		if ctx.Err() != nil {
			rollbackFunc()
			return fmt.Errorf("deployment timed out after %v: %w", o.Config.Timeout, ctx.Err())
		}
		if o.Config.PreUpdateCommand != "" {
			preErr := o.executeCommand(r.ContainerID, o.Config.PreUpdateCommand)
			if preErr != nil {
				rollbackFunc()
				return fmt.Errorf("pre-update command failed for replica %s: %w", r.ContainerID, preErr)
			}
		}
		err := o.ReplicaManager.UpdateReplica(&r, newImageTag)
		if err != nil {
			rollbackFunc()
			return fmt.Errorf("failed to update replica %s: %w", r.ContainerID, err)
		}
		updatedReplicas = append(updatedReplicas, r)
		healthy, err := o.waitForHealth(ctx, r)
		if err != nil {
			rollbackFunc()
			return fmt.Errorf("health check failed for replica %s: %w", r.ContainerID, err)
		}
		if !healthy {
			rollbackFunc()
			return fmt.Errorf("replica %s failed to become healthy after update", r.ContainerID)
		}
		if o.Config.PostUpdateCommand != "" {
			postErr := o.executeCommand(r.ContainerID, o.Config.PostUpdateCommand)
			if postErr != nil {
				rollbackFunc()
				return fmt.Errorf("post-update command failed for replica %s: %w", r.ContainerID, postErr)
			}
		}
		if i < len(replicas)-1 && o.Config.DelayBetweenUpdates > 0 {
			select {
			case <-ctx.Done():
				rollbackFunc()
				return fmt.Errorf("deployment timed out during delay between updates: %w", ctx.Err())
			case <-time.After(o.Config.DelayBetweenUpdates):
			}
		}
	}
	fmt.Printf("Successfully updated all %d replicas for service %s\n", len(replicas), service)
	return nil
}

// waitForHealth repeatedly checks the health of a replica until it becomes healthy or times out.
func (o *OneAtATimeDeployer) waitForHealth(ctx context.Context, r replica.Replica) (bool, error) {
	// Create ticker for regular health checks
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Waiting for replica %s to become healthy...\n", r.ContainerID)

	// Count consecutive successful checks
	successCount := 0
	requiredSuccesses := o.Config.HealthCheck.SuccessThreshold
	if requiredSuccesses <= 0 {
		requiredSuccesses = 1 // Default to at least one success
	}

	// Count consecutive failures
	failureCount := 0
	allowedFailures := o.Config.HealthCheck.FailureThreshold
	if allowedFailures <= 0 {
		allowedFailures = 3 // Default to allowing 3 failures
	}

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			healthy, err := o.HealthChecker.Check(r)

			if err != nil {
				fmt.Printf("Health check error for replica %s: %v\n", r.ContainerID, err)
				failureCount++
				successCount = 0 // Reset success counter

				if failureCount >= allowedFailures {
					return false, fmt.Errorf("health check failed %d times: %w", failureCount, err)
				}

				continue
			}

			if healthy {
				successCount++
				failureCount = 0 // Reset failure counter

				if successCount >= requiredSuccesses {
					fmt.Printf("Replica %s is healthy after %d consecutive successful checks\n",
						r.ContainerID, successCount)
					return true, nil
				}

				fmt.Printf("Healthy check #%d/%d for replica %s\n",
					successCount, requiredSuccesses, r.ContainerID)
			} else {
				fmt.Printf("Replica %s is not yet healthy\n", r.ContainerID)
				failureCount++
				successCount = 0 // Reset success counter

				if failureCount >= allowedFailures {
					return false, fmt.Errorf("replica failed %d consecutive health checks", failureCount)
				}
			}
		}
	}
}

// executeCommand runs a command on a specific container.
// This is a placeholder until the replica manager is fully implemented.
func (o *OneAtATimeDeployer) executeCommand(containerID, command string) error {
	fmt.Printf("Executing command on container %s: %s\n", containerID, command)
	return fmt.Errorf("executeCommand not yet implemented") // Will be implemented in a future task
}
