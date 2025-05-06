/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"dosync/internal/replica"
)

// DockerHealthChecker implements the HealthChecker interface using Docker's
// built-in health check status.
type DockerHealthChecker struct {
	*BaseChecker
	dockerClient client.APIClient
}

// NewDockerHealthChecker creates a new health checker that uses Docker's
// built-in health check mechanism.
func NewDockerHealthChecker(config HealthCheckConfig) (*DockerHealthChecker, error) {
	// Set the correct type if it's not already set
	if config.Type == "" {
		config.Type = DockerHealthCheck
	} else if config.Type != DockerHealthCheck {
		return nil, fmt.Errorf("invalid health check type for DockerHealthChecker: %s", config.Type)
	}

	// Create the base checker
	baseChecker, err := NewBaseChecker(DockerHealthCheck, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base checker: %w", err)
	}

	// Create the Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerHealthChecker{
		BaseChecker:  baseChecker,
		dockerClient: dockerClient,
	}, nil
}

// Check performs a health check on the specified replica using Docker's health check status.
// It implements the HealthChecker interface.
func (d *DockerHealthChecker) Check(replica replica.Replica) (bool, error) {
	// Check if it's time to perform a health check
	if !d.ShouldCheck() {
		// Return the current status if it's not time to check again
		healthy, _, _ := d.GetStatus()
		return healthy, nil
	}

	result, err := d.CheckWithDetails(replica)
	return result.Healthy, err
}

// CheckWithDetails performs a health check and returns detailed information.
// It overrides the BaseChecker implementation to provide Docker-specific details.
func (d *DockerHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	// Check if the container ID is valid
	if replica.ContainerID == "" {
		d.UpdateStatus(false, "Container ID is empty")
		return HealthCheckResult{
			Healthy:   false,
			Message:   "Container ID is empty",
			Timestamp: time.Now(),
		}, fmt.Errorf("container ID is empty")
	}

	// Query the container's health status
	ctx, cancel := context.WithTimeout(context.Background(), d.Config.Timeout)
	defer cancel()

	// Inspect the container to get its health state
	containerInfo, err := d.dockerClient.ContainerInspect(ctx, replica.ContainerID)
	if err != nil {
		message := fmt.Sprintf("Failed to inspect container %s: %v", replica.ContainerID, err)
		d.UpdateStatus(false, message)
		return HealthCheckResult{
			Healthy:   false,
			Message:   message,
			Timestamp: time.Now(),
		}, err
	}

	// Check if the container has a health check configured
	if containerInfo.State == nil || containerInfo.State.Health == nil {
		message := fmt.Sprintf("Container %s does not have a health check configured", replica.ContainerID)
		d.UpdateStatus(false, message)
		return HealthCheckResult{
			Healthy:   false,
			Message:   message,
			Timestamp: time.Now(),
		}, fmt.Errorf(message)
	}

	// Determine the health status based on the Docker health status
	var healthy bool
	var message string

	switch containerInfo.State.Health.Status {
	case container.Healthy:
		healthy = true
		message = fmt.Sprintf("Container %s is healthy", replica.ContainerID)
	case container.Unhealthy:
		healthy = false
		message = fmt.Sprintf("Container %s is unhealthy", replica.ContainerID)
	case container.Starting:
		// Container is still in the starting phase, consider it unhealthy for now
		healthy = false
		message = fmt.Sprintf("Container %s is starting", replica.ContainerID)
	default:
		healthy = false
		message = fmt.Sprintf("Container %s has unknown health status: %s", replica.ContainerID, containerInfo.State.Health.Status)
	}

	// Update the status
	d.UpdateStatus(healthy, message)

	// Return the health check result
	return HealthCheckResult{
		Healthy:   healthy,
		Message:   message,
		Timestamp: time.Now(),
	}, nil
}

// Close closes the Docker client connection if possible
func (d *DockerHealthChecker) Close() error {
	if d.dockerClient != nil {
		// Attempt to type assert to check if the client has a Close method
		if closer, ok := d.dockerClient.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}
