# Delta CLI Tests

This directory contains integration and end-to-end tests for Delta CLI.

## Structure

```
tests/
├── integration/          # Integration tests that test multiple components
│   └── validation/      # Validation subsystem integration tests
├── e2e/                 # End-to-end tests (future)
└── fixtures/            # Test data and fixtures (future)
```

## Running Tests

### Run all tests
```bash
go test ./tests/...
```

### Run specific integration tests
```bash
go test ./tests/integration/validation/...
```

### Run with verbose output
```bash
go test -v ./tests/...
```

## Writing Tests

### Integration Tests

Integration tests should:
- Test interactions between multiple components
- Use real implementations (not mocks)
- Be placed in appropriate subdirectories under `tests/integration/`
- Use the `_test` package suffix to ensure they test the public API

Example:
```go
package validation_test

import (
    "testing"
    "delta/validation"
)

func TestFeatureIntegration(t *testing.T) {
    // Test multiple components working together
}
```

### Unit Tests

Unit tests should remain in the same directory as the code they test, following Go conventions:
- `foo.go` → `foo_test.go`
- Test specific functions or types in isolation
- Use the same package name (not `_test` suffix) for testing internal functions

## Test Categories

- **Unit Tests**: In package directories (e.g., `/validation/*_test.go`)
- **Integration Tests**: In `/tests/integration/`
- **E2E Tests**: In `/tests/e2e/` (future)
- **Benchmarks**: Can be in either location, named `*_bench_test.go`

## CI/CD Integration

All tests are run as part of the CI/CD pipeline. Make sure new tests:
- Pass locally before pushing
- Don't depend on external services
- Clean up any resources they create
- Are deterministic and repeatable