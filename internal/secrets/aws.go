package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AWSSecretsProvider implements SecretProvider for AWS Secrets Manager
// This implementation uses the AWS CLI to fetch secrets
type AWSSecretsProvider struct {
	region string
}

// NewAWSSecretsProvider creates a new AWS Secrets Manager provider
// region: AWS region (e.g., "us-east-1")
func NewAWSSecretsProvider(region string) *AWSSecretsProvider {
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1" // Default region
		}
	}
	return &AWSSecretsProvider{
		region: region,
	}
}

// Name returns the provider name
func (a *AWSSecretsProvider) Name() string {
	return "aws"
}

// GetSecret retrieves a secret from AWS Secrets Manager
// secretPath: secret name (e.g., "prod/github/pat")
func (a *AWSSecretsProvider) GetSecret(ctx context.Context, secretPath string) (string, error) {
	// Use AWS CLI to fetch the secret
	cmd := exec.CommandContext(ctx, "aws", "secretsmanager", "get-secret-value",
		"--secret-id", secretPath,
		"--region", a.region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("AWS CLI error: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute AWS CLI: %w", err)
	}

	// Parse the JSON response
	var result struct {
		SecretString string `json:"SecretString"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse AWS Secrets Manager response: %w", err)
	}

	return strings.TrimSpace(result.SecretString), nil
}
