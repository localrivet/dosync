/*
Copyright © 2025 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DependsOnCondition represents the condition for a dependency
type DependsOnCondition struct {
	Condition string `yaml:"condition"`
}

// ServiceWithDeps represents a service with its dependencies
type ServiceWithDeps struct {
	Image     string                        `yaml:"image"`
	DependsOn map[string]DependsOnCondition `yaml:"depends_on"`
}

// ComposeWithDeps represents a compose file with dependency information
type ComposeWithDeps struct {
	Services map[string]ServiceWithDeps `yaml:"services"`
}

// ContainerHealthState represents the health state from docker inspect
type ContainerHealthState struct {
	Status string `json:"Status"`
}

// ContainerState represents the state from docker inspect
type ContainerState struct {
	Status  string                `json:"Status"`
	Running bool                  `json:"Running"`
	Health  *ContainerHealthState `json:"Health,omitempty"`
}

// ContainerInspect represents the docker inspect output
type ContainerInspect struct {
	State ContainerState `json:"State"`
}

// DependencyHealthConfig configures dependency health polling
type DependencyHealthConfig struct {
	Timeout       time.Duration // Maximum time to wait for dependencies
	PollInterval  time.Duration // How often to check
	Verbose       bool          // Enable verbose logging
	ProjectName   string        // Docker Compose project name for container naming
}

// DefaultDependencyHealthConfig returns sensible defaults
func DefaultDependencyHealthConfig() DependencyHealthConfig {
	return DependencyHealthConfig{
		Timeout:      120 * time.Second, // 2 minutes max wait
		PollInterval: 2 * time.Second,   // Check every 2 seconds
		Verbose:      false,
	}
}

// GetServiceDependencies parses the compose file and returns dependencies
// that require health checks (condition: service_healthy)
func GetServiceDependencies(composeContent []byte, serviceName string) ([]string, error) {
	var compose ComposeWithDeps
	if err := yaml.Unmarshal(composeContent, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	service, exists := compose.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found in compose file", serviceName)
	}

	var healthDeps []string
	for depName, depConfig := range service.DependsOn {
		// Only include dependencies with condition: service_healthy
		if depConfig.Condition == "service_healthy" {
			healthDeps = append(healthDeps, depName)
		}
	}

	return healthDeps, nil
}

// WaitForDependencies waits for all dependencies to be healthy before proceeding.
// This is called BEFORE docker compose up --no-deps to ensure dependencies are ready.
// Returns nil if all dependencies are healthy, or an error if timeout/failure.
func WaitForDependencies(composeContent []byte, serviceName string, config DependencyHealthConfig) error {
	deps, err := GetServiceDependencies(composeContent, serviceName)
	if err != nil {
		// If we can't parse deps, log warning but don't block (backwards compatibility)
		if config.Verbose {
			fmt.Printf("Warning: could not parse dependencies for %s: %v\n", serviceName, err)
		}
		return nil
	}

	if len(deps) == 0 {
		if config.Verbose {
			fmt.Printf("Service %s has no health-check dependencies, proceeding immediately\n", serviceName)
		}
		return nil
	}

	if config.Verbose {
		fmt.Printf("Service %s depends on %v with health checks, waiting for them to be healthy\n", serviceName, deps)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for dependencies %v to be healthy", deps)
		case <-ticker.C:
			allHealthy := true
			for _, dep := range deps {
				healthy, status, err := checkContainerHealth(dep, config.ProjectName)
				if err != nil {
					if config.Verbose {
						fmt.Printf("  Dependency %s: error checking health: %v\n", dep, err)
					}
					allHealthy = false
					continue
				}
				if !healthy {
					if config.Verbose {
						fmt.Printf("  Dependency %s: not healthy (status: %s)\n", dep, status)
					}
					allHealthy = false
				} else if config.Verbose {
					fmt.Printf("  Dependency %s: healthy\n", dep)
				}
			}
			if allHealthy {
				if config.Verbose {
					fmt.Printf("All dependencies for %s are healthy, proceeding with update\n", serviceName)
				}
				return nil
			}
		}
	}
}

// checkContainerHealth checks if a container is healthy using docker inspect
// Returns (healthy, status, error)
func checkContainerHealth(serviceName string, projectName string) (bool, string, error) {
	// Try to find the container by various naming conventions
	containerNames := []string{
		fmt.Sprintf("%s_%s", projectName, serviceName),      // project_service
		fmt.Sprintf("%s-%s", projectName, serviceName),      // project-service
		fmt.Sprintf("%s_%s_1", projectName, serviceName),    // project_service_1 (old docker-compose)
		fmt.Sprintf("%s-%s-1", projectName, serviceName),    // project-service-1 (new docker compose)
		serviceName,                                          // just the service name
	}

	var lastErr error
	for _, containerName := range containerNames {
		healthy, status, err := inspectContainerHealth(containerName)
		if err == nil {
			return healthy, status, nil
		}
		lastErr = err
	}

	return false, "", fmt.Errorf("could not find container for service %s (tried: %v): %w",
		serviceName, containerNames, lastErr)
}

// inspectContainerHealth runs docker inspect and checks health status
func inspectContainerHealth(containerName string) (bool, string, error) {
	cmd := exec.Command("docker", "inspect", containerName)
	output, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("docker inspect failed: %w", err)
	}

	var containers []ContainerInspect
	if err := json.Unmarshal(output, &containers); err != nil {
		return false, "", fmt.Errorf("failed to parse docker inspect output: %w", err)
	}

	if len(containers) == 0 {
		return false, "", fmt.Errorf("no container found with name %s", containerName)
	}

	container := containers[0]

	// Check if container is running
	if !container.State.Running {
		return false, "not running", nil
	}

	// Check if container has health check
	if container.State.Health == nil {
		// No health check configured - consider it healthy if running
		return true, "running (no healthcheck)", nil
	}

	// Check health status
	status := strings.ToLower(container.State.Health.Status)
	switch status {
	case "healthy":
		return true, "healthy", nil
	case "starting":
		return false, "starting", nil
	case "unhealthy":
		return false, "unhealthy", nil
	default:
		return false, status, nil
	}
}
