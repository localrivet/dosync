package notification

import (
	"errors"
	"testing"
	"time"
)

func TestMockNotifier(t *testing.T) {
	config := NotificationConfig{
		Type:     string(SlackNotification),
		Endpoint: "https://hooks.slack.com/services/x/y/z",
		Token:    "xoxb-token",
		Channel:  "deployments",
	}

	t.Run("SendDeploymentStarted", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewMockNotifier(config)
		mockNotifier.ErrorToReturn = errors.New("test error")

		// Set test values
		service := "web-service"
		version := "v1.2.3"

		// Call method
		err := mockNotifier.SendDeploymentStarted(service, version)

		// Verify results
		if err != mockNotifier.ErrorToReturn {
			t.Errorf("Expected error %v, got %v", mockNotifier.ErrorToReturn, err)
		}
		if !mockNotifier.DeploymentStartedCalled {
			t.Error("DeploymentStartedCalled should be true")
		}
		if mockNotifier.LastService != service {
			t.Errorf("Expected LastService to be %s, got %s", service, mockNotifier.LastService)
		}
		if mockNotifier.LastVersion != version {
			t.Errorf("Expected LastVersion to be %s, got %s", version, mockNotifier.LastVersion)
		}
	})

	t.Run("SendDeploymentSuccess", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewMockNotifier(config)

		// Set test values
		service := "web-service"
		version := "v1.2.3"
		duration := 5 * time.Second

		// Call method
		err := mockNotifier.SendDeploymentSuccess(service, version, duration)

		// Verify results
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mockNotifier.DeploymentSuccessCalled {
			t.Error("DeploymentSuccessCalled should be true")
		}
		if mockNotifier.LastService != service {
			t.Errorf("Expected LastService to be %s, got %s", service, mockNotifier.LastService)
		}
		if mockNotifier.LastVersion != version {
			t.Errorf("Expected LastVersion to be %s, got %s", version, mockNotifier.LastVersion)
		}
		if mockNotifier.LastDuration != duration {
			t.Errorf("Expected LastDuration to be %v, got %v", duration, mockNotifier.LastDuration)
		}
	})

	t.Run("SendDeploymentFailure", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewMockNotifier(config)

		// Set test values
		service := "web-service"
		version := "v1.2.3"
		errorMessage := "deployment failed: container crashed"

		// Call method
		err := mockNotifier.SendDeploymentFailure(service, version, errorMessage)

		// Verify results
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mockNotifier.DeploymentFailureCalled {
			t.Error("DeploymentFailureCalled should be true")
		}
		if mockNotifier.LastService != service {
			t.Errorf("Expected LastService to be %s, got %s", service, mockNotifier.LastService)
		}
		if mockNotifier.LastVersion != version {
			t.Errorf("Expected LastVersion to be %s, got %s", version, mockNotifier.LastVersion)
		}
		if mockNotifier.LastErrorMessage != errorMessage {
			t.Errorf("Expected LastErrorMessage to be %s, got %s", errorMessage, mockNotifier.LastErrorMessage)
		}
	})

	t.Run("SendRollback", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewMockNotifier(config)

		// Set test values
		service := "web-service"
		fromVersion := "v1.2.3"
		toVersion := "v1.2.2"

		// Call method
		err := mockNotifier.SendRollback(service, fromVersion, toVersion)

		// Verify results
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !mockNotifier.RollbackCalled {
			t.Error("RollbackCalled should be true")
		}
		if mockNotifier.LastService != service {
			t.Errorf("Expected LastService to be %s, got %s", service, mockNotifier.LastService)
		}
		if mockNotifier.LastFromVersion != fromVersion {
			t.Errorf("Expected LastFromVersion to be %s, got %s", fromVersion, mockNotifier.LastFromVersion)
		}
		if mockNotifier.LastToVersion != toVersion {
			t.Errorf("Expected LastToVersion to be %s, got %s", toVersion, mockNotifier.LastToVersion)
		}
	})
}
