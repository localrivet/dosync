# GHCR First-Class Support Test

This directory contains tests for the improved GitHub Container Registry (GHCR) support in DOSync.

## What Was Fixed

### 1. Authentication Method
- **Before**: Used `Bearer` token authentication (incorrect)
- **After**: Uses proper Basic Auth with username and Personal Access Token

### 2. API Validation
- **Before**: Called non-existent `https://ghcr.io/token` endpoint
- **After**: Uses standard Docker Registry v2 API endpoint `https://ghcr.io/v2/`

### 3. Error Handling
- **Before**: Generic "API request failed with status code: 404" 
- **After**: Detailed error messages with specific debugging information:
  - Repository not found vs authentication issues
  - Includes actual API URLs being called
  - Provides actionable error messages

### 4. Configuration Support
- **Before**: Only supported `token` field
- **After**: Supports both `token` and optional `username` fields

## Running the Test

```bash
cd examples/ghcr-test
go run main.go
```

## Expected Output

The test will:
1. ✅ Load configuration with environment variable expansion
2. ✅ Create GHCR client with proper authentication
3. ✅ Validate authentication against GHCR
4. ⚠️  Test API calls (may fail for private repositories)
5. ❌ Show improved error messages for debugging

## Configuration Example

```yaml
registry:
  ghcr:
    token: "${GITHUB_PAT}"      # Required: GitHub Personal Access Token
    username: "localrivet"      # Optional: GitHub username
    imagePolicy:                # Optional: Tag selection policy
      policy:
        alphabetical:
          order: desc
```

## Key Improvements

1. **Proper Authentication**: GHCR now uses correct Basic Auth method
2. **Better Error Messages**: Detailed debugging information for API failures
3. **Username Support**: Optional username field for explicit user specification
4. **Docker Registry v2 Compliance**: Follows standard Docker Registry API specification
5. **Enhanced Validation**: Proper endpoint validation for authentication testing

## Common Issues Resolved

- ❌ "API request failed with status code: 404" (generic)
- ✅ "GHCR repository not found: tax-equity/solar-equity-hub (URL: https://ghcr.io/v2/tax-equity/solar-equity-hub/tags/list). Verify the repository exists and is accessible"

- ❌ "invalid GitHub token" (unclear)
- ✅ "GHCR authentication failed for tax-equity/solar-equity-hub: invalid token or insufficient permissions"

The improved GHCR support provides first-class integration with proper authentication, detailed error reporting, and full Docker Registry v2 API compliance. 