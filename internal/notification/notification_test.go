package notification

import (
	"reflect"
	"testing"
)

func TestNotificationConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  NotificationConfig
		wantErr bool
	}{
		{
			name: "Valid slack config",
			config: NotificationConfig{
				Type:     string(SlackNotification),
				Endpoint: "https://hooks.slack.com/services/x/y/z",
				Token:    "xoxb-token",
				Channel:  "deployments",
			},
			wantErr: false,
		},
		{
			name: "Valid email config",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com:587",
				Recipients: []string{"admin@example.com"},
			},
			wantErr: false,
		},
		{
			name: "Valid webhook config",
			config: NotificationConfig{
				Type:     string(WebhookNotification),
				Endpoint: "https://api.example.com/webhook",
			},
			wantErr: false,
		},
		{
			name: "Empty type",
			config: NotificationConfig{
				Endpoint: "https://api.example.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "Empty endpoint",
			config: NotificationConfig{
				Type: string(WebhookNotification),
			},
			wantErr: true,
		},
		{
			name: "Slack without token",
			config: NotificationConfig{
				Type:     string(SlackNotification),
				Endpoint: "https://hooks.slack.com/services/x/y/z",
				Channel:  "deployments",
			},
			wantErr: true,
		},
		{
			name: "Slack without channel",
			config: NotificationConfig{
				Type:     string(SlackNotification),
				Endpoint: "https://hooks.slack.com/services/x/y/z",
				Token:    "xoxb-token",
			},
			wantErr: true,
		},
		{
			name: "Email without recipients",
			config: NotificationConfig{
				Type:     string(EmailNotification),
				Endpoint: "smtp.example.com:587",
			},
			wantErr: true,
		},
		{
			name: "Unsupported notification type",
			config: NotificationConfig{
				Type:     "sms",
				Endpoint: "sms.example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NotificationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseNotifierConfigure(t *testing.T) {
	tests := []struct {
		name    string
		config  NotificationConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: NotificationConfig{
				Type:     string(SlackNotification),
				Endpoint: "https://hooks.slack.com/services/x/y/z",
				Token:    "xoxb-token",
				Channel:  "deployments",
			},
			wantErr: false,
		},
		{
			name: "Invalid config",
			config: NotificationConfig{
				Type: string(SlackNotification),
				// Missing required fields
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := &BaseNotifier{}
			err := notifier.Configure(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("BaseNotifier.Configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !reflect.DeepEqual(notifier.Config, tt.config) {
				t.Errorf("BaseNotifier.Configure() didn't properly set config")
			}
		})
	}
}

func TestBaseNotifierShouldNotify(t *testing.T) {
	config := NotificationConfig{
		Type:       string(SlackNotification),
		Endpoint:   "https://hooks.slack.com/services/x/y/z",
		Token:      "xoxb-token",
		Channel:    "deployments",
		OnSuccess:  true,
		OnFailure:  false,
		OnRollback: true,
	}

	notifier := &BaseNotifier{}
	err := notifier.Configure(config)
	if err != nil {
		t.Fatalf("Unexpected error configuring notifier: %v", err)
	}

	if !notifier.ShouldNotifyOnSuccess() {
		t.Error("ShouldNotifyOnSuccess() = false, want true")
	}

	if notifier.ShouldNotifyOnFailure() {
		t.Error("ShouldNotifyOnFailure() = true, want false")
	}

	if !notifier.ShouldNotifyOnRollback() {
		t.Error("ShouldNotifyOnRollback() = false, want true")
	}
}
