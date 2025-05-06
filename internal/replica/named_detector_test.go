/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"os"
	"regexp"
	"testing"
)

// TestFindNamedServiceGroups verifies that the method can identify services with naming patterns
func TestFindNamedServiceGroups(t *testing.T) {
	// Create a test Docker Compose file with named service patterns
	composeContent := `
version: '3'
services:
  web-1:
    image: nginx:latest
  web-2:
    image: nginx:latest
  web-blue:
    image: nginx:latest
  api-1: 
    image: node:latest
  api-2:
    image: node:latest
  db.1:
    image: postgres:latest
  db.2:
    image: postgres:latest
  redis:
    image: redis:latest
`
	tmpFile := createTempComposeFile(t, composeContent)
	defer os.Remove(tmpFile)

	detector := &NamedServiceDetector{}

	// Test finding the service groups
	serviceGroups, err := detector.findNamedServiceGroups(tmpFile)
	if err != nil {
		t.Fatalf("Failed to find named service groups: %v", err)
	}

	// Verify results
	if len(serviceGroups) != 3 {
		t.Errorf("Expected 3 service groups (web, api, db), got %d", len(serviceGroups))
	}

	// Check web group
	if webGroup, exists := serviceGroups["web"]; !exists {
		t.Errorf("Expected to find 'web' service group")
	} else {
		if len(webGroup) != 3 {
			t.Errorf("Expected 3 replicas in web group, got %d", len(webGroup))
		}

		// Verify that we detected web-1, web-2, and web-blue
		replicaIDs := map[string]bool{}
		for _, info := range webGroup {
			replicaIDs[info.ReplicaID] = true
			if info.BaseServiceName != "web" {
				t.Errorf("Expected base service name 'web', got '%s'", info.BaseServiceName)
			}
		}

		expectedReplicaIDs := []string{"1", "2", "blue"}
		for _, id := range expectedReplicaIDs {
			if !replicaIDs[id] {
				t.Errorf("Expected to find replica ID '%s' in web group", id)
			}
		}
	}

	// Check api group
	if apiGroup, exists := serviceGroups["api"]; !exists {
		t.Errorf("Expected to find 'api' service group")
	} else {
		if len(apiGroup) != 2 {
			t.Errorf("Expected 2 replicas in api group, got %d", len(apiGroup))
		}
	}

	// Check db group
	if dbGroup, exists := serviceGroups["db"]; !exists {
		t.Errorf("Expected to find 'db' service group")
	} else {
		if len(dbGroup) != 2 {
			t.Errorf("Expected 2 replicas in db group, got %d", len(dbGroup))
		}
	}

	// Verify that single service (redis) was not included
	if _, exists := serviceGroups["redis"]; exists {
		t.Errorf("Did not expect to find 'redis' as a service group (not a replica)")
	}
}

// TestNamedServiceRegexPatterns tests that the regex patterns work correctly
func TestNamedServiceRegexPatterns(t *testing.T) {
	// Test dash pattern
	testCases := []struct {
		serviceName     string
		pattern         *regexp.Regexp
		expectMatch     bool
		baseServiceName string
		replicaID       string
	}{
		{"web-1", dashPattern, true, "web", "1"},
		{"web-blue", dashPattern, true, "web", "blue"},
		{"api_v1", dashPattern, true, "api", "v1"},
		{"db.1", dotPattern, true, "db", "1"},
		{"auth.service", dotPattern, true, "auth", "service"},
		{"redis", dashPattern, false, "", ""},
		{"redis", dotPattern, false, "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.serviceName, func(t *testing.T) {
			matches := tc.pattern.FindStringSubmatch(tc.serviceName)

			if tc.expectMatch {
				if len(matches) != 3 {
					t.Errorf("Expected pattern to match '%s', but it didn't", tc.serviceName)
					return
				}

				if matches[1] != tc.baseServiceName {
					t.Errorf("Expected base service name '%s', got '%s'", tc.baseServiceName, matches[1])
				}

				if matches[2] != tc.replicaID {
					t.Errorf("Expected replica ID '%s', got '%s'", tc.replicaID, matches[2])
				}
			} else {
				if len(matches) > 0 {
					t.Errorf("Expected pattern not to match '%s', but it did", tc.serviceName)
				}
			}
		})
	}
}

// TestStubNamedServiceDetector creates a stub implementation for testing without Docker API
func TestStubNamedServiceDetector(t *testing.T) {
	// Create a stub named service detector
	stub := &StubNamedServiceDetector{
		ServiceGroups: map[string][]ServiceNameInfo{
			"web": {
				{BaseServiceName: "web", ReplicaID: "1", FullServiceName: "web-1"},
				{BaseServiceName: "web", ReplicaID: "2", FullServiceName: "web-2"},
				{BaseServiceName: "web", ReplicaID: "blue", FullServiceName: "web-blue"},
			},
			"api": {
				{BaseServiceName: "api", ReplicaID: "1", FullServiceName: "api-1"},
				{BaseServiceName: "api", ReplicaID: "2", FullServiceName: "api-2"},
			},
		},
	}

	// Add mock replicas for testing
	stub.Replicas = map[string][]Replica{
		"web": {
			{ServiceName: "web", ReplicaID: "1", ContainerID: "web1", Status: "running"},
			{ServiceName: "web", ReplicaID: "2", ContainerID: "web2", Status: "running"},
			{ServiceName: "web", ReplicaID: "blue", ContainerID: "web-blue", Status: "running"},
		},
		"api": {
			{ServiceName: "api", ReplicaID: "1", ContainerID: "api1", Status: "running"},
			{ServiceName: "api", ReplicaID: "2", ContainerID: "api2", Status: "running"},
		},
	}

	// Test the stub
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

	if len(replicas["api"]) != 2 {
		t.Errorf("Expected 2 replicas for api service, got %d", len(replicas["api"]))
	}

	// Verify replica IDs
	webReplicaIDs := map[string]bool{}
	for _, replica := range replicas["web"] {
		webReplicaIDs[replica.ReplicaID] = true
	}

	expectedWebReplicaIDs := []string{"1", "2", "blue"}
	for _, id := range expectedWebReplicaIDs {
		if !webReplicaIDs[id] {
			t.Errorf("Expected to find web replica with ID '%s'", id)
		}
	}
}

// TestNamedServiceDetectorIntegration is an integration test for the full workflow
func TestNamedServiceDetectorIntegration(t *testing.T) {
	// Skip in automated tests, but can be run manually when Docker is available
	t.Skip("Skipping integration test that requires Docker")

	// This test requires:
	// 1. Docker daemon running
	// 2. Docker Compose file with named services
	// 3. Running containers that match the named services

	// Create a test Docker Compose file
	composeContent := `
version: '3'
services:
  web-1:
    image: nginx:latest
  web-2:
    image: nginx:latest
  api-blue:
    image: node:latest
  api-green:
    image: node:latest
`
	composeFile := createTempComposeFile(t, composeContent)
	defer os.Remove(composeFile)

	// In a real test, we would:
	// 1. Run `docker-compose up -d` to start services
	// 2. Wait for containers to start
	// 3. Run our detector
	// 4. Verify the results
	// 5. Clean up with `docker-compose down`

	// Create a real detector
	detector, err := NewNamedServiceDetector()
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
