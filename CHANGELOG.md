# Changelog

## [v0.1.6] - 2025-01-07

### Fixed
- **GHCR Authentication**: Fixed GitHub Container Registry authentication to use proper Docker Registry v2 OAuth2 Bearer token flow instead of Basic Auth
- **GHCR Private Repositories**: Full support for private GHCR repositories with proper token scoping
- **GHCR Error Messages**: Enhanced error messages with specific debugging information and actual API URLs

### Added
- **GHCR First-Class Support**: Comprehensive GHCR configuration documentation with examples
- **GHCR Token Validation**: Automatic Bearer token acquisition per repository scope
- **GHCR Troubleshooting Guide**: Detailed troubleshooting section for common GHCR issues

### Changed
- **GHCR API Endpoints**: Updated to use correct Docker Registry v2 API endpoints (`https://ghcr.io/v2/`)
- **GHCR Repository Path Handling**: Fixed repository path to properly strip tags before API calls
- **GHCR Error Reporting**: Error messages now include specific URLs and actionable debugging information

## [v0.1.5] - 2025-06-25

### Fixed
- Configuration loading panic when using user-friendly YAML field names
- Support for `checkInterval`, `interval`, and `CHECK_INTERVAL` field names
- Support for `verbose` and `VERBOSE` field names  
- Support for `imagePolicy` and `image_policy` field names
- Improved error handling in configuration loading (no more panics)
- Docker container CMD to work properly with new configuration system

### Added
- Comprehensive test suite for environment variable expansion in `examples/config-tests/`
- Example configuration files demonstrating proper YAML structure
- Documentation for configuration loading and environment variable usage

### Changed
- Organized examples into separate subdirectories to avoid main function conflicts
- Moved `replica_detection.go` to `examples/replica-detection/main.go`
- Environment variable expansion was already working correctly

## [v0.1.4] - 2025-05-07

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.3] - 2025-05-06

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.2] - 2025-05-06

### Added
- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.1] - 2025-05-06

### Added

- Latest release of DOSync
- See previous releases for full feature list

## [v0.1.0] - 2025-05-06

### Added

- Latest release of DOSync
- See previous releases for full feature list
