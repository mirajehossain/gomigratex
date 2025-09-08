package lock

import "testing"

func TestKeyFor(t *testing.T) {
	if KeyFor("db", "t") != "gomigratex:db:t" {
		t.Fatal("key format mismatch")
	}
}
