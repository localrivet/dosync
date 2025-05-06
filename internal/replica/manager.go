/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"fmt"
)

// ReplicaManager provides a unified API for handling service replicas
// It manages both scale-based and name-based replicas
type ReplicaManager struct {
	// detectors is a map of replica types to their corresponding detector implementations
	detectors map[ReplicaType]ReplicaDetector

	// replicas is a cache of detected service replicas
	replicas map[string][]Replica

	// composeFile is the path to the Docker Compose file
	composeFile string
}

// NewReplicaManager creates a new ReplicaManager for the specified Docker Compose file
func NewReplicaManager(composeFile string) (*ReplicaManager, error) {
	// Create a new ReplicaManager with the given compose file
	manager := &ReplicaManager{
		detectors:   make(map[ReplicaType]ReplicaDetector),
		replicas:    make(map[string][]Replica),
		composeFile: composeFile,
	}

	// At this stage, we're just initializing the manager.
	// The actual detector implementations will be registered later.
	return manager, nil
}

// RegisterDetector registers a replica detector for a specific replica type
func (rm *ReplicaManager) RegisterDetector(replicaType ReplicaType, detector ReplicaDetector) {
	rm.detectors[replicaType] = detector
}

// HasDetector checks if a detector for the specified replica type is registered
func (rm *ReplicaManager) HasDetector(replicaType ReplicaType) bool {
	_, exists := rm.detectors[replicaType]
	return exists
}

// GetDetector returns the detector registered for the specified replica type
// Returns the detector if found, or nil if no detector is registered for that type
func (rm *ReplicaManager) GetDetector(replicaType ReplicaType) ReplicaDetector {
	if detector, exists := rm.detectors[replicaType]; exists {
		return detector
	}
	return nil
}

// UnregisterDetector removes a detector for the specified replica type
// Returns true if a detector was removed, false if no detector was registered for that type
func (rm *ReplicaManager) UnregisterDetector(replicaType ReplicaType) bool {
	if _, exists := rm.detectors[replicaType]; exists {
		delete(rm.detectors, replicaType)
		return true
	}
	return false
}

// GetServiceReplicas returns all replicas for a specific service
func (rm *ReplicaManager) GetServiceReplicas(serviceName string) ([]Replica, error) {
	// Check if we have replicas for this service in the cache
	if replicas, ok := rm.replicas[serviceName]; ok {
		return replicas, nil
	}

	// If not, try to detect replicas from all registered detectors
	if err := rm.detectReplicas(); err != nil {
		return nil, fmt.Errorf("failed to detect replicas: %w", err)
	}

	// Now check if we have the service after detection
	if replicas, ok := rm.replicas[serviceName]; ok {
		return replicas, nil
	}

	// Return an empty slice if no replicas found for this service
	return []Replica{}, nil
}

// GetAllReplicas returns all detected replicas across all services
func (rm *ReplicaManager) GetAllReplicas() (map[string][]Replica, error) {
	// If the cache is empty, try to detect replicas
	if len(rm.replicas) == 0 {
		if err := rm.detectReplicas(); err != nil {
			return nil, fmt.Errorf("failed to detect replicas: %w", err)
		}
	}

	return rm.replicas, nil
}

// RefreshReplicas forces a refresh of the replica cache
func (rm *ReplicaManager) RefreshReplicas() error {
	return rm.detectReplicas()
}

// detectReplicas uses all registered detectors to find service replicas
func (rm *ReplicaManager) detectReplicas() error {
	// Create a new map to store the results
	allReplicas := make(map[string][]Replica)

	// Check if we have any detectors registered
	if len(rm.detectors) == 0 {
		return fmt.Errorf("no replica detectors registered")
	}

	// Use each detector to find replicas
	for _, detector := range rm.detectors {
		replicas, err := detector.DetectReplicas(rm.composeFile)
		if err != nil {
			return fmt.Errorf("detector failed: %w", err)
		}

		// Merge the results
		for service, serviceReplicas := range replicas {
			// Append the replicas to any existing ones for this service
			allReplicas[service] = append(allReplicas[service], serviceReplicas...)
		}
	}

	// Update the cache with the new results
	rm.replicas = allReplicas

	return nil
}

// UpdateReplica updates the given replica to the specified new image tag
func (rm *ReplicaManager) UpdateReplica(r *Replica, newImageTag string) error {
	// Call UpdateDockerComposeAndRestart directly (same package)
	return UpdateDockerComposeAndRestart(r.ServiceName, newImageTag, rm.composeFile, false)
}

// RollbackReplica rolls back the given replica to the previous image/tag
func (rm *ReplicaManager) RollbackReplica(r *Replica) error {
	// Rollback logic must be handled by the orchestrator (e.g., rolling update controller)
	return fmt.Errorf("RollbackReplica is not implemented in ReplicaManager; handle rollback in the orchestrator layer")
}
