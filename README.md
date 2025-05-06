# DOSync

[![Go Reference](https://pkg.go.dev/badge/github.com/localrivet/dosync.svg)](https://pkg.go.dev/github.com/localrivet/dosync)
[![Go Report Card](https://goreportcard.com/badge/github.com/localrivet/dosync)](https://goreportcard.com/report/github.com/localrivet/dosync)

**DOSync** is a tool that synchronizes Docker Compose services with the latest images from any supported container registry. It automates the process of checking for new image tags, updating your Docker Compose file, and restarting the relevant services to ensure your deployments are always running the latest versions.

## Features

- Automatic polling of all major container registries (Docker Hub, GCR, GHCR, ACR, Quay.io, Harbor, DigitalOcean, AWS ECR, custom, etc.) for new image tags
- Docker Compose file updating with new image tags
- Service restarting after image updates
- Docker image pruning to save disk space
- Backup creation of Docker Compose files before modifications
- **Intelligent service replica detection** for both scale-based and name-based replicas
- **Advanced tag selection policies** for controlling which tags to use (semver, numerical, regex)
- Simple systemd service integration
- Configurable polling interval
- **Self-contained container that can update other services in the same Docker Compose file**

## Installation

### Option 1: Docker Compose (Recommended)

The easiest way to use DOSync is to include it in your Docker Compose file. This way, it runs as a container alongside your other services and can update them when new images are available.

#### Automatic Setup with Helper Script

We provide a helper script that can automatically add DOSync to your existing Docker Compose project:

```bash
# Download the script
curl -sSL https://raw.githubusercontent.com/localrivet/dosync/main/add-to-compose.sh > add-to-compose.sh
chmod +x add-to-compose.sh

# Run it (providing any required registry credentials as environment variables)
./add-to-compose.sh

# Start DOSync
docker compose up -d dosync
```

#### Manual Setup

Alternatively, you can manually add the DOSync service to your Docker Compose file:

```yaml
services:
  # Your other services here
  webapp:
    image: ghcr.io/your-org/webapp:latest
    # ...
  api:
    image: gcr.io/your-project/api:latest
    # ...
  backend:
    image: registry.digitalocean.com/your-registry/backend:latest
    # ...
  frontend:
    image: 123456789012.dkr.ecr.us-east-1.amazonaws.com/frontend:latest
    # ...

  # Self-updating DOSync service
  dosync:
    image: localrivet/dosync:latest
    restart: unless-stopped
    volumes:
      # Mount the Docker socket to allow controlling the Docker daemon
      - /var/run/docker.sock:/var/run/docker.sock
      # Mount the actual docker-compose.yml file that's being used to run the stack
      - ./docker-compose.yml:/app/docker-compose.yml
      # Mount a directory for backups
      - ./backups:/app/backups
    environment:
      # Registry credentials as needed (see configuration below)
      - DO_TOKEN=${DO_TOKEN} # Only required for DigitalOcean
      - CHECK_INTERVAL=1m
      - VERBOSE=--verbose
```

See [docker-compose.example.yaml](docker-compose.example.yaml) for a complete example.

### Option 2: Install from Release

```bash
# Download and run the installation script
curl -sSL https://raw.githubusercontent.com/localrivet/dosync/main/install.sh | bash
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/localrivet/dosync.git
cd dosync

# Build the binary
make build

# Install the binary
sudo cp ./release/$(go env GOOS)/$(go env GOARCH)/dosync /usr/local/bin/dosync
sudo chmod +x /usr/local/bin/dosync
```

## Configuration

### Registry Credentials

Create a `.env` file or set environment variables with your registry credentials as needed. For example:

```env
# DigitalOcean (only if using DigitalOcean Container Registry)
DO_TOKEN=your_digitalocean_token_here
# Docker Hub
DOCKERHUB_USERNAME=youruser
DOCKERHUB_PASSWORD=yourpassword
# AWS ECR
AWS_ACCESS_KEY_ID=yourkey
AWS_SECRET_ACCESS_KEY=yoursecret
# ...and so on for other registries
```

### Docker Compose File

Your Docker Compose file can use images from any supported registry:

```yaml
services:
  backend:
    image: registry.digitalocean.com/your-registry/backend:latest
    # ...
  frontend:
    image: ghcr.io/your-org/frontend:latest
    # ...
  api:
    image: gcr.io/your-project/api:latest
    # ...
  worker:
    image: quay.io/yourorg/worker:latest
    # ...
```

### Multi-Registry Support

DOSync supports syncing images from multiple container registries, including Docker Hub, GCR, GHCR, ACR, Quay.io, Harbor, DigitalOcean Container Registry, AWS ECR, and custom/private registries.

To configure credentials for these registries, add a `registry` section to your dosync.yaml file. **All fields are optional**—only specify the registries you need. You can use environment variable expansion for secrets.

Example:

```yaml
registry:
  dockerhub:
    username: myuser
    password: ${DOCKERHUB_PASSWORD}
    imagePolicy: # Optional image policy configuration
      filterTags:
        pattern: '^main-' # Only consider tags starting with 'main-'
      policy:
        numerical:
          order: desc # Select the highest numerical value
  gcr:
    credentials_file: /path/to/gcp.json
  ghcr:
    token: ${GITHUB_PAT}
    imagePolicy:
      filterTags:
        pattern: '^v(?P<semver>[0-9]+\.[0-9]+\.[0-9]+)$'
        extract: '$semver'
      policy:
        semver:
          range: '>=1.0.0 <2.0.0' # Only use 1.x versions
  acr:
    tenant_id: your-tenant-id
    client_id: your-client-id
    client_secret: ${AZURE_CLIENT_SECRET}
    registry: yourregistry.azurecr.io
  quay:
    token: ${QUAY_TOKEN}
  harbor:
    url: https://myharbor.domain.com
    username: myuser
    password: ${HARBOR_PASSWORD}
  docr:
    token: ${DOCR_TOKEN}
  ecr:
    aws_access_key_id: ${AWS_ACCESS_KEY_ID}
    aws_secret_access_key: ${AWS_SECRET_ACCESS_KEY}
    region: us-east-1
    registry: 123456789012.dkr.ecr.us-east-1.amazonaws.com
  custom:
    url: https://custom.registry.com
    username: myuser
    password: ${CUSTOM_REGISTRY_PASSWORD}
```

See the code comments in `internal/config/config.go` for more details on each field.

### Image Policy Configuration

DOSync allows you to define sophisticated policies for selecting which image tags to use. This is especially useful for CI/CD pipelines where tag patterns may contain branch names, timestamps, or version information.

Each registry configuration can include an `imagePolicy` section with the following components:

1. **Tag Filtering (optional)**: Use regex patterns to filter which tags are considered
2. **Value Extraction (optional)**: Extract values from tags using named groups
3. **Policy Selection**: Choose how to sort and select the "best" tag (numerical, semver, alphabetical)

If no policy is specified, DOSync defaults to using the lexicographically highest tag, preferring non-prerelease tags if available (like traditional container registries).

#### Policy Types

##### Numerical Policy

Select tags based on numerical values, useful for tags containing timestamps or build numbers.

```yaml
imagePolicy:
  filterTags:
    pattern: '^main-[a-zA-F0-9]+-(?P<ts>\d+)$' # Match format: main-hash-timestamp
    extract: '$ts' # Extract the timestamp value
  policy:
    numerical:
      order: desc # Select highest number (newest)
```

Example: With tags `["main-abc123-100", "main-def456-200", "main-ghi789-150"]`, this policy selects `main-def456-200`.

##### Semver Policy

Select tags based on semantic versioning rules, optionally with version constraints.

```yaml
imagePolicy:
  policy:
    semver: # Select highest semver without constraints
      range: '' # Empty means any valid semver
```

Or with constraints:

```yaml
imagePolicy:
  policy:
    semver:
      range: '>=1.0.0 <2.0.0' # Only select from 1.x versions
```

Example: With tags `["v1.2.3", "v1.2.4", "v2.0.0", "v2.0.0-rc1"]`, the above policy selects `v1.2.4`.

You can extract the version from complex tag formats:

```yaml
imagePolicy:
  filterTags:
    pattern: '^v(?P<semver>[0-9]+\.[0-9]+\.[0-9]+)(-[a-z]+)?$'
    extract: '$semver'
  policy:
    semver:
      range: '>=1.0.0'
```

##### Alphabetical Policy

Select tags based on alphabetical ordering, useful for date-based formats like RELEASE.DATE.

```yaml
imagePolicy:
  filterTags:
    pattern: '^RELEASE\.(?P<timestamp>.*)Z$' # Match format: RELEASE.2024-01-01T00-00-00Z
    extract: '$timestamp' # Extract the timestamp portion
  policy:
    alphabetical:
      order: desc # Select alphabetically highest (newest)
```

Example: With tags `["RELEASE.2024-06-01T12-00-00Z", "RELEASE.2024-06-02T12-00-00Z"]`, this selects `RELEASE.2024-06-02T12-00-00Z`.

#### Common Policy Examples

##### Build Pipeline Tags with Git SHA and Timestamp

For tags like `main-abc1234-1718435261`:

```yaml
imagePolicy:
  filterTags:
    pattern: '^main-[a-fA-F0-9]+-(?P<ts>\d+)$'
    extract: '$ts'
  policy:
    numerical:
      order: desc # Highest timestamp wins
```

##### Semantic Versioning Tags

For standard semver tags like `v1.2.3`:

```yaml
imagePolicy:
  policy:
    semver:
      range: '>=1.0.0' # Any version 1.0.0 or higher
```

For only stable 1.x versions:

```yaml
imagePolicy:
  policy:
    semver:
      range: '>=1.0.0 <2.0.0' # Only 1.x versions
```

For including pre-releases:

```yaml
imagePolicy:
  policy:
    semver:
      range: '>=1.0.0-0' # Include pre-releases
```

##### Filtering Specific Release Candidates

For only using release candidates:

```yaml
imagePolicy:
  filterTags:
    pattern: '.*-rc.*'
  policy:
    semver:
      range: ''
```

##### Extracting Semver from Complex Tags

For tags like `1.2.3-alpine3.17`:

```yaml
imagePolicy:
  filterTags:
    pattern: '^(?P<semver>[0-9]*\.[0-9]*\.[0-9]*)-.*'
    extract: '$semver'
  policy:
    semver:
      range: '>=1.0.0'
```

##### Date-based Releases (like Minio)

For tags like `RELEASE.2023-01-31T08-42-01Z`:

```yaml
imagePolicy:
  filterTags:
    pattern: '^RELEASE\.(?P<timestamp>.*)Z$'
    extract: '$timestamp'
  policy:
    alphabetical:
      order: asc # Ascending for dates in this format
```

## Usage

### Basic Usage

```bash
# Run manually with default settings
dosync sync -f docker-compose.yml

# Run with environment file and verbose output
dosync sync -e .env -f docker-compose.yml --verbose

# Run with custom polling interval (check every 5 minutes)
dosync sync -f docker-compose.yml -i 5m
```

### Command Reference

```
dosync sync [flags]

Flags:
  -e, --env-file string     Path to .env file with registry credentials
  -f, --file string         Path to docker-compose.yml file (required)
  -h, --help                Help for sync command
  -i, --interval duration   Polling interval (default: 5m)
  -v, --verbose             Enable verbose output
```

### Running as a Service

After installation, the script creates a systemd service:

```bash
# Start the service
sudo systemctl start dosync.service

# Enable automatic start on boot
sudo systemctl enable dosync.service

# Check service status
sudo systemctl status dosync.service

# View service logs
sudo journalctl -u dosync.service -f
```

## How It Works

1. DOSync polls all configured container registries according to the specified interval
2. It checks for new image tags for each service defined in your Docker Compose file
3. When a new tag is found, it updates the Docker Compose file
4. It then uses `docker compose up -d --no-deps` to restart only the affected services
5. Old images are pruned to save disk space

## Replica Detection

DOSync includes a sophisticated replica detection system that can identify and manage different types of service replicas in Docker Compose environments:

### Why Replica Detection Matters

**The Problem:**
Modern applications often run multiple copies (replicas) of the same service for reliability, load balancing, and zero-downtime deployments. When updating these services, you need to know:

- Which containers belong to which service
- How many replicas exist
- Whether they're using scale-based replication or name-based patterns like blue-green deployments

Without this knowledge, updates can become inconsistent or require manual intervention.

**Our Solution:**
DOSync's replica detection automatically identifies all replicas of your services regardless of how they're deployed, allowing for:

- Consistent updates across all replicas of a service
- Proper handling of blue-green deployments
- Support for both Docker Compose scaling and custom naming patterns
- Zero-downtime rolling updates

### Scale-Based Replicas

Detects replicas created using Docker Compose's scale features:

```yaml
services:
  web:
    image: nginx:latest
    scale: 3 # Creates 3 replicas

  api:
    image: node:latest
    deploy:
      replicas: 2 # Creates 2 replicas using swarm mode syntax
```

### Name-Based Replicas

Detects replicas with naming patterns like blue-green deployments:

```yaml
services:
  database-blue:
    image: postgres:latest

  database-green:
    image: postgres:latest

  cache-1:
    image: redis:latest

  cache-2:
    image: redis:latest
```

### Example

We provide an interactive example to demonstrate replica detection:

```bash
cd examples
./run_example.sh
```

For more details, see the [replica package documentation](internal/replica/README.md).

## License

MIT License
