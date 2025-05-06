---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# Usage

This guide covers how to use DOSync from the command line and as a service.

## CLI Usage

Run DOSync manually to sync your services:

```bash
dosync sync -f docker-compose.yml
```

### Common Flags

- `-e, --env-file string` Path to .env file with registry credentials
- `-f, --file string` Path to docker-compose.yml file (required)
- `-i, --interval duration` Polling interval (default: 5m)
- `-v, --verbose` Enable verbose output

## Running as a Service

After installation, DOSync can run as a systemd service:

```bash
sudo systemctl start dosync.service
sudo systemctl enable dosync.service
sudo systemctl status dosync.service
sudo journalctl -u dosync.service -f
```

Or as a container in Docker Compose:

```bash
docker compose up -d dosync
```

## Advanced Usage

- Use custom polling intervals: `dosync sync -f docker-compose.yml -i 2m`
- Use with different Compose files: `dosync sync -f my-stack.yml`
- Enable verbose output for debugging: `dosync sync -f docker-compose.yml --verbose`

See [Configuration](configuration.md) and [Docker Compose Integration](docker-compose.md) for more details.

---

## [⬅️ Configuration](configuration.md) | [Next ➡️ Architecture](architecture.md)
