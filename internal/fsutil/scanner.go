package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var fileRe = regexp.MustCompile(`^(\d+)_([a-zA-Z0-9_\-]+)\.(up|down)\.sql$`)

type Pair struct {
	Version  string
	Name     string
	UpPath   string // path in fs
	DownPath string
}

type FS interface {
	fs.FS
}

// ScanDir scans a local directory on disk.
func ScanDir(dir string) (map[string]*Pair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	return scan(entries, func(name string) string { return filepath.Join(dir, name) })
}

// ScanEmbedded scans an embedded fs under a root dir path (logical path).
func ScanEmbedded(fsys fs.FS, root string) (map[string]*Pair, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	return scan(entries, func(name string) string { return filepath.Join(root, name) })
}

func scan(entries []fs.DirEntry, full func(name string) string) (map[string]*Pair, error) {
	out := map[string]*Pair{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		version, name, typ := m[1], m[2], m[3]
		key := version + ":" + name
		p := out[key]
		if p == nil {
			p = &Pair{Version: version, Name: name}
			out[key] = p
		}
		switch typ {
		case "up":
			if p.UpPath != "" {
				return nil, errors.New("duplicate up file for version " + version)
			}
			p.UpPath = full(e.Name())
		case "down":
			if p.DownPath != "" {
				return nil, errors.New("duplicate down file for version " + version)
			}
			p.DownPath = full(e.Name())
		}
	}
	// Validate all have both up/down
	for k, p := range out {
		if p.UpPath == "" || p.DownPath == "" {
			return nil, errors.New("missing pair for " + k)
		}
	}
	return out, nil
}

func SortKeys(m map[string]*Pair) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		// Compare on version then name
		vi := strings.SplitN(keys[i], ":", 2)[0]
		vj := strings.SplitN(keys[j], ":", 2)[0]
		if vi == vj {
			return keys[i] < keys[j]
		}
		return vi < vj
	})
	return keys
}
