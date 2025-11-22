# DOSync Testing Guide

## Overview

This document outlines the testing strategy for DOSync, identifying critical files that require test coverage and providing guidance on how to test the consolidated rolling update system.

## Testing Philosophy

Following the **ONE WAY OF DOING THINGS** rule, DOSync now has a single, unified implementation path for rolling updates, health checks, and rollback operations. Tests must verify:

1. **Single Implementation Path** - No duplicate logic exists
2. **System Integration** - Components work together correctly
3. **Error Handling** - Failures are handled gracefully with proper rollback
4. **State Consistency** - Docker state matches expected outcomes

## Critical Files Requiring Tests

### 1. Core Rolling Update System

#### `internal/strategy/*` (PRIMARY - Most Critical)
**Why Critical**: This is the ONLY system that handles rolling updates after v0.3.9 consolidation.

Files to test:
- `internal/strategy/one_at_a_time.go` - Sequential replica updates
- `internal/strategy/percentage.go` - Percentage-based updates
- `internal/strategy/blue_green.go` - Blue-green deployments
- `internal/strategy/canary.go` - Canary deployments
- `internal/strategy/factory.go` - Strategy creation logic
- `internal/strategy/types.go` - Base strategy implementation

**Test Coverage Requirements**:
- ✅ Strategy executes replicas in correct order
- ✅ Health checks pass between updates
- ✅ Rollback triggers on failure
- ✅ Timeout handling works correctly
- ✅ Pre/post update commands execute properly
- ✅ Multiple replicas update without conflicts

**Existing Tests**: `internal/strategy/integration_test.go`

#### `internal/replica/update.go` (SIMPLIFIED)
**Why Critical**: Entry point for all Docker Compose updates. Must ONLY update compose file and run `docker compose up`.

**Test Coverage Requirements**:
- ✅ Compose file updated correctly with new tag
- ✅ Backup created before modification
- ✅ `docker compose up -d --no-deps SERVICE` executed
- ✅ Registry authentication works (GHCR, Docker Hub, etc.)
- ✅ No rolling update logic exists (removed in v0.3.9)
- ✅ Errors propagate correctly to strategy layer

**Existing Tests**: None - **HIGH PRIORITY to add**

#### `internal/replica/manager.go`
**Why Critical**: Manages replica detection and updates. Critical for multi-replica deployments.

**Test Coverage Requirements**:
- ✅ Detects scale-based replicas (`scale: N`)
- ✅ Detects named replicas (blue-green patterns)
- ✅ Updates replica and refreshes container ID
- ✅ Handles container state transitions (created → running)
- ✅ Verbose logging works when enabled

**Existing Tests**: `internal/replica/manager_test.go`, `internal/replica/integration_test.go`

### 2. Health Check System

#### `internal/health/*` (ONLY Health Check Implementation)

Files to test:
- `internal/health/docker.go` - Docker health checks
- `internal/health/http.go` - HTTP endpoint checks
- `internal/health/tcp.go` - TCP port checks
- `internal/health/command.go` - Custom command checks
- `internal/health/factory.go` - Health checker creation

**Test Coverage Requirements**:
- ✅ Each health check type detects healthy/unhealthy correctly
- ✅ Success/failure thresholds work properly
- ✅ Timeout handling prevents infinite waits
- ✅ Retry logic with exponential backoff
- ✅ Config validation catches invalid settings

**Existing Tests**: `internal/health/*_test.go`

### 3. Rollback System

#### `internal/rollback/*` (ONLY Rollback Implementation)

Files to test:
- `internal/rollback/controller.go` - Rollback orchestration
- `internal/rollback/backup.go` - Compose file backup management
- `internal/rollback/detector.go` - Rollback point detection

**Test Coverage Requirements**:
- ✅ `PrepareRollback()` creates timestamped backups
- ✅ `Rollback()` restores most recent backup
- ✅ `RollbackToVersion()` restores specific version
- ✅ `docker compose up -d --no-deps` used (no dependency conflicts)
- ✅ Backup history maintained correctly
- ✅ Old backups cleaned up (max history limit)

**Existing Tests**: `internal/rollback/controller_test.go`

### 4. Integration Points

#### `cmd/sync.go` (Rolling Update Orchestration)
**Why Critical**: Coordinates all subsystems for rolling updates.

**Test Coverage Requirements**:
- ✅ Registry authentication configured correctly
- ✅ Tag selection by image policy works
- ✅ Rollback controller creates backups before updates
- ✅ Strategy system executes updates
- ✅ No duplicate rollback logic (removed in v0.3.9)
- ✅ Delays between service updates work
- ✅ Service skip configuration honored

**Existing Tests**: `cmd/sync_test.go`

#### `internal/manager/manager.go` (System Coordinator)
**Why Critical**: High-level orchestration of rolling updates.

**Test Coverage Requirements**:
- ✅ All subsystems initialized correctly
- ✅ Error recovery works
- ✅ Metrics collection doesn't block updates
- ✅ Notifications sent on success/failure
- ✅ Dependency resolution prevents conflicts

**Existing Tests**: `internal/manager/integration_test.go`

### 5. Registry and Tag Selection

#### `internal/syncer/syncer.go`
**Why Critical**: Tag selection logic determines which image version to deploy.

**Test Coverage Requirements**:
- ✅ Semantic versioning comparison works correctly
- ✅ Regex filter tags extract values properly
- ✅ Numerical/alphabetical ordering correct
- ✅ State drift prevention (checks actual running containers)
- ✅ Self-update safeguards prevent downgrades

**Existing Tests**: `internal/syncer/syncer_test.go`

#### `internal/registry/*`
**Why Critical**: Multi-registry support for GHCR, Docker Hub, GCR, ACR, ECR, etc.

**Test Coverage Requirements**:
- ✅ Auto-detection of registry type from image URL
- ✅ Authentication for each registry type
- ✅ Tag listing works for all registry types
- ✅ Full repository reference handling (ghcr.io/owner/repo)

**Existing Tests**: `internal/registry/*_test.go`

## Testing Scenarios

### Scenario 1: Single Replica Update
**Purpose**: Verify basic update flow without complexity.

**Steps**:
1. Start service with 1 replica
2. Trigger update to new tag
3. Verify compose file updated
4. Verify `docker compose up` executed
5. Verify health check passes
6. Verify no rollback triggered

**Expected**: Update completes successfully in ~10 seconds

### Scenario 2: Multi-Replica Rolling Update
**Purpose**: Verify one-at-a-time strategy with multiple replicas.

**Steps**:
1. Start service with 3 replicas
2. Trigger update to new tag
3. Verify replicas updated sequentially
4. Verify health check between each update
5. Verify delay between updates honored
6. Verify all replicas on new version

**Expected**: Update takes ~30 seconds (3 replicas × 10s each)

### Scenario 3: Health Check Failure with Rollback
**Purpose**: Verify rollback triggers on unhealthy deployment.

**Steps**:
1. Start service with 2 replicas
2. Deploy broken image (fails health check)
3. Verify health check fails after threshold
4. Verify rollback triggered
5. Verify compose file restored to backup
6. Verify `docker compose up` executed with old version
7. Verify service returns to healthy state

**Expected**: Rollback completes in ~15 seconds

### Scenario 4: Blue-Green Deployment
**Purpose**: Verify zero-downtime blue-green deployment.

**Steps**:
1. Start blue replica (running)
2. Deploy green replica with new version
3. Verify green health checks pass
4. Verify traffic switches to green
5. Verify blue removed after successful switch

**Expected**: Zero downtime during switch

### Scenario 5: State Drift Prevention
**Purpose**: Verify actual running containers checked, not just compose file.

**Steps**:
1. Manually edit compose file to new version
2. Don't restart containers (state drift)
3. Trigger DOSync check
4. Verify DOSync detects drift (running ≠ compose)
5. Verify update triggered

**Expected**: DOSync detects and fixes drift

### Scenario 6: Registry Authentication
**Purpose**: Verify private registry pulls work.

**Steps**:
1. Configure GHCR authentication
2. Deploy private image
3. Verify `docker login` executed
4. Verify image pulled successfully
5. Verify authentication failures logged but don't block

**Expected**: Private images deploy correctly

## Test Execution

### Run All Tests
```bash
go test ./...
```

### Run Specific Package Tests
```bash
# Strategy system (most critical)
go test ./internal/strategy -v

# Replica management
go test ./internal/replica -v

# Health checks
go test ./internal/health -v

# Rollback system
go test ./internal/rollback -v

# Tag selection
go test ./internal/syncer -v
```

### Run Integration Tests Only
```bash
go test ./internal/manager/integration_test.go -v
go test ./internal/replica/integration_test.go -v
go test ./internal/strategy/integration_test.go -v
```

### Run with Coverage
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Coverage Goals

### Priority 1 (Must Have 80%+ Coverage)
- `internal/strategy/*` - Core rolling update logic
- `internal/replica/update.go` - Docker Compose updates
- `internal/replica/manager.go` - Replica management
- `internal/health/*` - Health check system
- `internal/rollback/controller.go` - Rollback orchestration

### Priority 2 (Should Have 60%+ Coverage)
- `cmd/sync.go` - CLI orchestration
- `internal/syncer/syncer.go` - Tag selection
- `internal/registry/*` - Registry clients
- `internal/manager/manager.go` - System coordination

### Priority 3 (Nice to Have 40%+ Coverage)
- `internal/notification/*` - Notifications
- `internal/metrics/*` - Metrics collection
- `internal/dashboard/*` - Web dashboard

## Test Maintenance

### After Code Changes

**Before Merging**:
1. Run full test suite: `go test ./...`
2. Verify no regressions
3. Update tests if behavior changed intentionally
4. Add new tests for new functionality

**After Consolidation (v0.3.9)**:
- ❌ Remove tests for deleted duplicate logic
- ✅ Verify strategy system tests cover all rolling update scenarios
- ✅ Ensure no tests assume duplicate rollback exists

## Manual Testing Checklist

When automated tests aren't sufficient, perform manual validation:

### ✅ Pre-Release Manual Tests
- [ ] Deploy to staging environment
- [ ] Trigger rolling update with real Docker containers
- [ ] Verify zero downtime during update
- [ ] Intentionally deploy broken image
- [ ] Verify rollback restores service
- [ ] Check verbose logging shows detailed errors
- [ ] Verify multiple registries work (GHCR, Docker Hub)
- [ ] Test self-update (DOSync updates itself)

### ✅ Production Smoke Tests
- [ ] Monitor first production deployment closely
- [ ] Check DOSync logs for errors
- [ ] Verify health checks detecting issues
- [ ] Confirm rollback available if needed
- [ ] Validate backup files created

## Common Testing Pitfalls

### ❌ Don't Test Implementation Details
**Bad**: Testing that `performRollingUpdate()` is called
**Good**: Testing that replicas update one-at-a-time

### ❌ Don't Mock Everything
**Bad**: Mocking Docker client, compose files, containers
**Good**: Use real Docker daemon for integration tests

### ❌ Don't Skip Error Cases
**Bad**: Only testing happy path
**Good**: Test timeouts, failures, edge cases

### ✅ Do Test State Transitions
**Good**: Verify container goes: not exist → created → running → healthy

### ✅ Do Test Concurrency
**Good**: Multiple services updating simultaneously don't conflict

### ✅ Do Test Resource Cleanup
**Good**: Temporary containers removed after rollback

## Debugging Failed Tests

### Test Hangs or Times Out
```bash
# Check for orphaned Docker containers
docker ps -a | grep dosync-test

# Clean up test containers
docker compose -f internal/manager/testdata/docker-compose.yml down -v

# Run test with timeout
go test ./internal/strategy -timeout 5m -v
```

### Test Fails Intermittently
- Check for race conditions (use `-race` flag)
- Increase health check timeouts in test configs
- Verify Docker daemon not overloaded
- Look for timing-dependent assertions

### Integration Test Environment Issues
```bash
# Verify Docker daemon running
docker ps

# Check Docker Compose version
docker compose version

# Ensure no port conflicts
lsof -i :80 -i :443 -i :8080
```

## Future Test Improvements

### High Priority
1. Add unit tests for `internal/replica/update.go`
2. Add integration tests for GHCR authentication
3. Add load tests for concurrent updates
4. Add chaos tests (network failures, disk full, etc.)

### Medium Priority
1. Add benchmark tests for large-scale deployments
2. Add security tests (credential handling, injection attacks)
3. Add compatibility tests across Docker versions
4. Add migration tests (upgrade from old versions)

### Low Priority
1. Add performance regression tests
2. Add visual regression tests for dashboard
3. Add accessibility tests for web UI
4. Add localization tests

## Test Data Management

### Test Fixtures
- `internal/manager/testdata/` - Docker Compose files for integration tests
- `internal/replica/testdata/` - Replica detection test cases
- Test fixtures should be minimal and focused

### Cleanup Strategy
- All integration tests must clean up Docker resources
- Use `defer` for cleanup in test functions
- Never leave orphaned containers/networks/volumes
- Use unique names to avoid conflicts (include timestamp or random ID)

## Continuous Integration

### GitHub Actions Workflow
```yaml
# .github/workflows/test.yml
- Run unit tests on every PR
- Run integration tests on merge to main
- Generate coverage report
- Fail PR if coverage drops below thresholds
```

### Pre-commit Hooks
```bash
# Run before committing
go test ./... -short
go vet ./...
gofmt -s -w .
```

## Conclusion

Comprehensive testing ensures DOSync's consolidated architecture (v0.3.9+) remains reliable and maintainable. Focus testing efforts on the **single implementation paths**:
- `internal/strategy/*` for rolling updates
- `internal/health/*` for health checks
- `internal/rollback/*` for rollback operations

Any duplicate logic found during testing should be reported and removed to maintain the **ONE WAY OF DOING THINGS** principle.
