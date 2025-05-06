/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package replica

import (
	"testing"
)

// TestHasDetector tests the HasDetector method
func TestHasDetector(t *testing.T) {
	manager, err := NewReplicaManager("docker-compose.yml")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Initially, no detectors should be registered
	if manager.HasDetector(ScaleBased) {
		t.Errorf("Expected no ScaleBased detector to be registered")
	}
	if manager.HasDetector(NameBased) {
		t.Errorf("Expected no NameBased detector to be registered")
	}

	// Register a detector
	scaleDetector := NewStubScaleBasedDetector()
	manager.RegisterDetector(ScaleBased, scaleDetector)

	// Now ScaleBased should exist but not NameBased
	if !manager.HasDetector(ScaleBased) {
		t.Errorf("Expected ScaleBased detector to be registered")
	}
	if manager.HasDetector(NameBased) {
		t.Errorf("Expected no NameBased detector to be registered")
	}

	// Register the other detector
	nameDetector := NewStubNamedServiceDetector()
	manager.RegisterDetector(NameBased, nameDetector)

	// Now both should exist
	if !manager.HasDetector(ScaleBased) {
		t.Errorf("Expected ScaleBased detector to be registered")
	}
	if !manager.HasDetector(NameBased) {
		t.Errorf("Expected NameBased detector to be registered")
	}
}

// TestGetDetector tests the GetDetector method
func TestGetDetector(t *testing.T) {
	manager, err := NewReplicaManager("docker-compose.yml")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Initially, no detectors should be registered
	if detector := manager.GetDetector(ScaleBased); detector != nil {
		t.Errorf("Expected no ScaleBased detector to be registered")
	}

	// Register a detector
	scaleDetector := NewStubScaleBasedDetector()
	manager.RegisterDetector(ScaleBased, scaleDetector)

	// Now we should be able to get it
	if detector := manager.GetDetector(ScaleBased); detector == nil {
		t.Errorf("Expected to get ScaleBased detector")
	} else {
		// Verify it's the same detector
		stub, ok := detector.(*StubScaleBasedDetector)
		if !ok {
			t.Errorf("Expected StubScaleBasedDetector type")
		}
		if stub != scaleDetector {
			t.Errorf("Got different detector instance than what was registered")
		}
	}

	// Getting a non-existent detector should return nil
	if detector := manager.GetDetector(NameBased); detector != nil {
		t.Errorf("Expected nil for non-existent detector type")
	}
}

// TestUnregisterDetector tests the UnregisterDetector method
func TestUnregisterDetector(t *testing.T) {
	manager, err := NewReplicaManager("docker-compose.yml")
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Register detectors
	scaleDetector := NewStubScaleBasedDetector()
	nameDetector := NewStubNamedServiceDetector()
	manager.RegisterDetector(ScaleBased, scaleDetector)
	manager.RegisterDetector(NameBased, nameDetector)

	// Both should be registered
	if !manager.HasDetector(ScaleBased) {
		t.Errorf("Expected ScaleBased detector to be registered")
	}
	if !manager.HasDetector(NameBased) {
		t.Errorf("Expected NameBased detector to be registered")
	}

	// Unregister ScaleBased
	if !manager.UnregisterDetector(ScaleBased) {
		t.Errorf("UnregisterDetector should return true when detector was unregistered")
	}

	// Now ScaleBased should not exist but NameBased should
	if manager.HasDetector(ScaleBased) {
		t.Errorf("Expected ScaleBased detector to be unregistered")
	}
	if !manager.HasDetector(NameBased) {
		t.Errorf("Expected NameBased detector to still be registered")
	}

	// Trying to unregister a non-existent detector should return false
	if manager.UnregisterDetector(ScaleBased) {
		t.Errorf("UnregisterDetector should return false when no detector was registered")
	}

	// Unregister NameBased
	if !manager.UnregisterDetector(NameBased) {
		t.Errorf("UnregisterDetector should return true when detector was unregistered")
	}

	// Now no detectors should exist
	if manager.HasDetector(ScaleBased) {
		t.Errorf("Expected ScaleBased detector to be unregistered")
	}
	if manager.HasDetector(NameBased) {
		t.Errorf("Expected NameBased detector to be unregistered")
	}
}

// TestGetDetectorByType tests the GetDetectorByType convenience function
func TestGetDetectorByType(t *testing.T) {
	// Skip this test in automated runs since it would create real detectors
	t.Skip("Skipping test that creates real detectors")

	composeFile := "docker-compose.yml"

	// Test getting a scale-based detector
	scaleDetector, err := GetDetectorByType(composeFile, ScaleBased)
	if err != nil {
		t.Fatalf("Failed to get ScaleBased detector: %v", err)
	}
	if scaleDetector == nil {
		t.Errorf("Expected non-nil ScaleBased detector")
	}
	if scaleDetector.GetReplicaType() != ScaleBased {
		t.Errorf("Expected ScaleBased detector type, got %s", scaleDetector.GetReplicaType())
	}

	// Test getting a name-based detector
	nameDetector, err := GetDetectorByType(composeFile, NameBased)
	if err != nil {
		t.Fatalf("Failed to get NameBased detector: %v", err)
	}
	if nameDetector == nil {
		t.Errorf("Expected non-nil NameBased detector")
	}
	if nameDetector.GetReplicaType() != NameBased {
		t.Errorf("Expected NameBased detector type, got %s", nameDetector.GetReplicaType())
	}
}
