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
// envFilePath is the path to the .env file to pass to docker compose (critical for preserving environment variables).
func UpdateDockerComposeAndRestart(serviceName, newTag, filePath string, verbose bool, registryAuth *RegistryAuth, envFilePath string) error {
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

	// Derive project name to ensure consistent network naming
	// Priority: 1) Extract from container_name in compose file (e.g., "almatuck_app" -> "almatuck")
	//           2) Fall back to compose file directory name
	// This is critical because DOSync runs from /app inside the container, so filepath.Base
	// would return "app" instead of the actual project name like "almatuck" or "crowdgains"
	projectName := extractProjectNameFromCompose(input, serviceName)
	if projectName == "" {
		projectName = filepath.Base(composeDir)
	}
	logVerbose(verbose, fmt.Sprintf("Starting docker compose up for service: %s (project: %s)", serviceName, projectName))

	// Proactively remove stale containers before attempting docker compose up
	// This prevents "container name already in use" errors from orphaned containers
	removeStaleContainers(projectName, serviceName, verbose)

	// Simply run docker compose up to start the new container
	// The strategy system (internal/strategy/*) handles rolling updates, health checks, and rollback
	// Use --force-recreate to handle stale containers that block recreation
	// Use --project-name to ensure consistent network naming
	// CRITICAL: Use --env-file to preserve environment variables from .env file
	cmdArgs := buildDockerComposeArgs(filePath, projectName, envFilePath, serviceName, true)
	cmd := exec.Command("docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If we hit "container name already in use", try removing the stale container and retry
		outputStr := string(output)
		if strings.Contains(outputStr, "already in use") {
			logVerbose(verbose, "Container name conflict detected, attempting to remove stale container")
			// Extract the actual container name from the error message
			// Error format: The container name "/crowdgains_app" is already in use
			containerName := extractContainerNameFromError(outputStr)
			if containerName != "" {
				logVerbose(verbose, fmt.Sprintf("Removing stale container: %s", containerName))
				rmCmd := exec.Command("docker", "rm", "-f", containerName)
				rmCmd.Run() // Ignore error - best effort removal
			}
			// Retry the compose up with project name
			cmd = exec.Command("docker", cmdArgs...)
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

// removeStaleContainers proactively removes any containers matching the service pattern
// before attempting docker compose up. This prevents "container name already in use" errors
// caused by orphaned containers that weren't properly cleaned up.
func removeStaleContainers(projectName, serviceName string, verbose bool) {
	// Build pattern to match: projectname_servicename and projectname_servicename-N
	basePattern := fmt.Sprintf("%s_%s", projectName, serviceName)

	// List containers matching the pattern (including stopped ones)
	cmd := exec.Command("docker", "ps", "-a", "--filter",
		fmt.Sprintf("name=^%s", basePattern), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return // Best effort - continue even if listing fails
	}

	// Remove each matching container
	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, container := range containers {
		if container == "" {
			continue
		}
		logVerbose(verbose, fmt.Sprintf("Removing stale container: %s", container))
		rmCmd := exec.Command("docker", "rm", "-f", container)
		rmCmd.Run() // Ignore error - best effort
	}
}

// extractContainerNameFromError extracts the container name from a Docker error message
// Error format: The container name "/crowdgains_app" is already in use
func extractContainerNameFromError(errorOutput string) string {
	// Match: container name "/some_name" is already in use
	re := regexp.MustCompile(`container name "/?([^"]+)" is already in use`)
	matches := re.FindStringSubmatch(errorOutput)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractProjectNameFromCompose extracts the project name from the compose file.
// Priority order:
// 1. Top-level `name:` field in compose.yaml (Docker Compose v2.x standard)
// 2. container_name patterns like "projectname_servicename"
// 3. Falls back to empty string (caller will use directory name)
func extractProjectNameFromCompose(composeContent []byte, serviceName string) string {
	// Priority 1: Check for top-level `name:` field (Docker Compose v2.x standard)
	// This is the correct way to define project name in compose.yaml
	// Pattern matches: "name: ballast-app" at the start of a line
	nameRe := regexp.MustCompile(`(?m)^name:\s*["']?([a-zA-Z0-9_-]+)["']?\s*$`)
	nameMatches := nameRe.FindSubmatch(composeContent)
	if len(nameMatches) >= 2 {
		return string(nameMatches[1])
	}

	// Priority 2: Look for container_name pattern: container_name: projectname_servicename
	// Examples: "almatuck_app", "crowdgains_postgres", "patternadvisor_dosync"
	re := regexp.MustCompile(`container_name:\s*([a-zA-Z0-9_-]+)_` + regexp.QuoteMeta(serviceName) + `\s*[\r\n]`)
	matches := re.FindSubmatch(composeContent)
	if len(matches) >= 2 {
		return string(matches[1])
	}

	// Priority 3: Try to find any container_name and extract prefix
	// This handles cases where serviceName doesn't match exactly (e.g., service "app" vs container "almatuck_app")
	reAny := regexp.MustCompile(`container_name:\s*([a-zA-Z0-9_-]+)_(?:app|postgres|dosync|web|api|db)\s*[\r\n]`)
	matchesAny := reAny.FindSubmatch(composeContent)
	if len(matchesAny) >= 2 {
		return string(matchesAny[1])
	}

	return ""
}

// buildDockerComposeArgs builds the arguments for docker compose up command.
// CRITICAL: This function ensures --env-file is passed when envFilePath is provided,
// which is essential for preserving environment variables during service restarts.
// NOTE: Only passes --project-name if projectName is non-empty. If empty, Docker Compose
// will use the `name:` field from compose.yaml or fall back to directory name.
func buildDockerComposeArgs(filePath, projectName, envFilePath, serviceName string, forceRecreate bool) []string {
	args := []string{"compose", "-f", filePath}

	// Only pass --project-name if we have an explicit project name to use.
	// If empty, Docker Compose will use its native resolution:
	// 1. `name:` field in compose.yaml (Docker Compose v2.x standard)
	// 2. Directory name as fallback
	if projectName != "" {
		args = append(args, "--project-name", projectName)
	}

	// CRITICAL: Add --env-file flag if envFilePath is provided
	// This ensures environment variables from .env file are preserved during restarts
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}

	// CRITICAL: Always use --no-deps to prevent Docker Compose from recreating dependencies.
	// Without this, services marked skip:true (like databases) can still be recreated
	// when updating a dependent service, causing DATA LOSS.
	//
	// Trade-off: This means depends_on health checks are not waited on by Docker Compose.
	// Services may start before dependencies are ready (DNS lookup failures).
	// Workaround: Apps should have retry logic for database connections, or
	// manually restart after deployment if needed.
	//
	// Future improvement: DOSync should implement its own health polling to wait
	// for dependencies before starting services, giving us both safety and health checks.
	args = append(args, "up", "-d", "--no-deps")
	if forceRecreate {
		args = append(args, "--force-recreate")
	}
	args = append(args, serviceName)

	return args
}

// BuildDockerComposeArgs is the exported version for use by other packages (e.g., syncer).
// CRITICAL: This function ensures --env-file is passed when envFilePath is provided.
func BuildDockerComposeArgs(filePath, projectName, envFilePath, serviceName string, forceRecreate bool) []string {
	return buildDockerComposeArgs(filePath, projectName, envFilePath, serviceName, forceRecreate)
}
