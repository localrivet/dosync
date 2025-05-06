package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabase(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "dosync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Create a new database
	db, err := NewDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify the database was created
	assert.FileExists(t, dbPath)

	// Verify the database has the required table
	result, err := db.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='deployment_records';")
	require.NoError(t, err)
	defer result.Close()

	// Check that the table exists
	assert.True(t, result.Next())
}

func TestDefaultDatabasePath(t *testing.T) {
	// Set up a temporary environment variable
	tempDir, err := os.MkdirTemp("", "dosync-data-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Set the environment variable for the test
	oldDataDir := os.Getenv("DOSYNC_DATA_DIR")
	defer os.Setenv("DOSYNC_DATA_DIR", oldDataDir)
	os.Setenv("DOSYNC_DATA_DIR", tempDir)

	// Create a new database with default path
	db, err := NewDatabase("")
	require.NoError(t, err)
	defer db.Close()

	// Verify the database was created in the expected location
	expectedPath := filepath.Join(tempDir, defaultDBName)
	assert.FileExists(t, expectedPath)
}

func TestDatabaseSchemaInitialization(t *testing.T) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "dosync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test-schema.db")

	// Create a new database
	db, err := NewDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Check for all required tables and indices
	tables := []string{"deployment_records"}
	for _, table := range tables {
		t.Run("Table_"+table, func(t *testing.T) {
			result, err := db.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", table)
			require.NoError(t, err)
			defer result.Close()
			assert.True(t, result.Next(), "Table %s should exist", table)
		})
	}

	// Check for indices
	indices := []string{"idx_service_name", "idx_success", "idx_start_time"}
	for _, index := range indices {
		t.Run("Index_"+index, func(t *testing.T) {
			result, err := db.db.Query("SELECT name FROM sqlite_master WHERE type='index' AND name=?;", index)
			require.NoError(t, err)
			defer result.Close()
			assert.True(t, result.Next(), "Index %s should exist", index)
		})
	}
}
