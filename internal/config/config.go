package config

import (
	"fmt"
	"os"
	"regexp"
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
	CheckInterval string                  `mapstructure:"CHECK_INTERVAL"`
	Verbose       bool                    `mapstructure:"VERBOSE"`
	Rollback      rollback.RollbackConfig `mapstructure:"ROLLBACK"`
	Registry      *RegistryConfig         `mapstructure:"registry"`
	Dashboard     DashboardConfig         `mapstructure:"dashboard"`
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
		Extract string `mapstructure:"extract" yaml:"extract"` // Named group to extract (e.g., "$ts" or "$semver") (optional)
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
	Token       string       `mapstructure:"token"`                            // DigitalOcean API token
	ImagePolicy *ImagePolicy `mapstructure:"image_policy" yaml:"image_policy"` // Advanced tag selection policy (optional)
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
	cfgOnce sync.Once
)

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

// LoadConfig loads configuration from file, env, and flags (in that order of precedence)
func LoadConfig(configPath string, flags *pflag.FlagSet) (*Config, error) {
	cfgOnce.Do(func() {
		v := viper.New()

		// Set config file if provided
		if configPath != "" {
			v.SetConfigFile(configPath)
		} else {
			v.SetConfigName("config")
			v.AddConfigPath(".")
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
		v.SetDefault("VERBOSE", false)

		// Read config file if present
		_ = v.ReadInConfig() // Ignore error if not found

		// Unmarshal into struct
		var c Config
		if err := v.Unmarshal(&c); err != nil {
			panic(fmt.Errorf("failed to unmarshal config: %w", err))
		}

		// Expand environment variables in registry credentials
		if c.Registry != nil {
			if c.Registry.DockerHub != nil {
				c.Registry.DockerHub.Username = os.ExpandEnv(c.Registry.DockerHub.Username)
				c.Registry.DockerHub.Password = os.ExpandEnv(c.Registry.DockerHub.Password)
			}
			if c.Registry.GCR != nil {
				c.Registry.GCR.CredentialsFile = os.ExpandEnv(c.Registry.GCR.CredentialsFile)
			}
			if c.Registry.GHCR != nil {
				c.Registry.GHCR.Token = os.ExpandEnv(c.Registry.GHCR.Token)
			}
			if c.Registry.ACR != nil {
				c.Registry.ACR.TenantID = os.ExpandEnv(c.Registry.ACR.TenantID)
				c.Registry.ACR.ClientID = os.ExpandEnv(c.Registry.ACR.ClientID)
				c.Registry.ACR.ClientSecret = os.ExpandEnv(c.Registry.ACR.ClientSecret)
				c.Registry.ACR.Registry = os.ExpandEnv(c.Registry.ACR.Registry)
			}
			if c.Registry.Quay != nil {
				c.Registry.Quay.Token = os.ExpandEnv(c.Registry.Quay.Token)
			}
			if c.Registry.Harbor != nil {
				c.Registry.Harbor.URL = os.ExpandEnv(c.Registry.Harbor.URL)
				c.Registry.Harbor.Username = os.ExpandEnv(c.Registry.Harbor.Username)
				c.Registry.Harbor.Password = os.ExpandEnv(c.Registry.Harbor.Password)
			}
			if c.Registry.DOCR != nil {
				c.Registry.DOCR.Token = os.ExpandEnv(c.Registry.DOCR.Token)
			}
			if c.Registry.ECR != nil {
				c.Registry.ECR.AWSAccessKeyID = os.ExpandEnv(c.Registry.ECR.AWSAccessKeyID)
				c.Registry.ECR.AWSSecretAccessKey = os.ExpandEnv(c.Registry.ECR.AWSSecretAccessKey)
				c.Registry.ECR.Region = os.ExpandEnv(c.Registry.ECR.Region)
				c.Registry.ECR.Registry = os.ExpandEnv(c.Registry.ECR.Registry)
			}
			if c.Registry.Custom != nil {
				c.Registry.Custom.URL = os.ExpandEnv(c.Registry.Custom.URL)
				c.Registry.Custom.Username = os.ExpandEnv(c.Registry.Custom.Username)
				c.Registry.Custom.Password = os.ExpandEnv(c.Registry.Custom.Password)
			}
		}

		cfg = &c
		// Validate config after loading
		if err := ValidateConfig(cfg); err != nil {
			panic(fmt.Errorf("invalid config: %w", err))
		}
	})
	return cfg, nil
}

// GetConfig returns the loaded config singleton
func GetConfig() *Config {
	if cfg == nil {
		panic("config not loaded: call LoadConfig first")
	}
	return cfg
}
