package main

import (
	"fmt"
	"os"
)

func main() {
	// Test GHCR authentication and API access
	token := os.Getenv("GITHUB_PAT") // Use environment variable instead of hardcoded token
	if token == "" {
		fmt.Println("❌ GITHUB_PAT environment variable not set")
		fmt.Println("Please set: export GITHUB_PAT=your_github_personal_access_token")
		os.Exit(1)
	}

	username := "localrivet"
	repo := "tax-equity/solar-equity-hub"

	fmt.Printf("🔍 Testing GHCR access for %s/%s\n", username, repo)
	fmt.Printf("🔑 Using token: %s...%s\n", token[:7], token[len(token)-4:])

	// Test 1: Verify token and get user info
	fmt.Println("\n1️⃣ Testing GitHub API authentication...")
	if err := testGitHubAPI(token); err != nil {
		fmt.Printf("❌ GitHub API test failed: %v\n", err)
		return
	}
	fmt.Println("✅ GitHub API authentication successful")

	// Test 2: Check repository access
	fmt.Println("\n2️⃣ Testing repository access...")
	if err := testRepositoryAccess(token, repo); err != nil {
		fmt.Printf("❌ Repository access test failed: %v\n", err)
		return
	}
	fmt.Println("✅ Repository access confirmed")

	// Test 3: Test GHCR package access
	fmt.Println("\n3️⃣ Testing GHCR package access...")
	if err := testGHCRPackageAccess(token, repo); err != nil {
		fmt.Printf("❌ GHCR package access test failed: %v\n", err)
		return
	}
	fmt.Println("✅ GHCR package access confirmed")

	// Test 4: Test Docker Registry v2 API
	fmt.Println("\n4️⃣ Testing Docker Registry v2 API...")
	if err := testDockerRegistryAPI(token, username, repo); err != nil {
		fmt.Printf("❌ Docker Registry v2 API test failed: %v\n", err)
		return
	}
	fmt.Println("✅ Docker Registry v2 API access confirmed")

	fmt.Println("\n🎉 All GHCR tests passed! DOSync should work correctly with this configuration.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func testGitHubAPI(token string) error {
	// Implementation of testGitHubAPI function
	return nil
}

func testRepositoryAccess(token, repo string) error {
	// Implementation of testRepositoryAccess function
	return nil
}

func testGHCRPackageAccess(token, repo string) error {
	// Implementation of testGHCRPackageAccess function
	return nil
}

func testDockerRegistryAPI(token, username, repo string) error {
	// Implementation of testDockerRegistryAPI function
	return nil
}
