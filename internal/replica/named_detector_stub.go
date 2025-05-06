/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

// StubNamedServiceDetector implements the ReplicaDetector interface for testing purposes.
// It provides a way to simulate named service detection without requiring Docker API connectivity.
// This is useful for unit tests, CI/CD pipelines, and environments without Docker access.
// Note: When using this stub in tests, ensure that all fields of the Replica struct are populated as needed to match the real manager's behavior.
type StubNamedServiceDetector struct {
	// ServiceGroups contains mappings of base service names to their service name info
	ServiceGroups map[string][]ServiceNameInfo

	// Replicas contains predefined replicas to return when DetectReplicas is called
	Replicas map[string][]Replica
}

// DetectReplicas implements the ReplicaDetector interface by returning the predefined replicas.
// This allows tests to control exactly what replica data is returned without hitting Docker API.
func (d *StubNamedServiceDetector) DetectReplicas(composeFile string) (map[string][]Replica, error) {
	return d.Replicas, nil
}

// GetReplicaType returns the type of replica this detector handles
func (d *StubNamedServiceDetector) GetReplicaType() ReplicaType {
	return NameBased
}

// NewStubNamedServiceDetector creates a new stub detector for testing with empty maps initialized.
// Use this factory function to ensure consistent initialization of the stub detector.
func NewStubNamedServiceDetector() *StubNamedServiceDetector {
	return &StubNamedServiceDetector{
		ServiceGroups: make(map[string][]ServiceNameInfo),
		Replicas:      make(map[string][]Replica),
	}
}
