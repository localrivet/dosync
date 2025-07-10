# DOSync Deployment Troubleshooting Guide

This guide helps resolve common DOSync deployment issues encountered in production environments.

## Overview

DOSync is a zero-downtime deployment tool that monitors container registries and automatically updates your Docker services when new images are available. This guide addresses common configuration and deployment issues.

## Current Status Analysis

Based on your server logs, DOSync v0.1.7 is running successfully with the following status:

✅ **Working Components:**
- DOSync container is running and monitoring services
- Configuration is loaded correctly (`VERBOSE=true`, `CHECK_INTERVAL=2m`)
- Application containers are healthy and responding
- Zero-downtime updates are being detected and attempted

❌ **Issues Identified:**
1. **Docker Compose Command Error** - DOSync is using incorrect command syntax
2. **Registry API Rate Limiting** - Some Docker Hub API calls are failing
3. **Missing Environment Variables** - Several optional environment variables are not set

## Issue #1: Docker Compose Command Error

### Problem
```
Error updating service app: failed to restart service: exit status 125, output: unknown shorthand flag: 'f' in -f
```

### Root Cause
DOSync is using an older Docker Compose command syntax. The error occurs when DOSync tries to restart services using `docker -f` instead of `docker compose -f`.

### Solution

**Step 1: Update DOSync Configuration**

Edit your `dosync.yaml` file:

```bash
cd /opt/solar-equity-hub
nano dosync.yaml
```

Add or update the following configuration:

```yaml
# DOSync Configuration
CHECK_INTERVAL: "2m"
VERBOSE: true

# Registry configuration
registry:
  ghcr:
    token: "${GITHUB_TOKEN}"
    username: "your-github-username"

# Rollback configuration
rollback:
  composeFilePath: "/app/docker-compose.yml"  # Path inside DOSync container
  backupDir: "/app/backups"
  maxHistory: 5
  defaultRollbackOnFailure: false

# Dashboard (optional)
dashboard:
  enabled: false
  port: "8080"
```

**Step 2: Update Docker Compose File**

Ensure your `compose.prod.yaml` has the correct DOSync configuration:

```yaml
  # DOSync for zero-downtime deployments
  dosync:
    image: localrivet/dosync:v0.1.7
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./compose.prod.yaml:/app/docker-compose.yml
      - ./dosync.yaml:/app/dosync.yaml
      - ./backups:/app/backups
    environment:
      - CHECK_INTERVAL=2m
      - VERBOSE=true
      - GITHUB_TOKEN=${GITHUB_PAT}
    env_file:
      - .env
    networks:
      - app-network
```

**Step 3: Restart DOSync**

```bash
cd /opt/solar-equity-hub
docker compose -f compose.prod.yaml down dosync
docker compose -f compose.prod.yaml up -d dosync
```

## Issue #2: Registry API Rate Limiting

### Problem
```
Error getting tags for docker.io repo traefik: API request failed with status code: 404
Error getting tags for docker.io repo postgres: API request failed with status code: 404
```

### Root Cause
Docker Hub API rate limiting or authentication issues when checking for updates.

### Solution

**Step 1: Add Docker Hub Authentication (Recommended)**

Add Docker Hub credentials to your `.env` file:

```bash
cd /opt/solar-equity-hub
nano .env
```

Add these lines:

```env
# Docker Hub credentials (to avoid rate limiting)
DOCKERHUB_USERNAME=your-dockerhub-username
DOCKERHUB_PASSWORD=your-dockerhub-token
```

**Step 2: Update DOSync Configuration**

Update `dosync.yaml` to include Docker Hub registry:

```yaml
registry:
  dockerhub:
    username: "${DOCKERHUB_USERNAME}"
    password: "${DOCKERHUB_PASSWORD}"
  ghcr:
    token: "${GITHUB_TOKEN}"
    username: "your-github-username"
```

**Step 3: Alternative - Exclude Problematic Services**

If you don't want DOSync to monitor certain services, update your `compose.prod.yaml` to add labels:

```yaml
  traefik:
    image: traefik:v3.0
    labels:
      - "dosync.enable=false"  # Exclude from DOSync monitoring
    # ... rest of configuration

  postgres:
    image: postgres:15-alpine
    labels:
      - "dosync.enable=false"  # Exclude from DOSync monitoring
    # ... rest of configuration
```

## Issue #3: Missing Environment Variables

### Problem
```
WARN[0000] The "GOOGLE_SERVICE_ACCOUNT_KEY" variable is not set. Defaulting to a blank string.
WARN[0000] The "SIGNWELL_API_KEY" variable is not set. Defaulting to a blank string.
WARN[0000] The "SENDGRID_API_KEY" variable is not set. Defaulting to a blank string.
WARN[0000] The "POSTGRES_PASSWORD" variable is not set. Defaulting to a blank string.
```

### Solution

**Step 1: Check Your .env File**

```bash
cd /opt/solar-equity-hub
cat .env
```

**Step 2: Add Missing Required Variables**

Edit your `.env` file:

```bash
nano .env
```

Add the missing variables:

```env
# Database
POSTGRES_PASSWORD=your-secure-postgres-password

# Email (if using SendGrid)
SENDGRID_API_KEY=your-sendgrid-api-key

# Document signing (if using SignWell)
SIGNWELL_API_KEY=your-signwell-api-key

# Google integration (if using Google Docs)
GOOGLE_SERVICE_ACCOUNT_KEY=your-google-service-account-json

# GitHub for DOSync
GITHUB_PAT=your-github-personal-access-token

# Docker Hub for DOSync (optional but recommended)
DOCKERHUB_USERNAME=your-dockerhub-username
DOCKERHUB_PASSWORD=your-dockerhub-token
```

## Complete Deployment Steps

### Step 1: Stop Current Services

```bash
cd /opt/solar-equity-hub
docker compose -f compose.prod.yaml down
```

### Step 2: Update Configuration Files

1. **Update .env file** with missing variables (see above)
2. **Update dosync.yaml** with proper registry configuration
3. **Verify compose.prod.yaml** has correct DOSync setup

### Step 3: Restart Services

```bash
# Start all services
docker compose -f compose.prod.yaml up -d

# Check logs
docker compose -f compose.prod.yaml logs -f dosync
```

### Step 4: Verify DOSync is Working

```bash
# Check DOSync status
docker compose -f compose.prod.yaml logs --tail=20 dosync

# Check application health
docker compose -f compose.prod.yaml logs --tail=10 app
```

## Monitoring and Verification

### Check DOSync Logs

```bash
# Real-time logs
docker compose -f compose.prod.yaml logs -f dosync

# Last 50 lines
docker compose -f compose.prod.yaml logs --tail=50 dosync
```

### Verify Service Updates

DOSync should show logs like:
```
Service app is already running the latest tag: main-2fd2333-1752164905
```

### Test Manual Update

To test DOSync functionality:

```bash
# Trigger a manual check (restart DOSync)
docker compose -f compose.prod.yaml restart dosync
```

## Expected Behavior

After fixing the issues, you should see:

✅ **Successful DOSync logs:**
```
Loaded config: {CheckInterval:2m Verbose:true ...}
Starting synchronization process for all supported registries...
Processing service: app with image: ghcr.io/tax-equity/solar-equity-hub:main-xxx
Service app is already running the latest tag: main-xxx
```

✅ **No Docker command errors**
✅ **Registry API calls succeed**
✅ **Automatic updates when new images are pushed**

## Troubleshooting Commands

```bash
# Check container status
docker compose -f compose.prod.yaml ps

# Check DOSync configuration
docker compose -f compose.prod.yaml exec dosync cat /app/dosync.yaml

# Check environment variables
docker compose -f compose.prod.yaml exec dosync env | grep -E "(GITHUB|DOCKER|VERBOSE)"

# Restart specific service
docker compose -f compose.prod.yaml restart dosync

# View real-time logs
docker compose -f compose.prod.yaml logs -f dosync app
```

## Support

If issues persist:

1. **Check DOSync version**: Ensure you're using `localrivet/dosync:v0.1.7` or later
2. **Verify network connectivity**: Ensure the server can reach GitHub and Docker Hub
3. **Check file permissions**: Ensure DOSync can read configuration files
4. **Review GitHub token permissions**: Token needs `read:packages` for GHCR access

## Security Notes

- Store sensitive tokens in `.env` file, not in configuration files
- Use GitHub Personal Access Tokens with minimal required permissions
- Consider using Docker Hub access tokens instead of passwords
- Regularly rotate access tokens and passwords

---

**Need Help?** 
- Check the [DOSync documentation](https://github.com/localrivet/dosync)
- Review logs carefully for specific error messages
- Ensure all environment variables are properly set 