package metrics

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
)

// DAO (Data Access Object) provides methods to interact with the metrics database
type DAO struct {
	db *Database
}

// NewDAO creates a new Data Access Object for metrics
func NewDAO(db *Database) *DAO {
	return &DAO{
		db: db,
	}
}

// InsertDeploymentStart inserts a new deployment record with the start information
func (d *DAO) InsertDeploymentStart(serviceName, version string) (int64, error) {
	query := `
	INSERT INTO deployment_records 
	(service_name, version, start_time) 
	VALUES (?, ?, ?)
	`

	result, err := d.db.db.Exec(query, serviceName, version, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to insert deployment start record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}

// UpdateDeploymentSuccess updates a deployment record with success information
func (d *DAO) UpdateDeploymentSuccess(id int64, duration time.Duration) error {
	query := `
	UPDATE deployment_records 
	SET end_time = ?, success = 1, duration = ? 
	WHERE id = ?
	`

	_, err := d.db.db.Exec(query, time.Now(), duration.Nanoseconds(), id)
	if err != nil {
		return fmt.Errorf("failed to update deployment success: %w", err)
	}

	return nil
}

// UpdateDeploymentFailure updates a deployment record with failure information
func (d *DAO) UpdateDeploymentFailure(id int64, reason string, duration time.Duration) error {
	query := `
	UPDATE deployment_records 
	SET end_time = ?, success = 0, duration = ?, failure_reason = ? 
	WHERE id = ?
	`

	_, err := d.db.db.Exec(query, time.Now(), duration.Nanoseconds(), reason, id)
	if err != nil {
		return fmt.Errorf("failed to update deployment failure: %w", err)
	}

	return nil
}

// RecordRollback marks a deployment as rolled back and creates a rollback record
func (d *DAO) RecordRollback(serviceName, fromVersion, toVersion string) error {
	tx, err := d.db.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Find the most recent successful deployment to mark as rolled back
	findQuery := `
	SELECT id FROM deployment_records 
	WHERE service_name = ? AND version = ? AND success = 1
	ORDER BY end_time DESC
	LIMIT 1
	`

	var id int64
	err = tx.QueryRow(findQuery, serviceName, fromVersion).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return fmt.Errorf("failed to find original deployment: %w", err)
	}

	// Update the found record to mark as rolled back
	if err != sql.ErrNoRows {
		updateQuery := `
		UPDATE deployment_records 
		SET rollback = 1 
		WHERE id = ?
		`

		_, err = tx.Exec(updateQuery, id)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update original deployment: %w", err)
		}
	}

	// Insert a new record for the rollback itself
	insertQuery := `
	INSERT INTO deployment_records 
	(service_name, version, start_time, end_time, success, duration, rollback) 
	VALUES (?, ?, ?, ?, 1, 0, 1)
	`

	now := time.Now()
	_, err = tx.Exec(insertQuery, serviceName, toVersion, now, now)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert rollback record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDeploymentRecords retrieves all deployment records for a service
func (d *DAO) GetDeploymentRecords(serviceName string, limit, offset int) ([]DeploymentRecord, error) {
	query := `
	SELECT id, service_name, version, start_time, end_time, success, duration, failure_reason, rollback
	FROM deployment_records
	WHERE service_name = ?
	ORDER BY start_time DESC
	LIMIT ? OFFSET ?
	`

	rows, err := d.db.db.Query(query, serviceName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployment records: %w", err)
	}
	defer rows.Close()

	var records []DeploymentRecord
	for rows.Next() {
		var record DeploymentRecord
		var durationNanos sql.NullInt64
		var success, rollback int
		var endTime sql.NullTime
		var failureReason sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.ServiceName,
			&record.Version,
			&record.StartTime,
			&endTime,
			&success,
			&durationNanos,
			&failureReason,
			&rollback,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment record: %w", err)
		}

		// Convert database values to Go types
		if endTime.Valid {
			record.EndTime = endTime.Time
		}
		record.Success = success == 1
		if durationNanos.Valid {
			record.Duration = time.Duration(durationNanos.Int64)
		}
		if failureReason.Valid {
			record.FailureReason = failureReason.String
		}
		record.Rollback = rollback == 1

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating deployment records: %w", err)
	}

	return records, nil
}

// GetSuccessRateForService calculates the success rate for a service
func (d *DAO) GetSuccessRateForService(serviceName string) (float64, error) {
	query := `
	SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful
	FROM deployment_records
	WHERE service_name = ? AND end_time IS NOT NULL
	`

	var total, successful int
	err := d.db.db.QueryRow(query, serviceName).Scan(&total, &successful)
	if err != nil {
		return 0, fmt.Errorf("failed to query success rate: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	return float64(successful) / float64(total), nil
}

// GetAverageDeploymentTime calculates the average deployment time for a service
func (d *DAO) GetAverageDeploymentTime(serviceName string) (time.Duration, error) {
	query := `
	SELECT AVG(duration)
	FROM deployment_records
	WHERE service_name = ? AND success = 1 AND duration > 0
	`

	var avgDurationNanos sql.NullFloat64
	err := d.db.db.QueryRow(query, serviceName).Scan(&avgDurationNanos)
	if err != nil {
		return 0, fmt.Errorf("failed to query average duration: %w", err)
	}

	if !avgDurationNanos.Valid {
		return 0, nil
	}

	return time.Duration(avgDurationNanos.Float64), nil
}

// GetRollbackCountForService counts the number of rollbacks for a service
func (d *DAO) GetRollbackCountForService(serviceName string) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM deployment_records
	WHERE service_name = ? AND rollback = 1
	`

	var count int
	err := d.db.db.QueryRow(query, serviceName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query rollback count: %w", err)
	}

	return count, nil
}

// DeleteOldRecords deletes records older than the specified duration
func (d *DAO) DeleteOldRecords(age time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-age)

	query := `
	DELETE FROM deployment_records
	WHERE start_time < ?
	`

	result, err := d.db.db.Exec(query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old records: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}

// PruneOldRecords deletes records older than the specified maximum age
func (d *DAO) PruneOldRecords(maxAge time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-maxAge)

	query := `DELETE FROM deployment_records WHERE start_time < ?`
	result, err := d.db.db.Exec(query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old records: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows count: %w", err)
	}

	return affected, nil
}

// PruneExcessRecords limits the number of records per service
func (d *DAO) PruneExcessRecords(maxRecordsPerService int) (int64, error) {
	if maxRecordsPerService <= 0 {
		return 0, nil
	}

	// Get the list of services first, close the result set, then iterate.
	serviceQuery := `SELECT DISTINCT service_name FROM deployment_records`
	rows, err := d.db.db.Query(serviceQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to get service list: %w", err)
	}

	var serviceNames []string
	for rows.Next() {
		var serviceName string
		if err := rows.Scan(&serviceName); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan service name: %w", err)
		}
		serviceNames = append(serviceNames, serviceName)
	}
	rows.Close()

	var totalAffected int64

	// Now iterate over the collected service names so that each inner query runs
	// without an outer rows cursor blocking the single SQLite connection.
	for _, serviceName := range serviceNames {
		// Identify record IDs to keep and delete everything else
		keepQuery := `
		SELECT id FROM deployment_records
		WHERE service_name = ?
		ORDER BY start_time DESC
		LIMIT ?
		`

		keepRows, err := d.db.db.Query(keepQuery, serviceName, maxRecordsPerService)
		if err != nil {
			return totalAffected, fmt.Errorf("failed to query records to keep: %w", err)
		}

		var idsToKeep []interface{}
		for keepRows.Next() {
			var id int64
			if err := keepRows.Scan(&id); err != nil {
				keepRows.Close()
				return totalAffected, fmt.Errorf("failed to scan ID: %w", err)
			}
			idsToKeep = append(idsToKeep, id)
		}
		keepRows.Close()

		// If we don't have any records to keep or less than the limit, nothing to prune
		if len(idsToKeep) < maxRecordsPerService {
			continue
		}

		// Delete records not in the keep list
		deleteQuery := `
		DELETE FROM deployment_records
		WHERE service_name = ? AND id NOT IN (` + createPlaceholders(len(idsToKeep)) + `)
		`

		args := []interface{}{serviceName}
		args = append(args, idsToKeep...)

		result, err := d.db.db.Exec(deleteQuery, args...)
		if err != nil {
			return totalAffected, fmt.Errorf("failed to delete excess records: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return totalAffected, fmt.Errorf("failed to get affected rows count: %w", err)
		}

		totalAffected += affected
	}

	return totalAffected, nil
}

// Helper function to create a string of placeholders for SQL IN clauses
func createPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}

	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "?"
	}

	return strings.Join(placeholders, ",")
}

// PruneByDatabaseSize prunes records to keep the database under the specified maximum size
func (d *DAO) PruneByDatabaseSize(maxSize int64) (int64, error) {
	// Check current file size
	dbSize, err := d.getDatabaseFileSize()
	if err != nil {
		return 0, err
	}

	// If under the size limit, do nothing
	if dbSize <= maxSize {
		return 0, nil
	}

	// Start with a small batch size and increase if needed
	batchSize := 100
	totalDeleted := int64(0)

	for dbSize > maxSize {
		// Delete oldest records in batches
		query := `
		DELETE FROM deployment_records
		WHERE id IN (
			SELECT id FROM deployment_records
			ORDER BY start_time ASC
			LIMIT ?
		)
		`

		result, err := d.db.db.Exec(query, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to delete records by size: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to get affected rows count: %w", err)
		}

		totalDeleted += affected

		// If we didn't delete anything in this batch, we can't reduce size further
		if affected == 0 {
			break
		}

		// Check the new file size
		dbSize, err = d.getDatabaseFileSize()
		if err != nil {
			return totalDeleted, err
		}

		// Increase batch size for faster pruning if still over limit
		if dbSize > maxSize {
			batchSize *= 2
		}
	}

	// Run VACUUM to actually reclaim the disk space
	_, err = d.db.db.Exec("VACUUM")
	if err != nil {
		return totalDeleted, fmt.Errorf("failed to vacuum database: %w", err)
	}

	return totalDeleted, nil
}

// getDatabaseFileSize returns the current size of the database file in bytes
func (d *DAO) getDatabaseFileSize() (int64, error) {
	info, err := os.Stat(d.db.path)
	if err != nil {
		return 0, fmt.Errorf("failed to get database file info: %w", err)
	}
	return info.Size(), nil
}

// PruneDatabase applies all retention strategies based on the provided config
func (d *DAO) PruneDatabase(config RetentionConfig) (map[string]int64, error) {
	if !config.Enabled {
		return nil, nil
	}

	results := make(map[string]int64)

	// Apply time-based pruning
	if config.MaxAge > 0 {
		deleted, err := d.PruneOldRecords(config.MaxAge)
		if err != nil {
			return results, fmt.Errorf("time-based pruning failed: %w", err)
		}
		results["time_based"] = deleted
	}

	// Apply count-based pruning
	if config.MaxRecordsPerService > 0 {
		deleted, err := d.PruneExcessRecords(config.MaxRecordsPerService)
		if err != nil {
			return results, fmt.Errorf("count-based pruning failed: %w", err)
		}
		results["count_based"] = deleted
	}

	// Apply size-based pruning
	if config.MaxDatabaseSize > 0 {
		deleted, err := d.PruneByDatabaseSize(config.MaxDatabaseSize)
		if err != nil {
			return results, fmt.Errorf("size-based pruning failed: %w", err)
		}
		results["size_based"] = deleted
	}

	return results, nil
}

// GetServices returns a list of all service names that have metrics data
func (d *DAO) GetServices() ([]string, error) {
	query := `
	SELECT DISTINCT service_name
	FROM deployment_records
	ORDER BY service_name
	`

	rows, err := d.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, fmt.Errorf("failed to scan service name: %w", err)
		}
		services = append(services, service)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating services: %w", err)
	}

	return services, nil
}
