# Docker Compose Replica Detection

This package provides functionality to detect and manage service replicas in Docker Compose environments. It supports both scale-based replicas (using Docker Compose's `scale` property or `deploy.replicas`) and name-based replicas (services with naming patterns like `service-1`, `service-2`, or `service-blue`, `service-green`).

## Why This Package Matters

### The Problem

When working with Docker Compose environments, you often need to run multiple instances (replicas) of the same service for:

- **Reliability**: If one container crashes, others can still handle requests
- **Scalability**: Handling more traffic by adding more instances
- **Blue-Green Deployments**: Running old and new versions side-by-side for zero-downtime updates

Docker Compose offers multiple ways to create these replicas:

1. Using the `scale` property
2. Using `deploy.replicas` (Swarm mode syntax)
3. Manually defining services with naming patterns (like `service-blue`, `service-green`)

However, there's no built-in way to easily identify which containers belong to which logical service across these different approaches. This makes updating, monitoring, and managing these replicas challenging.

### The Solution

This package solves these problems by:

1. **Automatic Detection**: Identifying all replicas regardless of how they were created
2. **Unified API**: Providing a simple interface to work with replicas regardless of their type
3. **Extensible Design**: Supporting both current replica types and allowing for new detection strategies
4. **Testing Support**: Offering stub implementations for testing without Docker dependencies

With this package, applications can consistently manage replicas for updates, monitoring, or configuration changes without manual intervention or custom scripts for each replica type.

## Key Components

### Core Types

- `Replica`: Represents a single instance of a service with properties like `ServiceName`, `ReplicaID`, `ContainerID`, and `Status`.
- `ReplicaType`: Defines the kind of replica detection strategy (`ScaleBased` or `NameBased`).
- `ReplicaDetector`: Interface implemented by different replica detection strategies.

### Detectors

- `ScaleBasedDetector`: Detects replicas created using Docker Compose's scale property or deploy.replicas directive.
- `NamedServiceDetector`: Detects replicas based on naming patterns (e.g., `service-1`, `service-blue`).

### Manager

- `ReplicaManager`: Manages multiple detector types and provides a unified API for working with replicas.

## Usage Examples

### Basic Usage with Convenience Functions

```go
// Get all replicas from a Docker Compose file
allReplicas, err := replica.DetectAllReplicas("docker-compose.yml")
if err != nil {
    // Handle error
}

// Get replicas for a specific service
webReplicas, err := replica.DetectServiceReplicas("docker-compose.yml", "web")
if err != nil {
    // Handle error
}

// Get a specific detector type
scaleDetector, err := replica.GetDetectorByType("docker-compose.yml", replica.ScaleBased)
if err != nil {
    // Handle error
}
```

### Using the ReplicaManager API

```go
// Create a replica manager with all detector types
manager, err := replica.NewReplicaManagerWithAllDetectors("docker-compose.yml")
if err != nil {
    // Handle error
}

// Get replicas for a specific service
webReplicas, err := manager.GetServiceReplicas("web")
if err != nil {
    // Handle error
}

// Get all replicas
allReplicas, err := manager.GetAllReplicas()
if err != nil {
    // Handle error
}

// Refresh replica information
err = manager.RefreshReplicas()
if err != nil {
    // Handle error
}

// Check if a specific detector is registered
if manager.HasDetector(replica.ScaleBased) {
    // Scale-based detector is available
}

// Access a specific detector
if detector := manager.GetDetector(replica.NameBased); detector != nil {
    // Use the name-based detector
}

// Unregister a detector if needed
if manager.UnregisterDetector(replica.ScaleBased) {
    // Scale-based detector was successfully removed
}
```

### Using Stub Implementations for Testing

```go
// Create a stub replica manager for testing
stubManager, err := replica.CreateStubReplicaManager("docker-compose.yml")
if err != nil {
    // Handle error
}

// Access the stub detectors and configure them with test data
scaleDetector := stubManager.GetDetector(replica.ScaleBased).(*replica.StubScaleBasedDetector)
scaleDetector.Replicas = map[string][]replica.Replica{
    "web": {
        {ServiceName: "web", ReplicaID: "1", ContainerID: "container1", Status: "running"},
        {ServiceName: "web", ReplicaID: "2", ContainerID: "container2", Status: "running"},
    },
}

nameDetector := stubManager.GetDetector(replica.NameBased).(*replica.StubNamedServiceDetector)
nameDetector.Replicas = map[string][]replica.Replica{
    "database": {
        {ServiceName: "database", ReplicaID: "blue", ContainerID: "container3", Status: "running"},
        {ServiceName: "database", ReplicaID: "green", ContainerID: "container4", Status: "running"},
    },
}

// Use the stubbed manager in tests
replicas, err := stubManager.GetAllReplicas()
```

## Supported Naming Patterns

The `NamedServiceDetector` supports the following naming patterns:

1. Dash-separated: `service-1`, `service-2`, `service-blue`, `service-green`
2. Underscore-separated: `service_1`, `service_2`, `service_blue`
3. Dot-separated: `service.1`, `service.2`, `service.blue`

## Detection Process

1. Parse the Docker Compose file to identify services with scaling configurations or naming patterns
2. Connect to Docker API to find actual running containers for those services
3. Create `Replica` instances for each service instance with appropriate metadata
4. Cache and return the detected replicas

## Error Handling

The package provides detailed error information to help diagnose common issues:

- Docker connectivity problems
- Docker Compose file parsing issues
- Permission issues
- Missing services or containers

All errors are wrapped with context using `fmt.Errorf` with the `%w` verb to maintain error chains and provide better debugging information.

## Running the Example

The package includes a complete working example in the `examples` directory. For the simplest experience, use the provided shell script:

```bash
cd examples
./run_example.sh
```

See the [Examples README](../../examples/README.md) for more details on running the examples.

## Dependencies

- Docker Engine API (github.com/docker/docker/client)
- YAML parser (gopkg.in/yaml.v2)

## Testing

The package includes both unit tests and integration tests. Integration tests require Docker to be running and are skipped by default. To run integration tests, remove the `t.Skip()` calls in the test files.

### Running Tests

```bash
# Run unit tests only
go test ./internal/replica

# Run with verbose output
go test -v ./internal/replica

# Run specific test
go test -run TestNamedServiceRegexPatterns ./internal/replica
```

## Extension

To add a new replica detection strategy:

1. Create a new struct that implements the `ReplicaDetector` interface
2. Add a new `ReplicaType` constant in `types.go`
3. Register the new detector in `NewReplicaManagerWithAllDetectors`
