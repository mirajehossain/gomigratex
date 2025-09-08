package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirAndSortKeys(t *testing.T) {
	dir := t.TempDir()
	// valid pair
	if err := os.WriteFile(filepath.Join(dir, "20250101000000_init.up.sql"), []byte("-- up"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20250101000000_init.down.sql"), []byte("-- down"), 0o644); err != nil {
		t.Fatal(err)
	}
	// another pair
	_ = os.WriteFile(filepath.Join(dir, "20250102000000_add.up.sql"), []byte("-- up"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "20250102000000_add.down.sql"), []byte("-- down"), 0o644)

	pairs, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	keys := SortKeys(pairs)
	if len(keys) != 2 || keys[0] != "20250101000000:init" || keys[1] != "20250102000000:add" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}
