package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestCollector(t *testing.T) (*Collector, func()) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "dosync-collector-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, ".dosync.db")

	// Create a collector with minimal retention config
	config := RetentionConfig{
		MaxAge:               90 * 24 * time.Hour,
		MaxRecordsPerService: 1000,
		MaxDatabaseSize:      10 * 1024 * 1024,
		PruneInterval:        24 * time.Hour,
		Enabled:              false, // Disable scheduled pruning for tests
	}

	collector, err := NewCollector(dbPath, config)
	require.NoError(t, err)

	// Return the collector and a cleanup function
	cleanup := func() {
		collector.Close()
		os.RemoveAll(tempDir)
	}

	return collector, cleanup
}

func TestCollectorDeploymentTracking(t *testing.T) {
	collector, cleanup := setupTestCollector(t)
	defer cleanup()

	// Test recording a deployment cycle
	serviceName := "test-service"
	version := "v1.0.0"

	// 1. Record deployment start
	err := collector.RecordDeploymentStart(serviceName, version)
	require.NoError(t, err)

	// 2. Record deployment success
	time.Sleep(10 * time.Millisecond)                                // Ensure measurable duration
	err = collector.RecordDeploymentSuccess(serviceName, version, 0) // 0 to use auto-calculation
	require.NoError(t, err)

	// 3. Fetch records
	records, err := collector.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, serviceName, record.ServiceName)
	assert.Equal(t, version, record.Version)
	assert.True(t, record.Success)
	assert.False(t, record.Rollback)
	assert.NotZero(t, record.Duration)

	// 4. Test metrics
	successRate, err := collector.GetSuccessRate(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 1.0, successRate)

	avgTime, err := collector.GetAverageDeploymentTime(serviceName)
	require.NoError(t, err)
	assert.NotZero(t, avgTime)

	rollbackCount, err := collector.GetRollbackCount(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 0, rollbackCount)
}

func TestCollectorDeploymentFailure(t *testing.T) {
	collector, cleanup := setupTestCollector(t)
	defer cleanup()

	serviceName := "test-service"
	version := "v1.0.0"
	failureReason := "Test deployment failure"

	// 1. Record deployment start
	err := collector.RecordDeploymentStart(serviceName, version)
	require.NoError(t, err)

	// 2. Record deployment failure
	time.Sleep(10 * time.Millisecond)
	err = collector.RecordDeploymentFailure(serviceName, version, failureReason)
	require.NoError(t, err)

	// 3. Fetch records
	records, err := collector.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, serviceName, record.ServiceName)
	assert.Equal(t, version, record.Version)
	assert.False(t, record.Success)
	assert.Equal(t, failureReason, record.FailureReason)
	assert.NotZero(t, record.Duration)

	// 4. Test metrics
	successRate, err := collector.GetSuccessRate(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 0.0, successRate)
}

func TestCollectorRollback(t *testing.T) {
	collector, cleanup := setupTestCollector(t)
	defer cleanup()

	serviceName := "test-service"
	fromVersion := "v1.0.0"
	toVersion := "v0.9.0"

	// 1. Record successful deployment of the first version
	err := collector.RecordDeploymentStart(serviceName, fromVersion)
	require.NoError(t, err)
	err = collector.RecordDeploymentSuccess(serviceName, fromVersion, 100*time.Millisecond)
	require.NoError(t, err)

	// 2. Record rollback
	err = collector.RecordRollback(serviceName, fromVersion, toVersion)
	require.NoError(t, err)

	// 3. Fetch records - only get the 2 most recent ones
	records, err := collector.GetDeploymentRecords(serviceName, 2, 0)
	require.NoError(t, err)
	require.Len(t, records, 2, "Should have exactly 2 records")

	// The newest record (index 0) should be the rollback
	assert.Equal(t, toVersion, records[0].Version)
	assert.True(t, records[0].Rollback)
	assert.True(t, records[0].Success)

	// The older record (index 1) should be the original deployment
	assert.Equal(t, fromVersion, records[1].Version)
	assert.True(t, records[1].Rollback)
	assert.True(t, records[1].Success)

	// 4. Test metrics
	rollbackCount, err := collector.GetRollbackCount(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 2, rollbackCount) // Both records are marked as rollback

	// Services list should contain our test service
	services, err := collector.GetServicesWithMetrics()
	require.NoError(t, err)
	assert.Contains(t, services, serviceName)
}

func TestCollectorRetention(t *testing.T) {
	// Use direct DAO instead of going through the collector
	// to avoid retention manager overhead
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	serviceName := "test-service"

	// Insert test records directly via the DAO
	for i := range 5 {
		version := "v1.0." + string(rune('0'+i))
		_, err := dao.InsertDeploymentStart(serviceName, version)
		require.NoError(t, err)
		_, err = dao.InsertDeploymentStart(serviceName, version+"a")
		require.NoError(t, err)
	}

	// Verify records
	count, err := dao.CountRecords()
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Create retention config
	config := RetentionConfig{
		MaxAge:               90 * 24 * time.Hour,
		MaxRecordsPerService: 4,
		MaxDatabaseSize:      10 * 1024 * 1024,
		PruneInterval:        24 * time.Hour,
		Enabled:              true,
	}

	// Run the pruning directly
	results, err := dao.PruneDatabase(config)
	require.NoError(t, err)

	// Verify results
	assert.Greater(t, results["count_based"], int64(0))

	// Verify records after pruning
	count, err = dao.CountRecords()
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
}
