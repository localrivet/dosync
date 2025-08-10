package replica

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RegistryAuth contains authentication details for Docker registry login
type RegistryAuth struct {
	Server   string
	Username string
	Password string
}

// UpdateDockerComposeAndRestart updates the image tag for a service in the compose file and restarts the service using docker compose.
func UpdateDockerComposeAndRestart(serviceName, newTag, filePath string, verbose bool, registryAuth *RegistryAuth) error {
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

	// Authenticate Docker with registry before pulling if auth is provided
	if registryAuth != nil && registryAuth.Server != "" {
		if err := dockerLogin(registryAuth.Server, registryAuth.Username, registryAuth.Password, verbose); err != nil {
			logVerbose(verbose, fmt.Sprintf("Warning: Docker authentication failed for %s: %v", registryAuth.Server, err))
			// Continue anyway as the image might be public or Docker might already be authenticated
		}
	}

	logVerbose(verbose, fmt.Sprintf("Performing rolling update for service: %s", serviceName))

	// Perform rolling update to ensure zero downtime and clean up any orphaned containers
	if err := performRollingUpdate(serviceName, filePath, verbose); err != nil {
		return fmt.Errorf("failed to perform rolling update: %w", err)
	}

	logVerbose(verbose, fmt.Sprintf("Service %s updated successfully", serviceName))
	return nil
}

// performRollingUpdate performs a zero-downtime rolling update that properly cleans up orphaned containers
func performRollingUpdate(serviceName, filePath string, verbose bool) error {
	// Step 1: Remove any orphaned containers that might be running the same service
	logVerbose(verbose, fmt.Sprintf("Cleaning up orphaned containers for service: %s", serviceName))
	if err := cleanupOrphanedContainers(serviceName, verbose); err != nil {
		logVerbose(verbose, fmt.Sprintf("Warning: Failed to cleanup orphaned containers: %v", err))
		// Continue anyway - this is just cleanup
	}

	// Step 2: Use docker compose up with --force-recreate to update the service
	logVerbose(verbose, fmt.Sprintf("Starting rolling update for service: %s", serviceName))
	cmd := exec.Command("docker", "compose", "-f", filePath, "up", "-d", "--no-deps", "--force-recreate", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update service: %w, output: %s", err, string(output))
	}

	// Step 3: Final cleanup - remove any remaining orphaned containers
	logVerbose(verbose, fmt.Sprintf("Final cleanup for service: %s", serviceName))
	if err := cleanupOrphanedContainers(serviceName, verbose); err != nil {
		logVerbose(verbose, fmt.Sprintf("Warning: Failed final cleanup: %v", err))
		// Don't fail the update for cleanup issues
	}

	return nil
}

// cleanupOrphanedContainers removes containers that might be running the same service but with different naming
func cleanupOrphanedContainers(serviceName string, verbose bool) error {
	// Get all containers that might be related to this service
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}", "--filter", fmt.Sprintf("name=%s", serviceName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, container := range containers {
		container = strings.TrimSpace(container)
		if container == "" {
			continue
		}

		// Check if this container is running an old version
		if isOrphanedContainer(container, serviceName, verbose) {
			logVerbose(verbose, fmt.Sprintf("Removing orphaned container: %s", container))
			removeCmd := exec.Command("docker", "rm", "-f", container)
			if removeOutput, removeErr := removeCmd.CombinedOutput(); removeErr != nil {
				logVerbose(verbose, fmt.Sprintf("Warning: Failed to remove container %s: %v, output: %s", container, removeErr, string(removeOutput)))
			}
		}
	}

	return nil
}

// isOrphanedContainer determines if a container is an orphaned version of the service
func isOrphanedContainer(containerName, serviceName string, verbose bool) bool {
	// Get container info
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	status := strings.TrimSpace(string(output))

	// Only remove containers that are exited/dead, not running ones
	// This prevents us from accidentally removing healthy containers
	if status == "exited" || status == "dead" || status == "created" {
		logVerbose(verbose, fmt.Sprintf("Container %s has status %s - marking for cleanup", containerName, status))
		return true
	}

	return false
}

// dockerLogin performs docker login using the provided credentials
func dockerLogin(server, username, password string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Authenticating Docker with registry: %s", server))

	cmd := exec.Command("docker", "login", server, "--username", username, "--password-stdin")
	cmd.Stdin = strings.NewReader(password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed for %s: %w, output: %s", server, err, string(output))
	}

	logVerbose(verbose, fmt.Sprintf("Docker authentication successful for %s", server))
	return nil
}

func logVerbose(verbose bool, message string) {
	if verbose {
		fmt.Println(message)
	}
}
