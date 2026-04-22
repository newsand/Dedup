package report

import (
	"encoding/json"
	"fmt"
	"io"

	"deduplicator/internal/model"
)

// Data is the structured payload written to JSON reports.
type Data struct {
	Job        model.ScanJob                       `json:"job"`
	Groups     []GroupOut                          `json:"groups"`
	Uniques    []string                            `json:"uniques"`
	Mappings   map[string]model.OutputMapping      `json:"mappings,omitempty"`
	Plan       model.ActionPlan                    `json:"plan"`
	Logs       []model.ActionLog                   `json:"logs"`
	AuditPath  string                              `json:"audit_path,omitempty"`
	Errors     []string                            `json:"errors,omitempty"`
	DurationMS int64                               `json:"duration_ms"`
}

// GroupOut is the JSON-serialisable variant of DuplicateGroup + selection.
type GroupOut struct {
	BLAKE3Hex string                `json:"blake3_hex"`
	Members   []string              `json:"members"`
	Canonical string                `json:"canonical"`
	Reason    model.SelectionReason `json:"reason"`
}

// WriteJSON emits a fully deterministic JSON document. Time-derived fields
// (Duration, timestamps) are the only source of variation between runs.
func WriteJSON(w io.Writer, d Data) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// WriteText emits a human-friendly summary.
func WriteText(w io.Writer, d Data) error {
	fmt.Fprintf(w, "dedup — %s (dry-run=%v)\n", d.Job.Mode, d.Job.DryRun)
	fmt.Fprintf(w, "roots: %v\n", d.Job.Roots)
	fmt.Fprintf(w, "started: %s   finished: %s   duration: %dms\n",
		d.Job.StartedAt.Format("2006-01-02T15:04:05Z"),
		d.Job.FinishedAt.Format("2006-01-02T15:04:05Z"),
		d.DurationMS)
	fmt.Fprintf(w, "duplicate groups: %d   uniques: %d\n", len(d.Groups), len(d.Uniques))
	if len(d.Errors) > 0 {
		fmt.Fprintf(w, "errors: %d\n", len(d.Errors))
	}

	for _, g := range d.Groups {
		fmt.Fprintf(w, "\ngroup %s  (%d members, canonical=%s)\n", g.BLAKE3Hex[:12], len(g.Members), g.Canonical)
		for _, m := range g.Members {
			marker := "  "
			if m == g.Canonical {
				marker = "* "
			}
			fmt.Fprintf(w, "  %s%s\n", marker, m)
		}
	}

	if len(d.Uniques) > 0 {
		fmt.Fprintf(w, "\nuniques:\n")
		for _, u := range d.Uniques {
			fmt.Fprintf(w, "  %s\n", u)
		}
	}

	if d.AuditPath != "" {
		fmt.Fprintf(w, "\naudit: %s\n", d.AuditPath)
	}
	return nil
}

// BuildData converts a pipeline result into the serialisable Data value.
func BuildData(
	job model.ScanJob,
	groups []model.DuplicateGroup,
	uniques []model.DiscoveredFile,
	sels map[string]model.CanonicalSelection,
	mappings map[string]model.OutputMapping,
	plan model.ActionPlan,
	logs []model.ActionLog,
	auditPath string,
	errs []error,
	duration int64,
) Data {
	out := Data{
		Job:        job,
		Plan:       plan,
		Logs:       logs,
		AuditPath:  auditPath,
		DurationMS: duration,
	}
	out.Mappings = mappings
	for _, g := range groups {
		sel := sels[g.BLAKE3Hex]
		out.Groups = append(out.Groups, GroupOut{
			BLAKE3Hex: g.BLAKE3Hex,
			Members:   append([]string(nil), g.Members...),
			Canonical: sel.Canonical,
			Reason:    sel.Reason,
		})
	}
	for _, u := range uniques {
		out.Uniques = append(out.Uniques, u.AbsolutePath)
	}
	for _, e := range errs {
		out.Errors = append(out.Errors, e.Error())
	}
	return out
}
