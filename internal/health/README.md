# Health Check Package

This package provides a flexible health checking system for Docker Compose services. It supports multiple verification methods including Docker's built-in health checks, HTTP endpoints, TCP connections, and custom commands.

## Core Components

### HealthChecker Interface

The package is built around the `HealthChecker` interface which all specific health checkers implement:

```go
type HealthChecker interface {
    // Check performs a health check on the specified replica
    // Returns true if healthy, false otherwise, along with any error encountered
    Check(replica replica.Replica) (bool, error)

    // CheckWithDetails performs a health check and returns detailed result information
    CheckWithDetails(replica replica.Replica) (HealthCheckResult, error)

    // Configure sets up the health checker with the provided configuration
    Configure(config HealthCheckConfig) error

    // GetType returns the type of this health checker
    GetType() HealthCheckType
}
```

### Health Check Types

The package supports four types of health checks:

1. **Docker Health Check**: Uses Docker's built-in health check mechanism
2. **HTTP Health Check**: Makes HTTP requests to specified endpoints
3. **TCP Health Check**: Attempts to establish TCP connections to verify service availability
4. **Command Health Check**: Executes commands inside containers to check health

These types are defined as constants of the `HealthCheckType` type.

### Health Check Configuration

Health checks are configured through the `HealthCheckConfig` struct:

```go
type HealthCheckConfig struct {
    // Type defines which health checker to use
    Type HealthCheckType

    // Endpoint is the URL path for HTTP checks
    Endpoint string

    // Port is the port number for TCP checks
    Port int

    // Command is the command to execute for custom checks
    Command string

    // Timeout is the maximum duration to wait for a health check to complete
    Timeout time.Duration

    // RetryInterval is the time to wait between retries
    RetryInterval time.Duration

    // SuccessThreshold is the number of consecutive successful checks required
    SuccessThreshold int

    // FailureThreshold is the number of consecutive failed checks required
    FailureThreshold int
}
```

## Testing Support

The package includes stubs and mocks for testing:

- `StubHealthChecker`: A configurable implementation of the `HealthChecker` interface
- Helper functions like `NewStubDockerHealthChecker` to create specific stub types
- These stubs can be configured to return specific health states and errors

## Usage Examples

### Basic Usage

```go
// Create health check configuration
config := health.HealthCheckConfig{
    Type:             health.HTTPHealthCheck,
    Endpoint:         "/health",
    Port:             8080,
    Timeout:          time.Second * 5,
    RetryInterval:    time.Second,
    SuccessThreshold: 2,
    FailureThreshold: 3,
}

// Create health checker using factory (to be implemented in later subtasks)
checker, err := health.NewHealthChecker(config)
if err != nil {
    log.Fatalf("Failed to create health checker: %v", err)
}

// Check a replica's health
replica := replica.Replica{
    ServiceName: "web",
    ReplicaID:   "1",
    ContainerID: "container123",
    Status:      "running",
}

healthy, err := checker.Check(replica)
if err != nil {
    log.Fatalf("Health check failed: %v", err)
}

if healthy {
    fmt.Println("Service is healthy!")
} else {
    fmt.Println("Service is unhealthy!")
}
```

### Getting Detailed Results

```go
result, err := checker.CheckWithDetails(replica)
if err != nil {
    log.Fatalf("Health check failed: %v", err)
}

fmt.Printf("Health: %v\n", result.Healthy)
fmt.Printf("Message: %s\n", result.Message)
fmt.Printf("Checked at: %s\n", result.Timestamp.Format(time.RFC3339))
```

## Implementation Notes

- Each health checker is designed to be configurable and reusable
- Health checkers maintain separation of concerns for different check types
- The package is designed to work seamlessly with the replica package
- The code is extensively tested with unit tests
