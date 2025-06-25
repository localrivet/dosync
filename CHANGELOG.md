# Changelog

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
