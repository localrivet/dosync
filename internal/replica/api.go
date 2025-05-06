/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"fmt"
)

// NewReplicaManagerWithAllDetectors creates a new ReplicaManager with all available detector types registered.
// This is a convenience function that creates a ReplicaManager and initializes it with both
// scale-based and name-based replica detectors.
func NewReplicaManagerWithAllDetectors(composeFile string) (*ReplicaManager, error) {
	// Create the base ReplicaManager
	manager, err := NewReplicaManager(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReplicaManager: %w", err)
	}

	// Create and register the scale-based detector
	scaleDetector, err := NewScaleBasedDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to create scale-based detector: %w", err)
	}
	manager.RegisterDetector(ScaleBased, scaleDetector)

	// Create and register the name-based detector
	nameDetector, err := NewNamedServiceDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to create name-based detector: %w", err)
	}
	manager.RegisterDetector(NameBased, nameDetector)

	return manager, nil
}

// CreateStubReplicaManager creates a ReplicaManager with stub implementations for testing
func CreateStubReplicaManager(composeFile string) (*ReplicaManager, error) {
	// Create the base ReplicaManager
	manager, err := NewReplicaManager(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReplicaManager: %w", err)
	}

	// Create and register the stub detectors
	scaleDetector := NewStubScaleBasedDetector()
	manager.RegisterDetector(ScaleBased, scaleDetector)

	nameDetector := NewStubNamedServiceDetector()
	manager.RegisterDetector(NameBased, nameDetector)

	return manager, nil
}

// DetectServiceReplicas is a convenience function that creates a manager and immediately detects replicas
// for a specific service
func DetectServiceReplicas(composeFile string, serviceName string) ([]Replica, error) {
	manager, err := NewReplicaManagerWithAllDetectors(composeFile)
	if err != nil {
		return nil, err
	}

	return manager.GetServiceReplicas(serviceName)
}

// DetectAllReplicas is a convenience function that creates a manager and immediately detects all replicas
func DetectAllReplicas(composeFile string) (map[string][]Replica, error) {
	manager, err := NewReplicaManagerWithAllDetectors(composeFile)
	if err != nil {
		return nil, err
	}

	return manager.GetAllReplicas()
}

// GetDetectorByType is a convenience function to access a specific detector type from a compose file
// This can be useful for configuring detector-specific settings
func GetDetectorByType(composeFile string, detectorType ReplicaType) (ReplicaDetector, error) {
	manager, err := NewReplicaManagerWithAllDetectors(composeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	detector := manager.GetDetector(detectorType)
	if detector == nil {
		return nil, fmt.Errorf("no detector registered for type: %s", detectorType)
	}

	return detector, nil
}
