/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

// StubScaleBasedDetector provides a stub implementation for testing without Docker API dependencies
type StubScaleBasedDetector struct {
	// Services is a map of service names to their configuration
	Services map[string]DockerComposeService

	// Replicas is a predefined map of replicas to return for each service
	Replicas map[string][]Replica
}

// NewStubScaleBasedDetector creates a new stub detector
func NewStubScaleBasedDetector() *StubScaleBasedDetector {
	return &StubScaleBasedDetector{
		Services: make(map[string]DockerComposeService),
		Replicas: make(map[string][]Replica),
	}
}

// DetectReplicas implements the ReplicaDetector interface
func (d *StubScaleBasedDetector) DetectReplicas(composeFile string) (map[string][]Replica, error) {
	// Just return the predefined replicas
	return d.Replicas, nil
}

// GetReplicaType returns the type of replica this detector handles
func (d *StubScaleBasedDetector) GetReplicaType() ReplicaType {
	return ScaleBased
}

// parseComposeFile is a helper method to simulate parsing
func (d *StubScaleBasedDetector) parseComposeFile(composeFile string) (map[string]DockerComposeService, error) {
	// Just return the pre-configured services
	return d.Services, nil
}

// findScaledServices identifies services that have a scale property or deploy.replicas setting
func (d *StubScaleBasedDetector) findScaledServices(services map[string]DockerComposeService) map[string]int {
	// This mirrors the implementation in the real detector
	scaledServices := make(map[string]int)

	for serviceName, service := range services {
		// Check if the service has a scale property
		if service.Scale > 0 {
			scaledServices[serviceName] = service.Scale
			continue
		}

		// Check if the service has a deploy.replicas setting
		if service.Deploy.Replicas > 0 {
			scaledServices[serviceName] = service.Deploy.Replicas
		}
	}

	return scaledServices
}

// Note: When using this stub in tests, ensure that all fields of the Replica struct are populated as needed to match the real manager's behavior.
