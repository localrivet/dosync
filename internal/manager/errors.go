package manager

import (
	"errors"
	"fmt"
)

// Error definitions
var (
	// ErrMissingComposeFile indicates the compose file path was not specified
	ErrMissingComposeFile = errors.New("compose file path is required")

	// ErrInitFailed indicates the rolling update manager failed to initialize
	ErrInitFailed = errors.New("failed to initialize rolling update manager")

	// ErrInvalidConfig indicates an invalid configuration was provided
	ErrInvalidConfig = errors.New("invalid rolling update configuration")

	// ErrComponentInitFailed indicates a component initialization failed
	ErrComponentInitFailed = errors.New("failed to initialize component")

	// ErrServiceNotFound indicates the specified service was not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrUpdateFailed indicates the update operation failed
	ErrUpdateFailed = errors.New("update operation failed")

	// ErrRollbackFailed indicates the rollback operation failed
	ErrRollbackFailed = errors.New("rollback operation failed")

	// ErrHealthCheckFailed indicates a health check failed after deployment
	ErrHealthCheckFailed = errors.New("health check failed")

	// ErrDependencyCheckFailed indicates a dependency check failed
	ErrDependencyCheckFailed = errors.New("dependency check failed")

	// ErrReplicaDetectionFailed indicates failure to detect service replicas
	ErrReplicaDetectionFailed = errors.New("replica detection failed")

	// ErrNotificationFailed indicates a notification failed to send
	ErrNotificationFailed = errors.New("notification failed to send")

	// ErrMetricsRecordingFailed indicates metrics recording failed
	ErrMetricsRecordingFailed = errors.New("metrics recording failed")

	// ErrCleanupFailed indicates cleanup after failure failed
	ErrCleanupFailed = errors.New("cleanup after failure failed")

	// ErrStrategyExecutionFailed indicates the update strategy execution failed
	ErrStrategyExecutionFailed = errors.New("update strategy execution failed")

	// ErrCircularDependencyDetected indicates a circular dependency was detected
	ErrCircularDependencyDetected = errors.New("circular dependency detected")
)

// ErrorWithContext wraps an error with additional context for better diagnosis
type ErrorWithContext struct {
	// Err is the original error
	Err error

	// Context provides additional information about the error
	Context string

	// Component identifies which component produced the error
	Component string

	// ServiceName is the service that was being processed (if applicable)
	ServiceName string

	// Version is the version that was being deployed (if applicable)
	Version string

	// Critical indicates if this is a critical error that should abort the operation
	Critical bool

	// Recoverable indicates if the error is recoverable
	Recoverable bool
}

// Error implements the error interface
func (e *ErrorWithContext) Error() string {
	msg := fmt.Sprintf("[%s] %s: %v", e.Component, e.Context, e.Err)
	if e.ServiceName != "" {
		msg = fmt.Sprintf("%s (service: %s", msg, e.ServiceName)
		if e.Version != "" {
			msg = fmt.Sprintf("%s, version: %s)", msg, e.Version)
		} else {
			msg = fmt.Sprintf("%s)", msg)
		}
	}
	return msg
}

// Unwrap returns the underlying error for errors.Is and errors.As support
func (e *ErrorWithContext) Unwrap() error {
	return e.Err
}

// WrapError wraps an error with context information
func WrapError(err error, component, context, service, version string, critical, recoverable bool) *ErrorWithContext {
	return &ErrorWithContext{
		Err:         err,
		Context:     context,
		Component:   component,
		ServiceName: service,
		Version:     version,
		Critical:    critical,
		Recoverable: recoverable,
	}
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	var ec *ErrorWithContext
	if errors.As(err, &ec) {
		return ec.Recoverable
	}
	return false
}

// IsCritical checks if an error is critical
func IsCritical(err error) bool {
	var ec *ErrorWithContext
	if errors.As(err, &ec) {
		return ec.Critical
	}
	// Default non-wrapped errors are considered critical
	return true
}

// GetErrorComponent extracts the component name from an error
func GetErrorComponent(err error) string {
	var ec *ErrorWithContext
	if errors.As(err, &ec) {
		return ec.Component
	}
	return "unknown"
}

// GetErrorService extracts the service name from an error
func GetErrorService(err error) string {
	var ec *ErrorWithContext
	if errors.As(err, &ec) {
		return ec.ServiceName
	}
	return ""
}

// GetErrorVersion extracts the version from an error
func GetErrorVersion(err error) string {
	var ec *ErrorWithContext
	if errors.As(err, &ec) {
		return ec.Version
	}
	return ""
}
