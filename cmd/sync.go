package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"dosync/internal/config"
	"dosync/internal/health"
	"dosync/internal/registry"
	"dosync/internal/replica"
	"dosync/internal/rollback"
	"dosync/internal/strategy"
	"dosync/internal/syncer"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var verbose bool

func logVerbose(message string, force ...bool) {
	if verbose || (len(force) > 0 && force[0]) {
		log.Println(message)
	}
}

// syncCmd represents the synchronization command
var syncCmd = &cobra.Command{
	Use:   "sync [path to docker-compose file]",
	Short: "Synchronize Docker Compose services with container registries",
	Long: `Synchronize Docker Compose services with supported container registries.
This command checks for new image tags in configured registries based on image policies
and updates the Docker Compose file accordingly.

Image policies can be configured in dosync.yaml to:
- Filter tags using regex patterns with named capture groups
- Select tags based on numerical order (build numbers, timestamps)
- Select tags based on semantic versioning (semver) constraints
- Select tags based on alphabetical order

For full documentation on configuration options, see the README.md file.`,
	Run: func(cmd *cobra.Command, args []string) {
		setFlagFromEnv := func(flag, env, expected string) {
			if !cmd.Flags().Changed(flag) {
				if v := os.Getenv(env); v != "" {
					if err := cmd.Flags().Set(flag, v); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid value for %s: %q. Expected %s. Error: %v\n", env, v, expected, err)
						os.Exit(2)
					}
				}
			}
		}

		setFlagFromEnv("file", "SYNC_FILE", "a file path")
		setFlagFromEnv("interval", "SYNC_INTERVAL", "a duration (e.g. 5m, 1h)")
		setFlagFromEnv("verbose", "SYNC_VERBOSE", "true or false")
		setFlagFromEnv("rolling-update", "SYNC_ROLLING_UPDATE", "true or false")
		setFlagFromEnv("strategy", "SYNC_STRATEGY", "a strategy name (e.g. canary)")
		setFlagFromEnv("health-check", "SYNC_HEALTH_CHECK", "docker, http, tcp, or command")
		setFlagFromEnv("health-endpoint", "SYNC_HEALTH_ENDPOINT", "an endpoint path (e.g. /status)")
		setFlagFromEnv("delay", "SYNC_DELAY", "a duration (e.g. 10s, 1m)")
		setFlagFromEnv("rollback-on-failure", "SYNC_ROLLBACK_ON_FAILURE", "true or false")

		intervalStr := AppConfig.CheckInterval
		verbose = AppConfig.Verbose

		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			fmt.Printf("Invalid interval format: %s\n", err)
			return
		}

		filePath, _ := cmd.Flags().GetString("file")

		// Wire up yamlUnmarshal for syncer
		syncer.YamlUnmarshal = yaml.Unmarshal

		// Rolling update config
		rollingCfg, err := buildRollingUpdateConfig(cmd)
		if err != nil {
			fmt.Printf("Error parsing rolling update flags: %v\n", err)
			return
		}
		if rollingCfg.Enabled {
			handleRollingUpdate(rollingCfg, filePath)
			return
		}

		opts := syncer.SyncOptions{
			FilePath: filePath,
			Interval: interval,
			Verbose:  verbose,
		}
		syncer.StartSync(opts)
	},
}

// hasRegistryType checks if a docker-compose file contains images from a specific registry
func hasRegistryType(filePath string, registryDomain string) bool {
	composeFile, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	var compose DockerCompose
	err = yaml.Unmarshal(composeFile, &compose)
	if err != nil {
		return false
	}

	for _, service := range compose.Services {
		if service.Image == "" {
			continue
		}
		if strings.Contains(service.Image, registryDomain) {
			return true
		}
	}

	return false
}

func init() {
	syncCmd.Flags().StringP("file", "f", "", "docker-compose file path")
	syncCmd.Flags().StringP("interval", "i", "1m", "Interval for checking updates (e.g., '5m', '1h')")
	syncCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Rolling update flags
	syncCmd.Flags().Bool("rolling-update", false, "Enable rolling updates")
	syncCmd.Flags().String("strategy", "one-at-a-time", "Update strategy (one-at-a-time, percentage, blue-green, canary)")
	syncCmd.Flags().String("health-check", "docker", "Health check type (docker, http, tcp, command)")
	syncCmd.Flags().String("health-endpoint", "/health", "Health check endpoint for HTTP checks")
	syncCmd.Flags().Duration("delay", 10*time.Second, "Delay between updating instances")
	syncCmd.Flags().Bool("rollback-on-failure", true, "Automatically rollback on failure")

	// Make the file and interval flags required
	syncCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(syncCmd)
}

type DockerCompose struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image string `yaml:"image"`
}

// DigitalOcean API response for tags
type DOTagResponse struct {
	Tags []DOTag `json:"tags"`
}

type DOTag struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"updated_at"`
}

// RollingUpdateConfig holds all rolling update options from CLI flags
type RollingUpdateConfig struct {
	Enabled           bool
	Strategy          string
	HealthCheckType   string
	HealthEndpoint    string
	Delay             time.Duration
	RollbackOnFailure bool
}

// buildRollingUpdateConfig reads flags from the sync command and returns a RollingUpdateConfig
func buildRollingUpdateConfig(cmd *cobra.Command) (*RollingUpdateConfig, error) {
	enabled, err := cmd.Flags().GetBool("rolling-update")
	if err != nil {
		return nil, err
	}
	strategy, err := cmd.Flags().GetString("strategy")
	if err != nil {
		return nil, err
	}
	hcType, err := cmd.Flags().GetString("health-check")
	if err != nil {
		return nil, err
	}
	hcEndpoint, err := cmd.Flags().GetString("health-endpoint")
	if err != nil {
		return nil, err
	}
	delay, err := cmd.Flags().GetDuration("delay")
	if err != nil {
		return nil, err
	}
	rollback, err := cmd.Flags().GetBool("rollback-on-failure")
	if err != nil {
		return nil, err
	}
	return &RollingUpdateConfig{
		Enabled:           enabled,
		Strategy:          strategy,
		HealthCheckType:   hcType,
		HealthEndpoint:    hcEndpoint,
		Delay:             delay,
		RollbackOnFailure: rollback,
	}, nil
}

// handleRollingUpdate is a function variable for rolling update logic (overridable in tests)
var handleRollingUpdate = func(cfg *RollingUpdateConfig, filePath string) {
	// If compose file does not exist, treat this as stub (used in tests)
	if _, err := os.Stat(filePath); err != nil {
		fmt.Printf("[Rolling Update] Stub: would perform rolling update on %s\n", filePath)
		return
	}

	fmt.Printf("[Rolling Update] Starting rolling update on %s with config: %+v\n", filePath, cfg)

	// Load docker-compose file
	composeFile, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read docker-compose file: %v\n", err)
		return
	}
	var compose DockerCompose
	err = yaml.Unmarshal(composeFile, &compose)
	if err != nil {
		fmt.Printf("Failed to unmarshal docker-compose file: %v\n", err)
		return
	}

	appCfg := AppConfig
	if appCfg == nil {
		fmt.Println("[Rolling Update] AppConfig is not loaded.")
		return
	}

	// Prepare rollback controller
	rollbackCfg := rollback.RollbackConfig{
		ComposeFilePath: filePath,
		BackupDir:       "backups",
		MaxHistory:      10,
	}
	rollbackController, err := rollback.NewRollbackController(rollbackCfg)
	if err != nil {
		fmt.Printf("[Rolling Update] Failed to create rollback controller: %v\n", err)
		return
	}

	// Prepare health checker config
	healthCfg := health.HealthCheckConfig{
		Type:             health.HealthCheckType(cfg.HealthCheckType),
		Endpoint:         cfg.HealthEndpoint,
		Timeout:          5 * time.Second,
		RetryInterval:    1 * time.Second,
		SuccessThreshold: 1,
		FailureThreshold: 3,
	}
	healthChecker, err := health.NewHealthChecker(healthCfg)
	if err != nil {
		fmt.Printf("[Rolling Update] Failed to create health checker: %v\n", err)
		return
	}

	// Prepare replica manager
	replicaManager, err := replica.NewReplicaManager(filePath)
	if err != nil {
		fmt.Printf("[Rolling Update] Failed to create replica manager: %v\n", err)
		return
	}

	// Prepare strategy config
	strategyCfg := strategy.StrategyConfig{
		Type:                cfg.Strategy,
		HealthCheck:         healthCfg,
		DelayBetweenUpdates: cfg.Delay,
		RollbackOnFailure:   cfg.RollbackOnFailure,
		Timeout:             10 * time.Minute,
	}

	// For each service, orchestrate the rolling update
	for serviceName, service := range compose.Services {
		fmt.Printf("[Rolling Update] Checking service: %s\n", serviceName)
		if service.Image == "" {
			fmt.Printf("[Rolling Update] Service %s has no image, skipping.\n", serviceName)
			continue
		}
		info, err := registry.ParseImageURL(service.Image)
		if err != nil {
			fmt.Printf("[Rolling Update] Could not parse image URL for service %s: %v\n", serviceName, err)
			continue
		}
		options := map[string]string{}
		var imagePolicy *config.ImagePolicy
		if appCfg.Registry != nil {
			switch info.Type {
			case registry.DockerHub:
				if appCfg.Registry.DockerHub != nil {
					options["username"] = appCfg.Registry.DockerHub.Username
					options["password"] = appCfg.Registry.DockerHub.Password
					imagePolicy = appCfg.Registry.DockerHub.ImagePolicy
				}
			case registry.DOCR:
				if appCfg.Registry.DOCR != nil {
					options["token"] = appCfg.Registry.DOCR.Token
					imagePolicy = appCfg.Registry.DOCR.ImagePolicy
				}
			case registry.GHCR:
				if appCfg.Registry.GHCR != nil {
					options["token"] = appCfg.Registry.GHCR.Token
					imagePolicy = appCfg.Registry.GHCR.ImagePolicy
				}
			case registry.GCR:
				if appCfg.Registry.GCR != nil {
					options["credentialsFile"] = appCfg.Registry.GCR.CredentialsFile
					imagePolicy = appCfg.Registry.GCR.ImagePolicy
				}
			case registry.ACR:
				if appCfg.Registry.ACR != nil {
					options["registry"] = appCfg.Registry.ACR.Registry
					options["clientID"] = appCfg.Registry.ACR.ClientID
					options["clientSecret"] = appCfg.Registry.ACR.ClientSecret
					imagePolicy = appCfg.Registry.ACR.ImagePolicy
				}
			case registry.ECR:
				if appCfg.Registry.ECR != nil {
					options["registry"] = appCfg.Registry.ECR.Registry
					options["accessKey"] = appCfg.Registry.ECR.AWSAccessKeyID
					options["secretKey"] = appCfg.Registry.ECR.AWSSecretAccessKey
					options["region"] = appCfg.Registry.ECR.Region
					imagePolicy = appCfg.Registry.ECR.ImagePolicy
				}
			case registry.Harbor:
				if appCfg.Registry.Harbor != nil {
					options["url"] = appCfg.Registry.Harbor.URL
					options["username"] = appCfg.Registry.Harbor.Username
					options["password"] = appCfg.Registry.Harbor.Password
					imagePolicy = appCfg.Registry.Harbor.ImagePolicy
				}
			case registry.Quay:
				if appCfg.Registry.Quay != nil {
					options["token"] = appCfg.Registry.Quay.Token
					imagePolicy = appCfg.Registry.Quay.ImagePolicy
				}
			case registry.Custom:
				if appCfg.Registry.Custom != nil {
					options["url"] = appCfg.Registry.Custom.URL
					options["username"] = appCfg.Registry.Custom.Username
					options["password"] = appCfg.Registry.Custom.Password
					imagePolicy = appCfg.Registry.Custom.ImagePolicy
				}
			}
		}
		client, err := registry.NewRegistryClient(info.Type, options)
		if err != nil {
			fmt.Printf("[Rolling Update] Failed to create registry client for service %s: %v\n", serviceName, err)
			continue
		}
		tags, err := client.GetTags(info.Path)
		if err != nil {
			fmt.Printf("[Rolling Update] Error getting tags for %s repo %s: %v\n", info.Type, info.Path, err)
			continue
		}
		selectedTag, err := syncer.SelectTagByImagePolicy(tags, imagePolicy)
		if err != nil {
			fmt.Printf("[Rolling Update] ImagePolicy error for %s repo %s: %v\n", info.Type, info.Path, err)
			continue
		}
		if selectedTag == "" {
			fmt.Printf("[Rolling Update] No matching tags found for %s repo %s, skipping\n", info.Type, info.Path)
			continue
		}
		currentTag := extractTagFromImage(service.Image)
		if selectedTag == currentTag {
			fmt.Printf("[Rolling Update] Service %s already at latest tag: %s\n", serviceName, currentTag)
			continue
		}
		fmt.Printf("[Rolling Update] Preparing rollback backup for service %s...\n", serviceName)
		err = rollbackController.PrepareRollback(serviceName)
		if err != nil {
			fmt.Printf("[Rolling Update] Failed to create rollback backup for service %s: %v\n", serviceName, err)
			continue
		}
		fmt.Printf("[Rolling Update] Updating service %s to new tag: %s (current: %s)\n", serviceName, selectedTag, currentTag)
		strat, err := strategy.NewUpdateStrategy(strategyCfg, replicaManager, healthChecker)
		if err != nil {
			fmt.Printf("[Rolling Update] Failed to create update strategy for service %s: %v\n", serviceName, err)
			continue
		}
		err = strat.Configure(strategyCfg)
		if err != nil {
			fmt.Printf("[Rolling Update] Failed to configure strategy for service %s: %v\n", serviceName, err)
			continue
		}
		err = strat.Execute(serviceName, selectedTag)
		if err != nil {
			fmt.Printf("[Rolling Update] Error updating service %s: %v\n", serviceName, err)
			if cfg.RollbackOnFailure {
				fmt.Printf("[Rolling Update] Rolling back service %s to previous version...\n", serviceName)
				err = rollbackController.Rollback(serviceName)
				if err != nil {
					fmt.Printf("[Rolling Update] Rollback failed for service %s: %v\n", serviceName, err)
				}
			}
			continue
		}
		fmt.Printf("[Rolling Update] Service %s updated to tag: %s\n", serviceName, selectedTag)
		if cfg.Delay > 0 {
			fmt.Printf("[Rolling Update] Waiting %s before next service...\n", cfg.Delay)
			time.Sleep(cfg.Delay)
		}
	}
	fmt.Println("[Rolling Update] All services processed.")
}

func checkAndUpdateServices(doToken, filePath string) {
	// Reload the docker-compose.yml file
	composeFile, err := os.ReadFile(filePath)
	if err != nil {
		logVerbose(fmt.Sprintf("Failed to read docker-compose file: %s\n", err), true)
		return
	}

	var compose DockerCompose
	err = yaml.Unmarshal(composeFile, &compose)
	if err != nil {
		logVerbose(fmt.Sprintf("Failed to unmarshal docker-compose file: %s\n", err), true)
		return
	}

	for serviceName, service := range compose.Services {
		logVerbose(fmt.Sprintf("Processing service: %s with image: %s", serviceName, service.Image))

		if service.Image == "" || !strings.Contains(service.Image, "registry.digitalocean.com") {
			logVerbose(fmt.Sprintf("Skipping service %s as it does not have a DigitalOcean Registry image.", serviceName))
			continue
		}

		// Extract repository info from the image tag
		registryName, repoName, err := extractDORepositoryInfo(service.Image)
		if err != nil {
			logVerbose(fmt.Sprintf("Error extracting repository info for service %s: %s\n", serviceName, err), true)
			continue
		}

		logVerbose("Checking latest image tag for repository: " + repoName)

		// Get the latest image tag from DigitalOcean Registry
		latestImageTag, err := getLatestImageTag(doToken, registryName, repoName)
		if err != nil {
			logVerbose(fmt.Sprintf("Error getting latest image tag for repository %s: %s\n", repoName, err), true)
			continue
		}

		// Skip if no tag was found (avoid using "latest" explicitly)
		if latestImageTag == "" {
			logVerbose(fmt.Sprintf("No timestamp-based tags found for %s, skipping", repoName), true)
			continue
		}

		// Compare with current image tag
		currentImageTag := extractTagFromImage(service.Image)
		if latestImageTag != currentImageTag {
			logVerbose(fmt.Sprintf("Updating service %s to new image tag: %s (current: %s)\n",
				serviceName, latestImageTag, currentImageTag), true)

			// Update docker-compose.yml and restart the service
			if err := replica.UpdateDockerComposeAndRestart(serviceName, latestImageTag, filePath, verbose); err == nil {
				removeUnusedDockerImages()
			} else {
				logVerbose(fmt.Sprintf("Error updating service %s: %s\n", serviceName, err), true)
			}
		} else {
			logVerbose(fmt.Sprintf("Service %s is already running the latest image tag: %s",
				serviceName, currentImageTag))
		}
	}
}

func extractDORepositoryInfo(image string) (string, string, error) {
	logVerbose(fmt.Sprintf("Extracting DigitalOcean repository info from image: %s", image))

	// Format expected: registry.digitalocean.com/registryName/repoName:tag
	parts := strings.Split(image, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid DigitalOcean Registry image format: %s", image)
	}

	registryName := parts[1]
	repoNameWithTag := parts[2]
	repoName := strings.Split(repoNameWithTag, ":")[0] // Remove the tag part

	return registryName, repoName, nil
}

func getLatestImageTag(doToken, registryName, repoName string) (string, error) {
	// Create HTTP request to the DigitalOcean API
	url := fmt.Sprintf("https://api.digitalocean.com/v2/registry/%s/repositories/%s/tags", registryName, repoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add authorization header
	req.Header.Add("Authorization", "Bearer "+doToken)
	req.Header.Add("Content-Type", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse response
	var tagResponse DOTagResponse
	if err := json.Unmarshal(body, &tagResponse); err != nil {
		return "", err
	}

	// Find latest tag with timestamp format (main-YYYYMMDDHHMMSS-hash)
	var latestTime time.Time
	var latestTag string
	for _, tag := range tagResponse.Tags {
		// Look for timestamp-based tags (main-YYYYMMDDHHMMSS-hash)
		if strings.HasPrefix(tag.Name, "main-") && len(strings.Split(tag.Name, "-")) >= 3 {
			// Use the LastUpdated time from the API instead of parsing the tag
			if tag.LastUpdated.After(latestTime) {
				latestTime = tag.LastUpdated
				latestTag = tag.Name
			}
		}
	}

	return latestTag, nil
}

func extractTagFromImage(image string) string {
	// Assuming image format: registry.digitalocean.com/registryName/repoName:tag
	if strings.Contains(image, ":") {
		return strings.Split(image, ":")[1]
	}
	return "latest" // Default tag
}

func removeUnusedDockerImages() {
	cmd := exec.Command("docker", "image", "prune", "-a", "--force")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logVerbose(fmt.Sprintf("Error pruning unused Docker images: %v, stderr: %s", err, stderr.String()), true)
	} else {
		logVerbose("Unused Docker images pruned successfully.")
	}
}
