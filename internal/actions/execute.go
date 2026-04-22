package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"deduplicator/internal/config"
	dfs "deduplicator/internal/fs"
	"deduplicator/internal/model"
	"deduplicator/internal/report"
)

// Executor applies an ActionPlan.
type Executor struct {
	cfg   config.Config
	audit *report.AuditLog
	log   *slog.Logger
}

// NewExecutor wires an Executor. audit may be nil (no logging will happen).
func NewExecutor(cfg config.Config, audit *report.AuditLog, log *slog.Logger) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{cfg: cfg, audit: audit, log: log}
}

// Execute applies plan. In dry-run mode every item is recorded as
// status=planned and the filesystem is left untouched. Returns one
// ActionLog per item.
func (e *Executor) Execute(ctx context.Context, plan model.ActionPlan) ([]model.ActionLog, error) {
	if e.cfg.DryRun {
		return e.runDryRun(ctx, plan)
	}

	workers := e.cfg.Concurrency.ActionWorkers
	if workers <= 0 {
		workers = 4
	}

	logs := make([]model.ActionLog, len(plan.Items))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)
	var mu sync.Mutex

	for i, it := range plan.Items {
		i, it := i, it
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			logEntry := e.executeOne(it)
			mu.Lock()
			logs[i] = logEntry
			mu.Unlock()
			if e.audit != nil {
				if err := e.audit.Append(logEntry); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return logs, err
	}
	return logs, nil
}

func (e *Executor) runDryRun(_ context.Context, plan model.ActionPlan) ([]model.ActionLog, error) {
	out := make([]model.ActionLog, len(plan.Items))
	for i, it := range plan.Items {
		out[i] = model.ActionLog{
			Seq:     it.Seq,
			Kind:    it.Kind,
			SrcPath: it.SrcPath,
			DstPath: it.DstPath,
			Status:  model.StatusPlanned,
			Message: it.Rationale,
			Time:    time.Now().UTC(),
		}
	}
	return out, nil
}

func (e *Executor) executeOne(it model.ActionPlanItem) model.ActionLog {
	base := model.ActionLog{
		Seq:     it.Seq,
		Kind:    it.Kind,
		SrcPath: it.SrcPath,
		DstPath: it.DstPath,
		Time:    time.Now().UTC(),
	}
	switch it.Kind {
	case model.ActionKeep, model.ActionIgnore:
		base.Status = model.StatusExecuted
		base.Message = it.Rationale
		return base
	case model.ActionCopy:
		copied, err := dfs.CopyFile(it.SrcPath, it.DstPath)
		return finish(base, copied, err, "copied", "already exists")
	case model.ActionMove:
		moved, err := dfs.Move(it.SrcPath, it.DstPath)
		return finish(base, moved, err, "moved", "already exists")
	case model.ActionDelete:
		if err := dfs.Delete(it.SrcPath); err != nil {
			base.Status = model.StatusFailed
			base.Message = err.Error()
			return base
		}
		base.Status = model.StatusExecuted
		return base
	}
	base.Status = model.StatusFailed
	base.Message = fmt.Sprintf("unknown action kind %q", it.Kind)
	return base
}

func finish(base model.ActionLog, changed bool, err error, okMsg, skipMsg string) model.ActionLog {
	if err != nil {
		if errors.Is(err, dfs.ErrDestExistsMismatch) {
			base.Status = model.StatusFailed
			base.Message = err.Error()
			return base
		}
		base.Status = model.StatusFailed
		base.Message = err.Error()
		return base
	}
	if !changed {
		base.Status = model.StatusSkipped
		base.Message = skipMsg
		return base
	}
	base.Status = model.StatusExecuted
	base.Message = okMsg
	return base
}
