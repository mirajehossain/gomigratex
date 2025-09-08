package db

import "testing"

func TestOpenMySQLAppendsParseTime(t *testing.T) {
	dsn := "user:pass@tcp(localhost:3306)/db"
	db, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.Close()
}
