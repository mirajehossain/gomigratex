package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mirajehossain/gomigratex/internal/config"
	"github.com/mirajehossain/gomigratex/internal/db"
	"github.com/mirajehossain/gomigratex/internal/lock"
	"github.com/mirajehossain/gomigratex/internal/logger"
	"github.com/mirajehossain/gomigratex/internal/migrator"
)

const (
	exitOK        = 0
	exitDrift     = 2
	exitLocked    = 3
	exitFail      = 4
	exitPlanError = 5
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		usage()
		return exitOK
	}
	cmd := os.Args[1]
	global := flag.NewFlagSet("global", flag.ContinueOnError)
	dsn := global.String("dsn", "", "Database DSN (or set DB_DSN)")
	dir := global.String("dir", "./migrations", "Migrations directory (or MIGRATIONS_DIR)")
	jsonOut := global.Bool("json", false, "JSON logs")
	dryRun := global.Bool("dry-run", false, "Plan only; do not execute")
	conf := global.String("config", "", "Optional YAML config path")
	lockTimeout := global.Int("lock-timeout", 30, "Lock timeout seconds (or LOCK_TIMEOUT_SEC)")
	table := global.String("table", "schema_migrations", "Migrations table name")
	embedded := global.Bool("embedded", false, "Use embedded FS (examples/embedded) [library usage]")
	appliedBy := global.String("applied-by", "", "Override applied_by value")
	verbose := global.Bool("verbose", false, "Verbose per-migration logs")

	// Subcommands
	switch cmd {
	case "up", "status", "repair":
		// no extra args
	case "down":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "down requires N (number of steps) or 'all'")
			return exitPlanError
		}
	case "create":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "create requires a <name>")
			return exitPlanError
		}
	case "force":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "force requires <version>")
			return exitPlanError
		}
	default:
		usage()
		return exitOK
	}

	// Parse global flags after command word and its first arg if any
	argStart := 2
	if cmd == "down" || cmd == "create" || cmd == "force" {
		argStart = 3
	}
	if err := global.Parse(os.Args[argStart:]); err != nil {
		return exitPlanError
	}

	// Load config
	cfg, _ := config.LoadYAML(*conf)
	cfg = config.MergeEnv(cfg)
	// override with flags if provided
	if *dsn != "" {
		cfg.DSN = *dsn
	}
	if *dir != "" {
		cfg.Dir = *dir
	}
	cfg.JSON = *jsonOut
	cfg.DryRun = *dryRun
	cfg.LockTimeoutSec = *lockTimeout
	if *table != "" {
		cfg.MigrationsTable = *table
	}
	cfg.Embedded = *embedded
	if *appliedBy != "" {
		cfg.AppliedBy = *appliedBy
	}

	log := logger.New(cfg.JSON)

	switch cmd {
	case "create":
		name := os.Args[2]
		if err := createPair(cfg.Dir, name, log); err != nil {
			log.Error("create failed", map[string]any{"error": err.Error()})
			return exitFail
		}
		log.Info("created migration pair", map[string]any{"dir": cfg.Dir, "name": name})
		return exitOK
	}

	// Commands that need DB
	if cfg.DSN == "" {
		fmt.Fprintln(os.Stderr, "--dsn or DB_DSN is required")
		return exitPlanError
	}
	database, err := db.OpenMySQL(cfg.DSN)
	if err != nil {
		log.Error("db open failed", map[string]any{"error": err.Error()})
		return exitFail
	}
	defer database.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure table
	run := migrator.NewRunner(database, cfg.MigrationsTable, cfg.AppliedBy)
	if err := run.Ensure(ctx); err != nil {
		log.Error("ensure table failed", map[string]any{"error": err.Error()})
		return exitFail
	}

	// Advisory lock
	lockKey := lock.KeyFor(extractDBName(cfg.DSN), cfg.MigrationsTable)
	l := lock.NewMySQL(database, lockKey)
	if err := l.Acquire(ctx, database, cfg.LockTimeout()); err != nil {
		log.Error("failed to acquire lock", map[string]any{"error": err.Error(), "key": lockKey})
		return exitLocked
	}
	defer func() { _ = l.Release(ctx) }()

	// Build source and plan
	var src migrator.FileSource
	src.RootDir = cfg.Dir
	if cfg.Embedded {
		log.Warn("embedded mode requested - provide an embed.FS via library API; CLI uses filesystem", nil)
	}
	plan, err := migrator.DiscoverAndPlan(ctx, src, run.Storage)
	if err != nil {
		if errors.Is(err, migrator.ErrDrift) {
			log.Error("drift detected", map[string]any{"error": err.Error()})
			return exitDrift
		}
		log.Error("plan failed", map[string]any{"error": err.Error()})
		return exitPlanError
	}

	switch cmd {
	case "status":
		if *verbose {
			// Applied map is in plan.Applied
			appliedCount := 0
			for _, row := range plan.Applied {
				appliedCount++
				log.Info("status.applied", map[string]any{
					"version":         row.Version,
					"name":            row.Name,
					"checksum":        row.Checksum,
					"status":          row.Status,
					"applied_at":      row.AppliedAt.UTC().Format(time.RFC3339),
					"applied_by":      row.AppliedBy,
					"duration_ms":     row.DurationMS,
					"execution_order": row.ExecutionOrder,
				})
			}
			pendingCount := 0
			for _, fp := range plan.All {
				k := migrator.Key(fp.Version, fp.Name)
				if _, ok := plan.Applied[k]; !ok {
					pendingCount++
					log.Info("status.pending", map[string]any{
						"version":  fp.Version,
						"name":     fp.Name,
						"checksum": fp.Checksum,
					})
				}
			}
			log.Info("status.summary", map[string]any{
				"applied": appliedCount,
				"pending": pendingCount,
			})
			return exitOK
		}

		// Default (non-verbose) status behavior you already had:
		printStatus(plan, log)
		return exitOK
	case "up":
		if len(plan.Pending) == 0 {
			log.Info("no pending migrations", nil)
			return exitOK
		}
		// If verbose, list the plan
		if *verbose {
			for _, fp := range plan.Pending {
				log.Info("plan.apply", map[string]any{
					"version":  fp.Version,
					"name":     fp.Name,
					"checksum": fp.Checksum,
				})
			}
		}

		// Progress callback
		progress := func(stage string, fp migrator.FilePair, row *migrator.Row, err error) {
			if !*verbose {
				return
			}
			fields := map[string]any{
				"version": fp.Version,
				"name":    fp.Name,
			}
			if row != nil {
				fields["order"] = row.ExecutionOrder
			}
			if err != nil {
				fields["error"] = err.Error()
			}
			switch stage {
			case "start":
				log.Info("migrate.start", fields)
			case "success":
				if row != nil {
					fields["duration_ms"] = row.DurationMS
				}
				log.Info("migrate.success", fields)
			case "error":
				log.Error("migrate.error", fields)
			}
		}

		applied, err := run.ApplyUp(ctx, plan.Pending, cfg.DryRun, progress)
		if err != nil {
			log.Error("up failed", map[string]any{"error": err.Error()})
			return exitFail
		}

		log.Info("up complete", map[string]any{
			"applied": len(applied),
			"dry_run": cfg.DryRun,
		})
		return exitOK
	case "down":
		arg := os.Args[2]

		var rows []migrator.Row
		var err error
		if strings.ToLower(arg) == "all" {
			rows, err = run.LastApplied(ctx, 999999999) // all
		} else {
			n, convErr := strconv.Atoi(arg)
			if convErr != nil || n <= 0 {
				log.Error("invalid N for down", map[string]any{"arg": arg})
				return exitPlanError
			}
			rows, err = run.LastApplied(ctx, n)
		}
		if err != nil {
			log.Error("down query failed", map[string]any{"error": err.Error()})
			return exitFail
		}
		if len(rows) == 0 {
			log.Info("nothing to roll back", nil)
			return exitOK
		}

		// Build lookup for down SQL
		lookup := map[string]migrator.FilePair{}
		for _, fp := range plan.All {
			lookup[migrator.Key(fp.Version, fp.Name)] = fp
		}

		// Verbose plan list
		if *verbose {
			for _, r := range rows {
				log.Info("plan.rollback", map[string]any{
					"version": r.Version,
					"name":    r.Name,
					"order":   r.ExecutionOrder,
					"status":  r.Status,
				})
			}
		}

		// Progress callback
		progress := func(stage string, fp migrator.FilePair, row *migrator.Row, err error) {
			if !*verbose {
				return
			}
			fields := map[string]any{
				"version": fp.Version,
				"name":    fp.Name,
			}
			if row != nil {
				fields["order"] = row.ExecutionOrder
			}
			if err != nil {
				fields["error"] = err.Error()
			}
			switch stage {
			case "start":
				log.Info("migrate.down.start", fields)
			case "success":
				log.Info("migrate.down.success", fields)
			case "error":
				log.Error("migrate.down.error", fields)
			}
		}

		if err := run.ApplyDown(ctx, rows, lookup, cfg.DryRun, progress); err != nil {
			log.Error("down failed", map[string]any{"error": err.Error()})
			return exitFail
		}

		log.Info("down complete", map[string]any{"reverted": len(rows), "dry_run": cfg.DryRun})
		return exitOK
	case "repair":
		changed, err := repairChecksums(plan, run, cfg.DryRun)
		if err != nil {
			log.Error("repair failed", map[string]any{"error": err.Error()})
			return exitFail
		}
		log.Info("repair complete", map[string]any{"updated": changed, "dry_run": cfg.DryRun})
		return exitOK
	case "force":
		version := os.Args[2]
		fake := hasFlag("--fake")
		applied, err := run.ForceBaseline(ctx, plan.All, version, fake)
		if err != nil {
			log.Error("force failed", map[string]any{"error": err.Error()})
			return exitFail
		}
		log.Info("force complete", map[string]any{"count": len(applied), "fake": fake})
		return exitOK
	default:
		usage()
		return exitOK
	}
}

func usage() {
	fmt.Println(`gomigratex - Go migration CLI

USAGE:
  migratex <command> [args] [--flags]

COMMANDS:
  up                        Apply all pending migrations
  down <n>                  Roll back last n migrations
  status                    Show applied/pending state
  create <name>             Scaffold yyyyMMddHHmmss_name.{up,down}.sql
  repair                    Update stored checksums to current files (use after intentional edits)
  force <version> [--fake]  Mark all <= version as applied (baseline); with --fake skip running SQL

GLOBAL FLAGS:
  --dsn <dsn>               Database DSN (or DB_DSN)
  --dir <path>              Migrations directory (default ./migrations)
  --json                    JSON logs
  --dry-run                 Plan only; don't execute SQL
  --lock-timeout <sec>      Advisory lock timeout (default 30)
  --table <name>            Migrations table (default schema_migrations)
  --config <path>           Optional YAML config path
  --applied-by <name>       Override applied_by
  --verbose       	    Verbose per-migration logs

EXAMPLES:
  migratex up --dsn "$DSN" --dir ./migrations
  migratex down 1 --dsn "$DSN" --dir ./migrations
  migratex status --dsn "$DSN" --dir ./migrations --json
  migratex create add_user_table --dir ./migrations
  migratex repair --dsn "$DSN" --dir ./migrations --yes
  migratex force 20250825010101 --dsn "$DSN" --dir ./migrations --fake`)
}

func hasFlag(name string) bool {
	for _, a := range os.Args {
		if a == name {
			return true
		}
	}
	return false
}

func printStatus(plan *migrator.Plan, log *logger.Logger) {
	type item struct {
		Version  string `json:"version"`
		Name     string `json:"name"`
		Checksum string `json:"checksum"`
		Status   string `json:"status"` // applied|pending|failed
	}
	var out []item
	for _, fp := range plan.All {
		k := migrator.Key(fp.Version, fp.Name)
		if row, ok := plan.Applied[k]; ok {
			out = append(out, item{Version: fp.Version, Name: fp.Name, Checksum: fp.Checksum, Status: row.Status})
		} else {
			out = append(out, item{Version: fp.Version, Name: fp.Name, Checksum: fp.Checksum, Status: "pending"})
		}
	}
	if log != nil {
		// print as table-ish if not JSON
		if !log.JSONEnabled() {
			for _, it := range out {
				fmt.Printf("%s %-30s %-8s %s\n", it.Version, it.Name, it.Status, it.Checksum[:12])
			}
			return
		}
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(out)
	}
}

// deprecated: logJsonEnabled hack removed; use log.JSONEnabled()

func createPair(dir, name string, log *logger.Logger) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102150405")
	base := fmt.Sprintf("%s_%s", ts, sanitize(name))
	up := filepath.Join(dir, base+".up.sql")
	down := filepath.Join(dir, base+".down.sql")
	if err := os.WriteFile(up, []byte("-- write your UP migration here\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(down, []byte("-- write your DOWN migration here\n"), 0o644); err != nil {
		return err
	}
	if log != nil {
		log.Info("created files", map[string]any{"up": up, "down": down})
	}
	return nil
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func repairChecksums(plan *migrator.Plan, run *migrator.Runner, dry bool) (int, error) {
	changed := 0
	for _, fp := range plan.All {
		k := migrator.Key(fp.Version, fp.Name)
		row, ok := plan.Applied[k]
		if !ok {
			continue // pending; nothing to repair
		}
		if strings.EqualFold(row.Checksum, fp.Checksum) {
			continue
		}
		row.Checksum = fp.Checksum
		row.AppliedAt = time.Now()
		if dry {
			changed++
			continue
		}
		if err := run.Storage.Upsert(context.Background(), row); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

func extractDBName(dsn string) string {
	// naive extraction: find "/" then "?" or end
	// user:pass@tcp(127.0.0.1:3306)/dbname?params
	i := strings.LastIndex(dsn, "/")
	if i == -1 || i == len(dsn)-1 {
		return "db"
	}
	rest := dsn[i+1:]
	if j := strings.Index(rest, "?"); j != -1 {
		return rest[:j]
	}
	return rest
}
