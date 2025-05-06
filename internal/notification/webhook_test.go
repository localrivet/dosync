package notification

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewWebhookNotifier(t *testing.T) {
	config := NotificationConfig{
		Type:     string(WebhookNotification),
		Endpoint: "https://api.example.com/webhook",
		Token:    "test-token",
	}

	webhook := NewWebhookNotifier(config)

	if webhook == nil {
		t.Fatal("Expected NewWebhookNotifier to return a non-nil value")
	}

	if !reflect.DeepEqual(webhook.Config, config) {
		t.Errorf("Expected Config to be %v, got %v", config, webhook.Config)
	}
}

func TestWebhookNotifier_sendWebhook(t *testing.T) {
	// Create a test server that records the request
	var receivedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier with the test server URL
	config := NotificationConfig{
		Type:     string(WebhookNotification),
		Endpoint: server.URL,
		Token:    "test-token",
	}
	webhook := NewWebhookNotifier(config)

	// Create test payload with explicit server name for testing
	payload := WebhookPayload{
		Event:     "test_event",
		Service:   "test-service",
		Server:    "test-server-01",
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	// Send the webhook
	err := webhook.sendWebhook(payload)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that the request was made correctly
	if receivedRequest == nil {
		t.Fatal("No request was received by the test server")
	}

	// Check headers
	if receivedRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be 'application/json', got '%s'", receivedRequest.Header.Get("Content-Type"))
	}

	if receivedRequest.Header.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Expected Authorization header to be 'Bearer test-token', got '%s'", receivedRequest.Header.Get("Authorization"))
	}
}

func TestWebhookNotifier_AutomaticServerDetection(t *testing.T) {
	// Create a test server
	var bodyContent []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		var err error
		bodyContent, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier
	config := NotificationConfig{
		Type:      string(WebhookNotification),
		Endpoint:  server.URL,
		OnSuccess: true,
	}
	webhook := NewWebhookNotifier(config)

	// Create payload without server field to test auto-detection
	payload := WebhookPayload{
		Event:     "test_event",
		Service:   "test-service",
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	// Send the webhook
	err := webhook.sendWebhook(payload)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that a server field was added
	var receivedPayload WebhookPayload
	err = json.Unmarshal(bodyContent, &receivedPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	// The hostname should be automatically populated
	if receivedPayload.Server == "" {
		t.Errorf("Expected Server field to be automatically populated, got empty string")
	}
}

func TestWebhookNotifier_SendDeploymentStarted(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just check that we received a request with the correct method
		if r.Method != "POST" {
			t.Errorf("Expected HTTP method to be POST, got %s", r.Method)
		}

		// Check that Content-Type is set correctly
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier with notifications enabled
	config := NotificationConfig{
		Type:      string(WebhookNotification),
		Endpoint:  server.URL,
		Token:     "test-token",
		OnSuccess: true,
	}
	webhook := NewWebhookNotifier(config)

	// Call the SendDeploymentStarted method
	err := webhook.SendDeploymentStarted("test-service", "v1.0.0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWebhookNotifier_SendDeploymentSuccess(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier with notifications enabled
	config := NotificationConfig{
		Type:      string(WebhookNotification),
		Endpoint:  server.URL,
		Token:     "test-token",
		OnSuccess: true,
	}
	webhook := NewWebhookNotifier(config)

	// Call the SendDeploymentSuccess method
	duration := 5 * time.Second
	err := webhook.SendDeploymentSuccess("test-service", "v1.0.0", duration)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWebhookNotifier_SendDeploymentFailure(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier with notifications enabled
	config := NotificationConfig{
		Type:      string(WebhookNotification),
		Endpoint:  server.URL,
		Token:     "test-token",
		OnFailure: true,
	}
	webhook := NewWebhookNotifier(config)

	// Call the SendDeploymentFailure method
	err := webhook.SendDeploymentFailure("test-service", "v1.0.0", "Failed to start container")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWebhookNotifier_SendRollback(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook notifier with notifications enabled
	config := NotificationConfig{
		Type:       string(WebhookNotification),
		Endpoint:   server.URL,
		Token:      "test-token",
		OnRollback: true,
	}
	webhook := NewWebhookNotifier(config)

	// Call the SendRollback method
	err := webhook.SendRollback("test-service", "v1.0.0", "v0.9.0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWebhookNotifier_NotificationsDisabled(t *testing.T) {
	// Create webhook notifier with all notifications disabled
	config := NotificationConfig{
		Type:       string(WebhookNotification),
		Endpoint:   "https://api.example.com/webhook",
		Token:      "test-token",
		OnSuccess:  false,
		OnFailure:  false,
		OnRollback: false,
	}
	webhook := NewWebhookNotifier(config)

	// Test that no notification is sent when disabled
	err := webhook.SendDeploymentStarted("test-service", "v1.0.0")
	if err != nil {
		t.Errorf("Expected nil error when notifications disabled, got %v", err)
	}

	err = webhook.SendDeploymentSuccess("test-service", "v1.0.0", 5*time.Second)
	if err != nil {
		t.Errorf("Expected nil error when notifications disabled, got %v", err)
	}

	err = webhook.SendDeploymentFailure("test-service", "v1.0.0", "Failed to start container")
	if err != nil {
		t.Errorf("Expected nil error when notifications disabled, got %v", err)
	}

	err = webhook.SendRollback("test-service", "v1.0.0", "v0.9.0")
	if err != nil {
		t.Errorf("Expected nil error when notifications disabled, got %v", err)
	}
}
