package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DSN             string `yaml:"dsn"`
	Dir             string `yaml:"dir"`
	Embedded        bool   `yaml:"embedded"`
	JSON            bool   `yaml:"json"`
	DryRun          bool   `yaml:"dry_run"`
	LockTimeoutSec  int    `yaml:"lock_timeout_sec"`
	MigrationsTable string `yaml:"migrations_table"`
	AppliedBy       string `yaml:"applied_by"`
}

func Default() *Config {
	return &Config{
		LockTimeoutSec:  30,
		MigrationsTable: "schema_migrations",
	}
}

func LoadYAML(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func MergeEnv(cfg *Config) *Config {
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.DSN = v
	}
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		cfg.Dir = v
	}
	if v := os.Getenv("LOCK_TIMEOUT_SEC"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.LockTimeoutSec = i
		}
	}
	if v := os.Getenv("MIGRATIONS_TABLE"); v != "" {
		cfg.MigrationsTable = v
	}
	if v := os.Getenv("APPLIED_BY"); v != "" {
		cfg.AppliedBy = v
	}
	return cfg
}

func (c *Config) LockTimeout() time.Duration {
	if c.LockTimeoutSec <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.LockTimeoutSec) * time.Second
}
