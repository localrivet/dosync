package replica

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

// NewCompatibleDockerClient returns a Docker client that matches the server's API version.
func NewCompatibleDockerClient() (*client.Client, error) {
	// Try default (newest) version first
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err == nil {
		version, err := cli.ServerVersion(context.Background())
		if err == nil {
			return client.NewClientWithOpts(client.FromEnv, client.WithVersion(version.APIVersion))
		}
	}
	// If that fails, try with a fallback version (e.g., 1.47)
	fallbackVersion := "1.47"
	cli, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion(fallbackVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client with fallback version: %w", err)
	}
	version, err := cli.ServerVersion(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker server version with fallback: %w", err)
	}
	return client.NewClientWithOpts(client.FromEnv, client.WithVersion(version.APIVersion))
}
