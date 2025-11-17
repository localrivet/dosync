package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"dosync/internal/rollback"
)

// Example YAML configuration for registries:
//
// registry:
//   dockerhub:
//     username: myuser
//     password: ${DOCKERHUB_PASSWORD}
//     tag_pattern: "main-" # Optional: only consider tags starting with 'main-'
//   gcr:
//     credentials_file: /path/to/gcp.json
//     tag_pattern: "main-"
//   ghcr:
//     token: ${GITHUB_PAT}
//     tag_pattern: "main-"
//   acr:
//     tenant_id: your-tenant-id
//     client_id: your-client-id
//     client_secret: ${AZURE_CLIENT_SECRET}
//     registry: yourregistry.azurecr.io
//     tag_pattern: "main-"
//   quay:
//     token: ${QUAY_TOKEN}
//     tag_pattern: "main-"
//   harbor:
//     url: https://myharbor.domain.com
//     username: myuser
//     password: ${HARBOR_PASSWORD}
//     tag_pattern: "main-"
//   docr:
//     token: ${DOCR_TOKEN}
//     tag_pattern: "main-"
//   ecr:
//     aws_access_key_id: ${AWS_ACCESS_KEY_ID}
//     aws_secret_access_key: ${AWS_SECRET_ACCESS_KEY}
//     region: us-east-1
//     registry: 123456789012.dkr.ecr.us-east-1.amazonaws.com
//     tag_pattern: "main-"
//   custom:
//     url: https://custom.registry.com
//     username: myuser
//     password: ${CUSTOM_REGISTRY_PASSWORD}
//     tag_pattern: "main-"
//
// All fields are optional. Only specify the registries you need.
//
// You can use environment variable expansion for secrets.

// DashboardConfig holds settings for the web dashboard
type DashboardConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Port        string `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Pass        string `mapstructure:"pass"`
	IPWhitelist string `mapstructure:"ip_whitelist"`
}

// Config is the top-level configuration struct for the application
// Add new sections as needed (e.g., Logging, Deployment, etc.)
type Config struct {
	CheckInterval string                    `mapstructure:"CHECK_INTERVAL"`
	Verbose       bool                      `mapstructure:"VERBOSE"`
	Rollback      rollback.RollbackConfig   `mapstructure:"ROLLBACK"`
	Registry      *RegistryConfig           `mapstructure:"registry"`
	Dashboard     DashboardConfig           `mapstructure:"dashboard"`
	Services      map[string]*ServiceConfig `mapstructure:"services"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Skip bool `mapstructure:"skip"` // Skip this service from monitoring
}

// RegistryConfig holds optional config for all supported registries.
// Each field is a pointer to a registry-specific config struct. Only specify the registries you need.
type RegistryConfig struct {
	DockerHub *DockerHubConfig `mapstructure:"dockerhub"` // Docker Hub config (optional)
	GCR       *GCRConfig       `mapstructure:"gcr"`       // Google Container Registry config (optional)
	GHCR      *GHCRConfig      `mapstructure:"ghcr"`      // GitHub Container Registry config (optional)
	ACR       *ACRConfig       `mapstructure:"acr"`       // Azure Container Registry config (optional)
	Quay      *QuayConfig      `mapstructure:"quay"`      // Quay.io config (optional)
	Harbor    *HarborConfig    `mapstructure:"harbor"`    // Harbor config (optional)
	DOCR      *DOCRConfig      `mapstructure:"docr"`      // DigitalOcean Container Registry config (optional)
	ECR       *ECRConfig       `mapstructure:"ecr"`       // AWS ECR config (optional)
	Custom    *CustomConfig    `mapstructure:"custom"`    // Custom/private registry config (optional)
}

// ImagePolicy defines how to select the latest image tag for a repository.
type ImagePolicy struct {
	// FilterTags allows filtering and extracting values from tags using regex.
	FilterTags *struct {
		Pattern string `mapstructure:"pattern" yaml:"pattern"` // Regex pattern to filter tags (optional)
		Extract string `mapstructure:"extract" yaml:"extract"` // Named group to extract (e.g., "ts" or "semver") (optional)
	} `mapstructure:"filterTags" yaml:"filterTags"`

	// Policy defines the selection strategy: numerical, semver, or alphabetical.
	Policy *struct {
		Numerical *struct {
			Order string `mapstructure:"order" yaml:"order"` // "asc" or "desc"
		} `mapstructure:"numerical" yaml:"numerical"`
		Semver *struct {
			Range string `mapstructure:"range" yaml:"range"` // Semver range (e.g., ">=1.0.0 <2.0.0")
		} `mapstructure:"semver" yaml:"semver"`
		Alphabetical *struct {
			Order string `mapstructure:"order" yaml:"order"` // "asc" or "desc"
		} `mapstructure:"alphabetical" yaml:"alphabetical"`
	} `mapstructure:"policy" yaml:"policy"`
}

// DockerHubConfig holds Docker Hub credentials (all fields optional).
type DockerHubConfig struct {
	Username    string       `mapstructure:"username"`                         // Docker Hub username
	Password    string       `mapstructure:"password"`                         // Docker Hub password or token
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// GCRConfig holds Google Container Registry credentials.
type GCRConfig struct {
	CredentialsFile string       `mapstructure:"credentials_file"`                 // Path to GCP service account JSON
	ImagePolicy     *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// GHCRConfig holds GitHub Container Registry credentials.
type GHCRConfig struct {
	Token       string       `mapstructure:"token"`                            // GitHub Personal Access Token
	Username    string       `mapstructure:"username"`                         // GitHub username (optional, defaults to token owner)
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// ACRConfig holds Azure Container Registry credentials.
type ACRConfig struct {
	TenantID     string       `mapstructure:"tenant_id"`                        // Azure tenant ID
	ClientID     string       `mapstructure:"client_id"`                        // Azure client ID
	ClientSecret string       `mapstructure:"client_secret"`                    // Azure client secret
	Registry     string       `mapstructure:"registry"`                         // ACR registry domain
	ImagePolicy  *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// QuayConfig holds Quay.io credentials.
type QuayConfig struct {
	Token       string       `mapstructure:"token"`                            // Quay.io token
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// HarborConfig holds Harbor registry credentials.
type HarborConfig struct {
	URL         string       `mapstructure:"url"`                              // Harbor registry URL
	Username    string       `mapstructure:"username"`                         // Harbor username
	Password    string       `mapstructure:"password"`                         // Harbor password
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// DOCRConfig holds DigitalOcean Container Registry credentials.
type DOCRConfig struct {
	Token       string       `mapstructure:"token" yaml:"token"`
	Username    string       `mapstructure:"username" yaml:"username"`
	Password    string       `mapstructure:"password" yaml:"password"`
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"`
}

// ECRConfig holds AWS ECR credentials.
type ECRConfig struct {
	AWSAccessKeyID     string       `mapstructure:"aws_access_key_id"`                // AWS access key ID
	AWSSecretAccessKey string       `mapstructure:"aws_secret_access_key"`            // AWS secret access key
	Region             string       `mapstructure:"region"`                           // AWS region
	Registry           string       `mapstructure:"registry"`                         // ECR registry domain
	ImagePolicy        *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

// CustomConfig holds credentials for a custom/private registry.
type CustomConfig struct {
	URL         string       `mapstructure:"url"`                              // Registry URL
	Username    string       `mapstructure:"username"`                         // Username
	Password    string       `mapstructure:"password"`                         // Password
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
}

var (
	cfg     *Config
	cfgErr  error
	cfgOnce sync.Once
)

// sanitizeCredentials replaces sensitive credential fields with a redacted string
// to prevent accidental leakage in logs and error messages
func sanitizeCredentials(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	// Create a shallow copy
	sanitized := *cfg

	// Sanitize registry credentials if present
	if sanitized.Registry != nil {
		regCopy := *sanitized.Registry

		if regCopy.DockerHub != nil {
			dhCopy := *regCopy.DockerHub
			if dhCopy.Password != "" {
				dhCopy.Password = "***REDACTED***"
			}
			regCopy.DockerHub = &dhCopy
		}

		if regCopy.GHCR != nil {
			ghcrCopy := *regCopy.GHCR
			if ghcrCopy.Token != "" {
				ghcrCopy.Token = "***REDACTED***"
			}
			regCopy.GHCR = &ghcrCopy
		}

		if regCopy.DOCR != nil {
			docrCopy := *regCopy.DOCR
			if docrCopy.Token != "" {
				docrCopy.Token = "***REDACTED***"
			}
			if docrCopy.Password != "" {
				docrCopy.Password = "***REDACTED***"
			}
			regCopy.DOCR = &docrCopy
		}

		if regCopy.GCR != nil {
			gcrCopy := *regCopy.GCR
			if gcrCopy.CredentialsFile != "" {
				gcrCopy.CredentialsFile = "***REDACTED***"
			}
			regCopy.GCR = &gcrCopy
		}

		if regCopy.ACR != nil {
			acrCopy := *regCopy.ACR
			if acrCopy.ClientSecret != "" {
				acrCopy.ClientSecret = "***REDACTED***"
			}
			regCopy.ACR = &acrCopy
		}

		if regCopy.ECR != nil {
			ecrCopy := *regCopy.ECR
			if ecrCopy.AWSAccessKeyID != "" {
				ecrCopy.AWSAccessKeyID = "***REDACTED***"
			}
			if ecrCopy.AWSSecretAccessKey != "" {
				ecrCopy.AWSSecretAccessKey = "***REDACTED***"
			}
			regCopy.ECR = &ecrCopy
		}

		if regCopy.Harbor != nil {
			harborCopy := *regCopy.Harbor
			if harborCopy.Password != "" {
				harborCopy.Password = "***REDACTED***"
			}
			regCopy.Harbor = &harborCopy
		}

		if regCopy.Quay != nil {
			quayCopy := *regCopy.Quay
			if quayCopy.Token != "" {
				quayCopy.Token = "***REDACTED***"
			}
			regCopy.Quay = &quayCopy
		}

		if regCopy.Custom != nil {
			customCopy := *regCopy.Custom
			if customCopy.Password != "" {
				customCopy.Password = "***REDACTED***"
			}
			regCopy.Custom = &customCopy
		}

		sanitized.Registry = &regCopy
	}

	return &sanitized
}

// ValidateImagePolicy checks that the ImagePolicy is valid (regex, semver, order fields)
func ValidateImagePolicy(policy *ImagePolicy) error {
	if policy == nil {
		return nil
	}
	if policy.FilterTags != nil && policy.FilterTags.Pattern != "" {
		if _, err := regexp.Compile(policy.FilterTags.Pattern); err != nil {
			return fmt.Errorf("invalid image_policy.filterTags.pattern: %w", err)
		}
	}
	if policy.Policy != nil {
		if policy.Policy.Semver != nil && policy.Policy.Semver.Range != "" {
			if _, err := semver.NewConstraint(policy.Policy.Semver.Range); err != nil {
				return fmt.Errorf("invalid image_policy.policy.semver.range: %w", err)
			}
		}
		if policy.Policy.Numerical != nil {
			order := policy.Policy.Numerical.Order
			if order != "asc" && order != "desc" {
				return fmt.Errorf("invalid image_policy.policy.numerical.order: must be 'asc' or 'desc'")
			}
		}
		if policy.Policy.Alphabetical != nil {
			order := policy.Policy.Alphabetical.Order
			if order != "asc" && order != "desc" {
				return fmt.Errorf("invalid image_policy.policy.alphabetical.order: must be 'asc' or 'desc'")
			}
		}
	}
	return nil
}

// ValidateConfig checks all registry configs for valid image policies
func ValidateConfig(cfg *Config) error {
	if cfg == nil || cfg.Registry == nil {
		return nil
	}
	var err error
	check := func(policy *ImagePolicy, name string) {
		if e := ValidateImagePolicy(policy); e != nil && err == nil {
			err = fmt.Errorf("%s: %w", name, e)
		}
	}
	if cfg.Registry.DockerHub != nil {
		check(cfg.Registry.DockerHub.ImagePolicy, "dockerhub")
	}
	if cfg.Registry.GCR != nil {
		check(cfg.Registry.GCR.ImagePolicy, "gcr")
	}
	if cfg.Registry.GHCR != nil {
		check(cfg.Registry.GHCR.ImagePolicy, "ghcr")
	}
	if cfg.Registry.ACR != nil {
		check(cfg.Registry.ACR.ImagePolicy, "acr")
	}
	if cfg.Registry.Quay != nil {
		check(cfg.Registry.Quay.ImagePolicy, "quay")
	}
	if cfg.Registry.Harbor != nil {
		check(cfg.Registry.Harbor.ImagePolicy, "harbor")
	}
	if cfg.Registry.DOCR != nil {
		check(cfg.Registry.DOCR.ImagePolicy, "docr")
	}
	if cfg.Registry.ECR != nil {
		check(cfg.Registry.ECR.ImagePolicy, "ecr")
	}
	if cfg.Registry.Custom != nil {
		check(cfg.Registry.Custom.ImagePolicy, "custom")
	}
	return err
}

// Recursively expand env vars in all string fields of a struct
func ExpandEnvInStruct(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			field.SetString(os.ExpandEnv(field.String()))
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				ExpandEnvInStruct(field.Interface())
			}
		case reflect.Struct:
			ExpandEnvInStruct(field.Addr().Interface())
		}
	}
}

// LoadConfig loads configuration from file, env, and flags (in that order of precedence)
func LoadConfig(configPath string, flags *pflag.FlagSet) (*Config, error) {
	cfgOnce.Do(func() {
		v := viper.New()

		// Set config file if provided
		if configPath != "" {
			v.SetConfigFile(configPath)
		} else {
			// Look for dosync config files in common locations
			v.SetConfigName("dosync")
			v.AddConfigPath(".")    // Current directory
			v.AddConfigPath("/app") // Docker container path
			v.AddConfigPath("/etc/dosync/")
		}

		// Support YAML, JSON, TOML
		v.SetConfigType("yaml")

		// Bind environment variables (upper-case, underscores)
		v.AutomaticEnv()

		// Bind flags if provided
		if flags != nil {
			_ = v.BindPFlags(flags)
		}

		// Set defaults
		v.SetDefault("CHECK_INTERVAL", "1m")
		v.SetDefault("checkInterval", "1m")
		v.SetDefault("interval", "1m")
		v.SetDefault("VERBOSE", false)
		v.SetDefault("verbose", false)

		// Handle special VERBOSE environment variable cases before unmarshaling
		handleVerboseEnvironmentVariable(v)

		// Try to read config file
		if err := v.ReadInConfig(); err != nil {
			// Only fail if a specific config file was requested but not found
			if configPath != "" {
				cfg = nil
				cfgErr = fmt.Errorf("failed to read config file %s: %w", configPath, err)
				return
			}
			// Otherwise, continue with defaults and env vars
		}

		// Unmarshal into struct
		var c Config
		if err := v.Unmarshal(&c); err != nil {
			cfg = nil
			cfgErr = fmt.Errorf("failed to unmarshal config: %w", err)
			return
		}

		// Handle alternative field names manually
		handleAlternativeFieldNames(v, &c)

		// Recursively expand environment variables in all string fields
		ExpandEnvInStruct(&c)

		cfg = &c
		// Validate config after loading
		if err := ValidateConfig(cfg); err != nil {
			cfg = nil
			cfgErr = fmt.Errorf("invalid config: %w", err)
			return
		}
	})

	// Return any error that occurred during config loading
	if cfgErr != nil {
		return nil, cfgErr
	}
	if cfg == nil {
		return nil, fmt.Errorf("config loading failed")
	}
	return cfg, nil
}

// handleVerboseEnvironmentVariable processes VERBOSE env var before viper unmarshaling
func handleVerboseEnvironmentVariable(v *viper.Viper) {
	// Check if VERBOSE environment variable is set
	if verboseEnv := os.Getenv("VERBOSE"); verboseEnv != "" {
		// Parse the verbose flag and set it as a boolean in viper
		verboseBool := parseVerboseFlag(verboseEnv)
		v.Set("VERBOSE", verboseBool)
	}

	// Also check lowercase version
	if verboseEnv := os.Getenv("verbose"); verboseEnv != "" {
		verboseBool := parseVerboseFlag(verboseEnv)
		v.Set("verbose", verboseBool)
	}
}

// handleAlternativeFieldNames manually handles alternative field names that mapstructure doesn't support
func handleAlternativeFieldNames(v *viper.Viper, c *Config) {
	// Handle CheckInterval alternatives
	if c.CheckInterval == "1m" { // Check if it's still the default
		if val := v.GetString("checkInterval"); val != "" {
			c.CheckInterval = val
		} else if val := v.GetString("interval"); val != "" {
			c.CheckInterval = val
		}
	}

	// Handle Verbose alternatives and special cases
	if !c.Verbose { // Check if it's still the default
		// First try to get as boolean (normal case)
		if val := v.GetBool("verbose"); val {
			c.Verbose = val
		} else {
			// Handle special case where VERBOSE might be set to "--verbose" or similar flag format
			if verboseStr := v.GetString("VERBOSE"); verboseStr != "" {
				c.Verbose = parseVerboseFlag(verboseStr)
			} else if verboseStr := v.GetString("verbose"); verboseStr != "" {
				c.Verbose = parseVerboseFlag(verboseStr)
			}
		}
	}

	// Handle registry imagePolicy alternatives
	if c.Registry != nil {
		handleRegistryImagePolicyAlternatives(v, c.Registry)
	}
}

// parseVerboseFlag parses various verbose flag formats and returns a boolean
func parseVerboseFlag(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "--verbose", "-v", "verbose":
		return true // Handle flag-style values
	case "false", "0", "no", "off", "disabled", "":
		return false
	default:
		// Try to parse as boolean, fallback to false if invalid
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
		return false
	}
}

// handleRegistryImagePolicyAlternatives handles imagePolicy vs image_policy field names
func handleRegistryImagePolicyAlternatives(v *viper.Viper, registry *RegistryConfig) {
	// Helper function to check for imagePolicy alternative
	checkImagePolicy := func(registryName string, currentPolicy *ImagePolicy) *ImagePolicy {
		if currentPolicy != nil {
			return currentPolicy // Already set via image_policy
		}

		// Check for imagePolicy alternative
		if v.IsSet(registryName + ".imagePolicy") {
			var policy ImagePolicy
			if err := v.UnmarshalKey(registryName+".imagePolicy", &policy); err == nil {
				return &policy
			}
		}
		return nil
	}

	if registry.DockerHub != nil {
		registry.DockerHub.ImagePolicy = checkImagePolicy("registry.dockerhub", registry.DockerHub.ImagePolicy)
	}
	if registry.GCR != nil {
		registry.GCR.ImagePolicy = checkImagePolicy("registry.gcr", registry.GCR.ImagePolicy)
	}
	if registry.GHCR != nil {
		registry.GHCR.ImagePolicy = checkImagePolicy("registry.ghcr", registry.GHCR.ImagePolicy)
	}
	if registry.ACR != nil {
		registry.ACR.ImagePolicy = checkImagePolicy("registry.acr", registry.ACR.ImagePolicy)
	}
	if registry.Quay != nil {
		registry.Quay.ImagePolicy = checkImagePolicy("registry.quay", registry.Quay.ImagePolicy)
	}
	if registry.Harbor != nil {
		registry.Harbor.ImagePolicy = checkImagePolicy("registry.harbor", registry.Harbor.ImagePolicy)
	}
	if registry.DOCR != nil {
		registry.DOCR.ImagePolicy = checkImagePolicy("registry.docr", registry.DOCR.ImagePolicy)
	}
	if registry.ECR != nil {
		registry.ECR.ImagePolicy = checkImagePolicy("registry.ecr", registry.ECR.ImagePolicy)
	}
	if registry.Custom != nil {
		registry.Custom.ImagePolicy = checkImagePolicy("registry.custom", registry.Custom.ImagePolicy)
	}
}

// GetConfig returns the loaded config singleton
func GetConfig() *Config {
	if cfg == nil {
		panic("config not loaded: call LoadConfig first")
	}
	return cfg
}
