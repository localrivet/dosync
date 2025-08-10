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

// performRollingUpdate restarts the service with the updated image tag using blue-green deployment
func performRollingUpdate(serviceName, filePath string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Starting blue-green deployment for service: %s", serviceName))
	
	// Step 1: Rename existing containers to temporary names
	if err := renameExistingContainers(serviceName, verbose); err != nil {
		return fmt.Errorf("failed to rename existing containers: %w", err)
	}
	
	// Step 2: Start new containers with updated image
	logVerbose(verbose, fmt.Sprintf("Starting new containers for service: %s", serviceName))
	
	// Extract project name from existing container names to maintain consistency
	projectName := extractProjectNameFromExistingContainers(serviceName, verbose)
	var cmd *exec.Cmd
	if projectName == "" {
		logVerbose(verbose, "Could not detect project name, using default docker compose behavior")
		cmd = exec.Command("docker", "compose", "-f", filePath, "up", "-d", "--no-deps", serviceName)
	} else {
		logVerbose(verbose, fmt.Sprintf("Using project name: %s", projectName))
		cmd = exec.Command("docker", "compose", "-f", filePath, "-p", projectName, "up", "-d", "--no-deps", serviceName)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		logVerbose(verbose, "New containers failed to start, restoring original containers")
		// Rollback: restore original containers
		if rollbackErr := restoreOriginalContainers(serviceName, verbose); rollbackErr != nil {
			return fmt.Errorf("failed to start new containers and rollback failed: %w, rollback error: %v", err, rollbackErr)
		}
		return fmt.Errorf("failed to start new containers: %w, output: %s", err, string(output))
	}
	
	// Step 3: Verify new containers are healthy
	if err := verifyContainersHealthy(serviceName, verbose); err != nil {
		logVerbose(verbose, "New containers failed health check, rolling back")
		// Rollback: stop new containers and restore originals
		stopCmd := exec.Command("docker", "compose", "-f", filePath, "stop", serviceName)
		stopCmd.Run() // Best effort
		if rollbackErr := restoreOriginalContainers(serviceName, verbose); rollbackErr != nil {
			return fmt.Errorf("health check failed and rollback failed: %w, rollback error: %v", err, rollbackErr)
		}
		return fmt.Errorf("new containers failed health check: %w", err)
	}
	
	// Step 4: Success! Remove the temporary containers
	if err := removeTemporaryContainers(serviceName, verbose); err != nil {
		logVerbose(verbose, fmt.Sprintf("Warning: failed to cleanup temporary containers: %v", err))
		// This is not a fatal error
	}
	
	logVerbose(verbose, fmt.Sprintf("Blue-green deployment completed successfully for service: %s", serviceName))
	return nil
}

// renameExistingContainers renames all containers for a service to temporary names
func renameExistingContainers(serviceName string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Renaming existing containers for service: %s", serviceName))
	
	// Find containers for this service
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.service=%s", serviceName))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find containers for service %s: %w", serviceName, err)
	}
	
	containerIDs := strings.Fields(string(output))
	for _, containerID := range containerIDs {
		// Get current container name
		nameCmd := exec.Command("docker", "inspect", "--format", "{{.Name}}", containerID)
		nameOutput, err := nameCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get container name for %s: %w", containerID, err)
		}
		
		currentName := strings.TrimSpace(strings.TrimPrefix(string(nameOutput), "/"))
		tempName := currentName + "-tmp"
		
		// Rename container
		renameCmd := exec.Command("docker", "rename", containerID, tempName)
		if err := renameCmd.Run(); err != nil {
			return fmt.Errorf("failed to rename container %s to %s: %w", currentName, tempName, err)
		}
		
		logVerbose(verbose, fmt.Sprintf("Renamed container %s to %s", currentName, tempName))
	}
	
	return nil
}

// restoreOriginalContainers restores temporary containers to their original names
func restoreOriginalContainers(serviceName string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Restoring original containers for service: %s", serviceName))
	
	// Find temporary containers for this service
	cmd := exec.Command("docker", "ps", "-a", "-q", "--filter", "name=-tmp")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find temporary containers: %w", err)
	}
	
	containerIDs := strings.Fields(string(output))
	for _, containerID := range containerIDs {
		// Get current container name
		nameCmd := exec.Command("docker", "inspect", "--format", "{{.Name}}", containerID)
		nameOutput, err := nameCmd.Output()
		if err != nil {
			continue // Skip if we can't get the name
		}
		
		currentName := strings.TrimSpace(strings.TrimPrefix(string(nameOutput), "/"))
		if !strings.HasSuffix(currentName, "-tmp") {
			continue // Skip if not a temp container
		}
		
		originalName := strings.TrimSuffix(currentName, "-tmp")
		
		// Rename back to original
		renameCmd := exec.Command("docker", "rename", containerID, originalName)
		if err := renameCmd.Run(); err != nil {
			logVerbose(verbose, fmt.Sprintf("Warning: failed to restore container %s: %v", currentName, err))
			continue
		}
		
		// Start the container
		startCmd := exec.Command("docker", "start", containerID)
		if err := startCmd.Run(); err != nil {
			logVerbose(verbose, fmt.Sprintf("Warning: failed to start restored container %s: %v", originalName, err))
		}
		
		logVerbose(verbose, fmt.Sprintf("Restored container %s to %s", currentName, originalName))
	}
	
	return nil
}

// verifyContainersHealthy checks if new containers are running and healthy
func verifyContainersHealthy(serviceName string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Verifying health of new containers for service: %s", serviceName))
	
	// Find containers for this service
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.service=%s", serviceName))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find containers for service %s: %w", serviceName, err)
	}
	
	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return fmt.Errorf("no running containers found for service %s", serviceName)
	}
	
	for _, containerID := range containerIDs {
		// Check container status
		statusCmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerID)
		statusOutput, err := statusCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check status of container %s: %w", containerID, err)
		}
		
		status := strings.TrimSpace(string(statusOutput))
		if status != "running" {
			return fmt.Errorf("container %s is not running (status: %s)", containerID, status)
		}
		
		// Check health if available
		healthCmd := exec.Command("docker", "inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}", containerID)
		healthOutput, err := healthCmd.Output()
		if err == nil {
			health := strings.TrimSpace(string(healthOutput))
			if health != "no-healthcheck" && health != "healthy" && health != "starting" {
				return fmt.Errorf("container %s health check failed (status: %s)", containerID, health)
			}
		}
		
		logVerbose(verbose, fmt.Sprintf("Container %s is healthy (status: %s)", containerID, status))
	}
	
	return nil
}

// extractProjectNameFromExistingContainers extracts project name from existing container names
func extractProjectNameFromExistingContainers(serviceName string, verbose bool) string {
	logVerbose(verbose, fmt.Sprintf("Extracting project name from existing containers for service: %s", serviceName))
	
	// Find containers for this service (including temp ones)
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}", "--filter", fmt.Sprintf("label=com.docker.compose.service=%s", serviceName))
	output, err := cmd.Output()
	if err != nil {
		logVerbose(verbose, fmt.Sprintf("Failed to find containers for service %s: %v", serviceName, err))
		return ""
	}
	
	containerNames := strings.Fields(string(output))
	for _, name := range containerNames {
		// Container names follow pattern: projectName-serviceName-replica
		// e.g., "solar-equity-hub-app-1" -> project: "solar-equity-hub", service: "app", replica: "1"
		
		// Remove -tmp suffix if present
		cleanName := strings.TrimSuffix(name, "-tmp")
		
		// Find the service name in the container name
		serviceIndex := strings.Index(cleanName, "-"+serviceName+"-")
		if serviceIndex > 0 {
			projectName := cleanName[:serviceIndex]
			logVerbose(verbose, fmt.Sprintf("Extracted project name '%s' from container '%s'", projectName, name))
			return projectName
		}
	}
	
	logVerbose(verbose, fmt.Sprintf("Could not extract project name from containers for service %s", serviceName))
	return ""
}

// removeTemporaryContainers removes containers with -tmp suffix
func removeTemporaryContainers(serviceName string, verbose bool) error {
	logVerbose(verbose, fmt.Sprintf("Removing temporary containers for service: %s", serviceName))
	
	// Find temporary containers
	cmd := exec.Command("docker", "ps", "-a", "-q", "--filter", "name=-tmp")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find temporary containers: %w", err)
	}
	
	containerIDs := strings.Fields(string(output))
	for _, containerID := range containerIDs {
		// Remove container
		removeCmd := exec.Command("docker", "rm", "-f", containerID)
		if err := removeCmd.Run(); err != nil {
			logVerbose(verbose, fmt.Sprintf("Warning: failed to remove temporary container %s: %v", containerID, err))
			continue
		}
		
		logVerbose(verbose, fmt.Sprintf("Removed temporary container %s", containerID))
	}
	
	return nil
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
