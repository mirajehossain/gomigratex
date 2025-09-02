package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func OpenMySQL(dsn string) (*sql.DB, error) {
	// Ensure parseTime is on, recommend multiStatements true
	if !strings.Contains(strings.ToLower(dsn), "parsetime=") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func EnsureTable(ctx context.Context, db *sql.DB, table string) error {
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`, table)
	_, err := db.ExecContext(ctx, ddl)
	return err
}

var ErrLockTimeout = errors.New("advisory lock wait timeout")
