package registry

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// RegistryClient defines the interface for interacting with container registries.
// All registry types (Docker Hub, GHCR, GCR, ACR, ECR, Harbor, Quay, DOCR, etc.)
// use the same implementation backed by Google's go-containerregistry library.
type RegistryClient interface {
	// GetTags retrieves all tags for a repository
	GetTags(repository string) ([]string, error)

	// GetManifest retrieves the manifest for a specific image tag
	GetManifest(repository, tag string) ([]byte, error)

	// Type returns the registry type this client is for
	Type() RegistryType
}

// registryClient implements RegistryClient using Google's go-containerregistry library.
// This is the ONE implementation that handles ALL OCI-compliant registries.
type registryClient struct {
	auth     authn.Authenticator
	username string
	password string
}

// NewRegistryClient creates a registry client for the specified registry type.
// Following the ONE WAY OF DOING THINGS rule, ALL registry types use the same
// implementation backed by Google's go-containerregistry library.
func NewRegistryClient(regType RegistryType, options map[string]string) (RegistryClient, error) {
	username := options["username"]
	password := options["password"]

	// For GHCR, password is the GitHub PAT (can be passed as "token" for backwards compatibility)
	if regType == GHCR && password == "" {
		password = options["token"]
	}

	// For ECR, use access key as username and secret key as password
	if regType == ECR {
		if username == "" {
			username = options["accessKey"]
		}
		if password == "" {
			password = options["secretKey"]
		}
	}

	// For ACR, use client ID as username and client secret as password
	if regType == ACR {
		if username == "" {
			username = options["clientID"]
		}
		if password == "" {
			password = options["clientSecret"]
		}
	}

	return newRegistryClient(username, password)
}

// newRegistryClient creates a new registry client using go-containerregistry.
// It uses credentials from:
// 1. Explicitly provided username/password (Basic Auth)
// 2. Docker config file (~/.docker/config.json) - via DefaultKeychain
func newRegistryClient(username, password string) (*registryClient, error) {
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

	return &registryClient{
		auth:     auth,
		username: username,
		password: password,
	}, nil
}

// getRemoteOption returns the appropriate remote option for authentication
func (c *registryClient) getRemoteOption() remote.Option {
	if c.username != "" && c.password != "" {
		// Use explicit credentials
		return remote.WithAuth(c.auth)
	}
	// Use DefaultKeychain which reads from ~/.docker/config.json
	return remote.WithAuthFromKeychain(authn.DefaultKeychain)
}

// GetTags retrieves all tags for a repository.
// repository should be in format: "owner/repo" or "registry/owner/repo"
// Examples:
//   - Docker Hub: "library/ubuntu" or "nginx"
//   - GHCR: "ghcr.io/localrivet/dosync"
//   - GCR: "gcr.io/my-project/my-image"
func (c *registryClient) GetTags(repository string) ([]string, error) {
	// Parse the repository reference
	// This automatically detects the registry from the repository format
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
func (c *registryClient) GetManifest(repository, tag string) ([]byte, error) {
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

// Type returns Custom since this client handles all registry types universally
func (c *registryClient) Type() RegistryType {
	return Custom
}

// RepositoryParts holds the parsed parts of a repository name
type RepositoryParts struct {
	Registry string // e.g., "myregistry" for DOCR
	Name     string // e.g., "myapp" for DOCR
	FullPath string // The full repository path without registry domain
}

// ParseRepositoryParts parses a repository string into its components
func ParseRepositoryParts(repository string) RepositoryParts {
	parts := RepositoryParts{
		FullPath: repository,
	}

	// For DOCR, the format is typically "myregistry/myapp"
	repoSplit := strings.Split(repository, "/")
	if len(repoSplit) > 1 {
		parts.Registry = repoSplit[0]
		parts.Name = strings.Join(repoSplit[1:], "/")
	}

	return parts
}
