package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"dosync/internal/config"

	"github.com/spf13/pflag"
)

func TestSyncCmdFlags(t *testing.T) {
	cmd := syncCmd
	flags := cmd.Flags()

	// Check that all rolling update flags exist and have correct defaults
	if v, _ := flags.GetBool("rolling-update"); v != false {
		t.Errorf("expected rolling-update default false, got %v", v)
	}
	if v, _ := flags.GetString("strategy"); v != "one-at-a-time" {
		t.Errorf("expected strategy default 'one-at-a-time', got %v", v)
	}
	if v, _ := flags.GetString("health-check"); v != "docker" {
		t.Errorf("expected health-check default 'docker', got %v", v)
	}
	if v, _ := flags.GetString("health-endpoint"); v != "/health" {
		t.Errorf("expected health-endpoint default '/health', got %v", v)
	}
	if v, _ := flags.GetDuration("delay"); v != 10_000_000_000 {
		t.Errorf("expected delay default 10s, got %v", v)
	}
	if v, _ := flags.GetBool("rollback-on-failure"); v != true {
		t.Errorf("expected rollback-on-failure default true, got %v", v)
	}
}

func TestBuildRollingUpdateConfig(t *testing.T) {
	cmd := syncCmd
	cmd.Flags().Set("rolling-update", "true")
	cmd.Flags().Set("strategy", "canary")
	cmd.Flags().Set("health-check", "http")
	cmd.Flags().Set("health-endpoint", "/status")
	cmd.Flags().Set("delay", "5s")
	cmd.Flags().Set("rollback-on-failure", "false")

	cfg, err := buildRollingUpdateConfig(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled true")
	}
	if cfg.Strategy != "canary" {
		t.Errorf("expected Strategy 'canary', got %v", cfg.Strategy)
	}
	if cfg.HealthCheckType != "http" {
		t.Errorf("expected HealthCheckType 'http', got %v", cfg.HealthCheckType)
	}
	if cfg.HealthEndpoint != "/status" {
		t.Errorf("expected HealthEndpoint '/status', got %v", cfg.HealthEndpoint)
	}
	if cfg.Delay != 5*time.Second {
		t.Errorf("expected Delay 5s, got %v", cfg.Delay)
	}
	if cfg.RollbackOnFailure {
		t.Error("expected RollbackOnFailure false")
	}
}

func TestSyncCmdDispatchesToRollingUpdate(t *testing.T) {
	// Save original handleRollingUpdate
	origHandle := handleRollingUpdate
	defer func() { handleRollingUpdate = origHandle }()

	// Save and restore AppConfig
	origAppConfig := AppConfig
	defer func() { AppConfig = origAppConfig }()

	called := false
	handleRollingUpdate = func(cfg *RollingUpdateConfig, filePath string) {
		called = true
	}

	// Create a temp compose file
	tmpFile, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("services: {}\n")
	tmpFile.Close()

	// Reset all flags to default values
	syncCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Value.Set(f.DefValue) })

	syncCmd.Flags().Set("rolling-update", "true")
	syncCmd.Flags().Set("file", tmpFile.Name())

	// Assign a valid AppConfig
	AppConfig = &config.Config{
		CheckInterval: "1m",
		Verbose:       false,
		Registry: &config.RegistryConfig{
			DOCR: &config.DOCRConfig{Token: "dummy"},
		},
	}

	syncCmd.Run(syncCmd, []string{})

	if !called {
		t.Error("expected handleRollingUpdate to be called when rolling-update is enabled")
	}
}

func TestHandleRollingUpdateStub(t *testing.T) {
	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &RollingUpdateConfig{
		Enabled:           true,
		Strategy:          "canary",
		HealthCheckType:   "http",
		HealthEndpoint:    "/status",
		Delay:             5 * time.Second,
		RollbackOnFailure: false,
	}
	filePath := "test-compose.yml"

	handleRollingUpdate(cfg, filePath)

	w.Close()
	os.Stdout = origStdout
	buf.ReadFrom(r)
	output := buf.String()

	if want := "[Rolling Update] Stub: would perform rolling update on test-compose.yml"; !bytes.Contains([]byte(output), []byte(want)) {
		t.Errorf("expected output to contain %q, got %q", want, output)
	}
}

func TestSyncCmdEnvVarInvalidBool(t *testing.T) {
	// Save and restore original environment
	orig := os.Getenv("SYNC_VERBOSE")
	defer os.Setenv("SYNC_VERBOSE", orig)

	os.Setenv("SYNC_VERBOSE", "--verbose")

	// Prepare command to run the sync command in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestSyncCmdEnvVarInvalidBoolHelper")
	cmd.Env = append(os.Environ(), "SYNC_VERBOSE=--verbose")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Fatalf("expected exit code 2, got %d", exitErr.ExitCode())
		}
		if !strings.Contains(stderr.String(), "Invalid value for SYNC_VERBOSE") {
			t.Fatalf("expected error message about invalid SYNC_VERBOSE, got: %s", stderr.String())
		}
	} else if err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else {
		t.Fatalf("expected process to exit with error, but it did not")
	}
}

// This helper is only run as a subprocess by the above test
func TestSyncCmdEnvVarInvalidBoolHelper(t *testing.T) {
	if os.Getenv("SYNC_VERBOSE") != "--verbose" {
		t.Skip("not running subprocess test")
	}
	// Set up minimal AppConfig to avoid nil pointer
	AppConfig = &config.Config{CheckInterval: "1m", Verbose: false}
	// Run the sync command, which should exit with code 2 due to invalid SYNC_VERBOSE
	syncCmd.Run(syncCmd, []string{})
}
