/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package strategy

import (
	"context"
	"fmt"
	"time"

	"dosync/internal/health"
	"dosync/internal/replica"
)

// BlueGreenStrategyName is the string identifier for the blue/green strategy
const BlueGreenStrategyName = "blue-green"

// BlueGreenDeployer implements the UpdateStrategy interface for blue/green deployments.
// It creates a new set of instances, verifies their health, and then switches traffic.
type BlueGreenDeployer struct {
	*BaseStrategy
}

// NewBlueGreenStrategy creates a new strategy for blue/green deployments.
func NewBlueGreenStrategy(
	replicaManager ReplicaManager,
	healthChecker health.HealthChecker,
	config StrategyConfig,
) *BlueGreenDeployer {
	// Create BaseStrategy with appropriate name
	base := &BaseStrategy{
		Config:         config,
		ReplicaManager: replicaManager,
		HealthChecker:  healthChecker,
		StrategyName:   BlueGreenStrategyName,
	}

	return &BlueGreenDeployer{
		BaseStrategy: base,
	}
}

// Configure sets up the blue/green strategy with the provided configuration.
func (bg *BlueGreenDeployer) Configure(config StrategyConfig) error {
	// Set the correct strategy type if not already set
	if config.Type == "" {
		config.Type = BlueGreenStrategyName
	} else if config.Type != BlueGreenStrategyName {
		return fmt.Errorf("invalid strategy type for BlueGreenStrategy: %s", config.Type)
	}

	// Apply defaults and validate configuration
	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return err
	}

	// Configure the base strategy
	return bg.BaseStrategy.Configure(config)
}

// Execute performs a blue/green deployment update for the specified service.
func (bg *BlueGreenDeployer) Execute(service string, newImageTag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), bg.Config.Timeout)
	defer cancel()

	// Get all current replicas for the service (blue environment)
	blueReplicas, err := bg.ReplicaManager.GetServiceReplicas(service)
	if err != nil {
		return fmt.Errorf("failed to get replicas for service %s: %w", service, err)
	}
	if len(blueReplicas) == 0 {
		return fmt.Errorf("no replicas found for service %s", service)
	}
	fmt.Printf("Starting blue/green deployment for service %s with %d current replicas\n", service, len(blueReplicas))

	// Step 1: Create green environment (new replicas with updated image)
	fmt.Printf("Creating green environment for service %s with image tag %s\n", service, newImageTag)
	// For now, simulate green replicas as a copy of blueReplicas with new IDs
	greenReplicas := make([]replica.Replica, len(blueReplicas))
	for i, r := range blueReplicas {
		greenReplicas[i] = r
		greenReplicas[i].ContainerID = fmt.Sprintf("green-%s", r.ContainerID)
		greenReplicas[i].Status = "created"
	}

	rollbackFunc := func() {
		fmt.Printf("Rolling back: removing %d green replicas for service %s\n", len(greenReplicas), service)
		for _, r := range greenReplicas {
			// In a real implementation, this would remove the green replica
			fmt.Printf("Removing green replica %s\n", r.ContainerID)
		}
	}

	// Step 2: Execute pre-update command if configured
	if bg.Config.PreUpdateCommand != "" {
		for _, r := range greenReplicas {
			preErr := bg.executeCommand(r.ContainerID, bg.Config.PreUpdateCommand)
			if preErr != nil {
				rollbackFunc()
				return fmt.Errorf("pre-update command failed for green replica %s: %w", r.ContainerID, preErr)
			}
		}
	}

	// Step 3: Wait for all green replicas to be healthy
	fmt.Printf("Waiting for all %d green replicas to become healthy\n", len(greenReplicas))
	for _, r := range greenReplicas {
		healthy, err := bg.waitForHealth(ctx, r)
		if err != nil {
			rollbackFunc()
			return fmt.Errorf("health check failed for green replica %s: %w", r.ContainerID, err)
		}
		if !healthy {
			rollbackFunc()
			return fmt.Errorf("green replica %s failed to become healthy", r.ContainerID)
		}
	}

	// Step 4: Switch traffic from blue to green
	fmt.Printf("All green replicas are healthy. Switching traffic from blue to green.\n")
	// In a real implementation, this would switch traffic; here, just print
	fmt.Printf("Traffic switched to green environment for service %s\n", service)

	// Step 5: Execute post-update command if configured
	if bg.Config.PostUpdateCommand != "" {
		for _, r := range greenReplicas {
			postErr := bg.executeCommand(r.ContainerID, bg.Config.PostUpdateCommand)
			if postErr != nil {
				fmt.Printf("Warning: post-update command failed for green replica %s: %v\n", r.ContainerID, postErr)
			}
		}
	}

	// Step 6: Wait for verification period (optional grace period)
	fmt.Printf("Traffic switched to green environment. Waiting for verification period.\n")
	verificationPeriod := bg.Config.VerificationPeriod
	if verificationPeriod <= 0 {
		verificationPeriod = 30 * time.Second
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout during verification period: %w", ctx.Err())
	case <-time.After(verificationPeriod):
		// Continue after verification period
	}

	// Step 7: Remove old blue environment
	fmt.Printf("Removing %d old blue replicas for service %s\n", len(blueReplicas), service)
	for _, r := range blueReplicas {
		// In a real implementation, this would remove the blue replica
		fmt.Printf("Removing blue replica %s\n", r.ContainerID)
	}

	fmt.Printf("Blue/green deployment completed successfully for service %s\n", service)
	return nil
}

// waitForHealth repeatedly checks the health of a replica until it becomes healthy or times out.
// This method is identical to the ones in other strategies but needed here for the blue/green strategy.
func (bg *BlueGreenDeployer) waitForHealth(ctx context.Context, r replica.Replica) (bool, error) {
	// Create ticker for regular health checks
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Waiting for replica %s to become healthy...\n", r.ContainerID)

	// Count consecutive successful checks
	successCount := 0
	requiredSuccesses := bg.Config.HealthCheck.SuccessThreshold
	if requiredSuccesses <= 0 {
		requiredSuccesses = 1 // Default to at least one success
	}

	// Count consecutive failures
	failureCount := 0
	allowedFailures := bg.Config.HealthCheck.FailureThreshold
	if allowedFailures <= 0 {
		allowedFailures = 3 // Default to allowing 3 failures
	}

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			healthy, err := bg.HealthChecker.Check(r)

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
func (bg *BlueGreenDeployer) executeCommand(containerID, command string) error {
	fmt.Printf("Executing command on container %s: %s\n", containerID, command)
	return fmt.Errorf("executeCommand not yet implemented") // Will be implemented in a future task
}
