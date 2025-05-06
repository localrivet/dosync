package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionManager(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	// Insert some test records
	now := time.Now()
	oldRecord := `
	INSERT INTO deployment_records 
	(service_name, version, start_time) 
	VALUES (?, ?, ?)
	`
	oldTime := now.Add(-91 * 24 * time.Hour) // 91 days old
	for i := 0; i < 10; i++ {
		_, err := dao.db.db.Exec(oldRecord, "service1", "v0."+string(rune('0'+i)), oldTime)
		require.NoError(t, err)
	}

	// Create a test config with a short interval
	config := RetentionConfig{
		MaxAge:               90 * 24 * time.Hour, // 90 days
		MaxRecordsPerService: 1000,
		MaxDatabaseSize:      100 * 1024 * 1024, // 100MB
		PruneInterval:        10 * time.Millisecond,
		Enabled:              true,
	}

	// Create the retention manager
	manager := NewRetentionManager(dao, config)

	// Test manual pruning
	results, err := manager.RunPruningNow()
	require.NoError(t, err)
	assert.Equal(t, int64(10), results["time_based"]) // Should prune all 10 old records

	// Verify records were deleted
	count, err := dao.CountRecords()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Insert some new records
	for i := 0; i < 5; i++ {
		_, err := dao.InsertDeploymentStart("service1", "v1."+string(rune('0'+i)))
		require.NoError(t, err)
	}

	// Start the manager
	err = manager.Start()
	require.NoError(t, err)

	// Let it run for a bit (wait for at least one scheduled run)
	time.Sleep(50 * time.Millisecond)

	// Stop the manager
	manager.Stop()

	// Test updating the config
	newConfig := config
	newConfig.Enabled = false
	err = manager.UpdateConfig(newConfig)
	require.NoError(t, err)
	assert.False(t, manager.running)

	// Re-enable
	newConfig.Enabled = true
	err = manager.UpdateConfig(newConfig)
	require.NoError(t, err)
	assert.True(t, manager.running)

	manager.Stop()
}
