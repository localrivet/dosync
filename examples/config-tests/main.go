package main

import (
	"fmt"
	"os"

	"dosync/internal/config"
)

func main() {
	// Set environment variables for testing
	os.Setenv("GITHUB_PAT", "test-token-value")
	os.Setenv("TEST_INTERVAL", "5m")
	os.Setenv("TEST_VERBOSE", "true")

	fmt.Println("=== Testing Environment Variable Expansion ===")
	fmt.Printf("GITHUB_PAT environment variable: %s\n", os.Getenv("GITHUB_PAT"))
	fmt.Printf("TEST_INTERVAL environment variable: %s\n", os.Getenv("TEST_INTERVAL"))
	fmt.Printf("TEST_VERBOSE environment variable: %s\n", os.Getenv("TEST_VERBOSE"))

	// Test with a simple config file that uses environment variables
	fmt.Println("\n=== Loading Config with Environment Variables ===")
	cfg, err := config.LoadConfig("dosync-env-test.yaml", nil)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded config successfully!\n")
	fmt.Printf("CheckInterval: %s\n", cfg.CheckInterval)
	fmt.Printf("Verbose: %t\n", cfg.Verbose)

	if cfg.Registry != nil && cfg.Registry.GHCR != nil {
		fmt.Printf("GHCR Token: %s\n", cfg.Registry.GHCR.Token)
		if cfg.Registry.GHCR.Token == "test-token-value" {
			fmt.Println("✅ Environment variable expansion WORKED!")
		} else {
			fmt.Printf("❌ Environment variable expansion FAILED! Expected 'test-token-value', got '%s'\n", cfg.Registry.GHCR.Token)
		}
	} else {
		fmt.Println("❌ GHCR config not found")
	}

	fmt.Println("\n=== Testing Complex Config ===")
	cfg2, err := config.LoadConfig("dosync-complex-env-test.yaml", nil)
	if err != nil {
		fmt.Printf("Error loading complex config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Complex config loaded successfully!\n")
	if cfg2.Registry != nil && cfg2.Registry.GHCR != nil {
		fmt.Printf("Complex GHCR Token: %s\n", cfg2.Registry.GHCR.Token)
	}
}
