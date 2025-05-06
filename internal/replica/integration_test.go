/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"os"
	"testing"
)

// TestIntegrationFullWorkflow tests the complete workflow using both scale-based and named replica detection
func TestIntegrationFullWorkflow(t *testing.T) {
	// Skip this test in automated runs since it requires Docker
	t.Skip("Skipping integration test that requires Docker")

	// Create a test Docker Compose file with both scale-based and named replicas
	composeContent := `
version: '3'
services:
  web:
    image: nginx:latest
    scale: 2
  api-blue:
    image: node:latest
  api-green:
    image: node:latest
  db:
    image: postgres:latest
    deploy:
      replicas: 1
`
	composeFile := createTempComposeFile(t, composeContent)
	defer os.Remove(composeFile)

	// In a real integration test, we would run:
	// 1. docker-compose up -d
	// 2. wait for containers to start
	// 3. run our tests
	// 4. docker-compose down

	// Create a manager with all detectors
	manager, err := NewReplicaManagerWithAllDetectors(composeFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test GetAllReplicas
	allReplicas, err := manager.GetAllReplicas()
	if err != nil {
		t.Fatalf("GetAllReplicas failed: %v", err)
	}

	// Verify the results (in a real test, would need to match actual running containers)
	t.Logf("Found replicas: %+v", allReplicas)

	// Test GetServiceReplicas for scale-based replicas
	webReplicas, err := manager.GetServiceReplicas("web")
	if err != nil {
		t.Fatalf("GetServiceReplicas for 'web' failed: %v", err)
	}
	t.Logf("Web replicas: %+v", webReplicas)

	// Test GetServiceReplicas for name-based replicas
	apiReplicas, err := manager.GetServiceReplicas("api")
	if err != nil {
		t.Fatalf("GetServiceReplicas for 'api' failed: %v", err)
	}
	t.Logf("API replicas: %+v", apiReplicas)

	// Test RefreshReplicas
	err = manager.RefreshReplicas()
	if err != nil {
		t.Fatalf("RefreshReplicas failed: %v", err)
	}

	// Verify refreshed data
	refreshedReplicas, err := manager.GetAllReplicas()
	if err != nil {
		t.Fatalf("GetAllReplicas after refresh failed: %v", err)
	}
	t.Logf("Refreshed replicas: %+v", refreshedReplicas)
}

// TestConvenienceFunctionsWorkflow tests the convenience functions for the replica API
func TestConvenienceFunctionsWorkflow(t *testing.T) {
	// Skip this test in automated runs since it requires Docker
	t.Skip("Skipping integration test that requires Docker")

	// Create a test Docker Compose file
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
`
	composeFile := createTempComposeFile(t, composeContent)
	defer os.Remove(composeFile)

	// In a real integration test, we'd start Docker Compose services here

	// Test DetectAllReplicas
	allReplicas, err := DetectAllReplicas(composeFile)
	if err != nil {
		t.Fatalf("DetectAllReplicas failed: %v", err)
	}
	t.Logf("All detected replicas: %+v", allReplicas)

	// Test DetectServiceReplicas
	webReplicas, err := DetectServiceReplicas(composeFile, "web")
	if err != nil {
		t.Fatalf("DetectServiceReplicas for 'web' failed: %v", err)
	}
	t.Logf("Web replicas: %+v", webReplicas)

	dbReplicas, err := DetectServiceReplicas(composeFile, "db")
	if err != nil {
		t.Fatalf("DetectServiceReplicas for 'db' failed: %v", err)
	}
	t.Logf("DB replicas: %+v", dbReplicas)
}
