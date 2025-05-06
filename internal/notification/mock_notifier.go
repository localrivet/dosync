package notification

import (
	"time"
)

// MockNotifier is an implementation of the Notifier interface for testing purposes
type MockNotifier struct {
	BaseNotifier
	DeploymentStartedCalled bool
	DeploymentSuccessCalled bool
	DeploymentFailureCalled bool
	RollbackCalled          bool
	LastService             string
	LastVersion             string
	LastFromVersion         string
	LastToVersion           string
	LastErrorMessage        string
	LastDuration            time.Duration
	ErrorToReturn           error
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier(config NotificationConfig) *MockNotifier {
	m := &MockNotifier{}
	_ = m.Configure(config)
	return m
}

// SendDeploymentStarted records that the method was called and returns ErrorToReturn
func (m *MockNotifier) SendDeploymentStarted(service string, version string) error {
	m.DeploymentStartedCalled = true
	m.LastService = service
	m.LastVersion = version
	return m.ErrorToReturn
}

// SendDeploymentSuccess records that the method was called and returns ErrorToReturn
func (m *MockNotifier) SendDeploymentSuccess(service string, version string, duration time.Duration) error {
	m.DeploymentSuccessCalled = true
	m.LastService = service
	m.LastVersion = version
	m.LastDuration = duration
	return m.ErrorToReturn
}

// SendDeploymentFailure records that the method was called and returns ErrorToReturn
func (m *MockNotifier) SendDeploymentFailure(service string, version string, errorMessage string) error {
	m.DeploymentFailureCalled = true
	m.LastService = service
	m.LastVersion = version
	m.LastErrorMessage = errorMessage
	return m.ErrorToReturn
}

// SendRollback records that the method was called and returns ErrorToReturn
func (m *MockNotifier) SendRollback(service string, fromVersion string, toVersion string) error {
	m.RollbackCalled = true
	m.LastService = service
	m.LastFromVersion = fromVersion
	m.LastToVersion = toVersion
	return m.ErrorToReturn
}
