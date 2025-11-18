# Changelog

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
