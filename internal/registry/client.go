package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RegistryClient defines the interface for interacting with container registries
type RegistryClient interface {
	// GetTags retrieves all tags for a repository
	GetTags(repository string) ([]string, error)

	// GetManifest retrieves the manifest for a specific image tag
	GetManifest(repository, tag string) ([]byte, error)

	// Type returns the registry type this client is for
	Type() RegistryType
}

// BasicRegistryClient implements common registry client functionality
type BasicRegistryClient struct {
	authenticator Authenticator
	baseURL       string
}

// NewRegistryClient creates a registry client for the specified registry type
func NewRegistryClient(regType RegistryType, options map[string]string) (RegistryClient, error) {
	auth, err := CreateAuthenticator(regType, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	switch regType {
	case DockerHub:
		return &DockerHubClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       "https://registry.hub.docker.com/v2",
			},
		}, nil
	case GHCR:
		return &GHCRClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       "https://ghcr.io/v2",
			},
		}, nil
	case DOCR:
		return &DOCRClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       "https://api.digitalocean.com/v2/registry",
			},
		}, nil
	case GCR:
		return &GCRClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       "https://gcr.io/v2",
			},
			CredentialsFile: options["credentialsFile"],
		}, nil
	case ACR:
		return &ACRClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       options["registry"],
			},
			ClientID:     options["clientID"],
			ClientSecret: options["clientSecret"],
		}, nil
	case ECR:
		return &ECRClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       options["registry"],
			},
			AccessKey: options["accessKey"],
			SecretKey: options["secretKey"],
			Region:    options["region"],
		}, nil
	case Harbor:
		return &HarborClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       options["url"],
			},
			Username: options["username"],
			Password: options["password"],
		}, nil
	case Custom:
		url := options["url"]
		if url == "" {
			return nil, fmt.Errorf("url is required for custom registry")
		}
		return &CustomRegistryClient{
			BasicRegistryClient: BasicRegistryClient{
				authenticator: auth,
				baseURL:       url,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", regType)
	}
}

// DockerHubClient implements RegistryClient for Docker Hub
type DockerHubClient struct {
	BasicRegistryClient
}

func (c *DockerHubClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=100", repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication if needed
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tags := make([]string, 0, len(result.Results))
	for _, t := range result.Results {
		tags = append(tags, t.Name)
	}
	return tags, nil
}

func (c *DockerHubClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client - would require additional API calls
	return nil, fmt.Errorf("not implemented")
}

func (c *DockerHubClient) Type() RegistryType {
	return DockerHub
}

// GHCRClient implements RegistryClient for GitHub Container Registry
type GHCRClient struct {
	BasicRegistryClient
}

func (c *GHCRClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/tags/list", c.baseURL, repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication if needed
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Tags, nil
}

func (c *GHCRClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *GHCRClient) Type() RegistryType {
	return GHCR
}

// DOCRClient implements RegistryClient for DigitalOcean Container Registry
type DOCRClient struct {
	BasicRegistryClient
}

func (c *DOCRClient) GetTags(repository string) ([]string, error) {
	// First, extract registry and repo from the full repository name
	parts := ParseRepositoryParts(repository)
	if parts.Registry == "" || parts.Name == "" {
		return nil, fmt.Errorf("invalid repository format for DOCR: %s", repository)
	}

	url := fmt.Sprintf("%s/%s/repositories/%s/tags", c.baseURL, parts.Registry, parts.Name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tags := make([]string, 0, len(result.Tags))
	for _, t := range result.Tags {
		tags = append(tags, t.Name)
	}
	return tags, nil
}

func (c *DOCRClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *DOCRClient) Type() RegistryType {
	return DOCR
}

// CustomRegistryClient implements RegistryClient for custom/private registries
type CustomRegistryClient struct {
	BasicRegistryClient
}

func (c *CustomRegistryClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", c.baseURL, repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication if needed
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Tags, nil
}

func (c *CustomRegistryClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *CustomRegistryClient) Type() RegistryType {
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

// GCRClient implements RegistryClient for Google Container Registry
type GCRClient struct {
	BasicRegistryClient
	CredentialsFile string
}

func (c *GCRClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/tags/list", c.baseURL, repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication if needed
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GCR API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode GCR API response: %w", err)
	}

	return result.Tags, nil
}

func (c *GCRClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *GCRClient) Type() RegistryType {
	return GCR
}

// ACRClient implements RegistryClient for Azure Container Registry
type ACRClient struct {
	BasicRegistryClient
	ClientID     string
	ClientSecret string
}

func (c *ACRClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", c.baseURL, repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication if needed
	if err := c.authenticator.Authenticate(req); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ACR API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ACR API response: %w", err)
	}

	return result.Tags, nil
}

func (c *ACRClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *ACRClient) Type() RegistryType {
	return ACR
}

// ECRClient implements RegistryClient for AWS ECR
type ECRClient struct {
	BasicRegistryClient
	AccessKey string
	SecretKey string
	Region    string
}

func (c *ECRClient) GetTags(repository string) ([]string, error) {
	// For proper implementation, AWS SDK for Go should be used
	// This is a placeholder implementation
	if c.AccessKey == "" || c.SecretKey == "" || c.Region == "" {
		return nil, fmt.Errorf("AWS credentials (access key, secret key, region) are required for ECR access")
	}

	return nil, fmt.Errorf("ECR tag retrieval requires the AWS SDK - not implemented in this basic client")
}

func (c *ECRClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *ECRClient) Type() RegistryType {
	return ECR
}

// HarborClient implements RegistryClient for Harbor registry
type HarborClient struct {
	BasicRegistryClient
	Username string
	Password string
}

func (c *HarborClient) GetTags(repository string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", c.baseURL, repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Harbor API response: %w", err)
	}

	return result.Tags, nil
}

func (c *HarborClient) GetManifest(repository, tag string) ([]byte, error) {
	// Not implemented for basic client
	return nil, fmt.Errorf("not implemented")
}

func (c *HarborClient) Type() RegistryType {
	return Harbor
}
