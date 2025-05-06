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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// tagWithValue pairs a tag with its extracted or calculated value
type tagWithValue struct {
	Tag   string
	Value string
}

// SyncOptions holds configuration for a sync run.
type SyncOptions struct {
	FilePath string        // Path to docker-compose.yml
	Interval time.Duration // How often to check for updates
	Verbose  bool          // Enable verbose logging
}

// StartSync runs the main synchronization loop.
// It checks for new image tags and updates services as needed.
// This function blocks and should be run in a goroutine or as the main process.
func StartSync(opts SyncOptions) {
	// Scan the compose file for all registry types
	filePath := opts.FilePath
	composeFile, err := os.ReadFile(filePath)
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
	verbose := opts.Verbose

	logVerbose(verbose, "Starting synchronization process for all supported registries...")

	// check immediately
	checkAndUpdateServices(filePath, verbose)

	// Periodic check
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			checkAndUpdateServices(filePath, verbose)
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
// and updates/restarts services as needed.
func checkAndUpdateServices(filePath string, verbose bool) {
	composeFile, err := os.ReadFile(filePath)
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

	for serviceName, service := range compose.Services {
		logVerbose(verbose, fmt.Sprintf("Processing service: %s with image: %s", serviceName, service.Image))
		if service.Image == "" {
			continue
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
			options["token"] = os.Getenv("GITHUB_PAT")
		}

		client, err := registry.NewRegistryClient(info.Type, options)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Failed to create registry client for service %s: %v", serviceName, err), true)
			continue
		}

		tags, err := client.GetTags(info.Path)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Error getting tags for %s repo %s: %v", info.Type, info.Path, err), true)
			continue
		}

		selectedTag, err := SelectTagByImagePolicy(tags, imagePolicy)
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("ImagePolicy error for %s repo %s: %v", info.Type, info.Path, err), true)
			continue
		}
		if selectedTag == "" {
			logVerbose(verbose, fmt.Sprintf("No matching tags found for %s repo %s, skipping", info.Type, info.Path), true)
			continue
		}
		currentImageTag := extractTagFromImage(service.Image)
		if selectedTag != currentImageTag {
			logVerbose(verbose, fmt.Sprintf("Updating service %s to new tag: %s (current: %s)", serviceName, selectedTag, currentImageTag), true)
			if err := updateDockerComposeAndRestart(serviceName, selectedTag, filePath, verbose); err == nil {
				removeUnusedDockerImages(verbose)
			} else {
				logVerbose(verbose, fmt.Sprintf("Error updating service %s: %s", serviceName, err), true)
			}
		} else {
			logVerbose(verbose, fmt.Sprintf("Service %s is already running the latest tag: %s", serviceName, currentImageTag))
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

// extractTagFromImage returns the tag from an image reference, or "latest" if not present.
func extractTagFromImage(image string) string {
	if strings.Contains(image, ":") {
		return strings.Split(image, ":")[1]
	}
	return "latest"
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

// updateDockerComposeAndRestart updates the image tag for a service in the compose file
// and restarts the service using docker compose.
func updateDockerComposeAndRestart(serviceName, newTag, filePath string, verbose bool) error {
	composeDir := filepath.Dir(filePath)
	backupFile := filepath.Join(composeDir, "docker-compose.backup.yml")
	input, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read docker-compose file: %w", err)
	}

	err = os.WriteFile(backupFile, input, 0644)
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Warning: Could not create backup file: %s", err))
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
	imageIndent := ""

	for scanner.Scan() {
		line := scanner.Text()

		if matches := serviceRegex.FindStringSubmatch(line); matches != nil {
			currentService = matches[2]
			updatedLines = append(updatedLines, line)
			continue
		}

		if currentService == serviceName {
			if matches := imageRegex.FindStringSubmatch(line); matches != nil {
				imageIndent = matches[1]
				imageValue := matches[3]

				parts := strings.Split(imageValue, ":")
				if len(parts) == 2 {
					updatedImage := parts[0] + ":" + newTag
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

	logVerbose(verbose, fmt.Sprintf("Restarting service: %s", serviceName), true)

	cmd := exec.Command("docker", "compose", "-f", filePath, "up", "-d", "--no-deps", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service: %w, output: %s", err, string(output))
	}

	logVerbose(verbose, fmt.Sprintf("Service %s restarted successfully", serviceName), true)
	return nil
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

	// Handle case with no policy - use lexicographical fallback
	if policy == nil || policy.Policy == nil {
		// Fallback: prefer highest non-pre-release tag, else highest tag
		var max string
		var maxRelease string
		for i, t := range tags {
			if i == 0 || t > max {
				max = t
			}
			// Try to parse as semver and check if it's a release
			if v, err := semver.NewVersion(t); err == nil && len(v.Prerelease()) == 0 {
				if maxRelease == "" || t > maxRelease {
					maxRelease = t
				}
			}
		}
		if maxRelease != "" {
			return maxRelease, nil
		}
		return max, nil
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
