// Package report writes the human-readable reports and the machine-readable
// audit log.
//
// The audit log is a JSONL file: one ActionLog per line. We write the full
// plan upfront (status=planned) then append the outcome of each step
// (status=executed|skipped|failed). That guarantees we can reconstruct what
// the tool intended to do even if the process was killed halfway through.
package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"deduplicator/internal/model"
)

// AuditLog is a thread-safe JSONL writer over a file handle. Every Append
// immediately flushes and fsyncs so the log is durable before an action
// touches the filesystem.
type AuditLog struct {
	mu   sync.Mutex
	f    *os.File
	w    *bufio.Writer
	path string
}

// auditFileName produces a Windows-safe name based on the job start time.
// Format: 2006-01-02T150405Z — no colons (NTFS-unfriendly).
func auditFileName(startedAt time.Time) string {
	return startedAt.UTC().Format("2006-01-02T150405Z") + ".jsonl"
}

// NewAuditLog opens (or creates) an audit log under dir for the given job.
// The returned AuditLog must be closed by the caller.
func NewAuditLog(dir string, startedAt time.Time) (*AuditLog, error) {
	if dir == "" {
		dir = ".dedup-audit"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("audit: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, auditFileName(startedAt))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("audit: open %s: %w", path, err)
	}
	return &AuditLog{f: f, w: bufio.NewWriter(f), path: path}, nil
}

// Path returns the full path of the audit log file.
func (a *AuditLog) Path() string {
	if a == nil {
		return ""
	}
	return a.path
}

// AppendPlan writes every item of the plan with status=planned and fsyncs.
// Invariant: the audit log must contain the full planned set before any
// destructive action is applied.
func (a *AuditLog) AppendPlan(plan model.ActionPlan) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, it := range plan.Items {
		entry := model.ActionLog{
			Seq:     it.Seq,
			Kind:    it.Kind,
			SrcPath: it.SrcPath,
			DstPath: it.DstPath,
			Status:  model.StatusPlanned,
			Message: it.Rationale,
			Time:    plan.StartedAt,
		}
		if err := a.writeLocked(entry); err != nil {
			return err
		}
	}
	return a.syncLocked()
}

// Append writes a single ActionLog entry.
func (a *AuditLog) Append(log model.ActionLog) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.writeLocked(log); err != nil {
		return err
	}
	return a.syncLocked()
}

func (a *AuditLog) writeLocked(entry model.ActionLog) error {
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := a.w.Write(b); err != nil {
		return err
	}
	if err := a.w.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func (a *AuditLog) syncLocked() error {
	if err := a.w.Flush(); err != nil {
		return err
	}
	return a.f.Sync()
}

// Close flushes and closes the underlying file.
func (a *AuditLog) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.f == nil {
		return nil
	}
	_ = a.w.Flush()
	_ = a.f.Sync()
	err := a.f.Close()
	a.f = nil
	return err
}
