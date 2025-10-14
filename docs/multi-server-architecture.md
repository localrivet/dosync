# Multi-Server Architecture

This document describes how to deploy DOSync across multiple servers for high-availability, horizontal scaling, and production-grade deployments without Kubernetes complexity.

## Overview

DOSync supports a **distributed, independent architecture** where each server runs its own DOSync instance with no coordination required. This approach provides:

- **High availability**: Server failures are isolated
- **Horizontal scaling**: Add/remove servers without configuration changes
- **Operational simplicity**: Standard Docker Compose, no service discovery needed
- **Cost efficiency**: $10-20/server vs $40+ for Kubernetes nodes
- **Easy debugging**: SSH to any server, check logs, done

## Architecture Diagram

```
                    Load Balancer (Traefik/nginx/Caddy)
                              |
        +---------------------+---------------------+---------------------+
        |                     |                     |                     |
    Server 1              Server 2              Server 3              Server 4
  ┌───────────┐         ┌───────────┐         ┌───────────┐         ┌───────────┐
  │  DOSync   │         │  DOSync   │         │  DOSync   │         │  DOSync   │
  │    +      │         │    +      │         │    +      │         │    +      │
  │   web-1   │         │   web-1   │         │   web-1   │         │   web-1   │
  │   web-2   │         │   web-2   │         │   web-2   │         │   web-2   │
  │   web-3   │         │   web-3   │         │   web-3   │         │   web-3   │
  │    +      │         │    +      │         │    +      │         │    +      │
  │  postgres │         │  postgres │         │  postgres │         │  postgres │
  └───────────┘         └───────────┘         └───────────┘         └───────────┘
   Local Docker          Local Docker          Local Docker          Local Docker
   via /var/run/         via /var/run/         via /var/run/         via /var/run/
   docker.sock           docker.sock           docker.sock           docker.sock

Total capacity: 4 servers × 3 web replicas = 12 containers
```

## How It Works

### 1. Independent DOSync Instances

Each server runs its own DOSync container that:
- Monitors its local Docker daemon via `/var/run/docker.sock`
- Polls container registries for new image tags
- Updates its local `docker-compose.yml` file
- Performs rolling updates on its own replicas
- **No communication with other servers required**

### 2. Replica Detection

DOSync automatically detects replicas on each server using two strategies:

**Scale-based detection:**
```yaml
services:
  web:
    image: myapp:latest
    deploy:
      replicas: 3  # DOSync detects web-1, web-2, web-3
```

**Name-based detection:**
```yaml
services:
  web-blue:
    image: myapp:latest
  web-green:
    image: myapp:latest
  # DOSync detects both as replicas of the same service
```

### 3. Rolling Updates Per Server

When a new image is pushed:
1. All DOSync instances detect it independently (within polling interval)
2. Each starts a rolling update of its own replicas:
   - Server 1: Updates web-1, then web-2, then web-3
   - Server 2: Updates web-1, then web-2, then web-3
   - Server 3: Updates web-1, then web-2, then web-3
   - Server 4: Updates web-1, then web-2, then web-3

### 4. Load Balancer Health Checks

The load balancer ensures zero downtime:
- Monitors health checks for all containers
- Removes unhealthy containers from rotation
- Adds containers back when they become healthy
- No manual intervention required

## Complete Example

### Infrastructure Setup

**Prerequisites:**
- 4 VMs (Hetzner, DigitalOcean, AWS, etc.)
- Docker and Docker Compose installed on each
- Load balancer (Traefik recommended, or nginx/Caddy)
- Container registry (GHCR, DockerHub, etc.)

### Step 1: Create docker-compose.yml

This file is **identical on all servers**:

```yaml
version: '3.8'

services:
  dosync:
    image: localrivet/dosync:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./docker-compose.yml:/app/docker-compose.yml
      - ./backups:/app/backups
      - ./dosync.yaml:/app/dosync.yaml
    environment:
      - GHCR_TOKEN=${GHCR_TOKEN}
      - CONFIG_PATH=/app/dosync.yaml
      - SYNC_FILE=/app/docker-compose.yml
      - SYNC_INTERVAL=2m
      - SYNC_ROLLING_UPDATE=true
      - SYNC_STRATEGY=one-at-a-time
      - SYNC_HEALTH_CHECK=docker

  web:
    image: ghcr.io/yourorg/app:latest
    deploy:
      replicas: 3
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 40s
    environment:
      - DATABASE_URL=postgresql://postgres:password@postgres:5432/myapp
    labels:
      # Traefik labels for automatic service discovery
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host(`example.com`)"
      - "traefik.http.routers.web.entrypoints=websecure"
      - "traefik.http.routers.web.tls.certresolver=letsencrypt"
      - "traefik.http.services.web.loadbalancer.healthcheck.path=/health"
      - "traefik.http.services.web.loadbalancer.healthcheck.interval=10s"
    networks:
      - web

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: myapp
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - web

networks:
  web:
    external: true

volumes:
  postgres_data:
```

### Step 2: Create dosync.yaml

Configuration for tag selection and rollback:

```yaml
registry:
  ghcr:
    token: ${GITHUB_PAT}
    image_policy:
      policy:
        semver:
          range: '>=1.0.0 <2.0.0'  # Only deploy 1.x versions

rollback:
  max_history: 10
  backup_dir: /app/backups
  rollback_on_failure: true
```

### Step 3: Deploy to All Servers

```bash
# On your local machine, create infrastructure repo
mkdir infra && cd infra
# Add docker-compose.yml, dosync.yaml, .env

# Deploy to each server
for server in server1 server2 server3 server4; do
  scp -r . $server:/opt/myapp/
  ssh $server "cd /opt/myapp && docker compose up -d"
done
```

### Step 4: Configure Load Balancer

**Option A: Traefik (Recommended)**

Traefik automatically discovers services via Docker labels:

```yaml
# docker-compose.yml on a separate load balancer server
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./letsencrypt:/letsencrypt
```

**Option B: nginx**

```nginx
upstream backend {
    # Health checks (requires nginx plus or open source with module)
    server server1.example.com:8080 max_fails=3 fail_timeout=30s;
    server server2.example.com:8080 max_fails=3 fail_timeout=30s;
    server server3.example.com:8080 max_fails=3 fail_timeout=30s;
    server server4.example.com:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name example.com;

    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    location /health {
        proxy_pass http://backend;
        access_log off;
    }
}
```

## Deployment Workflow

### Initial Deployment

```bash
# 1. Push initial image
docker build -t ghcr.io/yourorg/app:v1.0.0 .
docker push ghcr.io/yourorg/app:v1.0.0

# 2. All DOSync instances detect and deploy
# Wait 2-3 minutes (polling interval)
# Check deployment status on any server:
ssh server1 "docker logs dosync"
```

### Updating Your Application

```bash
# 1. Build and push new version
docker build -t ghcr.io/yourorg/app:v1.1.0 .
docker push ghcr.io/yourorg/app:v1.1.0

# 2. DOSync automatically:
#    - Detects new version (within 2 minutes)
#    - Updates docker-compose.yml
#    - Performs rolling update (one replica at a time per server)
#    - Health checks ensure zero downtime
#    - Rollback automatically if health checks fail

# 3. Monitor deployment across fleet
for server in server{1..4}; do
  echo "=== $server ==="
  ssh $server "docker ps --filter 'label=com.docker.compose.service=web' --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'"
done
```

### Rollback

If a deployment fails:

```bash
# DOSync automatically rolls back on health check failures
# Or manually rollback on any server:
ssh server1 "docker exec dosync dosync rollback web"
```

## Scaling Your Fleet

### Adding a Server

```bash
# 1. Provision new server (server5)
# 2. Deploy compose stack
scp -r infra/ server5:/opt/myapp/
ssh server5 "cd /opt/myapp && docker compose up -d"

# 3. Add to load balancer
# For Traefik: automatic via Docker labels
# For nginx: add server5 to upstream block
```

**No DOSync configuration changes needed!**

### Removing a Server

```bash
# 1. Remove from load balancer
# For Traefik: stop the containers
# For nginx: remove from upstream block

# 2. Stop services
ssh server4 "cd /opt/myapp && docker compose down"

# 3. Decommission server
```

## Monitoring & Observability

### Per-Server Monitoring

```bash
# Check DOSync logs
docker logs -f dosync

# View deployment history
ls -lah /opt/myapp/backups/

# Check service health
docker ps --filter "label=com.docker.compose.service=web"

# View specific container logs
docker logs web-1
```

### Fleet-Wide Monitoring

**Prometheus + Grafana:**

```yaml
# Add to docker-compose.yml on monitoring server
prometheus:
  image: prom/prometheus
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml
  command:
    - '--config.file=/etc/prometheus/prometheus.yml'

# prometheus.yml
scrape_configs:
  - job_name: 'dosync'
    static_configs:
      - targets: ['server1:9090', 'server2:9090', 'server3:9090', 'server4:9090']
```

**Loki for Log Aggregation:**

```yaml
loki:
  image: grafana/loki
  ports:
    - "3100:3100"

promtail:
  image: grafana/promtail
  volumes:
    - /var/log:/var/log
    - /var/lib/docker/containers:/var/lib/docker/containers
```

## Best Practices

### 1. Health Checks Are Critical

Always define health checks in your compose file:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost/health"]
  interval: 10s
  timeout: 3s
  retries: 3
  start_period: 40s
```

### 2. Use Semantic Versioning

Configure image policies to avoid accidental deployments:

```yaml
# dosync.yaml
registry:
  ghcr:
    image_policy:
      policy:
        semver:
          range: '>=1.0.0 <2.0.0'  # Only 1.x versions
```

### 3. Test Deployments in Staging

Maintain a staging environment with the same multi-server setup:

```
Staging: 2 servers × 2 replicas = 4 containers
Production: 4 servers × 3 replicas = 12 containers
```

### 4. Backup Configuration

Keep your infrastructure as code in git:

```bash
infra/
├── docker-compose.yml
├── dosync.yaml
├── .env.example
└── README.md
```

### 5. Database Considerations

For stateful services like databases:
- Use one primary with read replicas
- Or external managed database (RDS, DigitalOcean DBaaS)
- Don't use `deploy.replicas` for databases (use explicit service names)

```yaml
services:
  postgres-primary:
    image: postgres:16
    volumes:
      - postgres_data:/var/lib/postgresql/data

  postgres-replica-1:
    image: postgres:16
    environment:
      - POSTGRES_PRIMARY_HOST=postgres-primary
```

## Troubleshooting

### Deployment Stuck

```bash
# Check DOSync logs
docker logs dosync

# Check if registry is accessible
docker pull ghcr.io/yourorg/app:latest

# Verify health checks are passing
docker inspect web-1 | grep -A10 Health
```

### Uneven Deployment Across Servers

```bash
# Check if all DOSync instances are running
for server in server{1..4}; do
  echo "=== $server ==="
  ssh $server "docker ps --filter name=dosync"
done

# Check for errors in logs
for server in server{1..4}; do
  echo "=== $server ==="
  ssh $server "docker logs dosync 2>&1 | grep -i error | tail -5"
done
```

### Rollback All Servers

```bash
# Manual rollback across fleet
for server in server{1..4}; do
  echo "Rolling back $server..."
  ssh $server "docker exec dosync dosync rollback web"
done
```

## Cost Analysis

### Single Server

```
1 server × $20/month = $20/month
+ Domain + SSL = $30/month total

Capacity: 3-5 replicas
Traffic: 100-1000 req/sec
Uptime: 99% (single point of failure)
```

### Multi-Server (DOSync)

```
4 servers × $20/month = $80/month
+ Load balancer $10/month
+ Domain + SSL = $100/month total

Capacity: 12-20 replicas
Traffic: 1000-10,000 req/sec
Uptime: 99.9% (fault tolerant)
```

### Kubernetes

```
3 control plane nodes × $40/month = $120/month
+ 4 worker nodes × $40/month = $160/month
+ Load balancer $10/month
+ Domain + SSL = $300/month total

Capacity: Similar to DOSync
Traffic: Similar to DOSync
Complexity: High
Uptime: 99.9% (but harder to debug)
```

**DOSync provides 66% cost savings vs Kubernetes with similar capabilities.**

## Conclusion

DOSync's multi-server architecture provides:
- **Kubernetes-like features** (rolling updates, health checks, rollback)
- **Without Kubernetes complexity** (no control plane, simple YAML)
- **At a fraction of the cost** ($100 vs $300/month for similar capacity)
- **With operational simplicity** (SSH to debug, standard Docker tools)

Perfect for teams running 5-50 servers who value boring technology that just works.
