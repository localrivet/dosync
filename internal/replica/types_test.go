/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"testing"
)

// TestReplicaStructure ensures that the Replica struct has the expected fields
func TestReplicaStructure(t *testing.T) {
	replica := Replica{
		ServiceName: "web",
		ReplicaID:   "1",
		ContainerID: "abc123",
		Status:      "running",
		Image:       "nginx:1.25.3",
		ImageTag:    "1.25.3",
		ServiceID:   "web-1",
		Parameters:  map[string]interface{}{"label:foo": "bar"},
		IPAddress:   "172.18.0.2",
		Version:     "1.2.3",
	}

	// Test field values
	if replica.ServiceName != "web" {
		t.Errorf("Expected ServiceName to be 'web', got '%s'", replica.ServiceName)
	}

	if replica.ReplicaID != "1" {
		t.Errorf("Expected ReplicaID to be '1', got '%s'", replica.ReplicaID)
	}

	if replica.ContainerID != "abc123" {
		t.Errorf("Expected ContainerID to be 'abc123', got '%s'", replica.ContainerID)
	}

	if replica.Status != "running" {
		t.Errorf("Expected Status to be 'running', got '%s'", replica.Status)
	}

	if replica.Image != "nginx:1.25.3" {
		t.Errorf("Expected Image to be 'nginx:1.25.3', got '%s'", replica.Image)
	}

	if replica.ImageTag != "1.25.3" {
		t.Errorf("Expected ImageTag to be '1.25.3', got '%s'", replica.ImageTag)
	}

	if replica.ServiceID != "web-1" {
		t.Errorf("Expected ServiceID to be 'web-1', got '%s'", replica.ServiceID)
	}

	if v, ok := replica.Parameters["label:foo"]; !ok || v != "bar" {
		t.Errorf("Expected Parameters to contain 'label:foo' with value 'bar', got '%v'", replica.Parameters)
	}

	if replica.IPAddress != "172.18.0.2" {
		t.Errorf("Expected IPAddress to be '172.18.0.2', got '%s'", replica.IPAddress)
	}

	if replica.Version != "1.2.3" {
		t.Errorf("Expected Version to be '1.2.3', got '%s'", replica.Version)
	}
}

// MockDetector implements the ReplicaDetector interface for testing
type MockDetector struct {
	Replicas     map[string][]Replica
	DetectorType ReplicaType
}

// DetectReplicas returns the pre-defined replicas for testing
func (m *MockDetector) DetectReplicas(composeFile string) (map[string][]Replica, error) {
	return m.Replicas, nil
}

// GetReplicaType returns the type of replica this detector handles
func (m *MockDetector) GetReplicaType() ReplicaType {
	// Default to ScaleBased if not specified
	if m.DetectorType == "" {
		return ScaleBased
	}
	return m.DetectorType
}

// TestReplicaDetectorInterface ensures that the ReplicaDetector interface can be implemented
func TestReplicaDetectorInterface(t *testing.T) {
	mockReplicas := map[string][]Replica{
		"web": {
			{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-1", Parameters: map[string]interface{}{"label:foo": "bar"}, IPAddress: "172.18.0.2", Version: "1.2.3"},
			{ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-2", Parameters: map[string]interface{}{"label:foo": "baz"}, IPAddress: "172.18.0.3", Version: "1.2.3"},
		},
	}
	detector := &MockDetector{
		Replicas:     mockReplicas,
		DetectorType: ScaleBased,
	}
	result, err := detector.DetectReplicas("dummy.yml")
	if err != nil {
		t.Errorf("DetectReplicas should not return an error, got: %v", err)
	}
	if len(result["web"]) != 2 {
		t.Errorf("Expected 2 web replicas, got %d", len(result["web"]))
	}
	// Check all fields for the first replica
	replica := result["web"][0]
	if replica.Image != "nginx:1.25.3" {
		t.Errorf("Expected Image to be 'nginx:1.25.3', got '%s'", replica.Image)
	}
	if replica.ImageTag != "1.25.3" {
		t.Errorf("Expected ImageTag to be '1.25.3', got '%s'", replica.ImageTag)
	}
	if replica.ServiceID != "web-1" {
		t.Errorf("Expected ServiceID to be 'web-1', got '%s'", replica.ServiceID)
	}
	if v, ok := replica.Parameters["label:foo"]; !ok || v != "bar" {
		t.Errorf("Expected Parameters to contain 'label:foo' with value 'bar', got '%v'", replica.Parameters)
	}
	if replica.IPAddress != "172.18.0.2" {
		t.Errorf("Expected IPAddress to be '172.18.0.2', got '%s'", replica.IPAddress)
	}
	if replica.Version != "1.2.3" {
		t.Errorf("Expected Version to be '1.2.3', got '%s'", replica.Version)
	}
}

// TestReplicaTypeConstants ensures that the ReplicaType constants are defined correctly
func TestReplicaTypeConstants(t *testing.T) {
	if ScaleBased != "scale" {
		t.Errorf("Expected ScaleBased to be 'scale', got '%s'", ScaleBased)
	}

	if NameBased != "name" {
		t.Errorf("Expected NameBased to be 'name', got '%s'", NameBased)
	}
}
