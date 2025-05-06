# Integration Test Environment for DoSync Manager

This directory contains test fixtures needed to run integration tests for the DoSync Rolling Update Manager.

## Test Fixtures

- `docker-compose.yml`: A basic Docker Compose configuration that includes:

  - `test-service`: A standalone service with 2 replicas
  - `test-service-with-deps`: A service that depends on test-service

- `docker-compose-fail.yml`: A Docker Compose configuration designed to test failure scenarios:
  - `test-service-fail`: A service that can be configured to fail health checks (controlled via environment variables)

## Running Integration Tests

The integration tests require Docker to be running on your machine. They will create and destroy containers as part of the testing process.

By default, integration tests will be skipped if the `SKIP_INTEGRATION_TESTS` environment variable is set. This is useful in CI environments where you might not want to run integration tests.

To run the integration tests:

```bash
# Run all tests including integration tests
go test -v ./internal/manager

# Skip integration tests
SKIP_INTEGRATION_TESTS=1 go test -v ./internal/manager

# Run a specific integration test
go test -v ./internal/manager -run TestEndToEndRollingUpdate
```

## Test Scenarios

The integration tests cover several key scenarios:

1. **End-to-End Rolling Update** (`TestEndToEndRollingUpdate`): Tests the entire update process from start to finish with a simulated Docker environment.

2. **Dependency Order Preservation** (`TestDependencyOrderPreservation`): Ensures that when updating services with dependencies, the dependency order is correctly respected.

3. **Rollback on Failure** (`TestRollbackOnFailure`): Verifies that the system can detect failures during the update process and automatically roll back to a previous stable state.

4. **Multiple Update Strategies** (`TestMultipleStrategies`): Tests the system's behavior with different update strategies (one-at-a-time, all-at-once, rolling-update).

## Mocking

The integration tests use mocks for certain components to facilitate testing specific scenarios:

- `MockHealthChecker`: Used to simulate health check failures without actually having failing containers.
- `MockLogger`: Captures log output for verification of operations and their sequence.

## Adding New Tests

When adding new integration tests:

1. Update the fixture files as needed
2. Make sure to include the skip logic for CI environments
3. Use mocks judiciously to isolate specific behaviors
4. Add clear verification steps to confirm expected behavior
