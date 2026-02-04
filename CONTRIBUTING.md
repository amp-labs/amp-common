# Contributing to amp-common

Thank you for your interest in contributing to amp-common! This document provides guidelines and information to help you contribute effectively.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Testing Requirements](#testing-requirements)
- [Code Style](#code-style)
- [Commit Conventions](#commit-conventions)
- [Pull Request Process](#pull-request-process)
- [Package Documentation](#package-documentation)
- [Adding New Packages](#adding-new-packages)

## Prerequisites

Before contributing, ensure you have:

- **Go 1.25.5 or later** installed
- **SSH key configured** for GitHub access to private amp-labs repositories
- **GOPRIVATE environment variable set**:

  ```bash
  export GOPRIVATE=github.com/amp-labs/*
  ```

- **Git configured to use SSH**:

  ```bash
  git config --global url."git@github.com:".insteadOf "https://github.com/"
  ```

Verify your setup:

```bash
ssh -T git@github.com  # Should show successful authentication
go env GOPRIVATE       # Should show: github.com/amp-labs/*
```

## Getting Started

1. **Fork and clone the repository**:

   ```bash
   git clone git@github.com:amp-labs/amp-common.git
   cd amp-common
   ```

2. **Create a feature branch**:

   ```bash
   git checkout -b feat/your-feature-name
   ```

   Branch naming conventions:
   - `feat/` - New features
   - `fix/` - Bug fixes
   - `docs/` - Documentation changes
   - `refactor/` - Code refactoring
   - `test/` - Test additions or updates

3. **Install dependencies**:

   ```bash
   go mod download
   ```

4. **Verify your setup**:

   ```bash
   make test
   make lint
   ```

## Development Workflow

1. Make your changes
2. Write or update tests (see [Testing Requirements](#testing-requirements))
3. Run the linter and formatter:

   ```bash
   make fix        # Run all formatters and linters with auto-fix
   ```

4. Run tests:

   ```bash
   make test       # Run all tests
   make race       # Run tests with race detection
   ```

5. Commit your changes (see [Commit Conventions](#commit-conventions))
6. Push to your fork and create a pull request

## Testing Requirements

### Mandatory t.Parallel()

**All tests MUST call `t.Parallel()`** at the beginning. This is enforced by the `paralleltest` linter.

```go
func TestMyFunction(t *testing.T) {
    t.Parallel()  // REQUIRED at the top of every test function

    t.Run("sub-test name", func(t *testing.T) {
        t.Parallel()  // REQUIRED in every sub-test too

        // Test code here
    })
}
```

**Why?**

- Ensures test isolation and thread-safety
- Catches concurrency bugs early
- Significantly speeds up test execution

**Exceptions:** Sequential tests that genuinely require serial execution (e.g., tests modifying global state or environment variables) should be clearly documented and justified.

### Assertion Library

Use `github.com/stretchr/testify` for all assertions:

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    t.Parallel()

    // require: Stops test on failure (for prerequisites)
    result, err := setupFunction()
    require.NoError(t, err)
    require.NotNil(t, result)

    // assert: Continues test on failure (for validations)
    assert.Equal(t, expected, actual)
    assert.True(t, condition)
}
```

### Coverage Goals

- Aim for high test coverage
- All exported functions should have tests
- Include edge cases and error conditions
- Add example tests for key packages (see [Package Documentation](#package-documentation))

### Running Tests

```bash
go test -v ./...                         # All tests with verbose output
go test -v -run TestName ./package-name  # Single test
make test                                # All tests
make race                                # With race detection
```

## Code Style

### Linting

This repository uses `golangci-lint` with strict configuration. All code must pass:

```bash
make lint      # Check without auto-fix
make fix       # Auto-fix issues
```

Key linters:

- **wsl_v5** - Whitespace linter (allows cuddle declarations)
- **gci** - Go import formatter (groups: standard, default, prefix github.com/amp-labs/amp-common)
- **revive** - Variable naming (accepts both "Id" and "ID")
- **paralleltest** - Enforces `t.Parallel()` in all tests
- **errcheck** - Ensures errors are checked
- **staticcheck** - Static analysis

### Import Order

Imports must be grouped in this order (enforced by `gci`):

```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. External dependencies
    "github.com/stretchr/testify/assert"

    // 3. amp-common packages
    "github.com/amp-labs/amp-common/errors"
    "github.com/amp-labs/amp-common/try"
)
```

### Formatting

```bash
make fix           # Run all formatters (gofmt, gci, golangci-lint --fix)
make fix/sort      # Same as fix but with sorted output
make fix-markdown  # Fix markdown files
```

### Code Guidelines

- Use meaningful variable names (short names acceptable within 15 lines)
- Prefer clear code over comments
- Add comments for non-obvious logic
- Use early returns to reduce nesting
- Keep functions focused and small

## Commit Conventions

Write clear, descriptive commit messages:

### Format

```
<subject line>

<optional body>

<optional footer>
```

### Subject Line

- Use imperative mood ("Add feature" not "Added feature" or "Adds feature")
- Keep under 72 characters
- Be specific and descriptive
- No period at the end

Examples:

- `Add validation support for nested structs`
- `Fix race condition in pool cleanup`
- `Update README with installation instructions`

### Body (optional)

- Explain **why** the change was made, not **what** (the diff shows what)
- Wrap at 72 characters
- Separate from subject with blank line

### Footer (optional)

- Reference issues: `Fixes #123` or `Closes #456`
- Note breaking changes: `BREAKING CHANGE: ...`

## Pull Request Process

1. **Update documentation** if you've changed APIs or added features
2. **Add tests** for new functionality
3. **Run `make fix` and `make test`** before submitting
4. **Fill out the PR template** completely
5. **Keep PRs focused** - one feature or fix per PR
6. **Respond to feedback** promptly and constructively

### PR Checklist

- [ ] Tests added or updated
- [ ] All tests call `t.Parallel()`
- [ ] `make fix` passes
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Documentation updated (if applicable)
- [ ] Example tests added (for new packages or significant features)
- [ ] Commit messages follow conventions

## Package Documentation

### Package Comments

Every package must have a package-level comment:

```go
// Package mypackage provides utilities for X.
//
// Longer description explaining purpose, key concepts,
// and basic usage patterns.
package mypackage
```

### READMEs

For core packages, create a comprehensive README.md with:

- **Purpose** - What problem does this package solve?
- **Installation** - How to use it
- **Quick Start** - Minimal working example
- **Usage** - Common patterns and examples
- **API Reference** - Key types and functions
- **Best Practices** - How to use it effectively
- **Troubleshooting** - Common issues

See `future/README.md` for an excellent example.

### Example Tests

For important packages, add `example_test.go`:

```go
package mypackage_test

import (
    "fmt"
    "github.com/amp-labs/amp-common/mypackage"
)

func ExampleNew() {
    obj := mypackage.New()
    fmt.Println(obj.Value())
    // Output: expected value
}
```

**Requirements:**

- Examples must be in `package_test` (not the main package)
- Must have `// Output:` comment for verification
- Keep examples simple and focused
- One concept per example

## Adding New Packages

When adding a new package to amp-common:

1. **Consider if it belongs here**
   - Is it reusable across multiple projects?
   - Does it fit the amp-common scope?
   - Would it benefit from shared maintenance?

2. **Package structure**

   ```
   mypackage/
   ├── mypackage.go        # Core implementation
   ├── mypackage_test.go   # Unit tests
   ├── example_test.go     # Example tests (optional but recommended)
   └── README.md           # Documentation (for core packages)
   ```

3. **Documentation requirements**
   - Package comment explaining purpose
   - Godoc for all exported types, functions, and constants
   - README.md for complex or frequently-used packages
   - Example tests demonstrating key features

4. **Testing requirements**
   - Comprehensive unit tests with `t.Parallel()`
   - Table-driven tests for multiple scenarios
   - Error cases and edge conditions
   - Example tests for public API

5. **Prometheus metrics** (if applicable)
   - Use `prometheus/promauto` for metric registration
   - Include subsystem label for multi-tenancy
   - Document metrics in package README

6. **Update CLAUDE.md**
   - Add package to appropriate section (Core Packages or Utility Packages)
   - Brief description of purpose and key features
   - Link to README if it exists

## Questions?

If you have questions or run into issues:

1. Check the [FAQ](FAQ.md)
2. Review existing code for examples
3. Open a GitHub issue with the `question` label
4. Reach out to the maintainers

Thank you for contributing to amp-common!
