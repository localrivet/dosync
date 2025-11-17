package secrets

import (
	"context"
	"fmt"
	"strings"
)

// SecretProvider defines the interface for fetching secrets from various providers
type SecretProvider interface {
	// GetSecret retrieves a secret by its path/name
	GetSecret(ctx context.Context, secretPath string) (string, error)
	// Name returns the provider name for logging
	Name() string
}

// SecretReference represents a reference to a secret in configuration
type SecretReference struct {
	Provider string // vault, aws, gcp
	Path     string // secret path/name
}

// ParseSecretReference parses a secret reference string
// Supported formats:
//   - vault:secret/data/github/pat
//   - aws:prod/github/pat
//   - gcp:projects/123/secrets/github-pat
func ParseSecretReference(ref string) (*SecretReference, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid secret reference format: %s (expected provider:path)", ref)
	}

	provider := strings.ToLower(parts[0])
	path := parts[1]

	if path == "" {
		return nil, fmt.Errorf("empty secret path in reference: %s", ref)
	}

	return &SecretReference{
		Provider: provider,
		Path:     path,
	}, nil
}

// Manager manages multiple secret providers and resolves secret references
type Manager struct {
	providers map[string]SecretProvider
}

// NewManager creates a new secrets manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]SecretProvider),
	}
}

// RegisterProvider registers a secret provider
func (m *Manager) RegisterProvider(provider SecretProvider) {
	m.providers[provider.Name()] = provider
}

// ResolveSecret resolves a secret reference to its actual value
func (m *Manager) ResolveSecret(ctx context.Context, reference string) (string, error) {
	ref, err := ParseSecretReference(reference)
	if err != nil {
		return "", err
	}

	provider, ok := m.providers[ref.Provider]
	if !ok {
		return "", fmt.Errorf("unknown secret provider: %s", ref.Provider)
	}

	return provider.GetSecret(ctx, ref.Path)
}

// ResolveSecretOrDefault resolves a secret reference, or returns the value as-is if not a reference
func (m *Manager) ResolveSecretOrDefault(ctx context.Context, value string) string {
	// Check if it's a secret reference (contains ":")
	if !strings.Contains(value, ":") {
		return value
	}

	// Try to parse as secret reference
	ref, err := ParseSecretReference(value)
	if err != nil {
		// Not a valid secret reference, return as-is
		return value
	}

	// Check if the provider exists
	if _, ok := m.providers[ref.Provider]; !ok {
		// Provider not registered, return as-is
		return value
	}

	// Resolve the secret
	secret, err := m.ResolveSecret(ctx, value)
	if err != nil {
		// Failed to resolve, return as-is (this will cause authentication to fail later)
		return value
	}

	return secret
}
