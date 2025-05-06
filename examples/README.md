# DOSync Examples

This directory contains example programs demonstrating various features of the DOSync library.

## Replica Detection Example

The `replica_detection.go` example demonstrates how to use the `replica` package to detect and manage service replicas in Docker Compose environments.

### Why Replica Detection Matters

In real-world Docker deployments, you often run multiple copies of the same service:

- Multiple web servers for load balancing
- Blue-green deployments for zero-downtime updates
- Scaled database instances for high availability

This creates a challenge: **How do you know which containers belong together as replicas of the same service?**

This example shows how DOSync solves this problem by automatically detecting:

- Replicas created with Docker Compose's `scale` property
- Replicas created with Docker Compose's `deploy.replicas` directive
- Replicas created using naming patterns (like service-1, service-blue)

With this capability, you can consistently update, monitor, and manage all instances of a service without complex manual tracking.

### Prerequisites

- Docker and Docker Compose installed
- Go 1.16 or higher

### Running the Example

#### Option 1: Using the Automated Script (Recommended)

For the easiest experience, use the provided shell script that handles all setup, execution, and cleanup:

```bash
cd examples
./run_example.sh
```

This script will:

1. Check if Docker and Docker Compose are available
2. Build the example if needed
3. Start the Docker Compose services
4. Run the replica detector example
5. Ask if you want to keep the containers running or clean them up

#### Option 2: Manual Steps

If you prefer to run the steps manually:

1. Start the example Docker Compose services:

```bash
cd examples
docker-compose up -d
```

This will start several services defined in `docker-compose.yml`, including:

- Web service (3 replicas using the `scale` property)
- API service (2 replicas using the `deploy.replicas` property)
- Database services (blue-green pattern using naming convention)
- Cache services (replicas using naming convention)

2. Build and run the example:

```bash
# From project root
go build -o examples/replica_detector examples/replica_detection.go
cd examples
./replica_detector
```

Or simply:

```bash
go run examples/replica_detection.go
```

3. To get replicas for a specific service, pass the service name as the second argument:

```bash
./replica_detector docker-compose.yml web
```

### Example Output

The example demonstrates several methods of detecting and handling replicas:

1. Using the convenience function `DetectAllReplicas`
2. Using the convenience function `DetectServiceReplicas` for a specific service
3. Using the full `ReplicaManager` API
4. Using stub implementations for testing

### Error Handling

The example includes robust error handling for common issues:

- Docker daemon not running
- Docker Compose file not found
- Permission issues
- Service not found

If you encounter errors, the program will provide specific suggestions to help resolve them.

### When No Replicas Are Found

If the program doesn't detect any replicas, it likely means that the Docker containers aren't running. The program will provide instructions on how to start them using Docker Compose.

### Cleanup

To stop and remove the Docker containers when you're done:

```bash
docker-compose down
```
