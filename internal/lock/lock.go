package lock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MySQL advisory lock using GET_LOCK/RELEASE_LOCK on a dedicated connection.
type MySQL struct {
	conn *sql.Conn
	key  string
	held bool
}

func NewMySQL(db *sql.DB, key string) *MySQL {
	return &MySQL{key: key}
}

func (m *MySQL) Acquire(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	if m.held {
		return nil
	}
	var err error
	m.conn, err = db.Conn(ctx)
	if err != nil {
		return err
	}
	// GET_LOCK(name, timeout_seconds)
	row := m.conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", m.key, int(timeout.Seconds()))
	var got sql.NullInt64
	if err := row.Scan(&got); err != nil {
		_ = m.conn.Close()
		return err
	}
	if !got.Valid || got.Int64 != 1 {
		_ = m.conn.Close()
		return errors.New("failed to acquire advisory lock (timeout or error)")
	}
	m.held = true
	return nil
}

func (m *MySQL) Release(ctx context.Context) error {
	if !m.held || m.conn == nil {
		return nil
	}
	row := m.conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", m.key)
	var rel sql.NullInt64
	_ = row.Scan(&rel) // do not fail on release
	m.held = false
	return m.conn.Close()
}

func (m *MySQL) Key() string { return m.key }

func KeyFor(database, table string) string {
	return fmt.Sprintf("gomigratex:%s:%s", database, table)
}
