# Replica Detection Example

This example demonstrates how to use DOSync's replica detection functionality to discover and manage Docker container replicas.

## What it does

The replica detection system can:
- Detect all running container replicas from a Docker Compose file
- Get replicas for specific services
- Provide detailed information about each replica (ID, container ID, status)
- Work with the full ReplicaManager API
- Support stub implementations for testing

## Running the Example

```bash
# Navigate to the replica-detection directory
cd examples/replica-detection

# Run with default docker-compose.yml
go run main.go

# Run with a specific compose file
go run main.go ../docker-compose.yml

# Run with a specific compose file and service
go run main.go ../docker-compose.yml web
```

## Prerequisites

- Docker must be running
- Docker containers should be started to see actual replicas:
  ```bash
  cd examples
  docker-compose up -d
  ```

## Example Output

```
Detecting replicas from /path/to/docker-compose.yml

=== Example 1: Get all replicas ===
Found 2 service(s) with replicas
Service: web (3 replicas)
  Replica #1: ID=web_1, Container=abc123, Status=running
  Replica #2: ID=web_2, Container=def456, Status=running
  Replica #3: ID=web_3, Container=ghi789, Status=running

=== Example 2: Get replicas for a specific service ===
Service: web (3 replicas)
  Replica #1: ID=web_1, Container=abc123, Status=running
  ...

=== Example 3: Using the ReplicaManager API ===
Replica count summary:
  web: 3 replicas
  db: 1 replicas

=== Example 4: Using stub implementations (for testing) ===
Stub replica count: 0 services
```

## Cleanup

When you're done testing:
```bash
cd examples
docker-compose down
``` 