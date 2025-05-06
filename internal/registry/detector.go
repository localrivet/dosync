package registry

import (
	"fmt"
	"regexp"
	"strings"
)

// RegistryType represents the type of container registry.
type RegistryType string

const (
	DockerHub RegistryType = "docker.io"
	GCR       RegistryType = "gcr.io"
	GHCR      RegistryType = "ghcr.io"
	ACR       RegistryType = "azurecr.io"
	Quay      RegistryType = "quay.io"
	Harbor    RegistryType = "harbor" // Harbor typically uses a custom domain
	DOCR      RegistryType = "registry.digitalocean.com"
	ECR       RegistryType = "ecr" // AWS ECR uses account_id.dkr.ecr.region.amazonaws.com
	Custom    RegistryType = "custom"
)

// RegistryInfo contains parsed information about an image registry.
type RegistryInfo struct {
	Type   RegistryType
	Domain string
	Path   string
}

// regex patterns for common registries
var (
	gcrPattern    = regexp.MustCompile(`^gcr\.io/(.+)$`)
	ghcrPattern   = regexp.MustCompile(`^ghcr\.io/(.+)$`)
	acrPattern    = regexp.MustCompile(`^(.+\.azurecr\.io)/(.+)$`)
	quayPattern   = regexp.MustCompile(`^quay\.io/(.+)$`)
	docrPattern   = regexp.MustCompile(`^registry\.digitalocean\.com/(.+)$`)
	ecrPattern    = regexp.MustCompile(`^(\d+\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com)/(.+)$`)
	harborPattern = regexp.MustCompile(`^(.+)/(.+)$`) // Generic pattern for custom/Harbor
)

// ParseImageURL parses an image URL and returns information about the registry.
func ParseImageURL(imageURL string) (*RegistryInfo, error) {
	// Default to Docker Hub if no registry is specified
	if !strings.Contains(imageURL, "/") {
		return &RegistryInfo{
			Type:   DockerHub,
			Domain: string(DockerHub),
			Path:   imageURL, // Image name only
		}, nil
	}

	parts := strings.SplitN(imageURL, "/", 2)
	domain := parts[0]
	path := parts[1]

	switch {
	case gcrPattern.MatchString(imageURL):
		return &RegistryInfo{
			Type:   GCR,
			Domain: domain,
			Path:   path,
		}, nil
	case ghcrPattern.MatchString(imageURL):
		return &RegistryInfo{
			Type:   GHCR,
			Domain: domain,
			Path:   path,
		}, nil
	case acrPattern.MatchString(imageURL):
		matches := acrPattern.FindStringSubmatch(imageURL)
		return &RegistryInfo{
			Type:   ACR,
			Domain: matches[1],
			Path:   matches[2],
		}, nil
	case quayPattern.MatchString(imageURL):
		return &RegistryInfo{
			Type:   Quay,
			Domain: domain,
			Path:   path,
		}, nil
	case docrPattern.MatchString(imageURL):
		return &RegistryInfo{
			Type:   DOCR,
			Domain: domain,
			Path:   path,
		}, nil
	case ecrPattern.MatchString(imageURL):
		matches := ecrPattern.FindStringSubmatch(imageURL)
		return &RegistryInfo{
			Type:   ECR,
			Domain: matches[1],
			Path:   matches[2],
		}, nil
	case domain == string(DockerHub):
		return &RegistryInfo{
			Type:   DockerHub,
			Domain: domain,
			Path:   path,
		}, nil
	case !strings.Contains(domain, ".") && (strings.Count(imageURL, "/") == 1):
		return &RegistryInfo{
			Type:   DockerHub,
			Domain: string(DockerHub),
			Path:   imageURL,
		}, nil
	case harborPattern.MatchString(imageURL):
		// This is a generic pattern, might need refinement for specific Harbor detection
		// For now, assume it's a custom or Harbor registry if it has a domain and path
		return &RegistryInfo{
			Type:   Custom, // Or Harbor, depending on how we want to differentiate
			Domain: domain,
			Path:   path,
		}, nil
	default:
		// If it has a slash but doesn't match known patterns, treat as Docker Hub with a user/org
		return &RegistryInfo{
			Type:   DockerHub,
			Domain: string(DockerHub),
			Path:   imageURL,
		}, nil
	}
}

// Example of how to use the ParseImageURL function
func main() {
	urls := []string{
		"ubuntu",
		"gcr.io/google-containers/busybox",
		"ghcr.io/myuser/myimage",
		"myregistry.azurecr.io/myimage:latest",
		"quay.io/coreos/etcd",
		"myharbor.domain.com/myproject/myimage",
		"registry.digitalocean.com/myuser/myimage:latest",
		"123456789012.dkr.ecr.us-east-1.amazonaws.com/myimage:latest", // AWS ECR
		"custom.registry.com/path/to/image",
		"library/ubuntu",                  // Docker Hub explicit
		"docker.io/library/ubuntu",        // Docker Hub explicit with domain
		"docker.io/myuser/myimage:latest", // Docker Hub explicit with domain and tag
	}

	for _, url := range urls {
		info, err := ParseImageURL(url)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", url, err)
		} else {
			fmt.Printf("URL: %s, Type: %s, Domain: %s, Path: %s\n", url, info.Type, info.Domain, info.Path)
		}
	}
}
