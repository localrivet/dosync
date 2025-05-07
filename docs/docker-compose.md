---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# Docker Compose Integration

DOSync is designed to work seamlessly with Docker Compose. This guide covers best practices for integration.

## Mounting the Docker Socket

To allow DOSync to control Docker, mount the Docker socket in your Compose file:

```yaml
services:
  dosync:
    image: localrivet/dosync:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./docker-compose.yml:/app/docker-compose.yml
      - ./backups:/app/backups
    environment:
      - DOCR_TOKEN=${DOCR_TOKEN}
      - CHECK_INTERVAL=1m
      - VERBOSE=true
```

## Updating Compose Files

- DOSync updates the Compose file in place and creates backups in the specified directory.
- Make sure the Compose file is mounted read-write.

## YAML Compatibility

- DOSync supports both map and list forms of the `environment` field.
- If you encounter YAML parsing issues, check your Compose file for non-standard fields or formats.

## Troubleshooting

- Ensure the Docker socket and Compose file are correctly mounted.
- Use high, uncommon ports in test environments to avoid conflicts.
- Check logs for errors: `docker logs dosync`

See [README](../README.md), [Configuration](configuration.md), and [Architecture](architecture.md) for more details.

## Environment Variable Support for Flags

DOSync supports setting all `sync` command flags and config/env file paths via environment variables. This is especially useful for Docker Compose and CI/CD environments.

**CLI flags always take precedence over environment variables.**

| Flag/Config Option    | Environment Variable     | Example Value           |
| --------------------- | ------------------------ | ----------------------- |
| --config, -c          | CONFIG_PATH              | /app/dosync.yaml        |
| --env-file, -e        | ENV_FILE                 | /app/.env               |
| --file, -f            | SYNC_FILE                | /app/docker-compose.yml |
| --interval, -i        | SYNC_INTERVAL            | 5m                      |
| --verbose, -v         | SYNC_VERBOSE             | true                    |
| --rolling-update      | SYNC_ROLLING_UPDATE      | false                   |
| --strategy            | SYNC_STRATEGY            | canary                  |
| --health-check        | SYNC_HEALTH_CHECK        | http                    |
| --health-endpoint     | SYNC_HEALTH_ENDPOINT     | /status                 |
| --delay               | SYNC_DELAY               | 30s                     |
| --rollback-on-failure | SYNC_ROLLBACK_ON_FAILURE | true                    |

**Example Docker Compose usage:**

```yaml
services:
  dosync:
    image: localrivet/dosync:latest
    container_name: dosync
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./docker-compose.yml:/app/docker-compose.yml
      - ./backups:/app/backups
      - ./deploy/dosync/dosync.yaml:/app/dosync.yaml
    env_file:
      - .env
    environment:
      - DOCR_TOKEN=${DOCR_TOKEN}
      - CONFIG_PATH=/app/dosync.yaml
      - ENV_FILE=/app/.env
      - SYNC_FILE=/app/docker-compose.yml
      - SYNC_INTERVAL=1m
      - SYNC_VERBOSE=true
      # - SYNC_ROLLING_UPDATE=false
      # - SYNC_STRATEGY=canary
      # - SYNC_HEALTH_CHECK=http
      # - SYNC_HEALTH_ENDPOINT=/status
      # - SYNC_DELAY=30s
      # - SYNC_ROLLBACK_ON_FAILURE=true
    networks:
      - proxy
```

You can set any of the above environment variables to control DOSync's behavior. CLI flags will always override environment variables if both are provided.

---

## [⬅️ Architecture](architecture.md) | [Next ➡️ Testing](testing.md)
