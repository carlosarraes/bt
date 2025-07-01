# Testing Infrastructure for Bitbucket CLI

This directory contains the comprehensive testing infrastructure for the `bt` CLI tool.

## Overview

The testing strategy focuses on **real API integration** rather than mocks to ensure reliability and accuracy. All tests are designed to work with the actual Bitbucket API v2.

## Quick Start

```bash
# Run unit tests only
make test

# Run CLI tests
make test-cli

# Run performance tests
make test-performance

# Set up integration tests (requires Bitbucket account)
export BT_INTEGRATION_TESTS=1
export BT_TEST_USERNAME="your-username"
export BT_TEST_APP_PASSWORD="your-app-password"
export BT_TEST_WORKSPACE="your-workspace"
export BT_TEST_REPO="your-test-repo"

# Run integration tests
make test-integration

# Run all tests
make test-all
```

## Test Categories

### 1. Unit Tests (`*_test.go` alongside source files)
- Test individual functions and methods
- Located alongside source code
- Fast execution, no external dependencies
- Coverage target: >80%

### 2. Integration Tests (`test/integration/`)
- **Real Bitbucket API integration**
- Test authentication, API calls, and data processing
- Require valid Bitbucket credentials
- Validate actual API response formats

### 3. CLI Tests (`test/cli/`)
- Command execution and output validation
- Flag parsing and error handling
- Golden file comparisons for output
- Performance validation (startup time < 100ms)

### 4. Performance Tests (`test/performance/`)
- Benchmark command startup time
- Memory usage validation
- API response time testing
- Throughput measurements

## Directory Structure

```
test/
├── README.md                 # This file
├── utils/                    # Shared testing utilities
│   └── test_helpers.go      # Test context, HTTP client, assertions
├── integration/             # Real API integration tests
│   ├── api_test.go         # API endpoint testing
│   └── auth_test.go        # Authentication testing
├── cli/                     # CLI command testing
│   └── commands_test.go    # Command execution tests
├── performance/             # Performance benchmarks
│   └── benchmark_test.go   # Performance validation
└── testdata/               # Test fixtures and golden files
    ├── api_responses/      # Example API responses
    ├── configs/            # Test configuration files
    ├── golden_files/       # Expected command outputs
    └── repositories/       # Test repository fixtures
```

## Integration Test Setup

Integration tests require a Bitbucket account and proper configuration:

### 1. Create App Password
1. Go to Bitbucket Settings → App passwords
2. Create password with permissions:
   - Account: Read
   - Repositories: Read, Write, Admin
   - Pull requests: Read, Write
   - Pipelines: Read

### 2. Set Environment Variables
```bash
export BT_INTEGRATION_TESTS=1
export BT_TEST_USERNAME="your-bitbucket-username"
export BT_TEST_APP_PASSWORD="your-app-password"
export BT_TEST_WORKSPACE="your-workspace-slug"
export BT_TEST_REPO="your-test-repository"
```

### 3. Create Test Repository
Create a repository in your workspace for testing:
- Name: `bt-test-repo` (or as specified in `BT_TEST_REPO`)
- Add some test content (commits, branches)
- Optionally create test pull requests
- Enable Pipelines if needed

## Test Utilities

### TestContext
Provides common test setup:
```go
testCtx := utils.NewTestContext(t)
defer testCtx.Close()
// Provides temp directory, config file, cleanup
```

### Command Execution
```go
result := utils.RunBTCommand("version")
utils.AssertCommandSuccess(t, result)
```

### API Testing
```go
client := utils.NewHTTPClient()
resp, err := client.Get("/user")
data, err := utils.ReadJSONResponse(resp)
```

## Performance Targets

| Metric | Target | Test Location |
|--------|--------|---------------|
| Command startup | < 100ms | `test/performance/` |
| API response time | < 500ms | `test/integration/` |
| Memory usage | < 50MB | `test/performance/` |
| Binary size | < 20MB | `test/performance/` |
| Commands/second | > 10 | `test/performance/` |

## Running Tests

### Local Development
```bash
# Quick tests (unit + CLI)
make test-quick

# All tests with coverage
make test-cover

# Watch mode (with entr)
find . -name "*.go" | entr -r make test-quick
```

### Continuous Integration
```bash
# CI-appropriate test suite
make test-ci

# With race detection
make test-race
```

### Performance Testing
```bash
# Run benchmarks
make bench

# Performance regression testing
make test-performance
```

## Test Data Management

### Golden Files (`test/testdata/golden_files/`)
Expected command outputs for comparison:
- Update when output format changes
- Version controlled for consistency
- Used in CLI tests for validation

### API Fixtures (`test/testdata/api_responses/`)
Example API responses for reference:
- Not used for mocking (we test real API)
- Help understand expected data structures
- Documentation for API response formats

### Configuration Files (`test/testdata/configs/`)
Test configuration scenarios:
- Valid and invalid configurations
- Different authentication methods
- Various output formats

## Troubleshooting

### Common Issues

#### Integration Tests Fail
```
Error: 401 Unauthorized
```
**Solution**: Check environment variables and app password permissions.

#### Command Not Found
```
Error: bt: command not found
```
**Solution**: Run `make build` to create the binary.

#### Rate Limiting
```
Error: 429 Too Many Requests
```
**Solution**: Wait for rate limit reset or reduce test frequency.

### Debug Commands
```bash
# Test specific function
go test -run TestVersionCommand ./test/cli/

# Verbose output
go test -v ./test/...

# Race condition detection
go test -race ./test/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Guidelines

### Do's
- ✅ Test real API integration
- ✅ Use descriptive test names
- ✅ Test error conditions
- ✅ Validate performance requirements
- ✅ Clean up test resources
- ✅ Use table-driven tests for variations

### Don'ts
- ❌ Don't use mocks for external APIs
- ❌ Don't hardcode sensitive data
- ❌ Don't rely on external state
- ❌ Don't skip cleanup
- ❌ Don't ignore test failures
- ❌ Don't test implementation details

## Contributing

When adding new features:

1. **Add unit tests** alongside your code
2. **Add integration tests** for API interactions
3. **Add CLI tests** for new commands
4. **Update golden files** if output changes
5. **Add performance tests** for critical paths
6. **Update documentation** as needed

### Test Naming Conventions
```go
// Unit tests
func TestFeatureName(t *testing.T)
func TestFeatureName_ErrorCase(t *testing.T)

// Integration tests
func TestAPIEndpoint(t *testing.T)
func (suite *IntegrationSuite) TestFeature()

// CLI tests
func TestCommandName(t *testing.T)
func TestCommandFlags(t *testing.T)

// Performance tests
func BenchmarkFeature(b *testing.B)
func TestPerformanceTarget(t *testing.T)
```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Framework](https://github.com/stretchr/testify)
- [Bitbucket API v2](https://developer.atlassian.com/bitbucket/api/2/reference/)
- [Testing Guidelines](./../.agent/shared/testing-guidelines.md)
- [Test Setup Guide](./../.agent/shared/test-setup.md)

## Maintenance

### Regular Tasks
- Update golden files when output changes
- Review and update performance targets
- Add tests for new features
- Remove obsolete tests
- Update test data and fixtures

### Monitoring
- Track test execution time
- Monitor test failure rates
- Review coverage reports
- Check performance regressions
- Validate test environment setup