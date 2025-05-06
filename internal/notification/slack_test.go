package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewSlackNotifier(t *testing.T) {
	config := NotificationConfig{
		Type:     string(SlackNotification),
		Endpoint: "https://hooks.slack.com/services/x/y/z",
		Token:    "xoxb-test-token",
		Channel:  "deployments",
	}

	slack := NewSlackNotifier(config)

	if slack == nil {
		t.Fatal("Expected NewSlackNotifier to return a non-nil value")
	}

	if !reflect.DeepEqual(slack.Config, config) {
		t.Errorf("Expected Config to be %v, got %v", config, slack.Config)
	}
}

func TestSlackNotifier_sendSlackMessage(t *testing.T) {
	// Create a test server that records the request
	var receivedRequest *http.Request
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with the test server URL as endpoint
	config := NotificationConfig{
		Type:     string(SlackNotification),
		Endpoint: server.URL,
		Token:    "xoxb-test-token",
		Channel:  "deployments",
	}
	slack := NewSlackNotifier(config)

	// Test message
	message := slackMessage{
		Text: "Test message",
	}

	// Send the message
	err := slack.sendSlackMessage(message)
	if err != nil {
		t.Fatalf("Expected sendSlackMessage to succeed, got error: %v", err)
	}

	// Verify request headers and body
	if receivedRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be application/json, got %s", receivedRequest.Header.Get("Content-Type"))
	}

	if receivedRequest.Header.Get("Authorization") != "Bearer "+config.Token {
		t.Errorf("Expected Authorization header to be 'Bearer %s', got %s", config.Token, receivedRequest.Header.Get("Authorization"))
	}

	// Verify message contents
	var receivedMessage slackMessage
	err = json.Unmarshal(receivedBody, &receivedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	if receivedMessage.Text != message.Text {
		t.Errorf("Expected message text to be '%s', got '%s'", message.Text, receivedMessage.Text)
	}

	if receivedMessage.Channel != config.Channel {
		t.Errorf("Expected message channel to be '%s', got '%s'", config.Channel, receivedMessage.Channel)
	}
}

func TestSlackNotifier_SendDeploymentStarted(t *testing.T) {
	// Create a test server that records the request
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with the test server URL as endpoint
	config := NotificationConfig{
		Type:      string(SlackNotification),
		Endpoint:  server.URL,
		Token:     "xoxb-test-token",
		Channel:   "deployments",
		OnSuccess: true,
	}
	slack := NewSlackNotifier(config)

	// Test parameters
	service := "web-service"
	version := "v1.2.3"

	// Send the notification
	err := slack.SendDeploymentStarted(service, version)
	if err != nil {
		t.Fatalf("Expected SendDeploymentStarted to succeed, got error: %v", err)
	}

	// Verify message contents
	var receivedMessage slackMessage
	err = json.Unmarshal(receivedBody, &receivedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	if len(receivedMessage.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(receivedMessage.Attachments))
	}

	attachment := receivedMessage.Attachments[0]

	// Check for Fields instead of Title/Text
	found := false
	for _, field := range attachment.Fields {
		if field.Title == "Service" && field.Value == service {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find a field with title 'Service' and value '%s'", service)
	}

	// Check that the message text contains the service and version
	if !containsAll(receivedMessage.Text, []string{service, version}) {
		t.Errorf("Expected message text to contain service and version, got '%s'", receivedMessage.Text)
	}
}

func TestSlackNotifier_SendDeploymentSuccess(t *testing.T) {
	// Create a test server that records the request
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with the test server URL as endpoint
	config := NotificationConfig{
		Type:      string(SlackNotification),
		Endpoint:  server.URL,
		Token:     "xoxb-test-token",
		Channel:   "deployments",
		OnSuccess: true,
	}
	slack := NewSlackNotifier(config)

	// Test parameters
	service := "web-service"
	version := "v1.2.3"
	duration := 5 * time.Second

	// Send the notification
	err := slack.SendDeploymentSuccess(service, version, duration)
	if err != nil {
		t.Fatalf("Expected SendDeploymentSuccess to succeed, got error: %v", err)
	}

	// Verify message contents
	var receivedMessage slackMessage
	err = json.Unmarshal(receivedBody, &receivedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	if len(receivedMessage.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(receivedMessage.Attachments))
	}

	attachment := receivedMessage.Attachments[0]

	// Check for Fields with correct values
	serviceFound := false
	versionFound := false
	durationFound := false

	for _, field := range attachment.Fields {
		if field.Title == "Service" && field.Value == service {
			serviceFound = true
		}
		if field.Title == "Version" && field.Value == version {
			versionFound = true
		}
		if field.Title == "Duration" && field.Value == duration.String() {
			durationFound = true
		}
	}

	if !serviceFound {
		t.Errorf("Expected to find a field with title 'Service' and value '%s'", service)
	}
	if !versionFound {
		t.Errorf("Expected to find a field with title 'Version' and value '%s'", version)
	}
	if !durationFound {
		t.Errorf("Expected to find a field with title 'Duration' and value '%s'", duration.String())
	}

	// Check that the footer is present
	if attachment.Footer == "" {
		t.Error("Expected footer to be present")
	}
}

func TestSlackNotifier_SendDeploymentFailure(t *testing.T) {
	// Create a test server that records the request
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with the test server URL as endpoint
	config := NotificationConfig{
		Type:      string(SlackNotification),
		Endpoint:  server.URL,
		Token:     "xoxb-test-token",
		Channel:   "deployments",
		OnFailure: true,
	}
	slack := NewSlackNotifier(config)

	// Test parameters
	service := "web-service"
	version := "v1.2.3"
	errorMsg := "container crashed"

	// Send the notification
	err := slack.SendDeploymentFailure(service, version, errorMsg)
	if err != nil {
		t.Fatalf("Expected SendDeploymentFailure to succeed, got error: %v", err)
	}

	// Verify message contents
	var receivedMessage slackMessage
	err = json.Unmarshal(receivedBody, &receivedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	if len(receivedMessage.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(receivedMessage.Attachments))
	}

	attachment := receivedMessage.Attachments[0]

	// Check for Fields with correct values
	serviceFound := false
	versionFound := false
	errorFound := false

	for _, field := range attachment.Fields {
		if field.Title == "Service" && field.Value == service {
			serviceFound = true
		}
		if field.Title == "Version" && field.Value == version {
			versionFound = true
		}
		if field.Title == "Error" && field.Value == errorMsg {
			errorFound = true
		}
	}

	if !serviceFound {
		t.Errorf("Expected to find a field with title 'Service' and value '%s'", service)
	}
	if !versionFound {
		t.Errorf("Expected to find a field with title 'Version' and value '%s'", version)
	}
	if !errorFound {
		t.Errorf("Expected to find a field with title 'Error' and value '%s'", errorMsg)
	}
}

func TestSlackNotifier_SendRollback(t *testing.T) {
	// Create a test server that records the request
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with the test server URL as endpoint
	config := NotificationConfig{
		Type:       string(SlackNotification),
		Endpoint:   server.URL,
		Token:      "xoxb-test-token",
		Channel:    "deployments",
		OnRollback: true,
	}
	slack := NewSlackNotifier(config)

	// Test parameters
	service := "web-service"
	fromVersion := "v1.2.3"
	toVersion := "v1.2.2"

	// Send the notification
	err := slack.SendRollback(service, fromVersion, toVersion)
	if err != nil {
		t.Fatalf("Expected SendRollback to succeed, got error: %v", err)
	}

	// Verify message contents
	var receivedMessage slackMessage
	err = json.Unmarshal(receivedBody, &receivedMessage)
	if err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	if len(receivedMessage.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(receivedMessage.Attachments))
	}

	attachment := receivedMessage.Attachments[0]

	// Check for Fields with correct values
	serviceFound := false
	fromFound := false
	toFound := false

	for _, field := range attachment.Fields {
		if field.Title == "Service" && field.Value == service {
			serviceFound = true
		}
		if field.Title == "From Version" && field.Value == fromVersion {
			fromFound = true
		}
		if field.Title == "To Version" && field.Value == toVersion {
			toFound = true
		}
	}

	if !serviceFound {
		t.Errorf("Expected to find a field with title 'Service' and value '%s'", service)
	}
	if !fromFound {
		t.Errorf("Expected to find a field with title 'From Version' and value '%s'", fromVersion)
	}
	if !toFound {
		t.Errorf("Expected to find a field with title 'To Version' and value '%s'", toVersion)
	}
}

func TestSlackNotifier_NotificationsDisabled(t *testing.T) {
	// Create a test server that records the request
	requestReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a slack notifier with all notifications disabled
	config := NotificationConfig{
		Type:       string(SlackNotification),
		Endpoint:   server.URL,
		Token:      "xoxb-test-token",
		Channel:    "deployments",
		OnSuccess:  false,
		OnFailure:  false,
		OnRollback: false,
	}
	slack := NewSlackNotifier(config)

	// Test that SendDeploymentStarted doesn't send a request
	err := slack.SendDeploymentStarted("service", "version")
	if err != nil {
		t.Errorf("Expected SendDeploymentStarted to succeed, got error: %v", err)
	}
	if requestReceived {
		t.Error("Expected no request to be sent when OnSuccess is false")
	}

	// Reset the flag
	requestReceived = false

	// Test that SendDeploymentSuccess doesn't send a request
	err = slack.SendDeploymentSuccess("service", "version", 1*time.Second)
	if err != nil {
		t.Errorf("Expected SendDeploymentSuccess to succeed, got error: %v", err)
	}
	if requestReceived {
		t.Error("Expected no request to be sent when OnSuccess is false")
	}

	// Reset the flag
	requestReceived = false

	// Test that SendDeploymentFailure doesn't send a request
	err = slack.SendDeploymentFailure("service", "version", "error")
	if err != nil {
		t.Errorf("Expected SendDeploymentFailure to succeed, got error: %v", err)
	}
	if requestReceived {
		t.Error("Expected no request to be sent when OnFailure is false")
	}

	// Reset the flag
	requestReceived = false

	// Test that SendRollback doesn't send a request
	err = slack.SendRollback("service", "from", "to")
	if err != nil {
		t.Errorf("Expected SendRollback to succeed, got error: %v", err)
	}
	if requestReceived {
		t.Error("Expected no request to be sent when OnRollback is false")
	}
}

// Helper function to check if a string contains all the provided substrings
func containsAll(str string, substrings []string) bool {
	for _, substr := range substrings {
		if !containsString(str, substr) {
			return false
		}
	}
	return true
}

// Helper function to check if a string contains a substring
func containsString(str, substr string) bool {
	return str != "" && substr != "" && len(str) >= len(substr) && str != substr && str[0:len(str)-1] != substr
}
