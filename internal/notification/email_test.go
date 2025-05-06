package notification

import (
	"reflect"
	"testing"
	"time"
)

func TestNewEmailNotifier(t *testing.T) {
	config := NotificationConfig{
		Type:       string(EmailNotification),
		Endpoint:   "smtp.example.com:587",
		Token:      "password",
		Recipients: []string{"admin@example.com"},
	}

	email := NewEmailNotifier(config)

	if email == nil {
		t.Fatal("Expected NewEmailNotifier to return a non-nil value")
	}

	if !reflect.DeepEqual(email.Config, config) {
		t.Errorf("Expected Config to be %v, got %v", config, email.Config)
	}

	if email.from != config.Token {
		t.Errorf("Expected from to be %s, got %s", config.Token, email.from)
	}
}

func TestEmailNotifier_Configure(t *testing.T) {
	tests := []struct {
		name    string
		config  NotificationConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com:587",
				Token:      "password",
				Recipients: []string{"admin@example.com"},
			},
			wantErr: false,
		},
		{
			name: "Invalid endpoint format",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com", // Missing port
				Token:      "password",
				Recipients: []string{"admin@example.com"},
			},
			wantErr: true,
		},
		{
			name: "Empty recipients",
			config: NotificationConfig{
				Type:     string(EmailNotification),
				Endpoint: "smtp.example.com:587",
				Token:    "password",
				// Missing recipients
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := &EmailNotifier{}
			err := email.Configure(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("EmailNotifier.Configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if !reflect.DeepEqual(email.Config, tt.config) {
					t.Errorf("EmailNotifier.Configure() didn't properly set config")
				}
				if email.from != tt.config.Token {
					t.Errorf("EmailNotifier.Configure() didn't properly set from field")
				}
			}
		})
	}
}

func TestEmailNotifier_ShouldNotify(t *testing.T) {
	tests := []struct {
		name     string
		config   NotificationConfig
		expected bool
	}{
		{
			name: "All notifications enabled",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com:587",
				Token:      "password",
				Recipients: []string{"admin@example.com"},
				OnSuccess:  true,
				OnFailure:  true,
				OnRollback: true,
			},
			expected: true,
		},
		{
			name: "Only success notifications",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com:587",
				Token:      "password",
				Recipients: []string{"admin@example.com"},
				OnSuccess:  true,
				OnFailure:  false,
				OnRollback: false,
			},
			expected: true,
		},
		{
			name: "All notifications disabled",
			config: NotificationConfig{
				Type:       string(EmailNotification),
				Endpoint:   "smtp.example.com:587",
				Token:      "password",
				Recipients: []string{"admin@example.com"},
				OnSuccess:  false,
				OnFailure:  false,
				OnRollback: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := NewEmailNotifier(tt.config)
			result := email.ShouldNotify()

			if result != tt.expected {
				t.Errorf("EmailNotifier.ShouldNotify() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestEmailNotifier_Send* functions would ideally use a mock SMTP server
// In a real implementation, we would create a mock SMTP server to test sending emails
// However, for simplicity, we'll just test that the methods behave correctly based on notification settings

func TestEmailNotifier_SendMethods(t *testing.T) {
	// Create a config with notifications disabled
	disabledConfig := NotificationConfig{
		Type:       string(EmailNotification),
		Endpoint:   "smtp.example.com:587",
		Token:      "password",
		Recipients: []string{"admin@example.com"},
		OnSuccess:  false,
		OnFailure:  false,
		OnRollback: false,
	}

	t.Run("SendDeploymentStarted with notifications disabled", func(t *testing.T) {
		email := NewEmailNotifier(disabledConfig)
		// This should return nil immediately without attempting to send
		err := email.SendDeploymentStarted("web", "v1.0.0")
		if err != nil {
			t.Errorf("Expected nil error when notifications disabled, got %v", err)
		}
	})

	t.Run("SendDeploymentSuccess with notifications disabled", func(t *testing.T) {
		email := NewEmailNotifier(disabledConfig)
		// This should return nil immediately without attempting to send
		err := email.SendDeploymentSuccess("web", "v1.0.0", 5*time.Second)
		if err != nil {
			t.Errorf("Expected nil error when notifications disabled, got %v", err)
		}
	})

	t.Run("SendDeploymentFailure with notifications disabled", func(t *testing.T) {
		email := NewEmailNotifier(disabledConfig)
		// This should return nil immediately without attempting to send
		err := email.SendDeploymentFailure("web", "v1.0.0", "test error")
		if err != nil {
			t.Errorf("Expected nil error when notifications disabled, got %v", err)
		}
	})

	t.Run("SendRollback with notifications disabled", func(t *testing.T) {
		email := NewEmailNotifier(disabledConfig)
		// This should return nil immediately without attempting to send
		err := email.SendRollback("web", "v1.0.0", "v0.9.0")
		if err != nil {
			t.Errorf("Expected nil error when notifications disabled, got %v", err)
		}
	})

	// Note: We can't easily test the success path without mocking SMTP or using a real server
	// In a real implementation, we would use a mock SMTP server or dependency injection
	// to test the actual sending functionality
}
