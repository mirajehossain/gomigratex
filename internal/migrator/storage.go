package migrator

import (
	"context"
	"database/sql"
	"fmt"
)

type Storage struct {
	DB    *sql.DB
	Table string
}

func (s *Storage) GetAll(ctx context.Context) (map[string]Row, error) {
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`SELECT version, name, checksum, applied_at, applied_by, duration_ms, status, execution_order FROM %s`, s.Table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Version, &r.Name, &r.Checksum, &r.AppliedAt, &r.AppliedBy, &r.DurationMS, &r.Status, &r.ExecutionOrder); err != nil {
			return nil, err
		}
		out[keyOf(r.Version, r.Name)] = r
	}
	return out, rows.Err()
}

func (s *Storage) MaxExecutionOrder(ctx context.Context) (int64, error) {
	row := s.DB.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE(MAX(execution_order), 0) FROM %s`, s.Table))
	var max int64
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	return max, nil
}

func (s *Storage) Upsert(ctx context.Context, r Row) error {
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (version, name, checksum, applied_at, applied_by, duration_ms, status, execution_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE checksum=VALUES(checksum), applied_at=VALUES(applied_at), applied_by=VALUES(applied_by), duration_ms=VALUES(duration_ms), status=VALUES(status), execution_order=VALUES(execution_order)
`, s.Table),
		r.Version, r.Name, r.Checksum, r.AppliedAt, r.AppliedBy, r.DurationMS, r.Status, r.ExecutionOrder,
	)
	return err
}

func (s *Storage) Delete(ctx context.Context, version, name string) error {
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE version=? AND name=?`, s.Table), version, name)
	return err
}

func keyOf(version, name string) string { return version + ":" + name }
