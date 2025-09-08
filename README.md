# gomigratex

A production-ready Go migration package + CLI for MySQL/MariaDB with **per-file tracking**, **safe out-of-order apply**, **advisory locking**, **transactional executes**, and **structured logs**.

## Features
- Tracks each migration (version, name, checksum, status, applied_at, applied_by, duration_ms, execution_order)
- Out-of-order applies: anything not recorded will run
- Advisory lock (`GET_LOCK`) prevents concurrent runners
- Transaction per migration (requires `multiStatements=true` in DSN)
- CLI: `up`, `down [n]`, `status`, `create <name>`, `repair`, `force <version>`
- Sources: filesystem or `embed.FS`
- Config: env-first + optional YAML

## Install
```bash
git clone https://github.com/mirajehossain/gomigratex
cd gomigratex
go mod tidy
go build ./cmd/migrate
```

## DSN (MySQL)
> **Important**: Enable multi-statements so `.sql` files with multiple statements run inside a transaction.
```
mysql://user:pass@tcp(127.0.0.1:3306)/dbname?parseTime=true&multiStatements=true
```
or classic DSN:
```
user:pass@tcp(127.0.0.1:3306)/dbname?parseTime=true&multiStatements=true
```

## Usage
```bash
# Show help
./migrate -h

# Apply all pending migrations (filesystem dir)
./migrate up --dsn "$DSN" --dir "./migrations"

# Roll back last 2
./migrate down 2 --dsn "$DSN" --dir "./migrations"

# Status
./migrate status --dsn "$DSN" --dir "./migrations" --json

# Create a new pair
./migrate create add_users_table --dir ./migrations

# Repair checksums after manual edits
./migrate repair --dsn "$DSN" --dir ./migrations --yes

# Force baseline to a version without running SQL (mark as applied)
./migrate force 20250825010101 --dsn "$DSN" --dir ./migrations --fake
```

## Embedded Migrations
See `examples/embedded/embedded.go` for how to pass an `embed.FS` into the library.

## Table Schema
Created automatically (`schema_migrations`). Unique on `(version, name)`.

## Down Behavior (Audit)
`down` deletes rows for the reverted migrations to reflect "not applied". Audit trail is preserved in **structured logs** your CI stores.

## Config
- Env: `DB_DSN`, `MIGRATIONS_DIR`, `LOCK_TIMEOUT_SEC`, `MIGRATIONS_TABLE` (default `schema_migrations`).
- YAML: pass `--config config.yaml` (env overrides YAML).

## Exit Codes
- `0` success
- `2` drift detected (checksum mismatch) — run `repair` if intentional
- `3` lock timeout / concurrency blocked
- `4` migration failure
- `5` planner error (duplicates/gaps/pair mismatches)
```

