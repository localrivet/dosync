package manager

import (
	"dosync/internal/logx"
	"dosync/internal/rollback"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestEndToEndRollingUpdate(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test due to SKIP_INTEGRATION_TESTS env var")
	}

	var composeFile string
	if _, err := os.Stat("testdata/docker-compose.yml"); err == nil {
		composeFile = "testdata/docker-compose.yml"
	} else if _, err := os.Stat("../../internal/manager/testdata/docker-compose.yml"); err == nil {
		composeFile = "../../internal/manager/testdata/docker-compose.yml"
	} else {
		t.Fatalf("Could not find docker-compose.yml in expected locations")
	}

	// Bring up the test environment
	cmdUp := exec.Command("docker-compose", "-f", composeFile, "up", "-d", "--remove-orphans")
	cmdUp.Stdout = os.Stdout
	cmdUp.Stderr = os.Stderr
	if err := cmdUp.Run(); err != nil {
		t.Fatalf("Failed to bring up test environment: %v", err)
	}
	defer func() {
		cmdDown := exec.Command("docker-compose", "-f", composeFile, "down", "-v")
		cmdDown.Stdout = os.Stdout
		cmdDown.Stderr = os.Stderr
		_ = cmdDown.Run()
	}()

	// Wait for services to be healthy
	t.Log("Waiting for services to become healthy...")
	time.Sleep(10 * time.Second)

	// Initialize RollingUpdateManager with test config
	config := &RollingUpdateConfig{
		ComposeFilePath:    composeFile,
		HealthCheckTimeout: 10 * time.Second,
		HealthCheckRetries: 3,
		UpdateStrategy:     "one-at-a-time",
		RollbackOnFailure:  true,
		RollbackConfig: rollback.RollbackConfig{
			ComposeFilePath:          composeFile,
			BackupDir:                "internal/manager/testdata/backups",
			MaxHistory:               5,
			ComposeFilePattern:       "docker-compose.yml",
			DefaultRollbackOnFailure: true,
		},
		MetricsDB: "internal/manager/testdata/metrics.db",
	}
	config.ApplyDefaults()

	logger := &testLogger{t}
	manager, err := NewRollingUpdateManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to initialize RollingUpdateManager: %v", err)
	}

	// Run a rolling update on test-service
	t.Log("Running rolling update on test-service to tag 'latest'")
	err = manager.Update("test-service", "latest")
	if err != nil {
		t.Fatalf("Rolling update failed: %v", err)
	}

	// TODO: Verify update succeeded and service is healthy
	// TODO: Add more scenarios (dependency order, rollback, etc.)
}

type testLogger struct{ t *testing.T }

func (l *testLogger) Info(format string, args ...interface{})  { l.t.Logf("INFO: "+format, args...) }
func (l *testLogger) Warn(format string, args ...interface{})  { l.t.Logf("WARN: "+format, args...) }
func (l *testLogger) Error(format string, args ...interface{}) { l.t.Logf("ERROR: "+format, args...) }
func (l *testLogger) Debug(format string, args ...interface{}) { l.t.Logf("DEBUG: "+format, args...) }
func (l *testLogger) SetLevel(level logx.LoggingLevel)         {}
