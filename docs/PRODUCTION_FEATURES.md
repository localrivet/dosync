# DOSync Production Features

This document describes the three production-critical features added to DOSync:

1. **Prometheus Metrics Export**
2. **Secrets Management Integration**
3. **Deployment Controls**

---

## 1. Prometheus Metrics Export

### Overview
DOSync now exposes metrics in Prometheus format, allowing you to monitor deployments using standard observability tools like Prometheus and Grafana.

### Endpoint
```
GET /metrics
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dosync_deployments_total` | Counter | Total deployments by service and status (`success`/`failed`) |
| `dosync_deployment_duration_seconds` | Histogram | Deployment duration in seconds |
| `dosync_health_check_failures_total` | Counter | Total health check failures by service |
| `dosync_current_version` | Gauge | Current deployed version (tag) of each service |
| `dosync_rollbacks_total` | Counter | Total rollbacks by service |

### Example Output
```prometheus
# HELP dosync_deployments_total Total number of deployments by service and status
# TYPE dosync_deployments_total counter
dosync_deployments_total{service="web",status="success"} 145
dosync_deployments_total{service="web",status="failed"} 3

# HELP dosync_deployment_duration_seconds Deployment duration in seconds
# TYPE dosync_deployment_duration_seconds histogram
dosync_deployment_duration_seconds_bucket{service="web",le="0.5"} 0
dosync_deployment_duration_seconds_bucket{service="web",le="1.0"} 12
dosync_deployment_duration_seconds_bucket{service="web",le="5.0"} 120
dosync_deployment_duration_seconds_bucket{service="web",le="+Inf"} 145

# HELP dosync_current_version Current deployed version of service
# TYPE dosync_current_version gauge
dosync_current_version{service="web",tag="v1.2.3"} 1
```

### Prometheus Configuration
Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'dosync'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Grafana Dashboard
Create alerts and dashboards using queries like:

```promql
# Deployment success rate
rate(dosync_deployments_total{status="success"}[5m])
  /
rate(dosync_deployments_total[5m])

# Average deployment duration
rate(dosync_deployment_duration_seconds_sum[5m])
  /
rate(dosync_deployment_duration_seconds_count[5m])

# Total rollbacks in last 24h
increase(dosync_rollbacks_total[24h])
```

---

## 2. Secrets Management Integration

### Overview
DOSync can now fetch secrets from enterprise secret management systems instead of storing them in config files or environment variables.

### Supported Providers
- **HashiCorp Vault**
- **AWS Secrets Manager**
- **GCP Secret Manager**

### Configuration

Add a `secrets` section to `dosync.yaml`:

```yaml
secrets:
  enabled: true

  # HashiCorp Vault (optional)
  vault:
    address: "https://vault.example.com:8200"
    token: "${VAULT_TOKEN}"  # Can still use env vars for Vault token

  # AWS Secrets Manager (optional)
  aws:
    region: "us-east-1"

  # GCP Secret Manager (optional)
  gcp:
    project: "my-project-123"
```

### Using Secret References

Instead of storing credentials directly, use secret references:

```yaml
registry:
  ghcr:
    token: "vault:secret/data/github/pat"
    # OR
    token: "aws:prod/github/pat"
    # OR
    token: "gcp:projects/123/secrets/github-pat"

  dockerhub:
    username: "myuser"
    password: "vault:secret/data/dockerhub/password"

  ecr:
    aws_access_key_id: "aws:prod/ecr/access-key"
    aws_secret_access_key: "aws:prod/ecr/secret-key"
```

### Secret Reference Format

```
{provider}:{secret-path}
```

**Examples:**
- `vault:secret/data/github/pat` - HashiCorp Vault KV v2
- `aws:prod/github/pat` - AWS Secrets Manager
- `gcp:projects/my-project/secrets/github-pat` - GCP Secret Manager (full path)
- `gcp:github-pat` - GCP Secret Manager (uses default project)

### How It Works

1. DOSync loads the configuration file
2. For each credential field, it checks if the value is a secret reference
3. If it is, DOSync contacts the appropriate secret provider
4. The secret value is fetched and used in memory (never written to disk)
5. Secrets are cached for the lifetime of the DOSync process

### Prerequisites

**For Vault:**
- Vault server accessible from DOSync
- Valid Vault token with read permissions

**For AWS:**
- AWS CLI installed
- AWS credentials configured (IAM role or credentials file)
- Permissions: `secretsmanager:GetSecretValue`

**For GCP:**
- gcloud CLI installed
- Service account with `secretmanager.versions.access` permission
- Application Default Credentials configured

### Security Best Practices

1. **Use IAM roles** (AWS/GCP) instead of access keys when possible
2. **Rotate Vault tokens** regularly
3. **Use separate secrets** for dev/staging/prod environments
4. **Audit secret access** through your provider's logging
5. **Limit secret permissions** to only what DOSync needs

---

## 3. Deployment Controls

### Overview
Deployment controls allow you to:
- Define deployment time windows
- Require manual approval
- Pause/resume specific services
- Enable dry-run mode

### Configuration

Add a `deployment_controls` section to `dosync.yaml`:

```yaml
deployment_controls:
  # Define when deployments are allowed
  deployment_windows:
    - days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
      start_time: "09:00"
      end_time: "17:00"
      timezone: "America/New_York"

    # Emergency window on weekends
    - days: ["Sat", "Sun"]
      start_time: "10:00"
      end_time: "14:00"
      timezone: "America/New_York"

  # Require manual approval for all deployments
  require_approval: true

  # Services that are currently paused
  paused_services:
    - "legacy-service"
    - "maintenance-app"

  # Dry-run mode: detect updates but don't deploy
  dry_run: false
```

### Deployment Windows

Deployments are only allowed during specified time windows:

```yaml
deployment_windows:
  - days: ["Mon", "Tue", "Wed", "Thu"]
    start_time: "09:00"  # 9 AM
    end_time: "17:00"    # 5 PM
    timezone: "America/New_York"
```

**Supported day formats:**
- Full: `Monday`, `Tuesday`, etc.
- Short: `Mon`, `Tue`, `Wed`, etc.

**Time format:** `HH:MM` (24-hour)

**If no windows are configured,** deployments are allowed at all times.

### Manual Approval

When `require_approval: true`, deployments require manual approval via API:

```bash
# Approve a deployment
curl -X POST -u admin:password \
  http://localhost:8080/api/controls/approve/web

# Check approval status (done automatically by DOSync)
curl -X GET -u admin:password \
  http://localhost:8080/api/controls/status
```

### Pause/Resume Services

```bash
# Pause a service (stop deployments)
curl -X POST -u admin:password \
  http://localhost:8080/api/controls/pause/web

# Resume a service
curl -X POST -u admin:password \
  http://localhost:8080/api/controls/resume/web
```

### Dry-Run Mode

When `dry_run: true`, DOSync will:
- ✅ Check for new image tags
- ✅ Log what would be deployed
- ✅ Run all validation
- ❌ **NOT** actually deploy anything

Perfect for testing configuration or verifying tag selection logic.

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/controls/pause/{service}` | POST | Pause deployments for a service |
| `/api/controls/resume/{service}` | POST | Resume deployments for a service |
| `/api/controls/approve/{service}` | POST | Approve deployment for a service |
| `/api/controls/status` | GET | Get deployment controls status |

### Example Workflow

1. **Configure deployment window** (Mon-Fri, 9AM-5PM)
2. **Enable approval requirement**
3. **DOSync detects** new image tag at 10 AM Tuesday
4. **Deployment is blocked** waiting for approval
5. **Operator reviews** the change and approves:
   ```bash
   curl -X POST -u admin:pass http://localhost:8080/api/controls/approve/web
   ```
6. **DOSync proceeds** with rolling update

### Use Cases

**Deployment Windows:**
- Prevent deployments during business hours
- Restrict to maintenance windows
- Avoid deploying on weekends

**Manual Approval:**
- Production deployments
- Critical services
- Regulatory compliance

**Pause/Resume:**
- Emergency stop
- During incidents
- Maintenance periods

**Dry-Run:**
- Test configuration
- Verify tag selection
- Training/demos

---

## Complete Configuration Example

```yaml
# DOSync configuration with all production features enabled

dashboard:
  enabled: true
  port: "8080"
  user: "admin"
  pass: "${DASHBOARD_PASSWORD}"

secrets:
  enabled: true
  vault:
    address: "https://vault.example.com:8200"
    token: "${VAULT_TOKEN}"
  aws:
    region: "us-east-1"

deployment_controls:
  deployment_windows:
    - days: ["Mon", "Tue", "Wed", "Thu"]
      start_time: "09:00"
      end_time: "17:00"
      timezone: "America/New_York"
  require_approval: true
  dry_run: false

registry:
  ghcr:
    # Use Vault to store GitHub PAT
    token: "vault:secret/data/github/pat"
    image_policy:
      filterTags:
        pattern: "^v(?P<semver>\\d+\\.\\d+\\.\\d+)$"
        extract: "semver"
      policy:
        semver:
          range: ">=1.0.0 <2.0.0"

  dockerhub:
    username: "myuser"
    # Use AWS Secrets Manager for DockerHub password
    password: "aws:prod/dockerhub/password"

services:
  legacy-app:
    skip: true  # Don't manage this service
```

---

## Monitoring Best Practices

### 1. Set up Prometheus Alerts

```yaml
# prometheus-alerts.yml
groups:
  - name: dosync
    rules:
      - alert: DeploymentFailureRate
        expr: |
          rate(dosync_deployments_total{status="failed"}[1h])
          /
          rate(dosync_deployments_total[1h]) > 0.1
        annotations:
          summary: "High deployment failure rate (>10%)"

      - alert: FrequentRollbacks
        expr: increase(dosync_rollbacks_total[24h]) > 5
        annotations:
          summary: "Too many rollbacks in 24h"
```

### 2. Create Grafana Dashboards

Key panels:
- Deployment success rate over time
- Average deployment duration
- Rollback count per service
- Current versions running

### 3. Integration with PagerDuty/Slack

Use Prometheus Alertmanager to send alerts to your team.

---

## Migration Guide

### From Basic Setup to Production

**Step 1: Enable Prometheus Metrics**
```yaml
# No config changes needed!
# Metrics are automatically available at /metrics
```

**Step 2: Move Secrets to Vault/AWS/GCP**
```yaml
# Before:
registry:
  ghcr:
    token: "ghp_xxxxxxxxxxxx"  # Hard-coded secret

# After:
secrets:
  enabled: true
  vault:
    address: "https://vault.example.com"
    token: "${VAULT_TOKEN}"

registry:
  ghcr:
    token: "vault:secret/data/github/pat"  # Secret reference
```

**Step 3: Add Deployment Controls**
```yaml
deployment_controls:
  deployment_windows:
    - days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
      start_time: "09:00"
      end_time: "17:00"
  require_approval: false  # Start with false, enable later
```

---

## Troubleshooting

### Prometheus Metrics Not Showing

**Problem:** `/metrics` endpoint returns empty or errors

**Solutions:**
1. Check dashboard is enabled: `dashboard.enabled: true`
2. Verify port is correct
3. Check if any deployments have occurred (metrics start after first deployment)

### Secret Resolution Fails

**Problem:** "Unknown secret provider" or "Failed to fetch secret"

**Solutions:**
1. Verify provider is configured in `secrets` section
2. Check provider credentials (Vault token, AWS credentials, etc.)
3. Verify secret path is correct
4. Check provider logs for authentication errors

### Deployments Blocked by Controls

**Problem:** "Outside of deployment window" or "Approval required"

**Solutions:**
1. Check current time against deployment windows
2. Approve deployment via API: `/api/controls/approve/{service}`
3. Temporarily disable controls or adjust windows
4. Check if service is paused

### Dry-Run Mode Active

**Problem:** DOSync detects updates but doesn't deploy

**Solutions:**
1. Check `deployment_controls.dry_run` is `false`
2. Restart DOSync after config change

---

## FAQ

**Q: Do Prometheus metrics require authentication?**
A: No, the `/metrics` endpoint is typically unauthenticated to allow Prometheus to scrape it. If you need authentication, modify the dashboard router code.

**Q: Can I use multiple secret providers?**
A: Yes! Configure all providers you need and use the appropriate prefix (`vault:`, `aws:`, `gcp:`).

**Q: What happens if secret fetching fails?**
A: DOSync will use the value as-is (treating it as a literal string), which will likely cause authentication to fail. Check logs for secret resolution errors.

**Q: Can deployment windows span midnight?**
A: Not currently. Create two separate windows if needed.

**Q: How do I test deployment controls without affecting production?**
A: Use `dry_run: true` to test without actually deploying.

---

## See Also

- [Main README](../README.md)
- [Configuration Guide](./CONFIGURATION.md)
- [Multi-Server Architecture](./MULTI_SERVER.md)
