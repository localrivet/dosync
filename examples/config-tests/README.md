# DOSync Configuration Tests

This directory contains tests and examples for DOSync configuration loading, specifically demonstrating environment variable expansion and alternative field name support.

## Bug Fixes Demonstrated

These tests verify the fixes for the configuration loading panic that occurred when using YAML configuration files. The original issues were:

1. **Field name mismatches** - Users expected `checkInterval` but the code only supported `CHECK_INTERVAL`
2. **Environment variable expansion** - `${VARIABLE}` syntax wasn't working reliably
3. **Configuration loading panics** - Errors caused the application to crash instead of providing helpful error messages

## Test Files

### `dosync-env-test.yaml`
A simple configuration that demonstrates basic environment variable expansion:
- `${CHECK_INTERVAL}` for timing configuration
- `${GITHUB_TOKEN}` for GHCR authentication
- `${DOCKERHUB_USER}` and `${DOCKERHUB_PASS}` for Docker Hub credentials

### `dosync-complex-env-test.yaml`
A more complex configuration showing advanced features:
- Alternative field names (`interval` vs `checkInterval`)
- Mixed environment variables and static values
- Default value fallbacks using `${VAR:-default}` syntax
- Multiple registry configurations
- Dashboard configuration with environment variables

## Running the Tests

```bash
# Navigate to the config-tests directory
cd examples/config-tests

# Run the environment variable expansion test
go run main.go
```

The test will:
1. Set up test environment variables
2. Load each configuration file
3. Verify that environment variables are properly expanded
4. Report success/failure for each test case

## Expected Output

```
=== DOSync Environment Variable Expansion Test ===

Setting test environment variables:
  GITHUB_TOKEN=ghp_test_token_12345
  DOCKERHUB_USER=testuser
  DOCKERHUB_PASS=testpass123
  CHECK_INTERVAL=5m

Testing configuration: dosync-env-test.yaml
----------------------------------------
✅ Config loaded successfully!
CheckInterval: 5m
Verbose: true
GHCR Token: ghp_test_token_12345
✅ GHCR token expansion: SUCCESS
DockerHub Username: testuser
DockerHub Password: testpass123
✅ DockerHub credentials expansion: SUCCESS

Testing configuration: dosync-complex-env-test.yaml
----------------------------------------
✅ Config loaded successfully!
CheckInterval: 5m
Verbose: true
GHCR Token: ghp_test_token_12345
✅ GHCR token expansion: SUCCESS
DockerHub Username: testuser
DockerHub Password: testpass123
✅ DockerHub credentials expansion: SUCCESS

=== Test Summary ===
Environment variable expansion allows you to use ${VAR_NAME} syntax
in your dosync.yaml configuration files. This is especially useful
for sensitive values like tokens and passwords.

Supported field name formats:
  - checkInterval, interval, or CHECK_INTERVAL
  - verbose or VERBOSE
  - imagePolicy or image_policy
```

## Key Features Verified

### Environment Variable Expansion
- ✅ `${VARIABLE}` syntax works in all string fields
- ✅ Nested configuration structures (e.g., `registry.ghcr.token`)
- ✅ Default value fallbacks with `${VAR:-default}` syntax
- ✅ Recursive expansion throughout the entire configuration tree

### Alternative Field Names
- ✅ `checkInterval`, `interval`, or `CHECK_INTERVAL` all work
- ✅ `verbose` or `VERBOSE` both work
- ✅ `imagePolicy` or `image_policy` both work

### Error Handling
- ✅ Graceful handling of missing configuration files
- ✅ Proper error messages instead of panics
- ✅ Automatic detection of `dosync.yaml` in common locations

## Usage in Docker

When using DOSync in Docker containers, you can now use environment variables in your configuration:

```yaml
# docker-compose.yml
services:
  dosync:
    image: localrivet/dosync:latest
    environment:
      - GITHUB_TOKEN=${GITHUB_PAT}
      - CHECK_INTERVAL=2m
    volumes:
      - ./dosync.yaml:/app/dosync.yaml
```

```yaml
# dosync.yaml
checkInterval: "${CHECK_INTERVAL}"
verbose: true
registry:
  ghcr:
    token: "${GITHUB_TOKEN}"
```

This approach keeps sensitive credentials out of your configuration files while maintaining flexibility. 