---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# Getting Started

Welcome to DOSync! This guide will help you get up and running quickly.

## Supported Container Registries

DOSync works with the following registries:

- Docker Hub
- AWS Elastic Container Registry (ECR)
- DigitalOcean Container Registry
- Azure Container Registry (ACR)
- Google Container Registry (GCR)
- GitHub Container Registry (GHCR)
- Harbor
- Quay.io
- Custom/private registries

See [Configuration](configuration.md) for details on setting up credentials for each registry.

## Starter Docker Compose Configuration

Here is a minimal example to run DOSync as a service:

```yaml
services:
  dosync:
    image: localrivet/dosync:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./docker-compose.yml:/app/docker-compose.yml
      - ./backups:/app/backups
      - ./dosync.yaml:/app/dosync.yaml # Mount your dosync.yaml config
    environment:
      - DOCKERHUB_USERNAME=${DOCKERHUB_USERNAME}
      - DOCKERHUB_PASSWORD=${DOCKERHUB_PASSWORD}
      - CHECK_INTERVAL=1m
```

## Starter dosync.yaml

A minimal `dosync.yaml` configuration:

```yaml
registry:
  dockerhub:
    username: ${DOCKERHUB_USERNAME}
    password: ${DOCKERHUB_PASSWORD}
backup:
  dir: ./backups
  keep: 5
```

For more advanced configuration, see [Configuration](configuration.md).

## Installation

### Option 1: Docker Compose (Recommended)

- Use the provided helper script or add DOSync manually to your Compose file.
- See the [README](../README.md) for full details.

### Option 2: Install from Release

- Download and run the install script.

### Option 3: Build from Source

- Clone the repo and build with `make build`.

## Quick Setup

1. Add DOSync to your Docker Compose file or run the binary.
2. Provide registry credentials as environment variables or in a `.env` file.
3. Start DOSync:
   - As a service: `docker compose up -d dosync`
   - Manually: `dosync sync -f docker-compose.yml`

## First Sync

Run DOSync to check for updates and synchronize your services:

```bash
dosync sync -f docker-compose.yml
```

For advanced configuration, see [Configuration](configuration.md).

[⬆️ Back to Home](index.md)

---

## [⬅️ Home](index.md) | [Next ➡️ Configuration](configuration.md)

---
