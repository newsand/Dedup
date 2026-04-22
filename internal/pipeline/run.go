// Package pipeline is the orchestrator that ties together scan -> compare ->
// canonical -> naming -> actions -> report.
//
// The pipeline has no knowledge of CLI concerns (flags, output formatting,
// exit codes). Its inputs and outputs are plain Go types, which makes the
// whole pipeline unit-testable without spawning a subprocess.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"

	"deduplicator/internal/actions"
	"deduplicator/internal/canonical"
	"deduplicator/internal/compare"
	"deduplicator/internal/config"
	"deduplicator/internal/filetype"
	"deduplicator/internal/model"
	"deduplicator/internal/naming"
	"deduplicator/internal/report"
	"deduplicator/internal/scan"
)

// Input is everything pipeline.Run needs.
type Input struct {
	Cfg     config.Config
	Version string
	Logger  *slog.Logger
}

// Output bundles the results of a pipeline run.
type Output struct {
	Job        model.ScanJob
	Groups     []model.DuplicateGroup
	Uniques    []model.DiscoveredFile
	Selections map[string]model.CanonicalSelection
	Mappings   map[string]model.OutputMapping
	Plan       model.ActionPlan
	Logs       []model.ActionLog
	AuditPath  string
	Errors     []error
	Duration   time.Duration
}

// Run executes the full pipeline and returns the aggregated Output.
func Run(ctx context.Context, in Input) (Output, error) {
	startedAt := time.Now().UTC()
	logger := in.Logger
	if logger == nil {
		logger = slog.Default()
	}

	job := model.ScanJob{
		StartedAt:    startedAt,
		Mode:         in.Cfg.Mode,
		DryRun:       in.Cfg.DryRun,
		SuppressName: in.Cfg.SuppressName,
		Roots:        absAll(in.Cfg.Roots),
		OutputDir:    absOrEmpty(in.Cfg.Output.Dir),
		DestDir:      absOrEmpty(in.Cfg.Destination.Dir),
		Version:      in.Version,
	}

	// --- scan ------------------------------------------------------------
	includeTypes := map[model.FileType]bool{
		model.FileTypeImage: true,
		model.FileTypePDF:   true,
	}
	discovered, scanErrs := scan.Walk(ctx, scan.Options{
		Roots:          in.Cfg.Roots,
		Filters:        scan.NewFilters(in.Cfg.Filters.Include, in.Cfg.Filters.Exclude),
		Detect:         filetype.Detect,
		IncludeTypes:   includeTypes,
		FollowSymlinks: in.Cfg.FollowSymlinks,
	})
	logger.Info("pipeline: scan complete",
		slog.Int("files", len(discovered)), slog.Int("errors", len(scanErrs)))

	// --- pre-size filter -------------------------------------------------
	candidates, singletons := compare.PreSize(discovered)

	// --- compare ---------------------------------------------------------
	workers := in.Cfg.Concurrency.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	delegate := compare.NewDelegate(
		compare.NewRegistry(compare.NewImageComparator(), compare.NewPDFComparator()),
		workers, in.Cfg.Strict, logger,
	)
	res, err := delegate.Run(ctx, candidates)
	if err != nil {
		return Output{Job: job}, err
	}
	// Merge singletons with delegate.Uniques.
	uniques := append([]model.DiscoveredFile{}, singletons...)
	uniques = append(uniques, res.Uniques...)

	// Index discovered files by path.
	files := make(map[string]model.DiscoveredFile, len(discovered))
	for _, f := range discovered {
		files[f.AbsolutePath] = f
	}

	// --- canonical -------------------------------------------------------
	selections := make(map[string]model.CanonicalSelection, len(res.Groups))
	for _, g := range res.Groups {
		sel, err := canonical.Select(canonical.Input{Group: g, Files: files})
		if err != nil {
			return Output{Job: job}, err
		}
		selections[g.BLAKE3Hex] = sel
	}

	// --- naming (only for copy-unique) ----------------------------------
	mappings := map[string]model.OutputMapping{}
	if in.Cfg.Mode == model.ModeCopyUnique {
		outAbs, err := filepath.Abs(in.Cfg.Output.Dir)
		if err != nil {
			return Output{Job: job}, err
		}
		toName := make([]model.DiscoveredFile, 0, len(uniques)+len(res.Groups))
		toName = append(toName, uniques...)
		for _, g := range res.Groups {
			if f, ok := files[selections[g.BLAKE3Hex].Canonical]; ok {
				toName = append(toName, f)
			}
		}
		mappings = naming.Build(toName, outAbs, naming.Rules{Suppress: in.Cfg.SuppressName})
	}

	// --- plan ------------------------------------------------------------
	job.StartedAt = startedAt // (re-affirm after possible mutations)
	plan, err := actions.Plan(actions.PlanInput{
		Job:      job,
		Groups:   res.Groups,
		Uniques:  uniques,
		Sels:     selections,
		Files:    files,
		Mappings: mappings,
		Cfg:      in.Cfg,
	})
	if err != nil {
		return Output{Job: job}, fmt.Errorf("plan: %w", err)
	}
	canonicalPaths := make(map[string]struct{}, len(selections))
	for _, s := range selections {
		canonicalPaths[s.Canonical] = struct{}{}
	}
	if err := actions.CheckInvariants(plan, canonicalPaths, in.Cfg); err != nil {
		return Output{Job: job, Plan: plan}, fmt.Errorf("invariant: %w", err)
	}

	// --- audit + execute ------------------------------------------------
	audit, err := report.NewAuditLog(in.Cfg.Audit.Dir, startedAt)
	if err != nil {
		return Output{Job: job, Plan: plan}, err
	}
	defer audit.Close()
	if err := audit.AppendPlan(plan); err != nil {
		return Output{Job: job, Plan: plan}, err
	}

	logs, err := actions.NewExecutor(in.Cfg, audit, logger).Execute(ctx, plan)
	if err != nil {
		return Output{Job: job, Plan: plan, Logs: logs, AuditPath: audit.Path()}, err
	}

	job.FinishedAt = time.Now().UTC()
	errsAll := make([]error, 0, len(scanErrs)+len(res.Errors))
	errsAll = append(errsAll, scanErrs...)
	for _, fe := range res.Errors {
		errsAll = append(errsAll, fmt.Errorf("hash %s: %w", fe.Path, fe.Err))
	}

	return Output{
		Job:        job,
		Groups:     res.Groups,
		Uniques:    uniques,
		Selections: selections,
		Mappings:   mappings,
		Plan:       plan,
		Logs:       logs,
		AuditPath:  audit.Path(),
		Errors:     errsAll,
		Duration:   job.FinishedAt.Sub(job.StartedAt),
	}, nil
}

func absAll(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		if a, err := filepath.Abs(r); err == nil {
			out = append(out, a)
		} else {
			out = append(out, r)
		}
	}
	return out
}

func absOrEmpty(p string) string {
	if p == "" {
		return ""
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}
