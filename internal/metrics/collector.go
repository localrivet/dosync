package metrics

import (
	"fmt"
	"sync"
	"time"
)

// Collector implements the MetricsCollector interface
type Collector struct {
	dao          *DAO
	retention    *RetentionManager
	deployments  map[string]deploymentInfo
	deploymentMu sync.Mutex
}

// deploymentInfo stores information about an in-progress deployment
type deploymentInfo struct {
	ID        int64
	StartTime time.Time
	Version   string
}

// NewCollector creates a new metrics collector with the given configuration
func NewCollector(dbPath string, config RetentionConfig) (*Collector, error) {
	// Initialize the database
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics database: %w", err)
	}

	// Create the DAO
	dao := NewDAO(db)

	// Create the collector
	collector := &Collector{
		dao:         dao,
		deployments: make(map[string]deploymentInfo),
	}

	// Set up retention management
	collector.retention = NewRetentionManager(dao, config)
	if err := collector.retention.Start(); err != nil {
		return nil, fmt.Errorf("failed to start retention manager: %w", err)
	}

	return collector, nil
}

// Close closes the collector and its database connection
func (c *Collector) Close() error {
	// Stop the retention manager
	c.retention.Stop()

	// Close the database
	return c.dao.db.Close()
}

// RecordDeploymentStart records the start of a deployment
func (c *Collector) RecordDeploymentStart(service string, version string) error {
	// Insert the record into the database
	id, err := c.dao.InsertDeploymentStart(service, version)
	if err != nil {
		return err
	}

	// Store the in-progress deployment info
	c.deploymentMu.Lock()
	defer c.deploymentMu.Unlock()

	c.deployments[service] = deploymentInfo{
		ID:        id,
		StartTime: time.Now(),
		Version:   version,
	}

	return nil
}

// RecordDeploymentSuccess records a successful deployment
func (c *Collector) RecordDeploymentSuccess(service string, version string, duration time.Duration) error {
	// If duration is 0, calculate from stored start time
	if duration == 0 {
		c.deploymentMu.Lock()
		info, exists := c.deployments[service]
		c.deploymentMu.Unlock()

		if exists && info.Version == version {
			duration = time.Since(info.StartTime)
			return c.dao.UpdateDeploymentSuccess(info.ID, duration)
		}
	}

	// If we don't have stored info, insert a new completed record
	id, err := c.dao.InsertDeploymentStart(service, version)
	if err != nil {
		return err
	}

	return c.dao.UpdateDeploymentSuccess(id, duration)
}

// RecordDeploymentFailure records a failed deployment
func (c *Collector) RecordDeploymentFailure(service string, version string, reason string) error {
	// Check if we have an in-progress deployment
	c.deploymentMu.Lock()
	info, exists := c.deployments[service]
	delete(c.deployments, service) // Remove from in-progress tracking
	c.deploymentMu.Unlock()

	if exists && info.Version == version {
		duration := time.Since(info.StartTime)
		return c.dao.UpdateDeploymentFailure(info.ID, reason, duration)
	}

	// If we don't have stored info, insert a new failed record
	id, err := c.dao.InsertDeploymentStart(service, version)
	if err != nil {
		return err
	}

	return c.dao.UpdateDeploymentFailure(id, reason, 0)
}

// RecordRollback records a rollback from one version to another
func (c *Collector) RecordRollback(service string, fromVersion string, toVersion string) error {
	return c.dao.RecordRollback(service, fromVersion, toVersion)
}

// GetDeploymentRecords retrieves deployment records for a service
func (c *Collector) GetDeploymentRecords(service string, limit, offset int) ([]DeploymentRecord, error) {
	return c.dao.GetDeploymentRecords(service, limit, offset)
}

// GetServicesWithMetrics returns a list of services that have metrics data
func (c *Collector) GetServicesWithMetrics() ([]string, error) {
	return c.dao.GetServices()
}

// GetSuccessRate calculates the deployment success rate for a service
func (c *Collector) GetSuccessRate(service string) (float64, error) {
	return c.dao.GetSuccessRateForService(service)
}

// GetAverageDeploymentTime calculates the average deployment time for a service
func (c *Collector) GetAverageDeploymentTime(service string) (time.Duration, error) {
	return c.dao.GetAverageDeploymentTime(service)
}

// GetRollbackCount returns the number of rollbacks for a service
func (c *Collector) GetRollbackCount(service string) (int, error) {
	return c.dao.GetRollbackCountForService(service)
}

// UpdateRetentionConfig updates the retention configuration
func (c *Collector) UpdateRetentionConfig(config RetentionConfig) error {
	return c.retention.UpdateConfig(config)
}

// RunRetentionNow triggers immediate pruning based on the current configuration
func (c *Collector) RunRetentionNow() (map[string]int64, error) {
	return c.retention.RunPruningNow()
}
