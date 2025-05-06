/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"strings"
)

// Replica represents a single instance of a service in a Docker Compose environment
type Replica struct {
	// ServiceName is the base service name (without replica identifier)
	ServiceName string

	// ReplicaID is the unique identifier for this replica
	// For scale-based replicas, this would be a number (e.g., "1", "2")
	// For named replicas, this would be the distinguishing part (e.g., "blue", "green")
	ReplicaID string

	// ContainerID is the Docker container ID for this replica
	ContainerID string

	// Status represents the current state of this replica (e.g., "running", "starting", "stopped")
	Status string

	// ServiceID is a unique identifier for the service-replica combination
	ServiceID string

	// Image is the full image name for this replica
	Image string

	// ImageTag is the specific tag of the image used by this replica
	ImageTag string

	// IPAddress is the IP address assigned to this replica (added for task 10.6)
	IPAddress string

	// Version is the version string for this replica (added for task 10.6)
	Version string

	// Parameters contains additional configuration parameters for this replica
	Parameters map[string]interface{}
}

// ReplicaDetector defines the interface for detecting service replicas in Docker Compose environments
type ReplicaDetector interface {
	// DetectReplicas analyzes a Docker Compose file and identifies all service replicas
	// It returns a map of service names to their replicas
	DetectReplicas(composeFile string) (map[string][]Replica, error)

	// GetReplicaType returns the type of replica this detector handles
	GetReplicaType() ReplicaType
}

// ReplicaType represents the kind of replica detection strategy to use
type ReplicaType string

const (
	// ScaleBased represents replicas created using Docker Compose scale property
	ScaleBased ReplicaType = "scale"

	// NameBased represents replicas identified by naming patterns
	NameBased ReplicaType = "name"
)

// DockerComposeFile represents the structure of a Docker Compose file
// Shared by all detectors
// Only define once here

// If not present, add:
type DockerComposeFile struct {
	Version  string                          `yaml:"version"`
	Services map[string]DockerComposeService `yaml:"services"`
}

type DockerComposeService struct {
	Image       string       `yaml:"image" mapstructure:"image"`
	Scale       int          `yaml:"scale,omitempty" mapstructure:"scale,omitempty"`
	Deploy      DeployConfig `yaml:"deploy,omitempty" mapstructure:"deploy,omitempty"`
	Environment interface{}  `yaml:"environment,omitempty" mapstructure:"environment,omitempty"`
}

type DeployConfig struct {
	Replicas int `yaml:"replicas,omitempty"`
}

// Helper to normalize environment to map[string]string
func parseEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}
	switch v := env.(type) {
	case map[interface{}]interface{}:
		for key, value := range v {
			k, ok1 := key.(string)
			val, ok2 := value.(string)
			if ok1 && ok2 {
				result[k] = val
			}
		}
	case map[string]interface{}:
		for k, val := range v {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
	case map[string]string:
		return v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					result[parts[0]] = parts[1]
				}
			}
		}
	}
	return result
}
