/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

// Common naming patterns for Docker Compose services that are replicas
var (
	// Matches patterns like: service-1, service-2, service-blue, service-green
	dashPattern = regexp.MustCompile(`^(.+?)[-_](\w+)$`)

	// Matches patterns with dot separators like: service.1, service.blue
	dotPattern = regexp.MustCompile(`^(.+?)\.(\w+)$`)
)

// NamedServiceDetector implements ReplicaDetector for services with naming patterns
type NamedServiceDetector struct {
	dockerClient *client.Client
}

// NewNamedServiceDetector creates a new detector for named service replicas
func NewNamedServiceDetector() (*NamedServiceDetector, error) {
	// Initialize the Docker client
	dockerClient, err := NewCompatibleDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &NamedServiceDetector{
		dockerClient: dockerClient,
	}, nil
}

// ServiceNameInfo holds information extracted from a service name
type ServiceNameInfo struct {
	BaseServiceName string // The base service name (e.g., "web" for "web-1")
	ReplicaID       string // The replica identifier (e.g., "1" for "web-1")
	FullServiceName string // The full service name from Docker Compose
}

// DetectReplicas finds replicas for services that follow naming patterns
func (d *NamedServiceDetector) DetectReplicas(composeFile string) (map[string][]Replica, error) {
	// Parse the Docker Compose file to find service groups with naming patterns
	serviceGroups, err := d.findNamedServiceGroups(composeFile)
	if err != nil {
		return nil, err
	}

	// Get container info for the services
	containersByService, err := d.getContainersByService(serviceGroups)
	if err != nil {
		return nil, err
	}

	// Convert to replicas
	replicas := make(map[string][]Replica)
	for baseServiceName, serviceInfos := range serviceGroups {
		serviceReplicas := make([]Replica, 0, len(serviceInfos))

		for _, serviceInfo := range serviceInfos {
			containers, exists := containersByService[serviceInfo.FullServiceName]
			if !exists || len(containers) == 0 {
				// Create a placeholder replica if no container is found
				replica := Replica{
					ServiceName: baseServiceName,
					ReplicaID:   serviceInfo.ReplicaID,
					ContainerID: "",
					Status:      "not_found",
				}
				serviceReplicas = append(serviceReplicas, replica)
				continue
			}

			// Use the first container for this service
			container := containers[0]

			// Inspect the container to get IP address and version
			ctx := context.Background()
			inspect, err := d.dockerClient.ContainerInspect(ctx, container.ID)
			ipAddress := ""
			if err == nil {
				// Try to get the first network's IP address
				for _, net := range inspect.NetworkSettings.Networks {
					ipAddress = net.IPAddress
					break
				}
			}

			// Get version from image tag if possible
			version := ""
			image := container.Image
			if idx := strings.Index(image, ":"); idx != -1 && idx+1 < len(image) {
				version = image[idx+1:]
			}

			// Get image and tag
			imageTag := ""
			if idx := strings.Index(image, ":"); idx != -1 && idx+1 < len(image) {
				imageTag = image[idx+1:]
			}

			// ServiceID as ServiceName-ReplicaID
			serviceID := baseServiceName + "-" + serviceInfo.ReplicaID

			// Parameters: use container labels if available
			params := map[string]interface{}{}
			for k, v := range container.Labels {
				params["label:"+k] = v
			}

			// envMap := parseEnvironment(service.Environment) // Uncomment and use if needed

			replica := Replica{
				ServiceName: baseServiceName,
				ReplicaID:   serviceInfo.ReplicaID,
				ContainerID: container.ID,
				Status:      container.State,
				Image:       image,
				ImageTag:    imageTag,
				ServiceID:   serviceID,
				Parameters:  params,
				IPAddress:   ipAddress,
				Version:     version,
			}
			serviceReplicas = append(serviceReplicas, replica)
		}

		replicas[baseServiceName] = serviceReplicas
	}

	return replicas, nil
}

// findNamedServiceGroups parses the Docker Compose file to identify services with naming patterns
func (d *NamedServiceDetector) findNamedServiceGroups(composeFile string) (map[string][]ServiceNameInfo, error) {
	// Read the Docker Compose file
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read Docker Compose file: %w", err)
	}

	// Parse the YAML
	var compose DockerComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse Docker Compose file: %w", err)
	}

	// Group services by base name using regex matching
	serviceGroups := make(map[string][]ServiceNameInfo)

	for serviceName := range compose.Services {
		// Try to match each pattern
		var baseServiceName, replicaID string
		var matched bool

		if matches := dashPattern.FindStringSubmatch(serviceName); len(matches) == 3 {
			baseServiceName, replicaID, matched = matches[1], matches[2], true
		} else if matches := dotPattern.FindStringSubmatch(serviceName); len(matches) == 3 {
			baseServiceName, replicaID, matched = matches[1], matches[2], true
		}

		if matched {
			serviceInfo := ServiceNameInfo{
				BaseServiceName: baseServiceName,
				ReplicaID:       replicaID,
				FullServiceName: serviceName,
			}

			// Add to the appropriate group
			if group, exists := serviceGroups[baseServiceName]; exists {
				serviceGroups[baseServiceName] = append(group, serviceInfo)
			} else {
				serviceGroups[baseServiceName] = []ServiceNameInfo{serviceInfo}
			}
		}
	}

	// Filter out single-service groups (not replicas)
	for baseServiceName, serviceInfos := range serviceGroups {
		if len(serviceInfos) < 2 {
			delete(serviceGroups, baseServiceName)
		}
	}

	return serviceGroups, nil
}

// getContainersByService retrieves all containers for each service
func (d *NamedServiceDetector) getContainersByService(serviceGroups map[string][]ServiceNameInfo) (map[string][]types.Container, error) {
	ctx := context.Background()
	containersByService := make(map[string][]types.Container)

	// Get all running containers
	containers, err := d.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true, // Include all containers (not just running ones)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Extract service names we're interested in
	serviceNames := make([]string, 0)
	for _, serviceInfos := range serviceGroups {
		for _, info := range serviceInfos {
			serviceNames = append(serviceNames, info.FullServiceName)
		}
	}

	// Group containers by service
	for _, serviceName := range serviceNames {
		serviceContainers := make([]types.Container, 0)
		for _, container := range containers {
			// Check if this container belongs to the service
			// Docker Compose names containers as: <project>_<service>_<number>
			for _, name := range container.Names {
				// Strip leading slash from container name
				name = strings.TrimPrefix(name, "/")
				nameParts := strings.Split(name, "_")
				if len(nameParts) >= 2 && nameParts[1] == serviceName {
					serviceContainers = append(serviceContainers, container)
					break
				}
			}
		}
		containersByService[serviceName] = serviceContainers
	}

	return containersByService, nil
}

// GetReplicaType returns the type of replica this detector handles
func (d *NamedServiceDetector) GetReplicaType() ReplicaType {
	return NameBased
}
