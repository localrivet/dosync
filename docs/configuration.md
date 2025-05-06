---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# Configuration

DOSync is highly configurable to fit a variety of deployment scenarios. This guide covers the main configuration options.

## dosync.yaml

The `dosync.yaml` file is the primary configuration file for DOSync. Place it in your project root or specify its path with the `--config` flag.

### Example

```yaml
registry:
  dockerhub:
    username: myuser
    password: ${DOCKERHUB_PASSWORD}
    imagePolicy:
      filterTags:
        pattern: '^main-'
      policy:
        numerical:
          order: desc
  gcr:
    credentials_file: /path/to/gcp.json
  # ...other registries
metrics:
  enabled: true
  listen: 0.0.0.0:9090
admin:
  enabled: true
  listen: 0.0.0.0:8080
  auth_token: ${ADMIN_TOKEN}
backup:
  dir: ./backups
  keep: 10
```

### Field Reference

- `registry`: Registry credentials and image policies (see below)
- `metrics`: Metrics API configuration
  - `enabled`: Enable/disable metrics endpoint
  - `listen`: Address/port to bind the metrics server
- `admin`: Admin API configuration
  - `enabled`: Enable/disable admin endpoint
  - `listen`: Address/port to bind the admin server
  - `auth_token`: Optional token for securing admin API
- `backup`: Backup settings
  - `dir`: Directory for Compose file backups
  - `keep`: Number of backups to retain

## Registry Credentials

Set credentials as environment variables or in a `.env` file:

```env
DO_TOKEN=your_digitalocean_token_here
DOCKERHUB_USERNAME=youruser
DOCKERHUB_PASSWORD=yourpassword
AWS_ACCESS_KEY_ID=yourkey
AWS_SECRET_ACCESS_KEY=yoursecret
```

## Image Policy Configuration

Define how DOSync selects tags for updates:

```yaml
imagePolicy:
  filterTags:
    pattern: '^v(?P<semver>[0-9]+\.[0-9]+\.[0-9]+)$'
    extract: '$semver'
  policy:
    semver:
      range: '>=1.0.0 <2.0.0'
```

## Image Tag Handling & Policies

DOSync provides flexible, powerful ways to control which image tags are selected for updates. This is managed via the `imagePolicy` section in your `dosync.yaml`.

### How Tag Selection Works

- **Filtering:** Use regex patterns to include/exclude tags.
- **Extraction:** Pull out values (like semver or timestamps) from complex tags using named groups.
- **Policy:** Choose how to sort and select the "best" tag (semver, numerical, alphabetical).

### Supported Policy Types

- **semver:** Selects tags based on semantic versioning, with optional version constraints.
- **numerical:** Selects tags based on extracted numbers (e.g., build numbers, timestamps).
- **alphabetical:** Selects tags based on lexicographical order (useful for date-based tags).

### Example: Semver Policy (Standard Tags)

```yaml
registry:
  dockerhub:
    imagePolicy:
      policy:
        semver:
          range: '>=1.0.0 <2.0.0' # Only 1.x versions
```

### Example: Semver Policy with Extraction

For tags like `v1.2.3-alpine3.17`:

```yaml
registry:
  ghcr:
    imagePolicy:
      filterTags:
        pattern: '^(?P<semver>[0-9]+\.[0-9]+\.[0-9]+)-.*'
        extract: '$semver'
      policy:
        semver:
          range: '>=1.0.0'
```

### Example: Numerical Policy (Build Timestamps)

For tags like `main-abc1234-1718435261`:

```yaml
registry:
  dockerhub:
    imagePolicy:
      filterTags:
        pattern: '^main-[a-fA-F0-9]+-(?P<ts>\d+)$'
        extract: '$ts'
      policy:
        numerical:
          order: desc # Highest timestamp wins
```

### Example: Alphabetical Policy (Date-Based Tags)

For tags like `RELEASE.2024-06-01T12-00-00Z`:

```yaml
registry:
  quay:
    imagePolicy:
      filterTags:
        pattern: '^RELEASE\.(?P<timestamp>.*)Z$'
        extract: '$timestamp'
      policy:
        alphabetical:
          order: desc # Most recent date wins
```

### Example: Filtering for Release Candidates Only

```yaml
registry:
  dockerhub:
    imagePolicy:
      filterTags:
        pattern: '.*-rc.*'
      policy:
        semver:
          range: ''
```

See [Architecture](architecture.md) for more on how tag selection fits into the update flow.

## Metrics API

DOSync exposes a Prometheus-compatible metrics endpoint if enabled in `dosync.yaml`:

- **Default endpoint:** `/metrics` (e.g., `http://localhost:9090/metrics`)
- **Configurable via:**
  ```yaml
  metrics:
    enabled: true
    listen: 0.0.0.0:9090
  ```
- **Metrics include:**
  - Sync counts, errors, durations, last sync time, etc.
- **See:** `internal/metrics/` for implementation details.

## Admin API

The Admin API provides endpoints for health checks, triggering syncs, and administrative actions.

- **Default endpoint:** `/admin` (e.g., `http://localhost:8080/admin`)
- **Configurable via:**
  ```yaml
  admin:
    enabled: true
    listen: 0.0.0.0:8080
    auth_token: ${ADMIN_TOKEN}
  ```
- **Endpoints include:**
  - `/admin/health` — Health check
  - `/admin/sync` — Trigger a manual sync
  - `/admin/status` — Get current sync status
- **See:** `internal/health/` and `internal/manager/` for implementation details.

See [Usage](usage.md) and [Architecture](architecture.md) for more on how configuration affects operation.

---

## [⬅️ Getting Started](getting-started.md) | [Next ➡️ Usage](usage.md)
