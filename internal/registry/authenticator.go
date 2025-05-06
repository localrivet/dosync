package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Authenticator defines the interface for registry authentication
type Authenticator interface {
	// Authenticate prepares an HTTP request with authentication (adds headers, etc.)
	Authenticate(req *http.Request) error

	// Validate checks if credentials are valid and returns an error if not
	Validate() error

	// Type returns the registry type this authenticator is for
	Type() RegistryType
}

// DockerHubAuthenticator implements authentication for Docker Hub
type DockerHubAuthenticator struct {
	Username string
	Password string
	Token    string // Cached token after authentication
}

func (a *DockerHubAuthenticator) Authenticate(req *http.Request) error {
	if a.Token == "" {
		// If we don't have a token, try to get one
		if a.Username == "" || a.Password == "" {
			// For public images, we can proceed without authentication
			return nil
		}
		token, err := a.getToken()
		if err != nil {
			return err
		}
		a.Token = token
	}

	// Add the token to the request
	req.Header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

func (a *DockerHubAuthenticator) getToken() (string, error) {
	// Docker Hub authentication endpoint
	authURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", "library/ubuntu") // Example scope

	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	// Add basic auth if credentials are provided
	if a.Username != "" && a.Password != "" {
		req.SetBasicAuth(a.Username, a.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	return result.Token, nil
}

func (a *DockerHubAuthenticator) Validate() error {
	if a.Username == "" || a.Password == "" {
		// For DockerHub, we can use anonymous access for public images
		return nil
	}

	// Try to get a token to validate credentials
	_, err := a.getToken()
	return err
}

func (a *DockerHubAuthenticator) Type() RegistryType {
	return DockerHub
}

// GHCRAuthenticator implements authentication for GitHub Container Registry
type GHCRAuthenticator struct {
	Token string
}

func (a *GHCRAuthenticator) Authenticate(req *http.Request) error {
	if a.Token == "" {
		// For public images, we can proceed without authentication
		return nil
	}

	// GitHub accepts token as a bearer token
	req.Header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

func (a *GHCRAuthenticator) Validate() error {
	if a.Token == "" {
		// For GHCR, we can use anonymous access for public images
		return nil
	}

	// Test validation by making a simple API call
	req, err := http.NewRequest("GET", "https://ghcr.io/token", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	if err := a.Authenticate(req); err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid GitHub token")
	}

	return nil
}

func (a *GHCRAuthenticator) Type() RegistryType {
	return GHCR
}

// DOCRAuthenticator implements authentication for DigitalOcean Container Registry
type DOCRAuthenticator struct {
	Token string
}

func (a *DOCRAuthenticator) Authenticate(req *http.Request) error {
	if a.Token == "" {
		return fmt.Errorf("DigitalOcean API token is required")
	}

	req.Header.Set("Authorization", "Bearer "+a.Token)
	return nil
}

func (a *DOCRAuthenticator) Validate() error {
	if a.Token == "" {
		return fmt.Errorf("DigitalOcean API token is required")
	}

	// Test validation by making a simple API call
	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/registry", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	if err := a.Authenticate(req); err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid DigitalOcean API token")
	}

	return nil
}

func (a *DOCRAuthenticator) Type() RegistryType {
	return DOCR
}

// CustomAuthenticator implements authentication for custom registries
type CustomAuthenticator struct {
	URL      string
	Username string
	Password string
}

func (a *CustomAuthenticator) Authenticate(req *http.Request) error {
	// If no credentials are provided, we assume public access
	if a.Username == "" || a.Password == "" {
		return nil
	}

	// Custom registries typically use basic auth
	req.SetBasicAuth(a.Username, a.Password)
	return nil
}

func (a *CustomAuthenticator) Validate() error {
	if a.URL == "" {
		return fmt.Errorf("registry URL is required")
	}

	// For custom registries, credentials might be optional (public images)
	if a.Username == "" || a.Password == "" {
		return nil
	}

	// Test validation by making a simple API call to the catalog endpoint
	url := strings.TrimSuffix(a.URL, "/") + "/v2/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	if err := a.Authenticate(req); err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid registry credentials")
	}

	return nil
}

func (a *CustomAuthenticator) Type() RegistryType {
	return Custom
}

// CreateAuthenticator is a factory function to create the appropriate authenticator
func CreateAuthenticator(regType RegistryType, options map[string]string) (Authenticator, error) {
	switch regType {
	case DockerHub:
		return &DockerHubAuthenticator{
			Username: options["username"],
			Password: options["password"],
		}, nil
	case GHCR:
		return &GHCRAuthenticator{
			Token: options["token"],
		}, nil
	case DOCR:
		return &DOCRAuthenticator{
			Token: options["token"],
		}, nil
	case Custom:
		return &CustomAuthenticator{
			URL:      options["url"],
			Username: options["username"],
			Password: options["password"],
		}, nil
	default:
		return nil, fmt.Errorf("no authenticator available for registry type: %s", regType)
	}
}
