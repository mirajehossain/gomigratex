package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultAndLockTimeout(t *testing.T) {
	c := Default()
	if c.MigrationsTable != "schema_migrations" {
		t.Fatal("default table mismatch")
	}
	if Default().LockTimeout() != 30*time.Second {
		t.Fatal("default timeout mismatch")
	}
}

func TestLoadYAMLAndMergeEnv(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(p, []byte("dsn: mysql://u:p@/db\ndir: ./migs\nlock_timeout_sec: 10\nmigrations_table: t\napplied_by: me\n"), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	cfg, err := LoadYAML(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Dir != "./migs" || cfg.MigrationsTable != "t" || cfg.LockTimeoutSec != 10 {
		t.Fatal("yaml load mismatch")
	}
	os.Setenv("MIGRATIONS_DIR", "./x")
	os.Setenv("LOCK_TIMEOUT_SEC", "20")
	os.Setenv("MIGRATIONS_TABLE", "y")
	os.Setenv("APPLIED_BY", "you")
	cfg = MergeEnv(cfg)
	if cfg.Dir != "./x" || cfg.MigrationsTable != "y" || cfg.LockTimeoutSec != 20 || cfg.AppliedBy != "you" {
		t.Fatal("env merge mismatch")
	}
}
