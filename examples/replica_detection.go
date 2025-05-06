/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package main

import (
	"dosync/internal/replica"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// The path to the Docker Compose file
	// Default is the example Docker Compose file in the examples directory
	examplesDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	defaultComposeFile := filepath.Join(examplesDir, "docker-compose.yml")

	composeFile := defaultComposeFile
	serviceName := ""

	// Process command line arguments
	if len(os.Args) > 1 {
		// Check if help is requested
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			printUsage()
			return
		}
		composeFile = os.Args[1]
	}

	if len(os.Args) > 2 {
		serviceName = os.Args[2]
	}

	// Verify the compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		fmt.Printf("Error: Docker Compose file not found at %s\n", composeFile)
		fmt.Println("Please provide a valid path to a docker-compose.yml file")
		os.Exit(1)
	}

	fmt.Printf("Detecting replicas from %s\n", composeFile)

	// Example 1: Using the convenience function to get all replicas
	fmt.Println("\n=== Example 1: Get all replicas ===")
	allReplicas, err := replica.DetectAllReplicas(composeFile)
	if err != nil {
		handleError("Error detecting replicas", err)
	}

	// Print the results
	fmt.Printf("Found %d service(s) with replicas\n", len(allReplicas))
	for service, replicas := range allReplicas {
		fmt.Printf("Service: %s (%d replicas)\n", service, len(replicas))
		for i, r := range replicas {
			fmt.Printf("  Replica #%d: ID=%s, Container=%s, Status=%s\n", i+1, r.ReplicaID, r.ContainerID, r.Status)
		}
	}

	// Example 2: Get replicas for a specific service
	if serviceName != "" {
		fmt.Println("\n=== Example 2: Get replicas for a specific service ===")
		serviceReplicas, err := replica.DetectServiceReplicas(composeFile, serviceName)
		if err != nil {
			handleError(fmt.Sprintf("Error detecting replicas for service %s", serviceName), err)
		}

		fmt.Printf("Service: %s (%d replicas)\n", serviceName, len(serviceReplicas))
		for i, r := range serviceReplicas {
			fmt.Printf("  Replica #%d: ID=%s, Container=%s, Status=%s\n", i+1, r.ReplicaID, r.ContainerID, r.Status)
		}
	}

	// Example 3: Using the full ReplicaManager API
	fmt.Println("\n=== Example 3: Using the ReplicaManager API ===")
	manager, err := replica.NewReplicaManagerWithAllDetectors(composeFile)
	if err != nil {
		handleError("Error creating ReplicaManager", err)
	}

	// Get all replicas
	managerReplicas, err := manager.GetAllReplicas()
	if err != nil {
		handleError("Error getting replicas from manager", err)
	}

	// Print a summary
	fmt.Println("Replica count summary:")
	for service, replicas := range managerReplicas {
		fmt.Printf("  %s: %d replicas\n", service, len(replicas))
	}

	// Example 4: Using the stub implementation for testing
	fmt.Println("\n=== Example 4: Using stub implementations (for testing) ===")
	stubManager, err := replica.CreateStubReplicaManager(composeFile)
	if err != nil {
		handleError("Error creating stub ReplicaManager", err)
	}

	// Since the stub is empty by default, we won't see any replicas
	// In a real test, you would populate the stub with test data
	stubReplicas, err := stubManager.GetAllReplicas()
	if err != nil {
		handleError("Error getting replicas from stub manager", err)
	}

	fmt.Printf("Stub replica count: %d services\n", len(stubReplicas))

	// Provide instructions for running Docker Compose if no replicas were found
	if len(allReplicas) == 0 {
		fmt.Println("\n=== No replicas found ===")
		fmt.Println("To see actual replicas, you need to start Docker containers using:")
		fmt.Printf("cd %s && docker-compose up -d\n", filepath.Dir(composeFile))
		fmt.Println("Then run this program again.")
	} else {
		// Provide cleanup instructions if replicas were found
		fmt.Println("\nTo clean up the Docker containers when you're done:")
		fmt.Printf("cd %s && docker-compose down\n", filepath.Dir(composeFile))
	}
}

// handleError processes errors with appropriate messages and suggestions
func handleError(context string, err error) {
	fmt.Printf("%s: %v\n", context, err)

	// Check for common error types and provide helpful messages
	errMsg := err.Error()

	if strings.Contains(errMsg, "Cannot connect to the Docker daemon") ||
		strings.Contains(errMsg, "Is the docker daemon running") {
		fmt.Println("\nERROR: Docker daemon is not running or not accessible.")
		fmt.Println("Please make sure Docker is installed and running before using this tool.")
		fmt.Println("Try running 'docker info' to verify Docker is working correctly.")
	} else if strings.Contains(errMsg, "permission denied") {
		fmt.Println("\nERROR: Permission denied when accessing Docker or the file system.")
		fmt.Println("You may need to run this program with elevated privileges or check file permissions.")
	} else if strings.Contains(errMsg, "no such file") {
		fmt.Println("\nERROR: A required file could not be found.")
		fmt.Println("Make sure all paths are correct and files exist.")
	} else if strings.Contains(errMsg, "no such service") {
		fmt.Println("\nERROR: The specified service was not found in the Docker Compose file.")
		fmt.Println("Check the service name and make sure it exists in the compose file.")
	}

	os.Exit(1)
}

// printUsage displays usage information for the program
func printUsage() {
	fmt.Println("Docker Compose Replica Detector - Example Program")
	fmt.Println("\nUsage:")
	fmt.Printf("  %s [docker-compose-file] [service-name]\n", os.Args[0])
	fmt.Println("\nArguments:")
	fmt.Println("  docker-compose-file  Path to a Docker Compose file (defaults to ./docker-compose.yml)")
	fmt.Println("  service-name         Optional: Specific service to get replicas for")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s                         # Use default compose file\n", os.Args[0])
	fmt.Printf("  %s ./my-compose.yml        # Use custom compose file\n", os.Args[0])
	fmt.Printf("  %s ./docker-compose.yml web # Get replicas for 'web' service\n", os.Args[0])
	fmt.Println("\nNote:")
	fmt.Println("  This example requires Docker to be running with containers matching")
	fmt.Println("  those defined in the Docker Compose file to show actual replicas.")
}
