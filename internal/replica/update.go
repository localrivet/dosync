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
// This function ONLY updates the compose file and runs docker compose up.
// Rolling updates, health checks, and rollback are handled by the strategy system (internal/strategy/*).
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

	logVerbose(verbose, fmt.Sprintf("Starting docker compose up for service: %s", serviceName))

	// Simply run docker compose up to start the new container
	// The strategy system (internal/strategy/*) handles rolling updates, health checks, and rollback
	// Use --force-recreate to handle stale containers that block recreation
	cmd := exec.Command("docker", "compose", "-f", filePath, "up", "-d", "--no-deps", "--force-recreate", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If we hit "container name already in use", try removing the stale container and retry
		if strings.Contains(string(output), "already in use") {
			logVerbose(verbose, "Container name conflict detected, attempting to remove stale container")
			// Extract container name from service and try to remove it
			rmCmd := exec.Command("docker", "rm", "-f", serviceName)
			rmCmd.Run() // Ignore error - container might not exist or have different name
			// Retry the compose up
			cmd = exec.Command("docker", "compose", "-f", filePath, "up", "-d", "--no-deps", "--force-recreate", serviceName)
			output, err = cmd.CombinedOutput()
		}
		if err != nil {
			logVerbose(verbose, fmt.Sprintf("Docker compose up failed: %s", string(output)))
			logVerbose(verbose, fmt.Sprintf("Docker compose up error: %v", err))
			return fmt.Errorf("failed to start service %s: %s, error: %w", serviceName, string(output), err)
		}
	}
	logVerbose(verbose, fmt.Sprintf("Docker compose up output: %s", string(output)))

	logVerbose(verbose, fmt.Sprintf("Service %s compose file updated and containers restarted", serviceName))
	return nil
}

// REMOVED: performRollingUpdate, renameExistingContainers, restoreOriginalContainers,
// verifyContainersHealthy, extractProjectNameFromExistingContainers, removeTemporaryContainers
//
// These functions implemented a duplicate rolling update system that conflicted with
// the advanced strategy system in internal/strategy/*.
//
// Following the ONE WAY OF DOING THINGS rule, rolling updates are now ONLY handled by:
// - internal/strategy/* - Rolling update strategies (one-at-a-time, percentage, blue-green, canary)
// - internal/health/* - Health checks (docker, http, tcp, command)
// - internal/rollback/* - Rollback and backup management
//
// UpdateDockerComposeAndRestart now simply updates the compose file and runs docker compose up.
// The strategy system handles all the complexity of rolling updates, health checks, and rollback.

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
