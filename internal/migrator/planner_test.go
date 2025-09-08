package migrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mirajehossain/gomigratex/internal/checksum"
)

// helper to create temp migration files
func writePair(t *testing.T, dir, ts, name, up, down string) {
	t.Helper()
	base := ts + "_" + name
	if err := os.WriteFile(filepath.Join(dir, base+".up.sql"), []byte(up), 0o644); err != nil {
		t.Fatalf("write up: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, base+".down.sql"), []byte(down), 0o644); err != nil {
		t.Fatalf("write down: %v", err)
	}
}

func TestDiscoverAndPlan_PendingAndApplied(t *testing.T) {
	// temp dir with 2 migrations
	dir := t.TempDir()
	writePair(t, dir, "20250101000000", "init", "CREATE TABLE t1(id INT);", "DROP TABLE t1;")
	writePair(t, dir, "20250102000000", "add_col", "ALTER TABLE t1 ADD COLUMN c INT;", "ALTER TABLE t1 DROP COLUMN c;")

	// mock DB with storage showing first applied successfully
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	columns := []string{"version", "name", "checksum", "applied_at", "applied_by", "duration_ms", "status", "execution_order"}
	// compute real checksum of the up file to avoid drift error
	upb, err := os.ReadFile(filepath.Join(dir, "20250101000000_init.up.sql"))
	if err != nil {
		t.Fatalf("read up: %v", err)
	}
	chk := checksum.SHA256(upb)
	rows := sqlmock.NewRows(columns).
		AddRow("20250101000000", "init", chk, time.Now(), "tester", int64(5), "success", int64(1))
	mock.ExpectQuery("SELECT version, name, checksum").WillReturnRows(rows)

	st := &Storage{DB: db, Table: "schema_migrations"}
	plan, err := DiscoverAndPlan(context.Background(), FileSource{RootDir: dir}, st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.All) != 2 {
		t.Fatalf("expected 2 files, got %d", len(plan.All))
	}
	if len(plan.Pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(plan.Pending))
	}
	if plan.Pending[0].Name != "add_col" {
		t.Fatalf("expected add_col pending, got %s", plan.Pending[0].Name)
	}
}

func TestKey(t *testing.T) {
	if Key("v", "n") != "v:n" {
		t.Fatal("key mismatch")
	}
}
