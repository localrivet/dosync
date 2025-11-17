package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// VaultProvider implements SecretProvider for HashiCorp Vault
type VaultProvider struct {
	address string
	token   string
	client  *http.Client
}

// NewVaultProvider creates a new Vault secret provider
// address: Vault server address (e.g., "https://vault.example.com:8200")
// token: Vault authentication token
func NewVaultProvider(address, token string) *VaultProvider {
	return &VaultProvider{
		address: strings.TrimSuffix(address, "/"),
		token:   token,
		client:  &http.Client{},
	}
}

// Name returns the provider name
func (v *VaultProvider) Name() string {
	return "vault"
}

// GetSecret retrieves a secret from Vault
// secretPath: path to the secret (e.g., "secret/data/github/pat")
func (v *VaultProvider) GetSecret(ctx context.Context, secretPath string) (string, error) {
	// Build the full URL
	url := fmt.Sprintf("%s/v1/%s", v.address, secretPath)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Vault request: %w", err)
	}

	// Add Vault token header
	req.Header.Set("X-Vault-Token", v.token)

	// Make request
	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret from Vault: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Vault API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Vault response: %w", err)
	}

	// Extract the secret value
	// For KV v2, the value is in data.data.value
	if value, ok := result.Data.Data["value"].(string); ok {
		return value, nil
	}

	// If not in "value" field, try to get the first field
	for _, v := range result.Data.Data {
		if str, ok := v.(string); ok {
			return str, nil
		}
	}

	return "", fmt.Errorf("no string value found in Vault secret at path: %s", secretPath)
}
