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
      - DO_TOKEN=${DO_TOKEN}
      - CHECK_INTERVAL=1m
      - VERBOSE=--verbose
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

---

## [⬅️ Architecture](architecture.md) | [Next ➡️ Testing](testing.md)
