/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package health

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"dosync/internal/replica"
)

// CommandHealthChecker implements the HealthChecker interface using commands
// executed locally that interact with the container.
type CommandHealthChecker struct {
	*BaseChecker
}

// NewCommandHealthChecker creates a new health checker that executes a custom
// command to determine health status.
func NewCommandHealthChecker(config HealthCheckConfig) (*CommandHealthChecker, error) {
	// Set the correct type if it's not already set
	if config.Type == "" {
		config.Type = CommandHealthCheck
	} else if config.Type != CommandHealthCheck {
		return nil, fmt.Errorf("invalid health check type for CommandHealthChecker: %s", config.Type)
	}

	// Validate and apply defaults
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid command configuration: %w", err)
	}

	// Create the base checker
	baseChecker, err := NewBaseChecker(CommandHealthCheck, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base checker: %w", err)
	}

	return &CommandHealthChecker{
		BaseChecker: baseChecker,
	}, nil
}

// Check performs a health check on the specified replica by executing a command
// and determining health based on the exit code.
func (c *CommandHealthChecker) Check(replica replica.Replica) (bool, error) {
	// Check if it's time to perform a health check
	if !c.ShouldCheck() {
		// Return the current status if it's not time to check again
		healthy, _, _ := c.GetStatus()
		return healthy, nil
	}

	result, err := c.CheckWithDetails(replica)
	return result.Healthy, err
}

// CheckWithDetails executes a command and returns detailed information.
func (c *CommandHealthChecker) CheckWithDetails(replica replica.Replica) (HealthCheckResult, error) {
	// Check if the container ID is valid
	if replica.ContainerID == "" {
		message := "Container ID is empty"
		c.UpdateStatus(false, message)
		return HealthCheckResult{
			Healthy:   false,
			Message:   message,
			Timestamp: time.Now(),
		}, fmt.Errorf(message)
	}

	// Check if a command was specified
	if c.Config.Command == "" {
		message := "No command specified for command health check"
		c.UpdateStatus(false, message)
		return HealthCheckResult{
			Healthy:   false,
			Message:   message,
			Timestamp: time.Now(),
		}, fmt.Errorf(message)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Timeout)
	defer cancel()

	// Format command to execute in docker container using 'docker exec'
	fullCommand := fmt.Sprintf("docker exec %s /bin/sh -c '%s'", replica.ContainerID, c.Config.Command)

	// Create the command
	cmd := exec.CommandContext(ctx, "sh", "-c", fullCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Determine health based on exit code (0 means healthy)
	healthy := err == nil
	var message string

	if healthy {
		message = fmt.Sprintf("Command '%s' executed successfully in container %s", c.Config.Command, replica.ContainerID)
		if stdout.Len() > 0 {
			// Add a summary of the output
			output := strings.TrimSpace(stdout.String())
			if len(output) > 100 {
				output = output[:97] + "..."
			}
			message += fmt.Sprintf(" (output: %s)", output)
		}
	} else {
		message = fmt.Sprintf("Command '%s' failed in container %s", c.Config.Command, replica.ContainerID)
		if stderr.Len() > 0 {
			// Add the error output
			errOutput := strings.TrimSpace(stderr.String())
			if len(errOutput) > 100 {
				errOutput = errOutput[:97] + "..."
			}
			message += fmt.Sprintf(" (error: %s)", errOutput)
		} else if ctx.Err() != nil {
			message += fmt.Sprintf(" (timeout after %v)", c.Config.Timeout)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			message += fmt.Sprintf(" (exit code: %d)", exitErr.ExitCode())
		} else {
			message += fmt.Sprintf(" (error: %v)", err)
		}
	}

	// Update the status
	c.UpdateStatus(healthy, message)

	// Return the health check result
	return HealthCheckResult{
		Healthy:   healthy,
		Message:   message,
		Timestamp: time.Now(),
	}, err
}

// Close is a no-op for the CommandHealthChecker
func (c *CommandHealthChecker) Close() error {
	return nil
}
