package metrics

import (
	"time"
)

// RetentionConfig defines how long metrics data should be kept
type RetentionConfig struct {
	// MaxAge defines the maximum age of records to keep
	// Records older than this will be deleted during pruning
	// Default is 90 days
	MaxAge time.Duration

	// MaxRecordsPerService limits the number of records kept per service
	// If more records exist, the oldest ones will be removed during pruning
	// Default is 1000 records per service
	MaxRecordsPerService int

	// MaxDatabaseSize defines the maximum size of the database file in bytes
	// When exceeded, records will be pruned starting with the oldest
	// Default is 100MB
	MaxDatabaseSize int64

	// PruneInterval defines how often the pruning job should run
	// Default is 24 hours
	PruneInterval time.Duration

	// Enabled determines if automatic pruning is enabled
	// Default is true
	Enabled bool
}

// DefaultRetentionConfig returns the default configuration for metrics retention
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		MaxAge:               90 * 24 * time.Hour, // 90 days
		MaxRecordsPerService: 1000,
		MaxDatabaseSize:      100 * 1024 * 1024, // 100MB
		PruneInterval:        24 * time.Hour,    // 1 day
		Enabled:              true,
	}
}
