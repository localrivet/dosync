package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDAO(t *testing.T) (*DAO, func()) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "dosync-dao-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "test-dao.db")

	// Create a new database
	db, err := NewDatabase(dbPath)
	require.NoError(t, err)

	// Create a new DAO
	dao := NewDAO(db)

	// Return the DAO and a cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return dao, cleanup
}

func TestInsertAndUpdateDeployment(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	// Insert a new deployment start record
	serviceName := "test-service"
	version := "1.0.0"
	id, err := dao.InsertDeploymentStart(serviceName, version)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Update with success information
	duration := 5 * time.Second
	err = dao.UpdateDeploymentSuccess(id, duration)
	require.NoError(t, err)

	// Retrieve the record
	records, err := dao.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)

	// Verify the record
	record := records[0]
	assert.Equal(t, id, record.ID)
	assert.Equal(t, serviceName, record.ServiceName)
	assert.Equal(t, version, record.Version)
	assert.True(t, record.Success)
	assert.Equal(t, duration, record.Duration)
	assert.False(t, record.Rollback)
}

func TestInsertAndUpdateFailedDeployment(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	// Insert a new deployment start record
	serviceName := "test-service"
	version := "1.0.0"
	failureReason := "Test failure reason"
	id, err := dao.InsertDeploymentStart(serviceName, version)
	require.NoError(t, err)

	// Update with failure information
	duration := 2 * time.Second
	err = dao.UpdateDeploymentFailure(id, failureReason, duration)
	require.NoError(t, err)

	// Retrieve the record
	records, err := dao.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)

	// Verify the record
	record := records[0]
	assert.Equal(t, id, record.ID)
	assert.Equal(t, serviceName, record.ServiceName)
	assert.Equal(t, version, record.Version)
	assert.False(t, record.Success)
	assert.Equal(t, failureReason, record.FailureReason)
	assert.Equal(t, duration, record.Duration)
}

func TestRecordRollback(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	serviceName := "test-service"
	fromVersion := "1.0.0"
	toVersion := "0.9.0"

	// Insert a successful deployment first
	id, err := dao.InsertDeploymentStart(serviceName, fromVersion)
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id, 1*time.Second)
	require.NoError(t, err)

	// Record a rollback
	err = dao.RecordRollback(serviceName, fromVersion, toVersion)
	require.NoError(t, err)

	// Retrieve all records
	records, err := dao.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 2)

	// The second record should be the rollback to the older version
	assert.Equal(t, toVersion, records[0].Version)
	assert.True(t, records[0].Rollback)
	assert.True(t, records[0].Success)

	// The first record should be marked as rolled back
	assert.Equal(t, fromVersion, records[1].Version)
	assert.True(t, records[1].Rollback)
}

func TestGetSuccessRateForService(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	serviceName := "test-service"

	// Insert one successful deployment
	id1, err := dao.InsertDeploymentStart(serviceName, "1.0.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id1, 1*time.Second)
	require.NoError(t, err)

	// Insert one failed deployment
	id2, err := dao.InsertDeploymentStart(serviceName, "1.1.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentFailure(id2, "Test failure", 1*time.Second)
	require.NoError(t, err)

	// Check success rate
	rate, err := dao.GetSuccessRateForService(serviceName)
	require.NoError(t, err)
	assert.Equal(t, 0.5, rate)
}

func TestGetAverageDeploymentTime(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	serviceName := "test-service"

	// Insert deployments with different durations
	id1, err := dao.InsertDeploymentStart(serviceName, "1.0.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id1, 1*time.Second)
	require.NoError(t, err)

	id2, err := dao.InsertDeploymentStart(serviceName, "1.1.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id2, 3*time.Second)
	require.NoError(t, err)

	// Check average time
	avgTime, err := dao.GetAverageDeploymentTime(serviceName)
	require.NoError(t, err)

	// Should be 2 seconds (average of 1 and 3)
	expectedAvg := 2 * time.Second
	assert.Equal(t, expectedAvg, avgTime)
}

func TestDeleteOldRecords(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	serviceName := "test-service"

	// Insert a record
	id, err := dao.InsertDeploymentStart(serviceName, "1.0.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id, 1*time.Second)
	require.NoError(t, err)

	// Check that the record exists
	records, err := dao.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	require.Len(t, records, 1)

	// Delete records older than 1 nanosecond (should delete everything)
	time.Sleep(2 * time.Millisecond) // Wait to ensure time has passed
	count, err := dao.DeleteOldRecords(1 * time.Nanosecond)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Check that no records exist anymore
	records, err = dao.GetDeploymentRecords(serviceName, 10, 0)
	require.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestMultipleServicesSeparation(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	// Insert records for multiple services
	service1 := "service-1"
	service2 := "service-2"

	// Add records for service 1
	id1, err := dao.InsertDeploymentStart(service1, "1.0.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentSuccess(id1, 1*time.Second)
	require.NoError(t, err)

	// Add records for service 2
	id2, err := dao.InsertDeploymentStart(service2, "1.0.0")
	require.NoError(t, err)
	err = dao.UpdateDeploymentFailure(id2, "Test failure", 1*time.Second)
	require.NoError(t, err)

	// Check records for service 1
	records1, err := dao.GetDeploymentRecords(service1, 10, 0)
	require.NoError(t, err)
	assert.Len(t, records1, 1)
	assert.Equal(t, service1, records1[0].ServiceName)
	assert.True(t, records1[0].Success)

	// Check records for service 2
	records2, err := dao.GetDeploymentRecords(service2, 10, 0)
	require.NoError(t, err)
	assert.Len(t, records2, 1)
	assert.Equal(t, service2, records2[0].ServiceName)
	assert.False(t, records2[0].Success)

	// Check success rate for each service
	rate1, err := dao.GetSuccessRateForService(service1)
	require.NoError(t, err)
	assert.Equal(t, 1.0, rate1)

	rate2, err := dao.GetSuccessRateForService(service2)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rate2)
}
