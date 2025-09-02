package migrator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/mirajehossain/gomigratex/internal/checksum"
	"github.com/mirajehossain/gomigratex/internal/fsutil"
)

type FileSource struct {
	FS       fs.FS // nil means local disk
	RootDir  string
	Embedded bool
}

type FilePair struct {
	Version   string
	Name      string
	UpPath    string
	DownPath  string
	UpBytes   []byte
	DownBytes []byte
	Checksum  string
}

type Plan struct {
	Pending []FilePair // to apply in order
	Applied map[string]Row
	All     []FilePair // all discovered
}

var (
	ErrDrift = errors.New("checksum drift detected")
)

// DiscoverAndPlan loads migration pairs and decides which to run.
// Out-of-order applies are supported: anything not (status=success) is considered pending.
func DiscoverAndPlan(ctx context.Context, src FileSource, st *Storage) (*Plan, error) {
	var pairs map[string]*fsutil.Pair
	var err error
	if src.Embedded && src.FS != nil {
		pairs, err = fsutil.ScanEmbedded(src.FS, src.RootDir)
	} else {
		pairs, err = fsutil.ScanDir(src.RootDir)
	}
	if err != nil {
		return nil, err
	}
	// Read file contents & checksum
	all := make([]FilePair, 0, len(pairs))
	keys := fsutil.SortKeys(pairs)
	for _, k := range keys {
		p := pairs[k]
		var upb, downb []byte
		if src.Embedded && src.FS != nil {
			upb, err = fs.ReadFile(src.FS, p.UpPath)
			if err != nil {
				return nil, err
			}
			downb, err = fs.ReadFile(src.FS, p.DownPath)
			if err != nil {
				return nil, err
			}
		} else {
			upb, err = os.ReadFile(p.UpPath)
			if err != nil {
				return nil, err
			}
			downb, err = os.ReadFile(p.DownPath)
			if err != nil {
				return nil, err
			}
		}
		chk := checksum.SHA256(upb) // checksum on up file
		all = append(all, FilePair{
			Version: p.Version, Name: p.Name, UpPath: p.UpPath, DownPath: p.DownPath,
			UpBytes: upb, DownBytes: downb, Checksum: chk,
		})
	}
	applied, err := st.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	pending := make([]FilePair, 0, len(all))
	for _, fp := range all {
		k := fp.Version + ":" + fp.Name
		if row, ok := applied[k]; ok {
			// If recorded success but checksum differs => drift
			if row.Status == "success" && !strings.EqualFold(row.Checksum, fp.Checksum) {
				return nil, fmt.Errorf("%w: %s (db=%s file=%s)", ErrDrift, k, row.Checksum, fp.Checksum)
			}
			// If failed previously, retry
			if row.Status == "failed" {
				pending = append(pending, fp)
			}
			continue // already applied
		}
		// Not present -> pending
		pending = append(pending, fp)
	}
	return &Plan{Pending: pending, Applied: applied, All: all}, nil
}
