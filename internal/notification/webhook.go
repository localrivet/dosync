package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// WebhookNotifier implements the Notifier interface for webhook notifications
type WebhookNotifier struct {
	BaseNotifier
}

// WebhookPayload represents the common structure for all webhook payloads
type WebhookPayload struct {
	Event       string                 `json:"event"`
	Service     string                 `json:"service"`
	Timestamp   string                 `json:"timestamp"`
	Details     map[string]interface{} `json:"details"`
	Environment string                 `json:"environment,omitempty"`
	Server      string                 `json:"server,omitempty"`
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(config NotificationConfig) *WebhookNotifier {
	webhook := &WebhookNotifier{}
	_ = webhook.Configure(config)
	return webhook
}

// sendWebhook sends a webhook notification
func (w *WebhookNotifier) sendWebhook(payload WebhookPayload) error {
	// Add hostname if not already set
	if payload.Server == "" {
		hostname, err := getHostname()
		if err == nil {
			payload.Server = hostname
		}
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", w.Config.Endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if w.Config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+w.Config.Token)
	}

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode)
	}

	return nil
}

// getHostname retrieves the hostname of the server
func getHostname() (string, error) {
	// Import here to avoid circular imports
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

// SendDeploymentStarted sends a webhook notification when a deployment starts
func (w *WebhookNotifier) SendDeploymentStarted(service string, version string) error {
	if !w.ShouldNotifyOnSuccess() {
		return nil
	}

	payload := WebhookPayload{
		Event:     "deployment_started",
		Service:   service,
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"version": version,
		},
	}

	return w.sendWebhook(payload)
}

// SendDeploymentSuccess sends a webhook notification when a deployment succeeds
func (w *WebhookNotifier) SendDeploymentSuccess(service string, version string, duration time.Duration) error {
	if !w.ShouldNotifyOnSuccess() {
		return nil
	}

	payload := WebhookPayload{
		Event:     "deployment_success",
		Service:   service,
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"version":  version,
			"duration": duration.String(),
		},
	}

	return w.sendWebhook(payload)
}

// SendDeploymentFailure sends a webhook notification when a deployment fails
func (w *WebhookNotifier) SendDeploymentFailure(service string, version string, errorMsg string) error {
	if !w.ShouldNotifyOnFailure() {
		return nil
	}

	payload := WebhookPayload{
		Event:     "deployment_failure",
		Service:   service,
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"version": version,
			"error":   errorMsg,
		},
	}

	return w.sendWebhook(payload)
}

// SendRollback sends a webhook notification when a rollback occurs
func (w *WebhookNotifier) SendRollback(service string, fromVersion string, toVersion string) error {
	if !w.ShouldNotifyOnRollback() {
		return nil
	}

	payload := WebhookPayload{
		Event:     "deployment_rollback",
		Service:   service,
		Timestamp: time.Now().Format(time.RFC3339),
		Details: map[string]interface{}{
			"from_version": fromVersion,
			"to_version":   toVersion,
		},
	}

	return w.sendWebhook(payload)
}
