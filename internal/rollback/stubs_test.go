/*
Copyright © 2024 LocalRivet <github.com/localrivet>
*/
package rollback

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubRollbackController_PrepareRollback(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		err := ctrl.PrepareRollback("test-service")
		assert.NoError(t, err)

		// Verify an entry was added
		entries, err := ctrl.GetRollbackHistory("test-service")
		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-service", entries[0].ServiceName)
		assert.Equal(t, "test-tag", entries[0].ImageTag)
	})

	t.Run("Error case", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		expectedErr := errors.New("prepare error")
		ctrl.PrepareError = expectedErr

		err := ctrl.PrepareRollback("test-service")
		assert.Equal(t, expectedErr, err)
	})
}

func TestStubRollbackController_GetRollbackHistory(t *testing.T) {
	t.Run("Returns empty list for non-existent service", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		entries, err := ctrl.GetRollbackHistory("non-existent")
		assert.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("Returns error when set", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		expectedErr := errors.New("history error")
		ctrl.HistoryError = expectedErr

		_, err := ctrl.GetRollbackHistory("test-service")
		assert.Equal(t, expectedErr, err)
	})

	t.Run("Returns entries for existing service", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		ctrl.PrepareRollback("test-service")
		ctrl.PrepareRollback("test-service")

		entries, err := ctrl.GetRollbackHistory("test-service")
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
	})
}

func TestStubRollbackController_Other(t *testing.T) {
	t.Run("Rollback returns configured error", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		expectedErr := errors.New("rollback error")
		ctrl.RollbackError = expectedErr

		err := ctrl.Rollback("test-service")
		assert.Equal(t, expectedErr, err)
	})

	t.Run("RollbackToVersion returns configured error", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		expectedErr := errors.New("version error")
		ctrl.VersionError = expectedErr

		err := ctrl.RollbackToVersion("test-service", "v1.0")
		assert.Equal(t, expectedErr, err)
	})

	t.Run("CleanupOldBackups returns configured error", func(t *testing.T) {
		ctrl := NewStubRollbackController()
		expectedErr := errors.New("cleanup error")
		ctrl.CleanupError = expectedErr

		err := ctrl.CleanupOldBackups()
		assert.Equal(t, expectedErr, err)
	})

	t.Run("ShouldRollback returns configured value", func(t *testing.T) {
		ctrl := NewStubRollbackController()

		// Default should be true
		assert.True(t, ctrl.ShouldRollback("service", false, true))

		// Change to false and test
		ctrl.ShouldRollbackVal = false
		assert.False(t, ctrl.ShouldRollback("service", false, true))
	})
}
