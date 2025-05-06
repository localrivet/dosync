/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"os"
	"testing"
)

// TestParseComposeFile verifies that Docker Compose file parsing works correctly
func TestFindScaledServices(t *testing.T) {
	// Create a temp compose file for testing
	composeContent := `
version: '3'
services:
  web:
    image: nginx:latest
    scale: 3
  db:
    image: postgres:latest
    deploy:
      replicas: 2
  redis:
    image: redis:latest
`
	tmpFile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpFile)

	// Create a detector
	detector := &ScaleBasedDetector{}

	// Test parsing the compose file
	scaledServices, err := detector.findScaledServices(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	// Verify the results
	if len(scaledServices) != 2 {
		t.Errorf("Expected 2 scaled services, got %d", len(scaledServices))
	}

	if scale, exists := scaledServices["web"]; !exists || scale != 3 {
		t.Errorf("Expected web service with scale 3, got %d", scale)
	}

	if scale, exists := scaledServices["db"]; !exists || scale != 2 {
		t.Errorf("Expected db service with scale 2, got %d", scale)
	}

	if _, exists := scaledServices["redis"]; exists {
		t.Error("Redis should not be detected as a scaled service")
	}
}

// Create a temporary Docker Compose file for testing
func createTempComposeFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

// TestStubScaleBasedDetector tests the stub implementation
func TestStubScaleBasedDetector(t *testing.T) {
	// Create a stub detector
	stub := NewStubScaleBasedDetector()

	// Configure with mock data
	stub.Services = map[string]DockerComposeService{
		"web": {
			Image: "nginx:latest",
			Scale: 3,
		},
		"db": {
			Image: "postgres:latest",
			Deploy: DeployConfig{
				Replicas: 2,
			},
		},
	}

	// Add mock replicas
	stub.Replicas = map[string][]Replica{
		"web": {
			{ServiceName: "web", ReplicaID: "1", ContainerID: "web1", Status: "running"},
			{ServiceName: "web", ReplicaID: "2", ContainerID: "web2", Status: "running"},
			{ServiceName: "web", ReplicaID: "3", ContainerID: "web3", Status: "running"},
		},
		"db": {
			{ServiceName: "db", ReplicaID: "1", ContainerID: "db1", Status: "running"},
			{ServiceName: "db", ReplicaID: "2", ContainerID: "db2", Status: "running"},
		},
	}

	// Test the detector
	replicas, err := stub.DetectReplicas("dummy-file.yml")
	if err != nil {
		t.Fatalf("Error detecting replicas: %v", err)
	}

	// Verify results
	if len(replicas) != 2 {
		t.Errorf("Expected 2 services with replicas, got %d", len(replicas))
	}

	if len(replicas["web"]) != 3 {
		t.Errorf("Expected 3 replicas for web service, got %d", len(replicas["web"]))
	}

	if len(replicas["db"]) != 2 {
		t.Errorf("Expected 2 replicas for db service, got %d", len(replicas["db"]))
	}
}

// TestDetectReplicasIntegration is an integration test for the full DetectReplicas workflow
func TestDetectReplicasIntegration(t *testing.T) {
	// Skip in automated tests, but can be run manually when Docker is available
	t.Skip("Skipping integration test that requires Docker")

	// This test requires:
	// 1. Docker daemon running
	// 2. Docker Compose file with scaled services
	// 3. Running containers that match the scaled services

	// Create a test Docker Compose file
	composeContent := `
version: '3'
services:
  web:
    image: nginx:latest
    scale: 2
  db:
    image: postgres:latest
    deploy:
      replicas: 1
`
	composeFile := createTempComposeFile(t, composeContent)
	defer os.Remove(composeFile)

	// In a real test, we would:
	// 1. Run `docker-compose up -d --scale web=2` to start services
	// 2. Wait for containers to start
	// 3. Run our detector
	// 4. Verify the results
	// 5. Clean up with `docker-compose down`

	// Create a real detector
	detector, err := NewScaleBasedDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	// Detect replicas
	replicas, err := detector.DetectReplicas(composeFile)
	if err != nil {
		t.Fatalf("Failed to detect replicas: %v", err)
	}

	// Verify results - would need to match the actual containers started by Docker Compose
	t.Logf("Found replicas: %+v", replicas)

	// In a real test, we would verify:
	// - That we found the right number of replicas for each service
	// - That container IDs match actual running containers
	// - That status values are correct
}
