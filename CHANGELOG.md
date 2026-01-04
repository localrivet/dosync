# Changelog

## [v0.3.25] - 2026-01-03

### Fixed
- **CRITICAL**: Respect `depends_on` health checks during container updates
- Previously used `--no-deps` flag which skipped dependency health checks
- Containers now wait for dependencies (e.g., postgres) to be healthy before starting
- Fixes DNS lookup failures like "lookup postgres on 127.0.0.11:53: server misbehaving"

### Technical Details
- Removed `--no-deps` from `buildDockerComposeArgs()` in `internal/replica/update.go`
- Rollback operations retain `--no-deps` to avoid cascading restarts
- Docker Compose now properly waits for `depends_on: condition: service_healthy`

## [v0.3.24] - 2026-01-03

### Fixed
- **Unresolved environment variable detection**: DOSync now detects and clearly reports when compose file has unresolved `${VAR}` patterns
- Previously would fail with confusing "invalid repository format" errors
- Now logs helpful error message explaining how to configure .env file mounting

### Technical Details
- Added check in `checkAndUpdateServices()` for unresolved environment variables in image URLs
- Improved `getResolvedComposeConfig()` warning message when `docker compose config` fails
- Error message includes example of proper .env mounting configuration

## [v0.3.23] - 2026-01-02

### Fixed
- **Compose file tag sync**: DOSync now updates compose file even when container is already running correct version
- Previously compose file could have stale tags, causing issues on server restart
- Now detects tag mismatch and syncs compose file without restarting container

### Technical Details
- Added `updateComposeFileImageTag()` function for tag-only updates
- Uses `LastIndex` for colon handling to support registry ports (e.g., `registry:5000/repo:tag`)
- Logs sync actions with "SYNC:" prefix for visibility

## [v0.3.22] - 2026-01-02

### Fixed
- **CRITICAL**: Respect compose file `name:` field for project name
- Previously DOSync would use directory name, causing containers to be created on wrong network
- Now correctly uses `name:` field from compose.yaml (Docker Compose v2.x standard)
- Prevents "dns lookup error" issues when services can't find each other

### Technical Details
- Updated `extractProjectNameFromCompose()` to check `name:` field first
- Priority order: `name:` field → container_name patterns → directory name
- Only passes `--project-name` flag when needed, allowing Docker Compose native resolution

## [v0.3.21] - 2026-01-02

### Added
- **Container Reconciliation**: DOSync now checks if all services defined in compose file are running and restarts any stopped/missing containers
- **Kubernetes-style reconciliation**: Ensures desired state matches actual state every sync interval
- **Automatic recovery**: If a container crashes between deployments, DOSync will restart it on the next sync

### Technical Details
- Added `reconcileStoppedContainers()` function to syncer
- Runs at end of each sync interval after image update checks
- Respects `skip: true` configuration - skipped services won't be reconciled
- Uses `--env-file` for proper environment variable loading during restarts

## [v0.3.20] - 2026-01-02

### Added
- **Proactive container cleanup**: DOSync now removes stale containers before deployment
- Prevents "container name already in use" errors from orphaned containers
- See previous releases for full feature list

## [v0.3.19] - 2026-01-02

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.18] - 2026-01-01

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.17] - 2026-01-01

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.16] - 2025-12-01

### Added
- **NEW FEATURE**: DOSync now automatically starts new services added to compose.yaml
- **USE CASE**: When you add a new service to your compose file and push a new image, DOSync will detect the service isn't running and start it automatically
- **PRODUCTION-READY**: This enables true zero-touch deployments across multiple servers - add a new service, push the image, and all DOSync instances will start it

### Technical Details
- Added `startNewServices()` function that runs before update checks
- Gets list of running containers via `docker ps`
- Compares against services defined in compose.yaml
- Starts any service that has an image but isn't running
- Respects `skip: true` configuration - skipped services won't be auto-started
- Uses correct project name from `container_name` pattern for network consistency

## [v0.3.15] - 2025-11-28

### Fixed
- **CRITICAL**: Fixed `skip: true` configuration being ignored in dosync.yaml
- **ROOT CAUSE**: Config services map was being checked correctly but debug logging was needed to verify
- **RESULT**: Services marked with `skip: true` (e.g., postgres, dosync) are now properly skipped during update checks

### Technical Details
- Added comprehensive debug logging to verify config loading and skip logic
- Logs now show: `DEBUG Config: service 'postgres' skip=true` and `SKIPPING service postgres as configured (skip=true)`
- Verified `cfg.Services` map is properly populated from dosync.yaml
- Skip check at `internal/syncer/syncer.go:129-133` now works correctly
- Requires `.env:/app/.env:ro` mount in DOSync container for environment variable substitution

## [v0.3.14] - 2025-11-27

### Fixed
- **CRITICAL**: Fixed project name extraction when DOSync runs inside Docker container
- **ROOT CAUSE**: DOSync mounts compose file to `/app/compose.yaml`, so `filepath.Base("/app")` returned `app` instead of actual project name
- **RESULT**: New containers created with wrong project name (`app_app-network` instead of `almatuck_app-network`), breaking inter-container networking

### Technical Details
- Added `extractProjectNameFromCompose()` function to extract project name from `container_name` field
- Pattern: `container_name: almatuck_app` → project name `almatuck`
- Falls back to directory-based name if no container_name pattern found
- This ensures DOSync deployments maintain consistent network naming regardless of where compose file is mounted

## [v0.3.13] - 2025-11-27

### Fixed
- **CRITICAL**: Fixed containers ending up on different Docker networks after DOSync deployment
- **ROOT CAUSE**: DOSync was running `docker compose up` without `--project-name`, causing Docker to derive the project name from the working directory (`/app`) instead of the compose file directory
- **RESULT**: Containers like `crowdgains_app` ended up on `app_app-network` while `crowdgains_postgres` stayed on `crowdgains_app-network`, breaking inter-container communication

### Technical Details
- Added `--project-name` flag to all `docker compose` commands
- Project name is derived from compose file directory: `filepath.Base(composeDir)`
- Example: `/opt/crowdgains/compose.yaml` → project name `crowdgains` → network `crowdgains_app-network`
- All containers now consistently use the same network

## [v0.3.12] - 2025-11-25

### Fixed
- **CRITICAL**: Fixed stale container removal not working due to wrong container name
- **FIXED**: Now extracts actual container name from Docker error message (e.g., `crowdgains_app` not `app`)
- **FIXED**: Docker Compose uses `{project}_{service}` naming, extraction now handles this

### Technical Details
- Root cause: v0.3.11 tried to remove `app` but actual container was `crowdgains_app`
- Solution: Parse error message with regex to extract actual container name
- Added `extractContainerNameFromError()` function with comprehensive test coverage
- Error format: `The container name "/crowdgains_app" is already in use`

## [v0.3.11] - 2025-11-25

### Fixed
- **CRITICAL**: Fixed "container name already in use" error blocking deployments
- **FIXED**: Added `--force-recreate` flag to docker compose up command
- **FIXED**: Auto-recovery from stale container conflicts by removing and retrying

### Technical Details
- Root cause: Stale containers with the same name blocked new container creation
- Solution: Use `--force-recreate` flag and retry with container removal on conflict
- Modified `internal/replica/update.go` to handle "already in use" errors gracefully

## [v0.3.10] - 2025-11-24

### Fixed - Complete ONE WAY OF DOING THINGS Consolidation
- **REGISTRY**: Consolidated 8 duplicate registry clients into ONE universal implementation
- **STRATEGY**: Consolidated duplicate UpdateStrategy interfaces (manager/types.go → strategy/types.go)
- **NOTIFIER**: Consolidated duplicate Notifier interfaces (manager/types.go → notification/notification.go)

### Removed
- `internal/registry/client_v2.go` - Merged into client.go
- Legacy registry clients: DockerHubClient, GHCRClient, GCRClient, ACRClient, ECRClient, HarborClient, DOCRClient, CustomRegistryClient
- Stub adapters: StrategyAdapter, NotifierAdapter (replaced with real implementations)
- ~500 lines of duplicate code

### Technical Details
- All container registries now use single `registryClient` backed by google/go-containerregistry
- `manager.UpdateStrategy` is now an alias to `strategy.UpdateStrategy`
- `manager.Notifier` is now an alias to `notification.Notifier`
- Manager now creates real strategy via `CreateStrategy()` and real notifiers via `CreateSlackNotifier()`
- Method naming unified: `SendDeploymentStart` → `SendDeploymentStarted`

### Breaking Changes
- None for users - API remains the same
- Internal refactoring only

## [v0.3.9] - 2025-11-21

### Fixed
- **CRITICAL**: Removed duplicate rolling update systems violating ONE WAY OF DOING THINGS rule
- **FIXED**: Eliminated ~300 lines of duplicate code from internal/replica/update.go
- **FIXED**: Removed duplicate health check, rollback, and rolling update logic
- **SIMPLIFIED**: UpdateDockerComposeAndRestart now only updates compose file and runs docker compose up

### Technical Details
- Root cause: TWO complete rolling update implementations making debugging impossible
- Duplicate system in `internal/replica/update.go` conflicted with advanced strategy system
- Removed functions: performRollingUpdate, renameExistingContainers, restoreOriginalContainers, verifyContainersHealthy, extractProjectNameFromExistingContainers, removeTemporaryContainers
- Removed ErrUpdateFailedButRolledBack error type
- Removed duplicate rollback execution in cmd/sync.go
- Rolling updates now ONLY handled by internal/strategy/* (one-at-a-time, percentage, blue-green, canary)
- Health checks now ONLY handled by internal/health/* (docker, http, tcp, command)
- Rollback now ONLY handled by internal/rollback/* (backup management, restore operations)
- Follows "ONE WAY OF DOING THINGS" rule: only one implementation path, add options to change behavior

## [v0.3.8] - 2025-11-21

### Fixed
- **CRITICAL**: Fixed verbose logging not being enabled during rolling updates
- **FIXED**: UpdateReplica now uses verbose flag to show detailed docker compose errors
- **FIXED**: Added SetVerbose() method to ReplicaManager to propagate verbose flag
- **FIXED**: cmd/sync.go now enables verbose logging on replicaManager when --verbose flag is set

### Technical Details
- Root cause: UpdateReplica in manager.go had hardcoded verbose=false
- Added verbose field to ReplicaManager struct
- Added SetVerbose() method to enable/disable verbose logging
- Modified UpdateReplica to use rm.verbose instead of hardcoded false
- Modified cmd/sync.go to call replicaManager.SetVerbose(verbose) after creation
- Now verbose docker compose errors will be logged when rolling updates fail

## [v0.3.7] - 2025-11-21

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.6] - 2025-11-21

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.5] - 2025-11-21

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.3.4] - 2025-11-18

### Fixed
- **CRITICAL**: Fixed "could not find new running container" error after updates
- **FIXED**: UpdateReplica now finds newest container regardless of state
- **FIXED**: Handles containers in "created", "restarting", or "running" states

### Technical Details
- Root cause: After UpdateDockerComposeAndRestart, container might not be "running" yet
- Container could be in "created" or "restarting" state during startup
- Previous code only looked for c.State == "running", which failed immediately after create
- Solution: Find newest container by creation timestamp regardless of state
- Modified `internal/replica/manager.go:UpdateReplica()` to sort by creation time
- Health check (which happens after) will verify the container becomes healthy
- Eliminates race condition between container creation and state query

## [v0.3.3] - 2025-11-18

### Fixed
- **CRITICAL**: Fixed rollback failing due to dependency conflicts
- **FIXED**: Rollback now uses `--no-deps` flag to avoid recreating dependent services
- **FIXED**: Prevents "container name already in use" errors during rollback

### Technical Details
- Root cause: `docker compose up -d app` tried to ensure postgres was running
- Postgres container already existed with hardcoded name, causing conflict
- Rollback failed, leaving site with no app container running
- Solution: Added `--no-deps` flag to rollback commands
- Modified `internal/rollback/controller.go` in both `Rollback()` and `RollbackToVersion()`
- Rollback now only restarts the failed service without touching dependencies
- Site stays up even when deployments fail

## [v0.3.2] - 2025-11-18

### Fixed
- **CRITICAL**: Fixed health check failure after rolling updates
- **FIXED**: UpdateReplica now updates container ID after creating new container
- **FIXED**: Health checks now target the new container instead of the stopped old container

### Technical Details
- Root cause: After stopping/renaming old container and starting new one, health check still used old container ID
- Old container no longer existed, causing "No such container" errors
- Rolling updates detected new images but failed health checks, triggering unnecessary rollbacks
- Solution: After successful rolling update, fetch and update the Replica's ContainerID to the new container
- Modified `internal/replica/manager.go:UpdateReplica()` to refresh container ID after update
- Health checks now correctly verify the new container's status

## [v0.3.1] - 2025-11-18

### Fixed
- **CRITICAL**: Fixed port binding conflict during rolling updates
- **FIXED**: Blue-green deployment now stops containers before renaming to release port bindings
- **FIXED**: Single-replica services with host port bindings can now update successfully

### Technical Details
- Root cause: `renameExistingContainers()` renamed containers to `-tmp` suffix but didn't stop them
- Docker doesn't release port bindings when renaming a running container
- New containers couldn't start because ports (80, 443, etc.) remained bound to old containers
- Solution: Added `docker stop` command before `docker rename` in `internal/replica/update.go`
- Update sequence: Stop old → Rename old → Start new → Verify health → Remove old

## [v0.3.0] - 2025-11-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.2.4] - 2025-11-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.2.3] - 2025-11-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.2.2] - 2025-11-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.2.1] - 2025-11-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.29] - 2025-11-17

### Fixed
- **CRITICAL**: Fixed GHCR authentication for rolling update mode
- **FIXED**: Missing registry path fix in cmd/sync.go rolling update code path
- Applied same fix to both syncer.go AND cmd/sync.go

### Technical Details
- v0.1.28 only fixed syncer.go (simple sync mode)
- Rolling update mode uses different code path in cmd/sync.go
- Both paths now reconstruct full repository reference (ghcr.io/owner/repo)
- Error showed `[Rolling Update]` prefix, indicating cmd/sync.go path was used

## [v0.1.28] - 2025-11-17

### Fixed
- **CRITICAL**: Fixed GHCR image authentication by preserving full registry path
- **FIXED**: Docker Hub misidentification for GHCR images (ghcr.io/owner/repo)
- Updated syncer to pass full repository reference to go-containerregistry library

### Technical Details
- Root cause: `ParseImageURL()` extracted path only, stripping `ghcr.io/` prefix
- Library needs full path (e.g., `ghcr.io/localrivet/almatuck.ai`) to determine registry
- Without prefix, library defaulted to Docker Hub authentication
- Solution: Reconstruct full repository reference for OCI registries before calling `GetTags()`

## [v0.1.27] - 2025-11-17

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.26] - 2025-11-17

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.25] - 2025-11-17

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.24] - 2025-11-17

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.23] - 2025-11-17

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.22] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.21] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.20] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.19] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.18] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.17] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.16] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.15] - 2025-08-10

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.13] - 2025-08-10

### Critical Bug Fixes
- **FIXED**: Tag selection algorithm now properly handles semantic versioning (v0.1.12 vs v0.1.9)
- **FIXED**: GHCR authentication and tag retrieval for private repositories  
- **FIXED**: YAML parsing issue where `$ts` was treated as variable substitution
- **FIXED**: Repository path parsing - proper tag stripping before API calls
- **ENHANCED**: Comprehensive debug logging for tag selection process

### Technical Details
- Updated default tag selection to prefer semantic versioning over lexicographic comparison
- Fixed GHCR OAuth2 authentication flow for Docker Registry v2 API compliance
- Corrected `extract` field syntax in ImagePolicy configs (use `'ts'` not `'$ts'`)
- Enhanced regex pattern matching and value extraction for timestamp-based tags
- Improved error messages and logging for better debugging

### Breaking Changes
- ImagePolicy configurations using `extract: '$ts'` must be updated to `extract: 'ts'`
- This affects all YAML configurations with filterTags extraction

### Updated Documentation
- Fixed all examples and documentation to use correct extract syntax
- Updated README.md, docs/configuration.md, and examples/dosync.yaml
- Added comprehensive troubleshooting guide for tag selection issues

All notable changes to DOSync will be documented in this file.

## [v0.1.12] - 2025-07-18

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.11] - 2024-12-20

### Critical Fixes
- **FIXED**: Semantic version comparison preventing DOSync from downgrading itself (v0.1.10 → v0.1.9 issue)
- **FIXED**: Lexicographic sorting bug where "v0.1.10" was considered less than "v0.1.9" 
- **ADDED**: Self-update protection for DOSync service to prevent downgrades
- **ADDED**: Service skip configuration for services like postgres, traefik that shouldn't be monitored
- **ADDED**: Comprehensive unit tests for semantic version comparison

### Technical Details
- Root cause: String comparison was treating "10" < "9" alphabetically
- Solution: Proper semantic version parsing using semver library
- Follows "no backward compatibility" and "one way of doing things" rules

## [v0.1.10] - 2024-12-20

### Critical Fixes  
- **FIXED**: Docker Compose v2 commands failing with "unknown shorthand flag: 'f' in -f" error
- **FIXED**: Missing docker-compose package in Alpine container
- **ADDED**: docker-compose package to Dockerfile for proper Docker Compose v2 support

### Technical Details
- Root cause: Container had docker-cli but missing docker-compose plugin
- Modern Docker CLI v28.3.0 requires separate docker-compose installation
- Now includes both docker-cli AND docker-compose v2.36.2

## [v0.1.9] - 2024-12-20

### Critical Fixes
- **FIXED**: Removed duplicate function `updateDockerComposeAndRestart()` from syncer package
- **FIXED**: Code duplication violation - now uses single function from replica package
- **FIXED**: Updated syncer to use `replica.UpdateDockerComposeAndRestart()` properly
- **FIXED**: Increased timeout in command health check tests to prevent flaky failures

### Technical Details
- Root cause: Syncer was calling its own duplicate function with old docker-compose syntax
- Solution: Follow "one way of doing things" rule - single function for all Docker operations
- All Docker command paths now use modern "docker compose" syntax

## [v0.1.8] - 2024-12-20

### Attempted Fixes (Not Effective)
- Updated rollback controller Docker commands
- Fixed integration test Docker commands  
- Updated example scripts
- These fixes addressed some instances but missed the main syncer code path

### Lessons Learned
- Multiple code paths were executing Docker commands
- Need comprehensive search for all command execution patterns
- Importance of following "one way of doing things" principle

## [v0.1.7] - 2024-12-20

### Issues Identified
- Production deployments failing with "unknown shorthand flag: 'f' in -f" error
- Docker command syntax errors during service restarts
- Legacy docker-compose command usage instead of modern docker compose syntax

---

**Upgrade Path**: Always upgrade to the latest version. DOSync now has proper semantic version comparison and self-update protection to prevent downgrades.

**Breaking Changes**: None - all changes maintain backward compatibility while fixing critical issues.

**Production Ready**: v0.1.11+ is stable for production use with comprehensive fixes for Docker command syntax and version comparison issues.
