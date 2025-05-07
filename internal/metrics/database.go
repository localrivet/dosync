package metrics

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	// Default database file path relative to the data directory
	defaultDBName = ".dosync.db"

	// Table creation SQL
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS deployment_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_name TEXT NOT NULL,
		version TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		success INTEGER DEFAULT 0,
		duration INTEGER,
		failure_reason TEXT,
		rollback INTEGER DEFAULT 0,
		UNIQUE(service_name, version, start_time)
	);
	
	CREATE INDEX IF NOT EXISTS idx_service_name ON deployment_records(service_name);
	CREATE INDEX IF NOT EXISTS idx_success ON deployment_records(success);
	CREATE INDEX IF NOT EXISTS idx_start_time ON deployment_records(start_time);
	`
)

// Database provides access to the metrics SQLite database
type Database struct {
	db   *sql.DB
	path string
}

// NewDatabase creates a new SQLite database connection and ensures tables exist
func NewDatabase(dbPath string) (*Database, error) {
	// If no path provided, use the default
	if dbPath == "" {
		dataDir := os.Getenv("DOSYNC_DATA_DIR")
		if dataDir == "" {
			// If no data directory is specified, use the current directory
			dir, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get current directory: %w", err)
			}
			dataDir = dir
		}

		// Ensure the data directory exists
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		dbPath = filepath.Join(dataDir, defaultDBName)
	}

	// Open the database file with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(1) // SQLite supports only one writer at a time

	// Create the database instance
	database := &Database{
		db:   db,
		path: dbPath,
	}

	// Initialize the database schema
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initSchema ensures the required tables and indices exist
func (d *Database) initSchema() error {
	_, err := d.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// GetDB returns the underlying database connection
// This should be used carefully, only in cases where direct access is needed
func (d *Database) GetDB() *sql.DB {
	return d.db
}
