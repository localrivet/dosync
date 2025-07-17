# DOSync Error Report: Docker Command Syntax Issue

**Date**: July 17, 2025  
**DOSync Version**: `localrivet/dosync:v0.1.7`  
**Environment**: Production Docker Compose deployment  
**Severity**: Medium - Service detection works but restart fails  

## Executive Summary

DOSync successfully detected a new container image version but failed to restart the service due to an invalid Docker command syntax error. The issue appears to be in DOSync's service restart logic where it generates a malformed Docker command containing an unsupported `-f` flag.

## Error Details

### Primary Error Message
```
Error updating service app: failed to restart service: exit status 125, output: unknown shorthand flag: 'f' in -f

Usage:  docker [OPTIONS] COMMAND [ARG...]

Run 'docker --help' for more information
```

### Complete Error Context
```
Processing service: app with image: ghcr.io/tax-equity/solar-equity-hub:main-13d2000-1752778457
Updating service app to new tag: main-3404e46-1752779523 (current: main-13d2000-1752778457)
Restarting service: app
Error updating service app: failed to restart service: exit status 125, output: unknown shorthand flag: 'f' in -f
```

## What Works vs What Fails

### ✅ Working Components
- **Image Detection**: DOSync correctly detects new image versions from GHCR
- **Version Comparison**: Properly identifies when updates are needed
- **Image Registry Authentication**: Successfully authenticates with GitHub Container Registry
- **Update Logic**: Correctly determines `main-13d2000-1752778457` → `main-3404e46-1752779523`

### ❌ Failing Component
- **Service Restart**: Docker command generation produces invalid syntax with unsupported `-f` flag

## Technical Analysis

### Root Cause
DOSync's service restart mechanism appears to be constructing a Docker command that incorrectly includes a `-f` flag in a context where Docker doesn't support it. Based on the error output, DOSync is likely trying to execute something similar to:

```bash
# What DOSync might be attempting (invalid):
docker [some-command] -f [arguments]

# What should work instead:
docker compose restart app
# or
docker compose up -d app
```

### Docker Exit Code 125
Exit code 125 specifically indicates "Docker daemon error" - typically caused by invalid command syntax or unsupported flags, which aligns with the "unknown shorthand flag" error message.

## Impact Assessment

### Immediate Impact
1. **Service Update Failure**: Containers continue running old versions despite DOSync detecting new images
2. **State Mismatch**: DOSync's internal state shows service as "updated" while actual containers remain outdated
3. **No Further Update Attempts**: DOSync won't retry updates because it believes the service is already current

### Operational Impact
- Manual intervention required for deployments
- Potential for production systems to run outdated code
- Monitoring confusion due to state mismatch

## Environment Configuration

### Docker Compose Setup
```yaml
# compose.yaml structure
services:
  app:
    image: ghcr.io/tax-equity/solar-equity-hub:latest
    # ... other configuration
  
  dosync:
    image: localrivet/dosync:v0.1.7
    command: "./dosync sync -f /app/compose.yaml"
    # ... other configuration
```

### DOSync Configuration
- **Config File**: `/app/compose.yaml`
- **Sync Command**: `./dosync sync -f /app/compose.yaml`
- **Registry**: GitHub Container Registry (ghcr.io)
- **Authentication**: GitHub token-based

## Workaround Applied

### Manual Resolution
We successfully worked around the issue by manually executing the intended Docker operations:

```bash
# Commands that work correctly:
docker compose pull app
docker compose up -d app

# Result: Service successfully updated to latest image
```

### Verification
```bash
$ docker compose ps
NAME                     IMAGE                                        STATUS
solar-equity-hub-app     ghcr.io/tax-equity/solar-equity-hub:latest   Up (healthy)
```

## Additional Observations

### Related Warnings (Non-blocking)
```
time="2025-07-17T19:18:38Z" level=warning msg="/opt/solar-equity-hub/compose.yaml: the attribute `version` is obsolete, it will be ignored, please remove it to avoid potential confusion"
```

### Other Registry Errors (Informational)
```
Error getting tags for docker.io repo postgres: API request failed with status code: 404
Error getting tags for docker.io repo traefik: API request failed with status code: 404
```
*Note: These appear to be informational and don't affect DOSync's primary functionality*

## Requested Actions

### For DOSync Development Team
1. **Investigate Docker Command Construction**: Review the service restart logic to identify where the invalid `-f` flag is being introduced
2. **Command Validation**: Add validation for generated Docker commands before execution
3. **Error Recovery**: Implement retry logic or fallback commands when restart fails
4. **State Consistency**: Ensure internal state only updates after successful service restart

### Suggested Fix Areas
- Docker command string building in restart functionality
- Flag validation before command execution
- Error handling and recovery mechanisms
- State management consistency

## Test Case for Reproduction

### Setup
1. Deploy DOSync v0.1.7 with Docker Compose configuration
2. Configure monitoring of GHCR repository
3. Push new image version to trigger update

### Expected vs Actual
- **Expected**: Service restarts with new image
- **Actual**: Error with "unknown shorthand flag: 'f' in -f"

## Contact Information

**Reporter**: Solar Equity Hub Development Team  
**Environment**: Production Docker deployment  
**Available for Follow-up**: Yes, can provide additional logs or testing

---

*This report documents a reproducible issue affecting automated deployments. We're available to assist with testing fixes or providing additional diagnostic information.* 