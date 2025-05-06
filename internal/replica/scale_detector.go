/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

// ScaleBasedDetector implements ReplicaDetector for services using scale property
type ScaleBasedDetector struct {
	dockerClient *client.Client
}

// NewScaleBasedDetector creates a new detector for scale-based replicas
func NewScaleBasedDetector() (*ScaleBasedDetector, error) {
	// Initialize the Docker client
	dockerClient, err := NewCompatibleDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &ScaleBasedDetector{
		dockerClient: dockerClient,
	}, nil
}

// DetectReplicas finds replicas for services that use the scale property
func (d *ScaleBasedDetector) DetectReplicas(composeFile string) (map[string][]Replica, error) {
	// Parse the Docker Compose file
	scaledServices, err := d.findScaledServices(composeFile)
	if err != nil {
		return nil, err
	}

	// Get the containers for each scaled service
	containersByService, err := d.getContainersByService(scaledServices)
	if err != nil {
		return nil, err
	}

	// Convert containers to replicas
	replicas := make(map[string][]Replica)
	for serviceName, containers := range containersByService {
		serviceReplicas := make([]Replica, 0, len(containers))
		for _, container := range containers {
			// Extract replica ID from container name
			// Docker Compose names containers as: <project>_<service>_<replica_number>
			nameParts := strings.Split(container.Names[0], "_")
			replicaID := ""
			if len(nameParts) >= 3 {
				replicaID = nameParts[len(nameParts)-1]
			}

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
			serviceID := serviceName + "-" + replicaID

			// Parameters: use container labels and env if available
			params := map[string]interface{}{}
			for k, v := range container.Labels {
				params["label:"+k] = v
			}
			// Example: get environment variables for this service from compose file
			// envMap := parseEnvironment(service.Environment) // Uncomment and use if needed

			replica := Replica{
				ServiceName: serviceName,
				ReplicaID:   replicaID,
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
		replicas[serviceName] = serviceReplicas
	}

	return replicas, nil
}

// findScaledServices parses the Docker Compose file to find services with scale property
func (d *ScaleBasedDetector) findScaledServices(composeFile string) (map[string]int, error) {
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

	// Find services with scale property
	scaledServices := make(map[string]int)
	for name, service := range compose.Services {
		if service.Scale > 0 {
			scaledServices[name] = service.Scale
		} else if service.Deploy.Replicas > 0 {
			scaledServices[name] = service.Deploy.Replicas
		}
	}

	return scaledServices, nil
}

// getContainersByService retrieves all containers that belong to scaled services
func (d *ScaleBasedDetector) getContainersByService(scaledServices map[string]int) (map[string][]types.Container, error) {
	ctx := context.Background()
	containersByService := make(map[string][]types.Container)

	// Get all running containers
	containers, err := d.dockerClient.ContainerList(ctx, container.ListOptions{
		All: true, // Include all containers (not just running ones)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Group containers by service
	for serviceName := range scaledServices {
		serviceContainers := make([]types.Container, 0)
		for _, container := range containers {
			// Check if this container belongs to the service
			// Docker Compose names containers as: <project>_<service>_<replica_number>
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
func (d *ScaleBasedDetector) GetReplicaType() ReplicaType {
	return ScaleBased
}
