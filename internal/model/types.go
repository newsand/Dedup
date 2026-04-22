// Package model defines pure data types that flow through the deduplication
// pipeline. Types are I/O-free DTOs; business logic lives in the pipeline,
// compare, canonical, naming, actions and report packages.
package model

import "time"

// FileType categorises a discovered file.
type FileType string

const (
	FileTypeImage   FileType = "image"
	FileTypePDF     FileType = "pdf"
	FileTypeUnknown FileType = "unknown"
)

// Mode selects which top-level action the CLI is running.
type Mode string

const (
	ModeReport           Mode = "report"
	ModeCopyUnique       Mode = "copy-unique"
	ModeMoveDuplicates   Mode = "move-duplicates"
	ModeDeleteDuplicates Mode = "delete-duplicates"
)

// ActionKind is the primitive operation an action-plan item represents.
type ActionKind string

const (
	ActionKeep   ActionKind = "keep"
	ActionCopy   ActionKind = "copy"
	ActionMove   ActionKind = "move"
	ActionDelete ActionKind = "delete"
	ActionIgnore ActionKind = "ignore"
)

// ActionStatus is the outcome recorded in the audit log.
type ActionStatus string

const (
	StatusPlanned  ActionStatus = "planned"
	StatusExecuted ActionStatus = "executed"
	StatusSkipped  ActionStatus = "skipped"
	StatusFailed   ActionStatus = "failed"
)

// ScanJob represents a single CLI invocation.
type ScanJob struct {
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	Mode         Mode      `json:"mode"`
	DryRun       bool      `json:"dry_run"`
	SuppressName bool      `json:"suppress_name"`
	Roots        []string  `json:"roots"`
	OutputDir    string    `json:"output_dir,omitempty"`
	DestDir      string    `json:"dest_dir,omitempty"`
	ConfigPath   string    `json:"config_path,omitempty"`
	Version      string    `json:"version"`
}

// DiscoveredFile is an entry produced by the scanner. Key: AbsolutePath.
type DiscoveredFile struct {
	AbsolutePath string    `json:"absolute_path"`
	InputRoot    string    `json:"input_root"`
	RelativePath string    `json:"relative_path"`
	FileType     FileType  `json:"file_type"`
	Extension    string    `json:"extension"`
	SizeBytes    int64     `json:"size_bytes"`
	MTime        time.Time `json:"mtime"`
}

// FileFingerprint is the hashing result for one file.
type FileFingerprint struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	BLAKE3Hex string `json:"blake3_hex"`
}

// DuplicateGroup groups two or more files with the same content hash.
type DuplicateGroup struct {
	BLAKE3Hex string   `json:"blake3_hex"`
	Members   []string `json:"members"`
}

// SelectionDetails carries context about how the canonical was chosen.
type SelectionDetails struct {
	MTimeCanonical  time.Time   `json:"mtime_canonical"`
	MTimesRunnersUp []time.Time `json:"mtimes_runners_up,omitempty"`
	PathCanonical   string      `json:"path_canonical"`
	PathsRunnersUp  []string    `json:"paths_runners_up,omitempty"`
}

// SelectionReason documents which rule picked the canonical.
type SelectionReason struct {
	ByMTime   bool             `json:"by_mtime"`
	ByLexical bool             `json:"by_lexical"`
	Details   SelectionDetails `json:"details"`
}

// CanonicalSelection is the chosen canonical of a DuplicateGroup.
type CanonicalSelection struct {
	BLAKE3Hex string          `json:"blake3_hex"`
	Canonical string          `json:"canonical"`
	Reason    SelectionReason `json:"reason"`
}

// OutputMapping describes where a file will land in the flattened output dir.
type OutputMapping struct {
	Path           string `json:"path"`
	OutputName     string `json:"output_name"`
	OutputPath     string `json:"output_path"`
	CollisionIndex int    `json:"collision_index"`
}

// ActionPlanItem is one step of the deterministic action plan.
type ActionPlanItem struct {
	Seq       int        `json:"seq"`
	Kind      ActionKind `json:"kind"`
	SrcPath   string     `json:"src_path"`
	DstPath   string     `json:"dst_path,omitempty"`
	Rationale string     `json:"rationale,omitempty"`
}

// ActionPlan is the ordered sequence of plan items for an execution.
type ActionPlan struct {
	StartedAt time.Time        `json:"started_at"`
	Items     []ActionPlanItem `json:"items"`
}

// ActionLog is a single audit-log entry written before and after each step.
type ActionLog struct {
	Seq     int          `json:"seq"`
	Kind    ActionKind   `json:"kind"`
	SrcPath string       `json:"src_path"`
	DstPath string       `json:"dst_path,omitempty"`
	Status  ActionStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Time    time.Time    `json:"time"`
}
