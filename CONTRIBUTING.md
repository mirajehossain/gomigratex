# Contributing to gomigratex

Thank you for your interest in contributing to gomigratex! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Project Structure](#project-structure)

## Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/) Code of Conduct. By participating, you agree to uphold this code.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git
- MySQL (for integration tests)
- Make (optional, for convenience commands)

### Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/gomigratex.git
   cd gomigratex
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/mirajehossain/gomigratex.git
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Verify setup**
   ```bash
   make test
   ```

## Making Changes

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number
```

### 2. Make Your Changes

- Write clear, self-documenting code
- Follow existing patterns and conventions
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/migrator

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

### 4. Build and Test CLI

```bash
# Build the CLI
go build -o migratex ./cmd/migrate

# Test with sample migrations
./migratex create test_migration --dir ./test-migrations
./migratex status --dsn "test:test@tcp(localhost:3306)/testdb" --dir ./test-migrations --dry-run
```

## Testing

### Unit Tests

All packages should have comprehensive unit tests:

```bash
# Test specific package
go test ./internal/migrator

# Test with coverage
go test -cover ./internal/migrator

# Test with race detection
go test -race ./internal/migrator
```

### Integration Tests

For database-related functionality:

```bash
# Start MySQL (using docker-compose)
docker-compose up -d

# Run integration tests
DB_DSN="admin:testpass1@tcp(127.0.0.1:3306)/test?parseTime=true&multiStatements=true" \
go test -tags=integration ./...
```

### Test Guidelines

- **Write tests first** (TDD approach when possible)
- **Test edge cases** and error conditions
- **Use descriptive test names** that explain what's being tested
- **Keep tests simple** and focused on one thing
- **Use table-driven tests** for multiple scenarios
- **Mock external dependencies** (databases, file systems)

### Example Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    invalidInput,
            expected: nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("FunctionName() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Submitting Changes

### 1. Commit Your Changes

```bash
# Stage your changes
git add .

# Commit with descriptive message
git commit -m "feat: add support for custom migration table names

- Add --table flag to CLI
- Update migrator to use configurable table name
- Add tests for new functionality
- Update documentation"
```

### 2. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 3. Create a Pull Request

1. Go to the [GitHub repository](https://github.com/mirajehossain/gomigratex)
2. Click "New Pull Request"
3. Select your branch
4. Fill out the PR template
5. Submit the PR

### 4. PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

## Code Style

### General Guidelines

- Follow standard Go conventions
- Use `gofmt` to format code
- Use `golint` to check for issues
- Write clear, self-documenting code
- Add comments for public APIs

### Naming Conventions

- **Packages**: lowercase, single word
- **Functions**: PascalCase for public, camelCase for private
- **Variables**: camelCase
- **Constants**: PascalCase or UPPER_CASE
- **Interfaces**: -er suffix (e.g., `Reader`, `Writer`)

### Code Organization

```go
// Package comment
package packagename

// Imports (standard, third-party, local)
import (
    "context"
    "fmt"

    "github.com/some/package"

    "github.com/mirajehossain/gomigratex/internal/other"
)

// Constants
const (
    DefaultTimeout = 30 * time.Second
)

// Types
type Config struct {
    // fields
}

// Functions (public first, then private)
func NewConfig() *Config {
    // implementation
}

func (c *Config) Validate() error {
    // implementation
}
```

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Better with context
if err != nil {
    return fmt.Errorf("failed to open database %s: %w", dsn, err)
}
```

## Project Structure

```
gomigratex/
├── cmd/
│   └── migrate/          # CLI application
├── internal/             # Private packages
│   ├── checksum/         # SHA256 checksum utilities
│   ├── config/           # Configuration management
│   ├── db/               # Database connection & schema
│   ├── fsutil/           # File system utilities
│   ├── lock/             # Advisory locking
│   ├── logger/           # Logging utilities
│   └── migrator/         # Core migration logic
├── examples/             # Usage examples
├── migrations/           # Sample migration files
├── .github/              # GitHub workflows
├── Makefile              # Build and test commands
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── README.md             # Project documentation
├── CONTRIBUTING.md       # This file
└── LICENSE               # License file
```

### Package Responsibilities

- **cmd/migrate**: CLI interface, argument parsing, command execution
- **internal/migrator**: Core migration logic, planning, execution
- **internal/db**: Database connections, schema management
- **internal/config**: Configuration loading, environment variables
- **internal/logger**: Logging utilities, JSON output
- **internal/lock**: Advisory locking for safe concurrent runs
- **internal/fsutil**: File system operations, migration discovery
- **internal/checksum**: SHA256 checksums for drift detection

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

- [ ] All tests pass
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version bumped in go.mod
- [ ] Git tag created
- [ ] GitHub release created

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/mirajehossain/gomigratex/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mirajehossain/gomigratex/discussions)
- **Email**: [Your email if you want to provide it]

## Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- GitHub contributors page

Thank you for contributing to gomigratex! 🚀
