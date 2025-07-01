# test/

Test infrastructure for the Bitbucket CLI.

## Structure

- **integration/** - Integration tests with real Bitbucket API
- **cli/** - CLI command testing with golden files
- **testdata/** - Test fixtures and example API responses

## Testing Strategy

### Real API Integration
All tests are designed to work with the real Bitbucket API v2, not mocks:
- Use dedicated test repositories for integration tests
- Test with actual authentication methods
- Validate real API response formats
- Test rate limiting and error handling

### Test Types

1. **Unit Tests** - Located alongside source files (`*_test.go`)
2. **Integration Tests** - Full API integration tests (`integration/*_test.go`)
3. **CLI Tests** - Command execution tests (`cli/*_test.go`)
4. **Performance Tests** - Benchmarks and performance validation

### Test Data

The `testdata/` directory contains:
- Example API responses (for documentation, not mocking)
- Test configuration files
- Golden files for CLI output testing
- Test repository fixtures

## Running Tests

```bash
# Unit tests
make test

# Integration tests (requires auth)
make test-integration

# All tests with coverage
make test-cover

# Race condition detection
make test-race
```

## Test Requirements

- Tests must pass with real Bitbucket API
- Integration tests require valid authentication
- No mock data used in tests
- Coverage target: >80%