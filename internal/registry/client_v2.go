package registry

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ContainerRegistryClient implements RegistryClient using Google's go-containerregistry library
// This handles all registry types (Docker Hub, GHCR, GCR, ACR, ECR, etc.) automatically
type ContainerRegistryClient struct {
	auth     authn.Authenticator
	username string
	password string
}

// NewContainerRegistryClient creates a new registry client using go-containerregistry
// It uses credentials from:
// 1. Explicitly provided username/password (Basic Auth)
// 2. Docker config file (~/.docker/config.json) - via DefaultKeychain
func NewContainerRegistryClient(username, password string) (*ContainerRegistryClient, error) {
	var auth authn.Authenticator

	if username != "" && password != "" {
		// Use explicit credentials (Basic Auth)
		auth = &authn.Basic{
			Username: username,
			Password: password,
		}
	} else {
		// Use anonymous auth - DefaultKeychain will be used per-request
		auth = authn.Anonymous
	}

	return &ContainerRegistryClient{
		auth:     auth,
		username: username,
		password: password,
	}, nil
}

// getRemoteOption returns the appropriate remote option for authentication
func (c *ContainerRegistryClient) getRemoteOption() remote.Option {
	if c.username != "" && c.password != "" {
		// Use explicit credentials
		return remote.WithAuth(c.auth)
	}
	// Use DefaultKeychain which reads from ~/.docker/config.json
	return remote.WithAuthFromKeychain(authn.DefaultKeychain)
}

// GetTags retrieves all tags for a repository
// repository should be in format: "owner/repo" or "registry/owner/repo"
// Examples:
//   - Docker Hub: "library/ubuntu" or "nginx"
//   - GHCR: "localrivet/almatuck.ai"
//   - GCR: "my-project/my-image"
func (c *ContainerRegistryClient) GetTags(repository string) ([]string, error) {
	// Parse the repository reference
	// This automatically detects the registry from the repository format
	// e.g., "ghcr.io/owner/repo" or "gcr.io/project/image"
	repo, err := name.NewRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("invalid repository format '%s': %w", repository, err)
	}

	// List tags using the remote package with authentication
	tags, err := remote.List(repo, c.getRemoteOption())
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for %s: %w", repository, err)
	}

	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags found for repository %s", repository)
	}

	return tags, nil
}

// GetManifest retrieves the manifest for a specific image tag
func (c *ContainerRegistryClient) GetManifest(repository, tag string) ([]byte, error) {
	// Parse the full image reference (repository + tag)
	ref, err := name.ParseReference(fmt.Sprintf("%s:%s", repository, tag))
	if err != nil {
		return nil, fmt.Errorf("invalid image reference '%s:%s': %w", repository, tag, err)
	}

	// Get the image descriptor
	desc, err := remote.Get(ref, c.getRemoteOption())
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest for %s:%s: %w", repository, tag, err)
	}

	// Return the raw manifest bytes
	return desc.Manifest, nil
}

// Type returns "universal" since this client handles all registry types
func (c *ContainerRegistryClient) Type() RegistryType {
	// We could add a new type "Universal" or just return the appropriate type
	// For now, return Custom to indicate it's a universal client
	return Custom
}
