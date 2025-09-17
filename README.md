# gomigratex

A lightweight, production-ready MySQL migration tool for Go applications. Supports both CLI usage and library integration with embedded migrations.

## Features

- ✅ **MySQL-first**: Optimized for MySQL with proper connection handling
- ✅ **CLI & Library**: Use as standalone tool or embed in your Go app
- ✅ **Embedded migrations**: Bundle SQL files into your binary
- ✅ **Checksum validation**: Detect drift between files and database
- ✅ **Advisory locking**: Safe concurrent migration runs
- ✅ **Rollback support**: Down migrations with proper ordering
- ✅ **Baseline support**: Mark existing schema as migrated
- ✅ **JSON logging**: Structured output for monitoring
- ✅ **Dry run mode**: Plan without executing

## Installation

### CLI Tool

**Install Latest Version:**
```bash
go install github.com/mirajehossain/gomigratex/cmd/migratex@latest
```

**Install Specific Version:**
```bash
# Install a specific release version
go install github.com/mirajehossain/gomigratex/cmd/migratex@v1.0.0

# Install a specific commit
go install github.com/mirajehossain/gomigratex/cmd/migratex@abc1234
```

**Verify Installation:**
```bash
migratex -v
```

### Library

**Add to Your Project (Latest):**
```bash
go get github.com/mirajehossain/gomigratex@latest
```

**Add Specific Version:**
```bash
# Use a specific release version
go get github.com/mirajehossain/gomigratex@v1.0.0

# Use a specific commit
go get github.com/mirajehossain/gomigratex@abc1234
```

**In your go.mod:**
```go
require github.com/mirajehossain/gomigratex v1.0.0
```

## Quick Start

### 1. Create Migration Files
```bash
migratex create add_users_table --dir ./migrations
```
This creates:
- `20250101120000_add_users_table.up.sql`
- `20250101120000_add_users_table.down.sql`

### 2. Write Your SQL
**up.sql:**
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**down.sql:**
```sql
DROP TABLE users;
```

### 3. Run Migrations
```bash
# Set your database connection
export DB_DSN="user:pass@tcp(localhost:3306)/mydb?parseTime=true&multiStatements=true"

# Apply all pending migrations
migratex up --dsn "$DB_DSN" --dir ./migrations

# Check status
migratex status --dsn "$DB_DSN" --dir ./migrations

# Rollback last migration
migratex down 1 --dsn "$DB_DSN" --dir ./migrations
```

## CLI Usage

### Commands

| Command           | Description                            |
| ----------------- | -------------------------------------- |
| `up`              | Apply all pending migrations           |
| `down <n>`        | Roll back last n migrations (or `all`) |
| `status`          | Show applied/pending state             |
| `create <name>`   | Create new migration pair              |
| `repair`          | Update checksums after file edits      |
| `force <version>` | Mark migrations as applied (baseline)  |

### Global Flags

| Flag             | Description              | Default             |
| ---------------- | ------------------------ | ------------------- |
| `--dsn`          | Database DSN             | `$DB_DSN`           |
| `--dir`          | Migrations directory     | `./migrations`      |
| `--table`        | Migrations table name    | `schema_migrations` |
| `--json`         | JSON output              | `false`             |
| `--dry-run`      | Plan only, don't execute | `false`             |
| `--verbose`      | Per-migration logs       | `false`             |
| `--lock-timeout` | Lock timeout (seconds)   | `30`                |

### Examples

```bash
# Basic usage
migratex up --dsn "$DB_DSN" --dir ./migrations

# With verbose logging
migratex up --dsn "$DB_DSN" --dir ./migrations --verbose

# JSON output for monitoring
migratex status --dsn "$DB_DSN" --dir ./migrations --json

# Dry run to see what would happen
migratex up --dsn "$DB_DSN" --dir ./migrations --dry-run

# Create migration with custom name
migratex create add_user_indexes --dir ./migrations

# Rollback all migrations
migratex down all --dsn "$DB_DSN" --dir ./migrations

# Baseline existing database
migratex force 20250101000000 --dsn "$DB_DSN" --dir ./migrations
```

## Library Usage

### Basic Integration

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/mirajehossain/gomigratex/internal/db"
    "github.com/mirajehossain/gomigratex/internal/migrator"
)

func main() {
    // Connect to database
    database, err := db.OpenMySQL("user:pass@tcp(localhost:3306)/mydb?parseTime=true&multiStatements=true")
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()

    // Create migrator
    runner := migrator.NewRunner(database, "schema_migrations", "myapp")

    // Ensure migrations table exists
    if err := runner.Ensure(context.Background()); err != nil {
        log.Fatal(err)
    }

    // Discover and plan migrations
    src := migrator.FileSource{RootDir: "./migrations"}
    plan, err := migrator.DiscoverAndPlan(context.Background(), src, runner.Storage)
    if err != nil {
        log.Fatal(err)
    }

    // Apply pending migrations
    if len(plan.Pending) > 0 {
        applied, err := runner.ApplyUp(context.Background(), plan.Pending, false, nil)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Applied %d migrations", len(applied))
    }
}
```

### Embedded Migrations

```go
package main

import (
    "context"
    "embed"
    "log"

    "github.com/mirajehossain/gomigratex/internal/db"
    "github.com/mirajehossain/gomigratex/internal/migrator"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
    database, err := db.OpenMySQL("user:pass@tcp(localhost:3306)/mydb?parseTime=true&multiStatements=true")
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()

    runner := migrator.NewRunner(database, "schema_migrations", "myapp")
    if err := runner.Ensure(context.Background()); err != nil {
        log.Fatal(err)
    }

    // Use embedded filesystem
    src := migrator.FileSource{
        FS: migrationFS,
        RootDir: "migrations",
        Embedded: true,
    }

    plan, err := migrator.DiscoverAndPlan(context.Background(), src, runner.Storage)
    if err != nil {
        log.Fatal(err)
    }

    applied, err := runner.ApplyUp(context.Background(), plan.Pending, false, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Applied %d migrations", len(applied))
}
```

## Migration File Naming

Migration files must follow this pattern:
```
{timestamp}_{name}.{up|down}.sql
```

- `timestamp`: 14-digit timestamp (YYYYMMDDHHMMSS)
- `name`: Descriptive name (lowercase, underscores)
- `up`: Forward migration
- `down`: Rollback migration

Examples:
- `20250101120000_add_users_table.up.sql`
- `20250101120000_add_users_table.down.sql`
- `20250101120001_add_user_indexes.up.sql`
- `20250101120001_add_user_indexes.down.sql`

## Database Schema

The tool creates a `schema_migrations` table (configurable) with:

```sql
CREATE TABLE schema_migrations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    version VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_by VARCHAR(255) NOT NULL,
    duration_ms BIGINT NOT NULL,
    status ENUM('success','failed') NOT NULL,
    execution_order BIGINT NOT NULL,
    UNIQUE KEY uniq_version_name (version, name)
);
```

## Best Practices

### 1. Always Write Down Migrations
```sql
-- up.sql
CREATE TABLE users (id INT, name VARCHAR(255));

-- down.sql
DROP TABLE users;
```

### 2. Use Transactions for Complex Migrations
```sql
-- up.sql
START TRANSACTION;
CREATE TABLE users (id INT PRIMARY KEY);
CREATE INDEX idx_users_name ON users(name);
COMMIT;
```

### 3. Test Your Migrations
```bash
# Test forward migration
migratex up --dsn "$DB_DSN" --dir ./migrations

# Test rollback
migratex down 1 --dsn "$DB_DSN" --dir ./migrations

# Test forward again
migratex up --dsn "$DB_DSN" --dir ./migrations
```

### 4. Use Descriptive Names
```bash
# Good
migratex create add_user_email_index

# Avoid
migratex create migration_001
```

### 5. Handle Data Migrations Carefully
```sql
-- up.sql
-- Add column with default
ALTER TABLE users ADD COLUMN email VARCHAR(255) DEFAULT '';

-- Update existing data
UPDATE users SET email = CONCAT('user', id, '@example.com') WHERE email = '';

-- Make NOT NULL
ALTER TABLE users MODIFY COLUMN email VARCHAR(255) NOT NULL;
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/) and uses Git tags for releases.

### Available Versions

You can see all available versions on the [GitHub Releases](https://github.com/mirajehossain/gomigratex/releases) page.

### Using Specific Versions

**CLI Installation:**
```bash
# Latest stable release
go install github.com/mirajehossain/gomigratex/cmd/migratex@latest

# Specific version
go install github.com/mirajehossain/gomigratex/cmd/migrate@v1.2.0

# Latest pre-release (if any)
go install github.com/mirajehossain/gomigratex/cmd/migrate@v0.9.0-beta.1
```

**Library Usage:**
```bash
# Add to go.mod
go get github.com/mirajehossain/gomigratex@v1.2.0

# Or use go mod edit
go mod edit -require=github.com/mirajehossain/gomigratex@v1.2.0
```

**Version Compatibility:**
- **v1.x.x**: Stable API, backward compatible within major version
- **v0.x.x**: Pre-1.0 releases, breaking changes may occur between minor versions
- **main branch**: Development version, may be unstable

### Checking Your Version

```bash
migratex version
```

This will show:
- Version number
- Build timestamp
- Git commit hash

## Configuration

### Environment Variables

| Variable           | Description                | Default             |
| ------------------ | -------------------------- | ------------------- |
| `DB_DSN`           | Database connection string | -                   |
| `MIGRATIONS_DIR`   | Migrations directory       | `./migrations`      |
| `MIGRATIONS_TABLE` | Migrations table name      | `schema_migrations` |
| `LOCK_TIMEOUT_SEC` | Lock timeout (seconds)     | `30`                |
| `APPLIED_BY`       | User who applied migration | Current user        |

### YAML Configuration

Create `migrate.yaml`:
```yaml
dsn: "user:pass@tcp(localhost:3306)/mydb?parseTime=true&multiStatements=true"
dir: "./migrations"
table: "schema_migrations"
lock_timeout_sec: 30
applied_by: "deployment"
json: true
```

Use with:
```bash
migratex up --config migrate.yaml
```

## Troubleshooting

### Common Issues

**1. "multiStatements=true" required**
```
Error: migration failed: You have an error in your SQL syntax
```
Solution: Add `multiStatements=true` to your DSN.

**2. Checksum drift detected**
```
Error: checksum drift detected: 20250101120000:add_users_table
```
Solution: Use `migratex repair` after intentional file edits.

**3. Lock timeout**
```
Error: failed to acquire advisory lock
```
Solution: Check for other migration processes or increase `--lock-timeout`.

**4. Missing down file**
```
Error: missing down file for 20250101120000:add_users_table
```
Solution: Create the corresponding `.down.sql` file.

### Debug Mode

Use `--verbose` for detailed logging:
```bash
migratex up --dsn "$DB_DSN" --dir ./migrations --verbose
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/mirajehossain/gomigratex.git
   cd gomigratex
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run tests**
   ```bash
   make test
   # or
   go test ./...
   ```

4. **Build the CLI**
   ```bash
   go build -o migratex ./cmd/migrate
   ```

### Project Structure

```
gomigratex/
├── cmd/migrate/          # CLI application
├── internal/
│   ├── checksum/         # SHA256 checksum utilities
│   ├── config/           # Configuration management
│   ├── db/               # Database connection & schema
│   ├── fsutil/           # File system utilities
│   ├── lock/             # Advisory locking
│   ├── logger/           # Logging utilities
│   └── migrator/         # Core migration logic
├── examples/             # Usage examples
├── migrations/           # Sample migration files
└── README.md
```

### Adding Features

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Write tests**
   - Add unit tests for new functionality
   - Ensure existing tests pass

3. **Update documentation**
   - Update README.md if needed
   - Add examples for new features

4. **Submit a pull request**
   - Describe your changes
   - Link any related issues

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/migrator

# Run with coverage
go test -cover ./...

# Run integration tests (requires MySQL)
DB_DSN="user:pass@tcp(localhost:3306)/testdb" go test -tags=integration ./...
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `golint`
- Write clear, self-documenting code
- Add comments for public APIs

### Release Process

**For Maintainers:**

1. **Prepare Release**
   ```bash
   # Ensure working directory is clean
   git status

   # Run full test suite
   make test

   # Update version in changelog if needed
   # Update any version references in documentation
   ```

2. **Create Release**
   ```bash
   # Create and push tag + build binaries
   make release-create VERSION=v1.2.0

   # Or step by step:
   make release-tag VERSION=v1.2.0    # Creates and pushes git tag
   make release-build VERSION=v1.2.0  # Builds binaries for all platforms
   ```

3. **Verify Release**
   ```bash
   # Test installation from the new tag
   go install github.com/mirajehossain/gomigratex/cmd/migrate@v1.2.0
   migratex version
   ```

4. **GitHub Release** (Manual)
   - Go to [GitHub Releases](https://github.com/mirajehossain/gomigratex/releases)
   - Create a new release from the tag
   - Upload the built binaries (`migratex-linux`, `migratex-darwin`, `migratex-windows.exe`)
   - Write release notes describing changes

**Version Numbering:**
- Follow [Semantic Versioning](https://semver.org/)
- Format: `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`)
- Use `v0.x.x` for pre-1.0 releases

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release
- MySQL migration support
- CLI and library interfaces
- Embedded migration support
- Checksum validation
- Advisory locking
- Rollback support

## Support

- **Issues**: [GitHub Issues](https://github.com/mirajehossain/gomigratex/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mirajehossain/gomigratex/discussions)
