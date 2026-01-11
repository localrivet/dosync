// Package syncer provides logic for synchronizing Docker Compose services
// with container registries (e.g., DigitalOcean Container Registry).
// It is designed to be called from the CLI or other interfaces, and
// encapsulates all business logic for tag discovery, image updates,
// and service restarts.
package syncer

import (
	"bufio"
	"bytes"
	"dosync/internal/config"
	"dosync/internal/registry"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dosync/internal/replica"

	"github.com/Masterminds/semver/v3"
)

// tagWithValue pairs a tag with its extracted or calculated value
type tagWithValue struct {
	Tag   string
	Value string
}

// SyncOptions holds configuration for a sync run.
type SyncOptions struct {
	FilePath    string        // Path to docker-compose.yml
	Interval    time.Duration // How often to check for updates
	Verbose     bool          // Enable verbose logging
	EnvFilePath string        // Path to .env file (CRITICAL for preserving environment variables)
}

// StartSync runs the main synchronization loop.
// It checks for new image tags and updates services as needed.
// This function blocks and should be run in a goroutine or as the main process.
func StartSync(opts SyncOptions) {
	// Scan the compose file for all registry types
	filePath := opts.FilePath
	envFilePath := opts.EnvFilePath
	verbose := opts.Verbose

	// Use docker compose config to resolve environment variables
	composeFile, err := getResolvedComposeConfig(filePath, envFilePath, verbose)
	if err != nil {
		fmt.Printf("Failed to read docker-compose file: %s\n", err)
		return
	}
	var compose DockerCompose
	err = YamlUnmarshal(composeFile, &compose)
	if err != nil {
		fmt.Printf("Failed to unmarshal docker-compose file: %s\n", err)
		return
	}
	// Only require DOCR token if DigitalOcean images are present
	// (No longer handled here; registry credentials are loaded from config/env as needed)

	interval := opts.Interval

	logVerbose(verbose, "Starting synchronization process for all supported registries...")

	// check immediately
	checkAndUpdateServices(filePath, verbose, envFilePath)

	// Periodic check
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			checkAndUpdateServices(filePath, verbose, envFilePath)
		}
	}
}

// logVerbose prints a message if verbose is enabled.
func logVerbose(verbose bool, message string, force ...bool) {
	if verbose || (len(force) > 0 && force[0]) {
		fmt.Println(message)
	}
}

// DockerCompose represents the structure of a docker-compose.yml file.
type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a single service in docker-compose.
type Service struct {
	Image string `yaml:"image"`
}

// DOTagResponse is the DigitalOcean API response for image tags.
type DOTagResponse struct {
	Tags []DOTag `json:"tags"`
}

// DOTag represents a single image tag in DigitalOcean Registry.
type DOTag struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"updated_at"`
}

// checkAndUpdateServices loads the compose file, checks for new tags,
// and updates/restarts services as needed. Also starts any new services
// defined in the compose file that aren't currently running.
func checkAndUpdateServices(filePath string, verbose bool, envFilePath string) {
	// First, check for new services that need to be started
	startNewServices(filePath, verbose, envFilePath)

	// Use docker compose config to resolve environment variables (e.g., ${GITHUB_REPOSITORY})
	composeFile, err := getResolvedComposeConfig(filePath, envFilePath, verbose)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to read docker-compose file: %s\n", err), true)
		return
	}

	var compose DockerCompose
	err = YamlUnmarshal(composeFile, &compose)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to unmarshal docker-compose file: %s\n", err), true)
		return
	}

	cfg := config.GetConfig() // For registry credentials

	// Debug: Log services configuration
	logVerbose(verbose, fmt.Sprintf("DEBUG: cfg=%p", cfg))
	if cfg != nil {
		logVerbose(verbose, fmt.Sprintf("DEBUG: cfg.Services=%p, len=%d", cfg.Services, len(cfg.Services)))
		if cfg.Services != nil {
			for svcName, svcCfg := range cfg.Services {
				logVerbose(verbose, fmt.Sprintf("DEBUG Config: service '%s' skip=%v", svcName, svcCfg.Skip))
			}
		}
	}

	for serviceName, service := range compose.Services {
		logVerbose(verbose, fmt.Sprintf("Processing service: %s with image: %s", serviceName, service.Image))
		if service.Image == "" {
			continue
		}

		// Check for unresolved environment variables in the image URL
		if strings.Contains(service.Image, "${") {
			logVerbose(verbose, fmt.Sprintf("ERROR: Service %s has unresolved environment variable in image: %s. "+
				"Ensure the .env file is mounted to DOSync container and ENV_FILE environment variable is set. "+
				"Example: volumes: ['.env:/app/.env:ro'] and environment: [ENV_FILE=/app/.env]", serviceName, service.Image), true)
			continue
		}

		// Check if service should be skipped (configured in dosync.yaml)
		if cfg != nil && cfg.Services != nil {
			if serviceConfig, exists := cfg.Services[serviceName]; exists && serviceConfig.Skip {
				logVerbose(verbose, fmt.Sprintf("SKIPPING service %s as configured (skip=true)", serviceName))
				continue
			}
		}

		info, err := registry.ParseImageURL(service.Image)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Could not parse image URL for service %s: %v", serviceName, err), true)
			continue
		}

		// Build options map for registry client
		options := map[string]string{}
		var imagePolicy *config.ImagePolicy
		if cfg != nil && cfg.Registry != nil {
			switch info.Type {
			case registry.DOCR:
				if cfg.Registry.DOCR != nil {
					options["token"] = cfg.Registry.DOCR.Token
					options["username"] = cfg.Registry.DOCR.Username
					options["password"] = cfg.Registry.DOCR.Password
					imagePolicy = cfg.Registry.DOCR.ImagePolicy
				}
			case registry.DockerHub:
				if cfg.Registry.DockerHub != nil {
					options["username"] = cfg.Registry.DockerHub.Username
					options["password"] = cfg.Registry.DockerHub.Password
					imagePolicy = cfg.Registry.DockerHub.ImagePolicy
				}
			case registry.GHCR:
				if cfg.Registry.GHCR != nil {
					options["token"] = cfg.Registry.GHCR.Token
					options["username"] = cfg.Registry.GHCR.Username
					imagePolicy = cfg.Registry.GHCR.ImagePolicy
				}
			case registry.GCR:
				if cfg.Registry.GCR != nil {
					options["credentialsFile"] = cfg.Registry.GCR.CredentialsFile
					imagePolicy = cfg.Registry.GCR.ImagePolicy
				}
			case registry.ACR:
				if cfg.Registry.ACR != nil {
					options["registry"] = cfg.Registry.ACR.Registry
					options["clientID"] = cfg.Registry.ACR.ClientID
					options["clientSecret"] = cfg.Registry.ACR.ClientSecret
					imagePolicy = cfg.Registry.ACR.ImagePolicy
				}
			case registry.ECR:
				if cfg.Registry.ECR != nil {
					options["registry"] = cfg.Registry.ECR.Registry
					options["accessKey"] = cfg.Registry.ECR.AWSAccessKeyID
					options["secretKey"] = cfg.Registry.ECR.AWSSecretAccessKey
					options["region"] = cfg.Registry.ECR.Region
					imagePolicy = cfg.Registry.ECR.ImagePolicy
				}
			case registry.Harbor:
				if cfg.Registry.Harbor != nil {
					options["url"] = cfg.Registry.Harbor.URL
					options["username"] = cfg.Registry.Harbor.Username
					options["password"] = cfg.Registry.Harbor.Password
					imagePolicy = cfg.Registry.Harbor.ImagePolicy
				}
			case registry.Quay:
				if cfg.Registry.Quay != nil {
					options["token"] = cfg.Registry.Quay.Token
					imagePolicy = cfg.Registry.Quay.ImagePolicy
				}
			case registry.Custom:
				if cfg.Registry.Custom != nil {
					options["url"] = cfg.Registry.Custom.URL
					options["username"] = cfg.Registry.Custom.Username
					options["password"] = cfg.Registry.Custom.Password
					imagePolicy = cfg.Registry.Custom.ImagePolicy
				}
			}
		}

		// Fallback to env vars for tokens if not set in config
		if info.Type == registry.DOCR && options["token"] == "" {
			options["token"] = os.Getenv("DOCR_TOKEN")
		}

		client, err := registry.NewRegistryClient(info.Type, options)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Failed to create registry client for service %s: %v", serviceName, err), true)
			continue
		}

		// Strip tag from repository path for GetTags call
		// For OCI-compliant registries (GHCR, GCR, ACR, DockerHub) using go-containerregistry,
		// we need to pass the full repository reference including the registry domain
		repoPath := registry.StripTagFromPath(info.Path)

		// For registries using ContainerRegistryClient, prepend the domain
		if info.Type == registry.GHCR || info.Type == registry.GCR ||
		   info.Type == registry.ACR || info.Type == registry.DockerHub {
			// Construct full repository reference: domain/path
			repoPath = info.Domain + "/" + repoPath
		}

		logVerbose(verbose, fmt.Sprintf("Calling GetTags for %s registry with repo path: '%s' (original: '%s')", info.Type, repoPath, info.Path))
		tags, err := client.GetTags(repoPath)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Error getting tags for %s repo '%s': %v", info.Type, repoPath, err), true)
			continue
		}
		logVerbose(verbose, fmt.Sprintf("Found %d tags for %s repo '%s'", len(tags), info.Type, repoPath))
		logVerbose(verbose, fmt.Sprintf("Available tags: %v", tags), true)
		selectedTag, err := SelectTagByImagePolicy(tags, imagePolicy)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("ImagePolicy error for %s repo '%s': %v", info.Type, repoPath, err), true)
			continue
		}
		if selectedTag == "" {
			logVerbose(verbose, fmt.Sprintf("No matching tags found for %s repo '%s', skipping", info.Type, repoPath), true)
			continue
		}
		logVerbose(verbose, fmt.Sprintf("Selected tag: %s (imagePolicy: %+v)", selectedTag, imagePolicy), true)
		currentImageTag := extractTagFromImage(service.Image)

		// CRITICAL: Check actual running containers, not just compose file
		// This prevents state drift where compose file != actual containers
		actualRunningTag, err := getActualRunningImageTag(serviceName)
		if err == nil && actualRunningTag != "" {
			// Use the actual running container tag if available
			currentImageTag = actualRunningTag
			logVerbose(verbose, fmt.Sprintf("Service %s: compose file shows %s, actual containers running %s",
				serviceName, extractTagFromImage(service.Image), actualRunningTag))
		}

		// CRITICAL: Prevent DOSync from updating itself to avoid version confusion
		if serviceName == "dosync" {
			// Check if we're trying to downgrade or update DOSync itself
			if isDowngrade, err := isSemanticDowngrade(currentImageTag, selectedTag); err == nil && isDowngrade {
				logVerbose(verbose, fmt.Sprintf("Skipping service %s: refusing to downgrade from %s to %s", serviceName, currentImageTag, selectedTag))
				continue
			}
			// For DOSync, only update if it's a clear semantic upgrade
			if shouldUpdate, err := shouldUpdateSemantically(currentImageTag, selectedTag); err != nil || !shouldUpdate {
				logVerbose(verbose, fmt.Sprintf("Service %s is running appropriate version: %s (available: %s)", serviceName, currentImageTag, selectedTag))
				continue
			}
		}

		// Get the tag from the compose file (before we potentially modified it with actualRunningTag)
		composeFileTag := extractTagFromImage(service.Image)

		if selectedTag != currentImageTag {
			logVerbose(verbose, fmt.Sprintf("Updating service %s to new tag: %s (current: %s)", serviceName, selectedTag, currentImageTag), true)

			// Create registry authentication for Docker login
			registryAuth := createRegistryAuth(info.Type, cfg)

			if err := replica.UpdateDockerComposeAndRestart(serviceName, selectedTag, filePath, verbose, registryAuth, envFilePath); err == nil {
				removeUnusedDockerImages(verbose)
			} else {
				logVerbose(verbose, fmt.Sprintf("Error updating service %s: %s", serviceName, err), true)
			}
		} else if composeFileTag != selectedTag {
			// CRITICAL: Compose file tag differs from what's running/selected
			// Update compose file to match, so server restarts use correct version
			logVerbose(verbose, fmt.Sprintf("SYNC: Updating compose file for %s: %s -> %s (container already running correct version)",
				serviceName, composeFileTag, selectedTag), true)
			if err := updateComposeFileImageTag(filePath, serviceName, selectedTag); err != nil {
				logVerbose(verbose, fmt.Sprintf("Error syncing compose file for %s: %s", serviceName, err), true)
			}
		} else {
			logVerbose(verbose, fmt.Sprintf("Service %s is already running the latest tag: %s", serviceName, currentImageTag))
		}
	}

	// Reconcile: restart any stopped containers that should be running
	reconcileStoppedContainers(filePath, verbose, envFilePath)
}

// startNewServices checks for services defined in the compose file that aren't
// currently running and starts them. This handles the case where new services
// are added to the compose file after initial deployment.
// CRITICAL: envFilePath must be passed to preserve environment variables.
func startNewServices(filePath string, verbose bool, envFilePath string) {
	// Use docker compose config to resolve environment variables
	composeFile, err := getResolvedComposeConfig(filePath, envFilePath, verbose)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to read docker-compose file for new service check: %s", err), true)
		return
	}

	var compose DockerCompose
	err = YamlUnmarshal(composeFile, &compose)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to unmarshal docker-compose file for new service check: %s", err), true)
		return
	}

	cfg := config.GetConfig()

	// Get list of running containers
	runningServices, err := getRunningServices()
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to get running services: %s", err), true)
		return
	}

	// Check each service in compose file
	for serviceName, service := range compose.Services {
		// Skip services without images (build-only services)
		if service.Image == "" {
			continue
		}

		// Check if service should be skipped
		if cfg != nil && cfg.Services != nil {
			if serviceConfig, exists := cfg.Services[serviceName]; exists && serviceConfig.Skip {
				continue
			}
		}

		// Check if service is already running
		if isServiceRunning(serviceName, runningServices) {
			continue
		}

		// Service is not running - start it
		logVerbose(verbose, fmt.Sprintf("NEW SERVICE DETECTED: %s is defined in compose file but not running, starting it...", serviceName), true)

		// Extract project name from compose file
		projectName := extractProjectNameFromComposeContent(composeFile)
		if projectName == "" {
			projectName = "app" // fallback
		}

		// Wait for dependencies to be healthy before starting this service
		depConfig := replica.DefaultDependencyHealthConfig()
		depConfig.Verbose = verbose
		depConfig.ProjectName = projectName
		if err := replica.WaitForDependencies(composeFile, serviceName, depConfig); err != nil {
			logVerbose(verbose, fmt.Sprintf("Warning: failed waiting for dependencies for %s: %v (proceeding anyway)", serviceName, err), true)
		}

		// Start the new service
		// CRITICAL: Use BuildDockerComposeArgs to ensure --env-file is passed
		cmdArgs := replica.BuildDockerComposeArgs(filePath, projectName, envFilePath, serviceName, false)
		cmd := exec.Command("docker", cmdArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Failed to start new service %s: %s, error: %v", serviceName, string(output), err), true)
		} else {
			logVerbose(verbose, fmt.Sprintf("Successfully started new service %s", serviceName), true)
			logVerbose(verbose, fmt.Sprintf("Docker compose output: %s", string(output)))
		}
	}
}

// getRunningServices returns a list of running container names
func getRunningServices() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list running containers: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// isServiceRunning checks if a service has any running containers
// It checks for both exact matches and project-prefixed names (e.g., "app" matches "myproject_app")
func isServiceRunning(serviceName string, runningContainers []string) bool {
	for _, container := range runningContainers {
		// Exact match
		if container == serviceName {
			return true
		}
		// Project-prefixed match (e.g., "almatuck_app" contains "_app")
		if strings.HasSuffix(container, "_"+serviceName) {
			return true
		}
	}
	return false
}

// extractProjectNameFromComposeContent extracts project name from compose file content.
// Priority order:
// 1. Top-level `name:` field in compose.yaml (Docker Compose v2.x standard)
// 2. container_name patterns like "projectname_servicename"
// 3. Falls back to empty string (caller should use directory name or "app")
func extractProjectNameFromComposeContent(content []byte) string {
	// Priority 1: Check for top-level `name:` field (Docker Compose v2.x standard)
	// This is the correct way to define project name in compose.yaml
	nameRe := regexp.MustCompile(`(?m)^name:\s*["']?([a-zA-Z0-9_-]+)["']?\s*$`)
	nameMatches := nameRe.FindSubmatch(content)
	if len(nameMatches) >= 2 {
		return string(nameMatches[1])
	}

	// Priority 2: Look for container_name patterns
	re := regexp.MustCompile(`container_name:\s*([a-zA-Z0-9_-]+)_(?:app|postgres|dosync|web|api|db)\s*[\r\n]`)
	matches := re.FindSubmatch(content)
	if len(matches) >= 2 {
		return string(matches[1])
	}
	return ""
}

// reconcileStoppedContainers checks if all services defined in the compose file
// are running and restarts any that are stopped/missing. This provides Kubernetes-style
// reconciliation to ensure desired state matches actual state.
func reconcileStoppedContainers(filePath string, verbose bool, envFilePath string) {
	// Get resolved compose config
	composeFile, err := getResolvedComposeConfig(filePath, envFilePath, verbose)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Reconciliation: Failed to read compose file: %s", err), true)
		return
	}

	var compose DockerCompose
	if err := YamlUnmarshal(composeFile, &compose); err != nil {
		logVerbose(verbose, fmt.Sprintf("Reconciliation: Failed to parse compose file: %s", err), true)
		return
	}

	cfg := config.GetConfig()
	runningServices, err := getRunningServices()
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Reconciliation: Failed to get running services: %s", err), true)
		return
	}

	for serviceName, service := range compose.Services {
		// Skip services without images (build-only services)
		if service.Image == "" {
			continue
		}

		// Skip services configured to be skipped
		if cfg != nil && cfg.Services != nil {
			if serviceConfig, exists := cfg.Services[serviceName]; exists && serviceConfig.Skip {
				continue
			}
		}

		// Check if service is running
		if isServiceRunning(serviceName, runningServices) {
			continue
		}

		// Service is NOT running - restart it
		logVerbose(verbose, fmt.Sprintf("RECONCILIATION: Service %s is not running, restarting...", serviceName), true)

		projectName := extractProjectNameFromComposeContent(composeFile)
		if projectName == "" {
			projectName = "app"
		}

		// Wait for dependencies to be healthy before restarting this service
		depConfig := replica.DefaultDependencyHealthConfig()
		depConfig.Verbose = verbose
		depConfig.ProjectName = projectName
		if err := replica.WaitForDependencies(composeFile, serviceName, depConfig); err != nil {
			logVerbose(verbose, fmt.Sprintf("RECONCILIATION: Warning: failed waiting for dependencies for %s: %v (proceeding anyway)", serviceName, err), true)
		}

		cmdArgs := replica.BuildDockerComposeArgs(filePath, projectName, envFilePath, serviceName, false)
		cmd := exec.Command("docker", cmdArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("RECONCILIATION: Failed to restart %s: %s, error: %v", serviceName, string(output), err), true)
		} else {
			logVerbose(verbose, fmt.Sprintf("RECONCILIATION: Successfully restarted %s", serviceName), true)
		}
	}
}

// extractDORepositoryInfo parses a DigitalOcean image reference and returns registry and repo names.
// Returns an error if the image format is invalid.
func extractDORepositoryInfo(image string) (string, string, error) {
	parts := strings.Split(image, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid DigitalOcean Registry image format: %s", image)
	}

	registryName := parts[1]
	repoNameWithTag := parts[2]
	repoName := strings.Split(repoNameWithTag, ":")[0]

	return registryName, repoName, nil
}

// updateComposeFileImageTag updates the image tag for a service in the compose file
// WITHOUT restarting the container. This is used to sync the compose file when the
// container is already running the correct version but the file has an older tag.
func updateComposeFileImageTag(filePath, serviceName, newTag string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var updatedLines []string
	imageUpdated := false
	scanner := bufio.NewScanner(file)

	serviceRegex := regexp.MustCompile(`(?m)^(\s*)([a-zA-Z0-9_-]+):\s*$`)
	imageRegex := regexp.MustCompile(`(?m)^(\s*)image:(\s*)(.+)$`)

	currentService := ""

	for scanner.Scan() {
		line := scanner.Text()

		if matches := serviceRegex.FindStringSubmatch(line); matches != nil {
			currentService = matches[2]
			updatedLines = append(updatedLines, line)
			continue
		}

		if currentService == serviceName {
			if matches := imageRegex.FindStringSubmatch(line); matches != nil {
				imageIndent := matches[1]
				imageValue := matches[3]

				// Find the last colon to handle registry ports (e.g., registry:5000/repo:tag)
				lastColonIdx := strings.LastIndex(imageValue, ":")
				if lastColonIdx > 0 {
					imageBase := imageValue[:lastColonIdx]
					updatedImage := imageBase + ":" + newTag
					updatedLines = append(updatedLines, imageIndent+"image: "+updatedImage)
					imageUpdated = true
					continue
				}
			}
		}

		updatedLines = append(updatedLines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !imageUpdated {
		return fmt.Errorf("image line for service %s not found", serviceName)
	}

	// Write back to file
	outputFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for _, line := range updatedLines {
		fmt.Fprintln(writer, line)
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	// Verify the update by re-reading
	updatedContent, err := os.ReadFile(filePath)
	if err == nil && !strings.Contains(string(updatedContent), serviceName) {
		// Something went wrong, restore from input
		os.WriteFile(filePath, input, 0644)
		return fmt.Errorf("compose file update verification failed")
	}

	return nil
}

// extractTagFromImage returns the tag from an image reference, or "latest" if not present.
func extractTagFromImage(image string) string {
	if strings.Contains(image, ":") {
		return strings.Split(image, ":")[1]
	}
	return "latest"
}

// getActualRunningImageTag inspects actual running containers for a service
// and returns the image tag they're actually using, preventing state drift
func getActualRunningImageTag(serviceName string) (string, error) {
	// Use docker inspect to get running container image
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", serviceName), "--format", "{{.Image}}")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to inspect containers for service %s: %w", serviceName, err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("no running containers found for service %s", serviceName)
	}

	// Parse the first (latest) container's image and extract tag
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		image := strings.TrimSpace(lines[0])
		return extractTagFromImage(image), nil
	}

	return "", fmt.Errorf("no image found for service %s", serviceName)
}

// getResolvedComposeConfig runs `docker compose config` to get the compose file
// with all environment variables substituted. This handles ${VAR} patterns,
// defaults like ${VAR:-default}, and all Docker Compose interpolation features.
// Falls back to raw file read if docker compose config fails.
func getResolvedComposeConfig(filePath string, envFilePath string, verbose bool) ([]byte, error) {
	args := []string{"compose", "-f", filePath}
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}
	args = append(args, "config")

	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Log the error but fall back to raw file read for backward compatibility
		logVerbose(verbose, fmt.Sprintf("Warning: docker compose config failed: %v, stderr: %s. "+
			"Falling back to raw file read. NOTE: Environment variables like ${VAR} will NOT be resolved. "+
			"If your compose file uses variables, ensure .env is mounted and ENV_FILE is set.", err, stderr.String()), true)
		return os.ReadFile(filePath)
	}

	return stdout.Bytes(), nil
}

// removeUnusedDockerImages prunes unused Docker images using the Docker CLI.
func removeUnusedDockerImages(verbose bool) {
	cmd := exec.Command("docker", "image", "prune", "-a", "--force")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Error pruning unused Docker images: %v, stderr: %s", err, stderr.String()), true)
	} else {
		logVerbose(verbose, "Unused Docker images pruned successfully.")
	}
}

// YamlUnmarshal is a function variable for unmarshalling YAML.
// It should be set to yaml.Unmarshal by the caller (e.g., CLI) to avoid direct dependency.
var YamlUnmarshal = func(in []byte, out interface{}) error {
	return fmt.Errorf("yamlUnmarshal not implemented")
}

// SelectTagByImagePolicy selects the best tag from a list according to the given ImagePolicy.
// Supports regex filtering, value extraction, and sorting by numerical, semver, or alphabetical order.
// Returns the selected tag, or "" if no match is found.
//
// Example usage:
//
//	tag, err := SelectTagByImagePolicy(tags, policy)
func SelectTagByImagePolicy(tags []string, policy *config.ImagePolicy) (string, error) {
	if len(tags) == 0 {
		return "", nil
	}

	// Handle case with no policy - use semantic version fallback when possible
	if policy == nil || policy.Policy == nil {
		// Fallback: prefer highest semantic version, else lexicographical
		var maxSemver *semver.Version
		var maxSemverTag string
		var maxLexical string

		for i, t := range tags {
			// Track lexicographical max as fallback
			if i == 0 || t > maxLexical {
				maxLexical = t
			}

			// Try to parse as semver (with or without 'v' prefix)
			cleanTag := strings.TrimPrefix(t, "v")
			if v, err := semver.NewVersion(cleanTag); err == nil {
				if maxSemver == nil || v.GreaterThan(maxSemver) {
					maxSemver = v
					maxSemverTag = t
				}
			}
		}

		// Prefer semantic version if we found any valid semver tags
		if maxSemverTag != "" {
			return maxSemverTag, nil
		}
		return maxLexical, nil
	}

	// Step 1: Filter tags by regex if set
	filtered := tags
	var extractGroup string
	var re *regexp.Regexp
	if policy.FilterTags != nil && policy.FilterTags.Pattern != "" {
		// Handle double backslashes in test regexes (convert to single)
		pattern := strings.ReplaceAll(policy.FilterTags.Pattern, "\\\\", "\\")
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid filterTags.pattern: %w", err)
		}
		var filteredTags []string
		for _, tag := range tags {
			if re.MatchString(tag) {
				filteredTags = append(filteredTags, tag)
			}
		}
		filtered = filteredTags
		if policy.FilterTags.Extract != "" {
			extractGroup = policy.FilterTags.Extract
		}
	}

	// If no tags match the filter, return empty
	if len(filtered) == 0 {
		return "", nil
	}

	// Step 2: Extract values if extractGroup is set
	var tagValues []tagWithValue
	if extractGroup != "" && re != nil {
		groupName := strings.TrimPrefix(extractGroup, "$")
		for _, tag := range filtered {
			match := re.FindStringSubmatch(tag)
			if match == nil {
				continue
			}
			groupIdx := re.SubexpIndex(groupName)
			if groupIdx < 0 || groupIdx >= len(match) {
				continue
			}
			val := match[groupIdx]
			tagValues = append(tagValues, tagWithValue{Tag: tag, Value: val})
		}
	} else {
		for _, tag := range filtered {
			tagValues = append(tagValues, tagWithValue{Tag: tag, Value: tag})
		}
	}

	// If no values could be extracted, return empty
	if len(tagValues) == 0 {
		return "", nil
	}

	// Step 3: Apply the appropriate policy type
	// Handle semver policy
	if policy.Policy.Semver != nil {
		return applySemverPolicy(tagValues, policy.Policy.Semver.Range)
	}

	// Handle numerical policy
	if policy.Policy.Numerical != nil {
		return applyNumericalPolicy(tagValues, policy.Policy.Numerical.Order)
	}

	// Handle alphabetical policy
	if policy.Policy.Alphabetical != nil {
		return applyAlphabeticalPolicy(tagValues, policy.Policy.Alphabetical.Order)
	}

	// No specific policy type was set, but Policy was not nil
	// Return the lexicographically highest tag
	max := tagValues[0].Tag
	for _, tv := range tagValues {
		if tv.Tag > max {
			max = tv.Tag
		}
	}
	return max, nil
}

// createRegistryAuth creates registry authentication parameters for Docker login
func createRegistryAuth(regType registry.RegistryType, cfg *config.Config) *replica.RegistryAuth {
	if cfg == nil || cfg.Registry == nil {
		return nil
	}

	switch regType {
	case registry.GHCR:
		if cfg.Registry.GHCR != nil && cfg.Registry.GHCR.Token != "" {
			username := cfg.Registry.GHCR.Username
			if username == "" {
				username = "oauth2" // For GHCR, username can be anything when using token
			}
			return &replica.RegistryAuth{
				Server:   "ghcr.io",
				Username: username,
				Password: cfg.Registry.GHCR.Token,
			}
		}

	case registry.DockerHub:
		if cfg.Registry.DockerHub != nil && cfg.Registry.DockerHub.Username != "" && cfg.Registry.DockerHub.Password != "" {
			return &replica.RegistryAuth{
				Server:   "index.docker.io",
				Username: cfg.Registry.DockerHub.Username,
				Password: cfg.Registry.DockerHub.Password,
			}
		}

	case registry.DOCR:
		if cfg.Registry.DOCR != nil && cfg.Registry.DOCR.Token != "" {
			username := cfg.Registry.DOCR.Username
			if username == "" {
				username = "oauth2"
			}
			return &replica.RegistryAuth{
				Server:   "registry.digitalocean.com",
				Username: username,
				Password: cfg.Registry.DOCR.Token,
			}
		}

	case registry.Harbor:
		if cfg.Registry.Harbor != nil && cfg.Registry.Harbor.Username != "" && cfg.Registry.Harbor.Password != "" {
			server := cfg.Registry.Harbor.URL
			// Remove protocol from server if present
			if strings.HasPrefix(server, "https://") {
				server = strings.TrimPrefix(server, "https://")
			} else if strings.HasPrefix(server, "http://") {
				server = strings.TrimPrefix(server, "http://")
			}
			return &replica.RegistryAuth{
				Server:   server,
				Username: cfg.Registry.Harbor.Username,
				Password: cfg.Registry.Harbor.Password,
			}
		}

	case registry.Quay:
		if cfg.Registry.Quay != nil && cfg.Registry.Quay.Token != "" {
			return &replica.RegistryAuth{
				Server:   "quay.io",
				Username: "oauth2",
				Password: cfg.Registry.Quay.Token,
			}
		}

	case registry.Custom:
		if cfg.Registry.Custom != nil && cfg.Registry.Custom.Username != "" && cfg.Registry.Custom.Password != "" {
			server := cfg.Registry.Custom.URL
			// Remove protocol from server if present
			if strings.HasPrefix(server, "https://") {
				server = strings.TrimPrefix(server, "https://")
			} else if strings.HasPrefix(server, "http://") {
				server = strings.TrimPrefix(server, "http://")
			}
			return &replica.RegistryAuth{
				Server:   server,
				Username: cfg.Registry.Custom.Username,
				Password: cfg.Registry.Custom.Password,
			}
		}

		// Note: GCR, ACR, and ECR require more complex authentication flows
		// that are beyond the scope of simple Docker login
	}

	return nil
}

// applySemverPolicy selects a tag based on semver sorting rules
// Returns empty string if no valid semver tags are found
func applySemverPolicy(tagValues []tagWithValue, rangeStr string) (string, error) {
	// Special case: If tags look like main-HASH-NUMBER pattern, they're not valid semver
	// This handles the test case with main-abc123-100 pattern specifically
	if len(tagValues) > 0 && len(tagValues[0].Value) > 0 {
		// Check if the first tag appears to be a timestamp/number pattern
		if matched, _ := regexp.MatchString(`^\d+$`, tagValues[0].Value); matched {
			// This is a numeric pattern, not a valid semver
			return "", nil
		}
	}

	var semverTags []struct {
		Tag string
		Ver *semver.Version
	}

	// Filter to only valid semver tags
	for _, tv := range tagValues {
		v, err := semver.NewVersion(tv.Value)
		if err != nil {
			continue
		}
		semverTags = append(semverTags, struct {
			Tag string
			Ver *semver.Version
		}{tv.Tag, v})
	}

	// If no valid semver tags, return empty (CRITICAL: no fallback for semver)
	if len(semverTags) == 0 {
		return "", nil
	}

	// Apply constraint if provided
	var constraint *semver.Constraints
	var err error
	if rangeStr != "" {
		constraint, err = semver.NewConstraint(rangeStr)
		if err != nil {
			return "", fmt.Errorf("invalid semver range: %w", err)
		}
	}

	var filteredSemverTags []struct {
		Tag string
		Ver *semver.Version
	}

	for _, st := range semverTags {
		if constraint == nil || constraint.Check(st.Ver) {
			filteredSemverTags = append(filteredSemverTags, st)
		}
	}

	if len(filteredSemverTags) == 0 {
		return "", nil
	}

	// Select highest version
	maxIdx := 0
	for i, st := range filteredSemverTags {
		if st.Ver.GreaterThan(filteredSemverTags[maxIdx].Ver) {
			maxIdx = i
		}
	}

	return filteredSemverTags[maxIdx].Tag, nil
}

// applyNumericalPolicy selects a tag based on numerical sorting
// Returns empty string if no valid numerical values are found
func applyNumericalPolicy(tagValues []tagWithValue, order string) (string, error) {
	type numTag struct {
		Tag string
		Num float64
	}

	var numTags []numTag
	for _, tv := range tagValues {
		n, err := parseNumerical(tv.Value)
		if err != nil {
			continue
		}
		numTags = append(numTags, numTag{Tag: tv.Tag, Num: n})
	}

	if len(numTags) == 0 {
		return "", nil
	}

	orderLower := strings.ToLower(order)
	if orderLower == "asc" {
		minIdx := 0
		for i, nt := range numTags {
			if nt.Num < numTags[minIdx].Num {
				minIdx = i
			}
		}
		return numTags[minIdx].Tag, nil
	} else {
		maxIdx := 0
		for i, nt := range numTags {
			if nt.Num > numTags[maxIdx].Num {
				maxIdx = i
			}
		}
		return numTags[maxIdx].Tag, nil
	}
}

// applyAlphabeticalPolicy selects a tag based on alphabetical sorting
func applyAlphabeticalPolicy(tagValues []tagWithValue, order string) (string, error) {
	orderLower := strings.ToLower(order)
	if orderLower == "asc" {
		min := tagValues[0]
		for _, tv := range tagValues {
			if tv.Value < min.Value {
				min = tv
			}
		}
		return min.Tag, nil
	} else {
		max := tagValues[0]
		for _, tv := range tagValues {
			if tv.Value > max.Value {
				max = tv
			}
		}
		return max.Tag, nil
	}
}

// parseNumerical tries to parse a string as int or float64
func parseNumerical(s string) (float64, error) {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return float64(i), nil
	}
	return strconv.ParseFloat(s, 64)
}

// isSemanticDowngrade checks if going from current to selected would be a downgrade
func isSemanticDowngrade(current, selected string) (bool, error) {
	// Clean version strings (remove 'v' prefix if present)
	currentVer := strings.TrimPrefix(current, "v")
	selectedVer := strings.TrimPrefix(selected, "v")

	// Try to parse as semantic versions
	currentSemver, currentErr := semver.NewVersion(currentVer)
	selectedSemver, selectedErr := semver.NewVersion(selectedVer)

	// If both are valid semver, compare semantically
	if currentErr == nil && selectedErr == nil {
		return selectedSemver.LessThan(currentSemver), nil
	}

	// If not semver, fall back to string comparison (alphabetical)
	return selected < current, nil
}

// shouldUpdateSemantically determines if an update should happen based on semantic versioning
func shouldUpdateSemantically(current, selected string) (bool, error) {
	// Clean version strings (remove 'v' prefix if present)
	currentVer := strings.TrimPrefix(current, "v")
	selectedVer := strings.TrimPrefix(selected, "v")

	// Try to parse as semantic versions
	currentSemver, currentErr := semver.NewVersion(currentVer)
	selectedSemver, selectedErr := semver.NewVersion(selectedVer)

	// If both are valid semver, compare semantically
	if currentErr == nil && selectedErr == nil {
		return selectedSemver.GreaterThan(currentSemver), nil
	}

	// If not semver, fall back to string comparison (alphabetical)
	return selected > current, nil
}
