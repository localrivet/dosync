package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// SlackNotifier implements the Notifier interface for Slack
type SlackNotifier struct {
	BaseNotifier
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(config NotificationConfig) *SlackNotifier {
	slack := &SlackNotifier{}
	_ = slack.Configure(config)
	return slack
}

// slackMessage represents a Slack message payload
type slackMessage struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
	Blocks      []interface{}     `json:"blocks,omitempty"`
}

// slackAttachment represents a Slack message attachment
type slackAttachment struct {
	Color     string       `json:"color"` // good, warning, danger
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []slackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Ts        json.Number  `json:"ts,omitempty"`
	Timestamp int64        `json:"timestamp,omitempty"`
}

// slackField represents a field in a Slack attachment
type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// sendSlackMessage sends a message to Slack
func (s *SlackNotifier) sendSlackMessage(message slackMessage) error {
	// Set channel if not specified
	if message.Channel == "" {
		message.Channel = s.Config.Channel
	}

	// Convert message to JSON
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", s.Config.Endpoint, bytes.NewBuffer(jsonMessage))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if s.Config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Config.Token)
	}

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned non-success status code: %d", resp.StatusCode)
	}

	return nil
}

// getHostnameField returns the server hostname for inclusion in messages
func getHostnameField() (string, string) {
	hostname, err := os.Hostname()
	if err != nil {
		return "Server", "unknown"
	}
	return "Server", hostname
}

// SendDeploymentStarted sends a Slack notification for a deployment start
func (s *SlackNotifier) SendDeploymentStarted(service, version string) error {
	if !s.ShouldNotifyOnSuccess() {
		return nil
	}

	message := fmt.Sprintf("Starting deployment for service *%s* to version *%s*", service, version)

	serverField, hostname := getHostnameField()

	attachment := slackAttachment{
		Color: "good",
		Fields: []slackField{
			{Title: "Service", Value: service, Short: true},
			{Title: "Version", Value: version, Short: true},
			{Title: serverField, Value: hostname, Short: true},
			{Title: "Event", Value: "Deployment Started", Short: true},
		},
		Footer: "DOSync Deployment Service",
		Ts:     json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	return s.sendSlackMessage(slackMessage{
		Channel:     s.Config.Channel,
		Text:        message,
		Attachments: []slackAttachment{attachment},
	})
}

// SendDeploymentSuccess sends a Slack notification for a successful deployment
func (s *SlackNotifier) SendDeploymentSuccess(service, version string, duration time.Duration) error {
	if !s.ShouldNotifyOnSuccess() {
		return nil
	}

	message := fmt.Sprintf("Successfully deployed service *%s* to version *%s* in %s", service, version, duration.String())

	serverField, hostname := getHostnameField()

	attachment := slackAttachment{
		Color: "good",
		Fields: []slackField{
			{Title: "Service", Value: service, Short: true},
			{Title: "Version", Value: version, Short: true},
			{Title: "Duration", Value: duration.String(), Short: true},
			{Title: serverField, Value: hostname, Short: true},
			{Title: "Event", Value: "Deployment Successful", Short: true},
		},
		Footer: "DOSync Deployment Service",
		Ts:     json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	return s.sendSlackMessage(slackMessage{
		Channel:     s.Config.Channel,
		Text:        message,
		Attachments: []slackAttachment{attachment},
	})
}

// SendDeploymentFailure sends a Slack notification for a failed deployment
func (s *SlackNotifier) SendDeploymentFailure(service, version, errorMsg string) error {
	if !s.ShouldNotifyOnFailure() {
		return nil
	}

	message := fmt.Sprintf("Failed to deploy service *%s* to version *%s*", service, version)

	serverField, hostname := getHostnameField()

	attachment := slackAttachment{
		Color: "danger",
		Fields: []slackField{
			{Title: "Service", Value: service, Short: true},
			{Title: "Version", Value: version, Short: true},
			{Title: serverField, Value: hostname, Short: true},
			{Title: "Error", Value: errorMsg, Short: false},
			{Title: "Event", Value: "Deployment Failed", Short: true},
		},
		Footer: "DOSync Deployment Service",
		Ts:     json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	return s.sendSlackMessage(slackMessage{
		Channel:     s.Config.Channel,
		Text:        message,
		Attachments: []slackAttachment{attachment},
	})
}

// SendRollback sends a Slack notification for a rollback
func (s *SlackNotifier) SendRollback(service, fromVersion, toVersion string) error {
	if !s.ShouldNotifyOnRollback() {
		return nil
	}

	message := fmt.Sprintf("Rolling back service *%s* from version *%s* to *%s*", service, fromVersion, toVersion)

	serverField, hostname := getHostnameField()

	attachment := slackAttachment{
		Color: "warning",
		Fields: []slackField{
			{Title: "Service", Value: service, Short: true},
			{Title: "From Version", Value: fromVersion, Short: true},
			{Title: "To Version", Value: toVersion, Short: true},
			{Title: serverField, Value: hostname, Short: true},
			{Title: "Event", Value: "Deployment Rollback", Short: true},
		},
		Footer: "DOSync Deployment Service",
		Ts:     json.Number(fmt.Sprintf("%d", time.Now().Unix())),
	}

	return s.sendSlackMessage(slackMessage{
		Channel:     s.Config.Channel,
		Text:        message,
		Attachments: []slackAttachment{attachment},
	})
}
