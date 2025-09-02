package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/user"
	"strings"
	"time"

	"github.com/mirajehossain/gomigratex/internal/db"
)

type Runner struct {
	DB        *sql.DB
	Storage   *Storage
	AppliedBy string
}

func NewRunner(database *sql.DB, table string, appliedBy string) *Runner {
	return &Runner{
		DB:        database,
		Storage:   &Storage{DB: database, Table: table},
		AppliedBy: appliedBy,
	}
}

func defaultAppliedBy() string {
	u, err := user.Current()
	if err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}

func (r *Runner) Ensure(ctx context.Context) error {
	if err := db.EnsureTable(ctx, r.DB, r.Storage.Table); err != nil {
		return err
	}
	if strings.TrimSpace(r.AppliedBy) == "" {
		r.AppliedBy = defaultAppliedBy()
	}
	return nil
}

func (r *Runner) ApplyUp(ctx context.Context, files []FilePair, dryRun bool, progress func(stage string, fp FilePair, row *Row, err error)) ([]Row, error) {
	applied := make([]Row, 0, len(files))
	maxOrder, err := r.Storage.MaxExecutionOrder(ctx)
	if err != nil {
		return nil, err
	}
	for _, fp := range files {
		maxOrder++
		row := Row{
			Version:        fp.Version,
			Name:           fp.Name,
			Checksum:       fp.Checksum,
			AppliedAt:      time.Now(),
			AppliedBy:      r.AppliedBy,
			Status:         "success",
			ExecutionOrder: maxOrder,
		}

		// progress: start
		if progress != nil {
			progress("start", fp, &row, nil)
		}

		if dryRun {
			if progress != nil {
				progress("success", fp, &row, nil)
			}
			applied = append(applied, row)
			continue
		}

		start := time.Now()
		tx, err := r.DB.BeginTx(ctx, nil)
		if err != nil {
			if progress != nil {
				progress("error", fp, &row, err)
			}
			return nil, err
		}

		// NOTE: DSN must include multiStatements=true if file has multiple statements
		if _, err := tx.ExecContext(ctx, string(fp.UpBytes)); err != nil {
			_ = tx.Rollback()
			row.Status = "failed"
			row.DurationMS = time.Since(start).Milliseconds()
			_ = r.Storage.Upsert(ctx, row)
			if progress != nil {
				progress("error", fp, &row, err)
			}
			return applied, fmt.Errorf("migration %s:%s failed: %w", fp.Version, fp.Name, err)
		}

		if err := tx.Commit(); err != nil {
			row.Status = "failed"
			row.DurationMS = time.Since(start).Milliseconds()
			_ = r.Storage.Upsert(ctx, row)
			if progress != nil {
				progress("error", fp, &row, err)
			}
			return applied, err
		}

		row.DurationMS = time.Since(start).Milliseconds()
		if err := r.Storage.Upsert(ctx, row); err != nil {
			if progress != nil {
				progress("error", fp, &row, err)
			}
			return applied, err
		}

		// progress: success
		if progress != nil {
			progress("success", fp, &row, nil)
		}
		applied = append(applied, row)
	}
	return applied, nil
}

func (r *Runner) ApplyDown(ctx context.Context, toRevert []Row, lookup map[string]FilePair, dryRun bool) error {
	for _, row := range toRevert {
		fp, ok := lookup[row.Version+":"+row.Name]
		if !ok {
			return fmt.Errorf("missing down file for %s:%s", row.Version, row.Name)
		}
		if dryRun {
			continue
		}
		tx, err := r.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(fp.DownBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("down migration %s:%s failed: %w", row.Version, row.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		// Remove record to indicate "not applied"
		if err := r.Storage.Delete(ctx, row.Version, row.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) LastApplied(ctx context.Context, n int) ([]Row, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT version, name, checksum, applied_at, applied_by, duration_ms, status, execution_order FROM "+r.Storage.Table+" WHERE status='success' ORDER BY execution_order DESC LIMIT ?", n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var rr Row
		if err := rows.Scan(&rr.Version, &rr.Name, &rr.Checksum, &rr.AppliedAt, &rr.AppliedBy, &rr.DurationMS, &rr.Status, &rr.ExecutionOrder); err != nil {
			return nil, err
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}

var ErrNoSuchVersion = errors.New("no such version")

func (r *Runner) ForceBaseline(ctx context.Context, all []FilePair, version string, fake bool) ([]Row, error) {
	applied := make([]Row, 0)
	maxOrder, err := r.Storage.MaxExecutionOrder(ctx)
	if err != nil {
		return nil, err
	}
	for _, fp := range all {
		// Apply up to and including version
		if fp.Version > version {
			continue
		}
		maxOrder++
		row := Row{
			Version: fp.Version, Name: fp.Name, Checksum: fp.Checksum,
			AppliedAt: time.Now(), AppliedBy: r.AppliedBy, DurationMS: 0,
			Status: "success", ExecutionOrder: maxOrder,
		}
		if !fake {
			// actually run .up.sql (baseline via executing)
			tx, err := r.DB.BeginTx(ctx, nil)
			if err != nil {
				return applied, err
			}
			if _, err := tx.ExecContext(ctx, string(fp.UpBytes)); err != nil {
				_ = tx.Rollback()
				return applied, err
			}
			if err := tx.Commit(); err != nil {
				return applied, err
			}
		}
		if err := r.Storage.Upsert(ctx, row); err != nil {
			return applied, err
		}
		applied = append(applied, row)
	}
	return applied, nil
}
