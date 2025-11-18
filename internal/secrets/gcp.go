package secrets

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GCPSecretsProvider implements SecretProvider for GCP Secret Manager
// This implementation uses the gcloud CLI to fetch secrets
type GCPSecretsProvider struct {
	project string
}

// NewGCPSecretsProvider creates a new GCP Secret Manager provider
// project: GCP project ID (optional, will use gcloud default if empty)
func NewGCPSecretsProvider(project string) *GCPSecretsProvider {
	return &GCPSecretsProvider{
		project: project,
	}
}

// Name returns the provider name
func (g *GCPSecretsProvider) Name() string {
	return "gcp"
}

// GetSecret retrieves a secret from GCP Secret Manager
// secretPath: secret name in format "projects/{project}/secrets/{secret}/versions/latest"
//
//	or just "secret-name" (will use default project)
func (g *GCPSecretsProvider) GetSecret(ctx context.Context, secretPath string) (string, error) {
	// Build the full secret name
	fullPath := secretPath
	if !strings.HasPrefix(secretPath, "projects/") {
		if g.project != "" {
			fullPath = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", g.project, secretPath)
		} else {
			// Let gcloud use the default project
			fullPath = fmt.Sprintf("secrets/%s/versions/latest", secretPath)
		}
	}

	// Use gcloud CLI to fetch the secret (returns raw secret value)
	cmd := exec.CommandContext(ctx, "gcloud", "secrets", "versions", "access",
		"latest", "--secret", extractSecretName(fullPath))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gcloud CLI error: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute gcloud CLI: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// extractSecretName extracts the secret name from a full GCP secret path
func extractSecretName(path string) string {
	// projects/{project}/secrets/{secret}/versions/{version} -> {secret}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "secrets" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return path
}
