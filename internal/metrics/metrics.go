package metrics

import (
	"time"
)

// DeploymentRecord represents a single deployment event
type DeploymentRecord struct {
	ID            int64
	ServiceName   string
	Version       string
	StartTime     time.Time
	EndTime       time.Time
	Success       bool
	Duration      time.Duration
	FailureReason string
	Rollback      bool
	DurationStr   string // human-readable duration for dashboard
}

// MetricsCollector defines the interface for recording and retrieving deployment metrics
type MetricsCollector interface {
	// Recording methods
	RecordDeploymentStart(service string, version string) error
	RecordDeploymentSuccess(service string, version string, duration time.Duration) error
	RecordDeploymentFailure(service string, version string, reason string) error
	RecordRollback(service string, fromVersion string, toVersion string) error

	// Retrieval methods
	GetDeploymentRecords(service string, limit, offset int) ([]DeploymentRecord, error)
	GetServicesWithMetrics() ([]string, error)
	GetSuccessRate(service string) (float64, error)
	GetAverageDeploymentTime(service string) (time.Duration, error)
	GetRollbackCount(service string) (int, error)

	// Retention management
	UpdateRetentionConfig(config RetentionConfig) error
	RunRetentionNow() (map[string]int64, error)

	// Cleanup
	Close() error
}

// DeploymentStats contains aggregated metrics about deployments
type DeploymentStats struct {
	TotalDeployments      int
	SuccessfulDeployments int
	FailedDeployments     int
	SuccessRate           float64
	AverageDeployTime     time.Duration
	RollbackCount         int
}
