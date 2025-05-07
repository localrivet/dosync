package cmd

import (
	"fmt"
	"os"

	"dosync/internal/config"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	envFilePath string
	configPath  string
	AppConfig   *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "dosync",
	Short: "Automatically sync Docker Compose services with container registries",
	Long: `DOSync is a command-line tool that automates the process of synchronizing Docker 
Compose services with the latest images available in various container registries.

Supported registries include:
- Docker Hub
- GitHub Container Registry (GHCR)
- Google Container Registry (GCR)
- Azure Container Registry (ACR)
- DigitalOcean Container Registry (DOCR)
- Amazon Elastic Container Registry (ECR)
- Harbor
- Quay.io
- Custom Docker-compatible registries

This tool periodically checks for new image tags in the configured registries, updates the Docker Compose file 
according to your image policy preferences, and restarts the relevant Docker services to ensure that your deployments 
are always running the latest image versions.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Allow env file path to be set via ENV_FILE env var if not provided by flag
		if envFilePath == "" {
			envFilePath = os.Getenv("ENV_FILE")
		}

		if envFilePath != "" {
			// Load the specified .env file
			err := godotenv.Load(envFilePath)
			if err != nil {
				fmt.Printf("Error loading .env file from '%s'\n", envFilePath)
				return err
			}
		}

		// Allow config path to be set via CONFIG_PATH env var if not provided by flag
		if configPath == "" {
			configPath = os.Getenv("CONFIG_PATH")
		}

		// Load unified config
		cfg, err := config.LoadConfig(configPath, cmd.Flags())
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		AppConfig = cfg
		fmt.Printf("Loaded config: %+v\n", *cfg)
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&envFilePath, "env-file", "e", "", "Path to the .env file (optional)")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (optional)")
	// Flag is optional, not required
}
