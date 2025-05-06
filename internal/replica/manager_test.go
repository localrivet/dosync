/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"testing"
)

// TestNewReplicaManager tests that a new ReplicaManager can be created
func TestNewReplicaManager(t *testing.T) {
	manager, err := NewReplicaManager("compose.yml")
	if err != nil {
		t.Errorf("NewReplicaManager should not return an error, got: %v", err)
	}

	if manager == nil {
		t.Fatal("NewReplicaManager should return a non-nil manager")
	}

	if manager.composeFile != "compose.yml" {
		t.Errorf("Expected composeFile to be 'compose.yml', got '%s'", manager.composeFile)
	}

	if manager.detectors == nil {
		t.Error("Expected detectors map to be initialized")
	}

	if manager.replicas == nil {
		t.Error("Expected replicas map to be initialized")
	}
}

// TestRegisterDetector tests that detectors can be registered with the manager
func TestRegisterDetector(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Register a mock detector
	mockDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"web": {
				{ServiceName: "web", ReplicaID: "1", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-1", Parameters: map[string]interface{}{"label:foo": "bar"}, IPAddress: "172.18.0.2", Version: "1.2.3"},
			},
		},
		DetectorType: ScaleBased,
	}

	manager.RegisterDetector(ScaleBased, mockDetector)

	// Check that the detector was registered
	if len(manager.detectors) != 1 {
		t.Errorf("Expected 1 detector to be registered, got %d", len(manager.detectors))
	}

	if manager.detectors[ScaleBased] != mockDetector {
		t.Error("Expected the registered detector to be the mock detector")
	}
}

// TestGetServiceReplicas tests retrieving replicas for a specific service
func TestGetServiceReplicas(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Register a mock detector
	mockDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"web": {
				{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-1", Parameters: map[string]interface{}{"label:foo": "bar"}, IPAddress: "172.18.0.2", Version: "1.2.3"},
				{ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-2", Parameters: map[string]interface{}{"label:foo": "baz"}, IPAddress: "172.18.0.3", Version: "1.2.3"},
			},
			"db": {
				{ServiceName: "db", ReplicaID: "1", ContainerID: "container3", Status: "running", Image: "postgres:latest", ImageTag: "latest", ServiceID: "db-1", Parameters: map[string]interface{}{"label:foo": "db"}, IPAddress: "172.18.0.4", Version: "15.0"},
			},
		},
		DetectorType: ScaleBased,
	}

	manager.RegisterDetector(ScaleBased, mockDetector)

	// Test getting replicas for a service
	webReplicas, err := manager.GetServiceReplicas("web")
	if err != nil {
		t.Errorf("GetServiceReplicas should not return an error, got: %v", err)
	}

	if len(webReplicas) != 2 {
		t.Errorf("Expected 2 web replicas, got %d", len(webReplicas))
	}
	if webReplicas[0].Image != "nginx:1.25.3" || webReplicas[0].IPAddress != "172.18.0.2" {
		t.Errorf("Expected web replica 1 to have correct Image and IPAddress")
	}

	// Test getting replicas for a different service
	dbReplicas, err := manager.GetServiceReplicas("db")
	if err != nil {
		t.Errorf("GetServiceReplicas should not return an error, got: %v", err)
	}

	if len(dbReplicas) != 1 {
		t.Errorf("Expected 1 db replica, got %d", len(dbReplicas))
	}
	if dbReplicas[0].Image != "postgres:latest" || dbReplicas[0].IPAddress != "172.18.0.4" {
		t.Errorf("Expected db replica to have correct Image and IPAddress")
	}

	// Test getting replicas for a non-existent service
	nonExistentReplicas, err := manager.GetServiceReplicas("nonexistent")
	if err != nil {
		t.Errorf("GetServiceReplicas should not return an error for non-existent services, got: %v", err)
	}

	if len(nonExistentReplicas) != 0 {
		t.Errorf("Expected 0 nonexistent replicas, got %d", len(nonExistentReplicas))
	}
}

// TestGetAllReplicas tests retrieving all replicas
func TestGetAllReplicas(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Register a mock detector
	mockDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"web": {
				{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-1", Parameters: map[string]interface{}{"label:foo": "bar"}, IPAddress: "172.18.0.2", Version: "1.2.3"},
				{ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-2", Parameters: map[string]interface{}{"label:foo": "baz"}, IPAddress: "172.18.0.3", Version: "1.2.3"},
			},
			"db": {
				{ServiceName: "db", ReplicaID: "1", ContainerID: "container3", Status: "running", Image: "postgres:latest", ImageTag: "latest", ServiceID: "db-1", Parameters: map[string]interface{}{"label:foo": "db"}, IPAddress: "172.18.0.4", Version: "15.0"},
			},
		},
		DetectorType: ScaleBased,
	}

	manager.RegisterDetector(ScaleBased, mockDetector)

	// Test getting all replicas
	allReplicas, err := manager.GetAllReplicas()
	if err != nil {
		t.Errorf("GetAllReplicas should not return an error, got: %v", err)
	}

	if len(allReplicas) != 2 {
		t.Errorf("Expected 2 services with replicas, got %d", len(allReplicas))
	}

	if len(allReplicas["web"]) != 2 {
		t.Errorf("Expected 2 web replicas, got %d", len(allReplicas["web"]))
	}
	if allReplicas["web"][0].Image != "nginx:1.25.3" || allReplicas["web"][0].IPAddress != "172.18.0.2" {
		t.Errorf("Expected web replica 1 to have correct Image and IPAddress")
	}

	if len(allReplicas["db"]) != 1 {
		t.Errorf("Expected 1 db replica, got %d", len(allReplicas["db"]))
	}
	if allReplicas["db"][0].Image != "postgres:latest" || allReplicas["db"][0].IPAddress != "172.18.0.4" {
		t.Errorf("Expected db replica to have correct Image and IPAddress")
	}
}

// TestMultipleDetectors tests using multiple detectors
func TestMultipleDetectors(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Register a scale-based detector
	scaleDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"web": {
				{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-1", Parameters: map[string]interface{}{"label:foo": "bar"}, IPAddress: "172.18.0.2", Version: "1.2.3"},
				{ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running", Image: "nginx:1.25.3", ImageTag: "1.25.3", ServiceID: "web-2", Parameters: map[string]interface{}{"label:foo": "baz"}, IPAddress: "172.18.0.3", Version: "1.2.3"},
			},
		},
		DetectorType: ScaleBased,
	}

	// Register a name-based detector
	nameDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"db": {
				{ServiceName: "db", ReplicaID: "blue", ContainerID: "container3", Status: "running", Image: "postgres:latest", ImageTag: "latest", ServiceID: "db-blue", Parameters: map[string]interface{}{"label:foo": "db"}, IPAddress: "172.18.0.5", Version: "15.0"},
				{ServiceName: "db", ReplicaID: "green", ContainerID: "container4", Status: "running", Image: "postgres:latest", ImageTag: "latest", ServiceID: "db-green", Parameters: map[string]interface{}{"label:foo": "db"}, IPAddress: "172.18.0.6", Version: "15.0"},
			},
		},
		DetectorType: NameBased,
	}

	manager.RegisterDetector(ScaleBased, scaleDetector)
	manager.RegisterDetector(NameBased, nameDetector)

	// Test getting all replicas
	allReplicas, err := manager.GetAllReplicas()
	if err != nil {
		t.Errorf("GetAllReplicas should not return an error, got: %v", err)
	}

	if len(allReplicas) != 2 {
		t.Errorf("Expected 2 services with replicas, got %d", len(allReplicas))
	}

	if len(allReplicas["web"]) != 2 {
		t.Errorf("Expected 2 web replicas, got %d", len(allReplicas["web"]))
	}
	if allReplicas["web"][0].Image != "nginx:1.25.3" || allReplicas["web"][0].IPAddress != "172.18.0.2" {
		t.Errorf("Expected web replica 1 to have correct Image and IPAddress")
	}

	if len(allReplicas["db"]) != 2 {
		t.Errorf("Expected 2 db replicas, got %d", len(allReplicas["db"]))
	}
	if allReplicas["db"][0].Image != "postgres:latest" || allReplicas["db"][0].IPAddress != "172.18.0.5" {
		t.Errorf("Expected db replica 1 to have correct Image and IPAddress")
	}
}

// TestNoDetectorsRegistered tests error handling when no detectors are registered
func TestNoDetectorsRegistered(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Try to get replicas without registering any detectors
	_, err := manager.GetAllReplicas()
	if err == nil {
		t.Error("GetAllReplicas should return an error when no detectors are registered")
	}
}

// TestRefreshReplicas tests that the replica cache can be refreshed
func TestRefreshReplicas(t *testing.T) {
	manager, _ := NewReplicaManager("compose.yml")

	// Register a detector with initial replicas
	mockDetector := &MockDetector{
		Replicas: map[string][]Replica{
			"web": {
				{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running"},
			},
		},
		DetectorType: ScaleBased,
	}

	manager.RegisterDetector(ScaleBased, mockDetector)

	// Get replicas to populate the cache
	initialReplicas, _ := manager.GetAllReplicas()
	if len(initialReplicas["web"]) != 1 {
		t.Errorf("Expected 1 web replica initially, got %d", len(initialReplicas["web"]))
	}

	// Change the mock detector's data
	mockDetector.Replicas = map[string][]Replica{
		"web": {
			{ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running"},
			{ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running"},
		},
	}

	// Refresh the cache
	err := manager.RefreshReplicas()
	if err != nil {
		t.Errorf("RefreshReplicas should not return an error, got: %v", err)
	}

	// Get the updated replicas
	updatedReplicas, _ := manager.GetAllReplicas()
	if len(updatedReplicas["web"]) != 2 {
		t.Errorf("Expected 2 web replicas after refresh, got %d", len(updatedReplicas["web"]))
	}
}
