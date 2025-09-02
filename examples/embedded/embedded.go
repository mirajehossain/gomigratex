package main

import (
	"context"
	"embed"
	"log"

	"github.com/mirajehossain/gomigratex/internal/db"
	"github.com/mirajehossain/gomigratex/internal/migrator"
)

//go:embed migrations/*.sql
var migFS embed.FS

func main() {
	dsn := "user:admin@testpass1(127.0.0.1:3306)/test?parseTime=true&multiStatements=true"
	sqlDB, err := db.OpenMySQL(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	run := migrator.NewRunner(sqlDB, "schema_migrations", "embedded-example")
	if err := run.Ensure(context.Background()); err != nil {
		log.Fatal(err)
	}

	src := migrator.FileSource{
		FS: migFS, RootDir: "migrations", Embedded: true,
	}
	plan, err := migrator.DiscoverAndPlan(context.Background(), src, run.Storage)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := run.ApplyUp(context.Background(), plan.Pending, false, nil); err != nil {
		log.Fatal(err)
	}
}
