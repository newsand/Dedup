package compare

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"deduplicator/internal/model"
)

// Delegate coordinates Comparator invocations over a set of discovered files.
//
// Concurrency model:
//   - `workers` goroutines run Key() in parallel via errgroup + semaphore.
//   - On strict=false, per-file errors are captured and do not abort the run.
//   - On strict=true, the first error cancels the group and is returned.
type Delegate struct {
	reg     *Registry
	workers int
	strict  bool
	log     *slog.Logger
}

// NewDelegate builds a Delegate. workers<=0 defaults to 1.
func NewDelegate(reg *Registry, workers int, strict bool, log *slog.Logger) *Delegate {
	if workers <= 0 {
		workers = 1
	}
	if log == nil {
		log = slog.Default()
	}
	return &Delegate{reg: reg, workers: workers, strict: strict, log: log}
}

// RunResult bundles everything Run produces so callers can consume it as a
// single value.
type RunResult struct {
	Groups   []model.DuplicateGroup
	Uniques  []model.DiscoveredFile
	Ignored  []model.DiscoveredFile // no Comparator registered
	Errors   []FileError
}

// FileError captures a per-file failure in non-strict mode.
type FileError struct {
	Path string
	Err  error
}

// Run computes keys for every file with a registered Comparator, groups them
// and returns the outcome.
func (d *Delegate) Run(ctx context.Context, files []model.DiscoveredFile) (RunResult, error) {
	var res RunResult

	type job struct {
		idx  int
		file model.DiscoveredFile
		comp Comparator
	}

	var jobs []job
	for i, f := range files {
		c, ok := d.reg.Resolve(f.FileType)
		if !ok {
			res.Ignored = append(res.Ignored, f)
			d.log.Debug("compare: ignoring unsupported file type",
				slog.String("path", f.AbsolutePath),
				slog.String("file_type", string(f.FileType)))
			continue
		}
		jobs = append(jobs, job{idx: i, file: f, comp: c})
	}

	comparables := make([]Comparable, len(jobs))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(d.workers)
	var mu sync.Mutex

	for i, j := range jobs {
		i, j := i, j
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			key, err := j.comp.Key(gctx, j.file)
			if err != nil {
				if d.strict {
					return err
				}
				mu.Lock()
				res.Errors = append(res.Errors, FileError{Path: j.file.AbsolutePath, Err: err})
				mu.Unlock()
				d.log.Warn("compare: hashing failed",
					slog.String("path", j.file.AbsolutePath),
					slog.String("error", err.Error()))
				return nil
			}
			comparables[i] = Comparable{File: j.file, Key: key}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return res, err
	}

	groupByKey := make(map[string][]model.DiscoveredFile)
	for _, c := range comparables {
		if c.Key == "" {
			continue // skipped (non-strict error)
		}
		groupByKey[c.Key] = append(groupByKey[c.Key], c.File)
	}

	for key, members := range groupByKey {
		if len(members) == 1 {
			res.Uniques = append(res.Uniques, members[0])
			continue
		}
		paths := make([]string, len(members))
		for i, m := range members {
			paths[i] = m.AbsolutePath
		}
		sort.Strings(paths)
		res.Groups = append(res.Groups, model.DuplicateGroup{
			BLAKE3Hex: key,
			Members:   paths,
		})
	}

	// Deterministic ordering.
	sort.Slice(res.Groups, func(i, j int) bool { return res.Groups[i].BLAKE3Hex < res.Groups[j].BLAKE3Hex })
	sort.Slice(res.Uniques, func(i, j int) bool { return res.Uniques[i].AbsolutePath < res.Uniques[j].AbsolutePath })
	sort.Slice(res.Ignored, func(i, j int) bool { return res.Ignored[i].AbsolutePath < res.Ignored[j].AbsolutePath })
	sort.Slice(res.Errors, func(i, j int) bool { return res.Errors[i].Path < res.Errors[j].Path })

	return res, nil
}
