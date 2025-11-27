package replica

import "testing"

func TestExtractProjectNameFromCompose(t *testing.T) {
	tests := []struct {
		name        string
		compose     string
		serviceName string
		expected    string
	}{
		{
			name: "Extract from container_name with app service",
			compose: "services:\n  app:\n    image: ghcr.io/localrivet/almatuck.com:latest\n    container_name: almatuck_app\n    restart: unless-stopped\n",
			serviceName: "app",
			expected:    "almatuck",
		},
		{
			name: "Extract from container_name with postgres service",
			compose: "services:\n  postgres:\n    image: postgres:16-alpine\n    container_name: crowdgains_postgres\n    restart: unless-stopped\n",
			serviceName: "postgres",
			expected:    "crowdgains",
		},
		{
			name: "Extract from any container_name when service doesn't match exactly",
			compose: "services:\n  database:\n    image: postgres:16-alpine\n    container_name: patternadvisor_postgres\n    restart: unless-stopped\n",
			serviceName: "database",
			expected:    "patternadvisor",
		},
		{
			name: "No container_name defined",
			compose: "services:\n  app:\n    image: ghcr.io/localrivet/almatuck.com:latest\n    restart: unless-stopped\n",
			serviceName: "app",
			expected:    "",
		},
		{
			name: "Container name without underscore separator",
			compose: "services:\n  app:\n    image: ghcr.io/localrivet/almatuck.com:latest\n    container_name: myapp\n    restart: unless-stopped\n",
			serviceName: "app",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectNameFromCompose([]byte(tt.compose), tt.serviceName)
			if result != tt.expected {
				t.Errorf("extractProjectNameFromCompose() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractContainerNameFromError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard Docker error with leading slash",
			input:    `The container name "/crowdgains_app" is already in use by container "abc123"`,
			expected: "crowdgains_app",
		},
		{
			name:     "Docker error without leading slash",
			input:    `The container name "myapp" is already in use by container "def456"`,
			expected: "myapp",
		},
		{
			name:     "Error response from daemon format",
			input:    `Error response from daemon: Conflict. The container name "/crowdgains_app" is already in use by container "2c7119bed2104cd1cd2dd01abbe7d15a2a414d1d1cc279aaf4f4a2bfb66dd4e8". You have to remove (or rename) that container to be able to reuse that name.`,
			expected: "crowdgains_app",
		},
		{
			name:     "Multiline output with container error",
			input:    "Container crowdgains_app  Creating\n Container crowdgains_app  Error response from daemon: Conflict. The container name \"/crowdgains_app\" is already in use",
			expected: "crowdgains_app",
		},
		{
			name:     "No match",
			input:    "Some other error message",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractContainerNameFromError(tt.input)
			if result != tt.expected {
				t.Errorf("extractContainerNameFromError() = %q, want %q", result, tt.expected)
			}
		})
	}
}
