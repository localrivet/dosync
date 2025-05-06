---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# FAQ

## What container registries does DOSync support?

DOSync supports Docker Hub, GCR, GHCR, ACR, Quay.io, Harbor, DigitalOcean, AWS ECR, and custom/private registries. See [Configuration](configuration.md).

## How do I fix YAML parsing errors?

- Ensure your Compose file uses standard YAML syntax.
- DOSync supports both map and list forms for the `environment` field.
- See [Docker Compose Integration](docker-compose.md) for more tips.

## How do I troubleshoot Docker issues?

- Check that the Docker socket and Compose file are mounted correctly.
- Use `docker logs dosync` to view logs.
- Make sure your Docker daemon is running and accessible.

## How can I contribute?

See [Contributing](contributing.md) for guidelines and workflow.

---

## [⬅️ Testing](testing.md) | [Next ➡️ Contributing](contributing.md)
