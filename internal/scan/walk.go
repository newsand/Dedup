// Package scan walks the filesystem and emits DiscoveredFile entries for
// regular files that match the configured filters.
package scan

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deduplicator/internal/model"
)

// DetectFn maps (path, extension) to a FileType. Injected to avoid a cyclic
// import with internal/filetype at the package boundary and to keep walk
// unit-testable.
type DetectFn func(path, ext string) model.FileType

// Options controls a single call to Walk.
type Options struct {
	Roots        []string // absolute or relative; each resolved to absolute
	Filters      Filters
	Detect       DetectFn
	IncludeTypes map[model.FileType]bool // when nil, all types allowed
	FollowSymlinks bool                  // default false
}

// Walk traverses the configured roots and returns discovered files.
//
// Behaviour:
//   - Each root is resolved to its absolute path.
//   - Symlinks are skipped by default (FollowSymlinks=false).
//   - Directories and special files (device, socket, named pipe) are skipped.
//   - Unreadable files are reported as errors but do not abort the walk
//     unless the context is cancelled.
//   - Results are sorted by AbsolutePath for determinism.
func Walk(ctx context.Context, opts Options) ([]model.DiscoveredFile, []error) {
	var (
		out  []model.DiscoveredFile
		errs []error
	)

	detect := opts.Detect
	if detect == nil {
		detect = func(_, _ string) model.FileType { return model.FileTypeUnknown }
	}

	for _, r := range opts.Roots {
		root, err := filepath.Abs(r)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			mode := info.Mode()
			if mode&os.ModeSymlink != 0 && !opts.FollowSymlinks {
				return nil
			}
			if !mode.IsRegular() {
				return nil
			}
			if !opts.Filters.AllowPath(path) {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			ft := detect(path, ext)
			if len(opts.IncludeTypes) > 0 && !opts.IncludeTypes[ft] {
				return nil
			}

			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = filepath.Base(path)
			}

			out = append(out, model.DiscoveredFile{
				AbsolutePath: path,
				InputRoot:    root,
				RelativePath: rel,
				FileType:     ft,
				Extension:    ext,
				SizeBytes:    info.Size(),
				MTime:        info.ModTime().UTC().Truncate(time.Second),
			})
			return nil
		})
		if walkErr != nil && ctx.Err() != nil {
			return out, append(errs, walkErr)
		}
	}

	// Deterministic ordering by absolute path.
	sortByPath(out)
	return out, errs
}

func sortByPath(f []model.DiscoveredFile) {
	// Insertion sort is fine for the sizes we target; but stdlib sort is O(n log n)
	// and we want to avoid accidental instability from non-stable sorts.
	sortSlice(f, func(i, j int) bool { return f[i].AbsolutePath < f[j].AbsolutePath })
}
