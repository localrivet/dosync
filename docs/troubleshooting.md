# DOSync Troubleshooting Guide

## Common Issues and Solutions

### "Stub: would perform rolling update" Error

**Symptom:** DOSync logs show `[Rolling Update] Stub: would perform rolling update on /app/docker-compose.yml` repeatedly and no actual updates happen.

**Cause:** DOSync cannot find the Docker Compose file at the expected path. This happens when:
1. The compose file is mounted with a different name (e.g., `compose.yaml` instead of `docker-compose.yml`)
2. The file path doesn't match the command-line argument

**Solution:** Override the command in your compose file to point to the correct path:

```yaml
dosync:
  image: localrivet/dosync:latest
  command: sync -f /app/compose.yaml  # <-- Add this line
  volumes:
    - ./compose.yaml:/app/compose.yaml:ro  # Your actual mount
    - ./dosync.yaml:/app/dosync.yaml:ro
  # ... rest of config
```

**Alternative Solution:** Mount the file at the default path:

```yaml
dosync:
  image: localrivet/dosync:latest
  volumes:
    - ./compose.yaml:/app/docker-compose.yml:ro  # Mount to default path
    - ./dosync.yaml:/app/dosync.yaml:ro
  # ... rest of config
```

---

### "No replica detectors registered" Error

**Symptom:** `failed to get replicas for service app: failed to detect replicas: no replica detectors registered`

**Cause:** This was a bug in versions prior to v0.2.1.

**Solution:** Upgrade to DOSync v0.2.1 or later:

```yaml
dosync:
  image: localrivet/dosync:v0.2.4  # Or :latest
```

---

### "No replicas found for service" Error

**Symptom:** `failed to update replica: no replicas found for service app`

**Cause:** In versions prior to v0.2.3, DOSync couldn't detect containers when using custom `container_name`.

**Solution:** Upgrade to DOSync v0.2.3 or later:

```yaml
dosync:
  image: localrivet/dosync:v0.2.4  # Or :latest
```

---

### DOSync Keeps Restarting Every 10-30 Seconds

**Symptom:** `docker ps` shows DOSync with status `Restarting (0) X seconds ago`

**Cause:** In versions prior to v0.2.4, rolling update mode would exit after one check cycle instead of running continuously.

**Solution:** Upgrade to DOSync v0.2.4 or later:

```yaml
dosync:
  image: localrivet/dosync:v0.2.4  # Or :latest
```

---

### Registry Authentication Failures

**Symptom:** `error from registry: unauthorized` or `failed to get tags`

**Cause:** DOSync cannot authenticate with the container registry.

**Solution 1 - Mount Docker config (Recommended):**

```yaml
dosync:
  image: localrivet/dosync:latest
  volumes:
    - /root/.docker/config.json:/root/.docker/config.json:ro  # Add this
    - /var/run/docker.sock:/var/run/docker.sock
    # ... other volumes
```

Then login to the registry on the host:

```bash
echo "${GITHUB_PAT}" | docker login ghcr.io -u username --password-stdin
```

**Solution 2 - Use dosync.yaml credentials:**

```yaml
# dosync.yaml
registry:
  ghcr:
    token: ${GITHUB_PAT}
    username: yourusername
```

---

### Services Not Being Updated

**Symptom:** DOSync logs show "Service X already at latest tag" but a newer version exists

**Possible Causes:**

1. **Image policy not matching tags**
   - Check that your `imagePolicy.filterTags.pattern` regex matches your tag format
   - Test your regex at regex101.com

2. **Wrong registry type detected**
   - Verify the correct registry is configured in dosync.yaml

3. **Service is configured to skip**
   - Check if the service has `skip: true` in dosync.yaml

**Debug Steps:**

```bash
# Check what tags DOSync sees
docker logs dosync 2>&1 | grep "Available tags"

# Check registry configuration
docker logs dosync 2>&1 | grep "registry"

# Enable verbose logging
# In compose.yaml:
environment:
  - VERBOSE=true
  - SYNC_INTERVAL=2m
```

---

### Rollback Failures with Container Name Conflicts

**Symptom:** `Container name "/service_name" is already in use` during rollback

**Cause:** The rollback mechanism tries to recreate containers but the old ones still exist.

**Solution:** This is typically a transient error that DOSync v0.2.4+ handles better. If it persists:

1. Check for orphaned containers:
   ```bash
   docker ps -a | grep service_name
   ```

2. Remove stuck containers:
   ```bash
   docker rm -f service_name
   ```

3. Let DOSync retry the deployment

---

### Database Services Being Updated Unexpectedly

**Symptom:** DOSync tries to update postgres, mysql, or other database images

**Cause:** By default, DOSync monitors all services in the compose file.

**Solution:** Add services to skip in dosync.yaml:

```yaml
# dosync.yaml
services:
  postgres:
    skip: true  # Never auto-update database

  redis:
    skip: true  # Never auto-update cache

  app:
    skip: false  # Allow auto-updates (default)
```

**Note:** Stateful services (databases, caches) should generally be updated manually to prevent data loss.

---

## Debugging Tips

### Enable Verbose Logging

```yaml
dosync:
  environment:
    - VERBOSE=true
```

### Check DOSync Logs

```bash
# Real-time logs
docker logs -f dosync

# Last 50 lines
docker logs dosync --tail 50

# Search for errors
docker logs dosync 2>&1 | grep -i error
```

### Verify File Mounts

```bash
docker exec dosync ls -la /app/
```

### Check Registry Connectivity

```bash
# Test GHCR
docker exec dosync wget -O- https://ghcr.io/v2/

# Test registry authentication
echo "${GITHUB_PAT}" | docker login ghcr.io -u username --password-stdin
```

### Verify Health Checks

```bash
docker inspect service_name --format='{{.State.Health.Status}}'
```

---

## Version Compatibility

| DOSync Version | Key Features | Notes |
|----------------|--------------|-------|
| v0.1.x | Initial release | Not recommended - multiple bugs |
| v0.2.0 | Multi-registry support | Replica detection issues |
| v0.2.1 | Fixed single-container detection | Recommended minimum |
| v0.2.2 | Added service skip support | - |
| v0.2.3 | Fixed custom container names | - |
| v0.2.4+ | Fixed continuous operation | **Recommended** |

Always use the latest version: `localrivet/dosync:latest`

---

## Getting Help

If you're still experiencing issues:

1. Check the [GitHub Issues](https://github.com/localrivet/dosync/issues)
2. Enable verbose logging and collect logs
3. Create a new issue with:
   - DOSync version
   - Your compose file (sanitized)
   - Your dosync.yaml (sanitized)
   - Error logs
   - Steps to reproduce

