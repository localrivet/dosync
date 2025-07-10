# DOSync Quick Fix Reference

## 🚨 Critical Issue: Docker Command Error

**Problem:** `unknown shorthand flag: 'f' in -f`

**Quick Fix:**
```bash
cd /opt/solar-equity-hub

# 1. Update DOSync to latest version
docker compose -f compose.prod.yaml pull dosync

# 2. Restart DOSync
docker compose -f compose.prod.yaml restart dosync
```

## 🔧 Registry API Errors (404s)

**Problem:** `API request failed with status code: 404`

**Quick Fix - Option 1: Add Docker Hub Auth**
```bash
# Add to .env file:
echo "DOCKERHUB_USERNAME=your-username" >> .env
echo "DOCKERHUB_PASSWORD=your-token" >> .env
```

**Quick Fix - Option 2: Exclude Services**
```yaml
# Add to compose.prod.yaml for services you don't want monitored:
labels:
  - "dosync.enable=false"
```

## ⚠️ Environment Variable Warnings

**Problem:** Missing POSTGRES_PASSWORD and other variables

**Quick Fix:**
```bash
# Check current .env
cat .env

# Add missing variables
nano .env
```

Required variables:
```env
POSTGRES_PASSWORD=your-secure-password
GITHUB_PAT=your-github-token
```

## 🔄 Complete Restart Sequence

```bash
cd /opt/solar-equity-hub

# Stop all services
docker compose -f compose.prod.yaml down

# Pull latest images
docker compose -f compose.prod.yaml pull

# Start services
docker compose -f compose.prod.yaml up -d

# Check DOSync logs
docker compose -f compose.prod.yaml logs -f dosync
```

## ✅ Verification Commands

```bash
# Check if DOSync is working
docker compose -f compose.prod.yaml logs --tail=10 dosync

# Should see: "Service app is already running the latest tag"
# Should NOT see: "unknown shorthand flag" or "exit status 125"
```

---
**Full troubleshooting guide:** See `troubleshooting-dosync-deployment.md` 