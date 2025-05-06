package registry

import (
	"testing"
	"time" // Import time for time.Duration in test cases
)

func TestParseImageURL(t *testing.T) {
	tests := []struct {
		name     string
		imageURL string
		expected *RegistryInfo
		wantErr  bool
	}{
		{
			name:     "Docker Hub implicit",
			imageURL: "ubuntu",
			expected: &RegistryInfo{
				Type:   DockerHub,
				Domain: string(DockerHub),
				Path:   "ubuntu",
			},
			wantErr: false,
		},
		{
			name:     "Docker Hub explicit",
			imageURL: "library/ubuntu",
			expected: &RegistryInfo{
				Type:   DockerHub,
				Domain: string(DockerHub),
				Path:   "library/ubuntu",
			},
			wantErr: false,
		},
		{
			name:     "GCR",
			imageURL: "gcr.io/google-containers/busybox",
			expected: &RegistryInfo{
				Type:   GCR,
				Domain: "gcr.io",
				Path:   "google-containers/busybox",
			},
			wantErr: false,
		},
		{
			name:     "GHCR",
			imageURL: "ghcr.io/myuser/myimage",
			expected: &RegistryInfo{
				Type:   GHCR,
				Domain: "ghcr.io",
				Path:   "myuser/myimage",
			},
			wantErr: false,
		},
		{
			name:     "ACR",
			imageURL: "myregistry.azurecr.io/myimage:latest",
			expected: &RegistryInfo{
				Type:   ACR,
				Domain: "myregistry.azurecr.io",
				Path:   "myimage:latest",
			},
			wantErr: false,
		},
		{
			name:     "Quay.io",
			imageURL: "quay.io/coreos/etcd",
			expected: &RegistryInfo{
				Type:   Quay,
				Domain: "quay.io",
				Path:   "coreos/etcd",
			},
			wantErr: false,
		},
		{
			name:     "DigitalOcean Container Registry",
			imageURL: "registry.digitalocean.com/myuser/myimage:latest",
			expected: &RegistryInfo{
				Type:   DOCR,
				Domain: "registry.digitalocean.com",
				Path:   "myuser/myimage:latest",
			},
			wantErr: false,
		},
		{
			name:     "Custom Registry",
			imageURL: "custom.registry.com/path/to/image",
			expected: &RegistryInfo{
				Type:   Custom,
				Domain: "custom.registry.com",
				Path:   "path/to/image",
			},
			wantErr: false,
		},
		{
			name:     "Image with tag",
			imageURL: "ubuntu:latest",
			expected: &RegistryInfo{
				Type:   DockerHub,
				Domain: string(DockerHub),
				Path:   "ubuntu:latest",
			},
			wantErr: false,
		},
		{
			name:     "Image with digest",
			imageURL: "ubuntu@sha256:...",
			expected: &RegistryInfo{
				Type:   DockerHub,
				Domain: string(DockerHub),
				Path:   "ubuntu@sha256:...",
			},
			wantErr: false,
		},
		{
			name:     "Image with user/org and tag",
			imageURL: "myuser/myimage:latest",
			expected: &RegistryInfo{
				Type:   DockerHub,
				Domain: string(DockerHub),
				Path:   "myuser/myimage:latest",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseImageURL(tt.imageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImageURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Type != tt.expected.Type || got.Domain != tt.expected.Domain || got.Path != tt.expected.Path) {
				t.Errorf("ParseImageURL() got = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

// Dummy structs for testing config loading (to avoid dependency on actual config package)
type HealthCheckConfig struct {
	Type string
}
type RollbackConfig struct {
	Automatic bool
}
type NotificationConfig struct {
	Enabled bool
}

// Dummy Config struct for testing purposes
type Config struct {
	CheckInterval time.Duration
	Verbose       bool
	Rollback      RollbackConfig
	Notifications NotificationConfig
}
