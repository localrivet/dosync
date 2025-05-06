---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# Testing

DOSync includes both unit and integration tests to ensure reliability.

## Running Tests

Run all tests with:

```bash
go test ./...
```

## Unit Tests

- Located throughout the codebase, especially in `internal/` modules.
- Use mocks for Docker and registry interactions.

## Integration Tests

- See `internal/manager/integration_test.go` for end-to-end tests.
- Test data is in `internal/manager/internal/manager/testdata/` and `internal/manager/testdata/`.
- Integration tests use real Docker Compose files and containers.

## Mocking

- Mock detectors and registry clients are used in unit tests to simulate real-world scenarios.

See [Contributing](contributing.md) and [Architecture](architecture.md) for more details.

---

## [⬅️ Docker Compose](docker-compose.md) | [Next ➡️ FAQ](faq.md)
