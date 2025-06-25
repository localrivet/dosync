package main

import (
	"dosync/internal/config"
	"fmt"
	"os"
)

func main() {
	fmt.Println("=== DOSync Environment Variable Expansion Test ===")
	fmt.Println()

	// Set test environment variables
	testVars := map[string]string{
		"GITHUB_TOKEN":   "ghp_test_token_12345",
		"DOCKERHUB_USER": "testuser",
		"DOCKERHUB_PASS": "testpass123",
		"CHECK_INTERVAL": "5m",
	}

	fmt.Println("Setting test environment variables:")
	for key, value := range testVars {
		os.Setenv(key, value)
		fmt.Printf("  %s=%s\n", key, value)
	}
	fmt.Println()

	// Test different config files
	testConfigs := []string{
		"dosync-env-test.yaml",
		"dosync-complex-env-test.yaml",
	}

	for _, configFile := range testConfigs {
		fmt.Printf("Testing configuration: %s\n", configFile)
		fmt.Println("----------------------------------------")

		// Load config
		cfg, err := config.LoadConfig(configFile, nil)
		if err != nil {
			fmt.Printf("❌ Error loading config: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ Config loaded successfully!\n")
		fmt.Printf("CheckInterval: %s\n", cfg.CheckInterval)
		fmt.Printf("Verbose: %t\n", cfg.Verbose)

		// Test GHCR token expansion
		if cfg.Registry != nil && cfg.Registry.GHCR != nil {
			fmt.Printf("GHCR Token: %s\n", cfg.Registry.GHCR.Token)
			if cfg.Registry.GHCR.Token == testVars["GITHUB_TOKEN"] {
				fmt.Printf("✅ GHCR token expansion: SUCCESS\n")
			} else {
				fmt.Printf("❌ GHCR token expansion: FAILED (expected '%s', got '%s')\n",
					testVars["GITHUB_TOKEN"], cfg.Registry.GHCR.Token)
			}
		}

		// Test DockerHub credentials expansion
		if cfg.Registry != nil && cfg.Registry.DockerHub != nil {
			fmt.Printf("DockerHub Username: %s\n", cfg.Registry.DockerHub.Username)
			fmt.Printf("DockerHub Password: %s\n", cfg.Registry.DockerHub.Password)

			usernameOK := cfg.Registry.DockerHub.Username == testVars["DOCKERHUB_USER"]
			passwordOK := cfg.Registry.DockerHub.Password == testVars["DOCKERHUB_PASS"]

			if usernameOK && passwordOK {
				fmt.Printf("✅ DockerHub credentials expansion: SUCCESS\n")
			} else {
				fmt.Printf("❌ DockerHub credentials expansion: FAILED\n")
			}
		}

		fmt.Println()
	}

	fmt.Println("=== Test Summary ===")
	fmt.Println("Environment variable expansion allows you to use ${VAR_NAME} syntax")
	fmt.Println("in your dosync.yaml configuration files. This is especially useful")
	fmt.Println("for sensitive values like tokens and passwords.")
	fmt.Println()
	fmt.Println("Supported field name formats:")
	fmt.Println("  - checkInterval, interval, or CHECK_INTERVAL")
	fmt.Println("  - verbose or VERBOSE")
	fmt.Println("  - imagePolicy or image_policy")
}
