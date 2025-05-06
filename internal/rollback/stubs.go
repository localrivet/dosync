/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"time"
)

// StubRollbackController is a test implementation of the RollbackController interface
type StubRollbackController struct {
	// Entries stores rollback entries by service name for testing
	Entries map[string][]RollbackEntry

	// Errors to return for each operation
	PrepareError      error
	RollbackError     error
	VersionError      error
	HistoryError      error
	CleanupError      error
	ShouldRollbackVal bool
}

// NewStubRollbackController creates a new stub rollback controller for testing
func NewStubRollbackController() *StubRollbackController {
	return &StubRollbackController{
		Entries:           make(map[string][]RollbackEntry),
		ShouldRollbackVal: true,
	}
}

// PrepareRollback creates a fake backup entry
func (s *StubRollbackController) PrepareRollback(service string) error {
	if s.PrepareError != nil {
		return s.PrepareError
	}

	entry := RollbackEntry{
		ServiceName: service,
		ImageTag:    "test-tag",
		Timestamp:   time.Now(),
		ComposeFile: "/path/to/backup.yml",
	}

	if _, ok := s.Entries[service]; !ok {
		s.Entries[service] = []RollbackEntry{}
	}
	s.Entries[service] = append(s.Entries[service], entry)
	return nil
}

// Rollback simulates rolling back a service
func (s *StubRollbackController) Rollback(service string) error {
	return s.RollbackError
}

// RollbackToVersion simulates rolling back to a specific version
func (s *StubRollbackController) RollbackToVersion(service string, version string) error {
	return s.VersionError
}

// GetRollbackHistory returns the stored entries for the service
func (s *StubRollbackController) GetRollbackHistory(service string) ([]RollbackEntry, error) {
	if s.HistoryError != nil {
		return nil, s.HistoryError
	}

	if entries, ok := s.Entries[service]; ok {
		return entries, nil
	}
	return []RollbackEntry{}, nil
}

// ShouldRollback returns the configured value for testing
func (s *StubRollbackController) ShouldRollback(service string, healthStatus bool, rollbackOnFailure bool) bool {
	return s.ShouldRollbackVal
}

// CleanupOldBackups simulates cleanup
func (s *StubRollbackController) CleanupOldBackups() error {
	return s.CleanupError
}
