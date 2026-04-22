// Package actions turns a compare/canonical/naming result into an
// ActionPlan and then executes it — with dry-run, audit logging, and
// invariants enforced before any filesystem mutation.
package actions

import (
	"fmt"
	"path/filepath"
	"sort"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

// PlanInput bundles everything a Planner needs. Maps are keyed by their
// natural key so callers don't have to worry about slice ordering.
type PlanInput struct {
	Job       model.ScanJob
	Groups    []model.DuplicateGroup
	Uniques   []model.DiscoveredFile
	Sels      map[string]model.CanonicalSelection  // BLAKE3Hex -> selection
	Files     map[string]model.DiscoveredFile      // AbsolutePath -> file
	Mappings  map[string]model.OutputMapping       // AbsolutePath -> mapping (copy-unique only)
	Cfg       config.Config
}

// Plan generates a deterministic ActionPlan for the configured mode. The
// returned plan is ordered by lexical SrcPath with Seq assigned 1-based.
func Plan(in PlanInput) (model.ActionPlan, error) {
	var items []model.ActionPlanItem

	canonicalSet := make(map[string]string, len(in.Sels)) // path -> blake3
	for hex, sel := range in.Sels {
		canonicalSet[sel.Canonical] = hex
	}

	switch in.Cfg.Mode {
	case model.ModeReport:
		items = planReport(in, canonicalSet)
	case model.ModeCopyUnique:
		var err error
		items, err = planCopyUnique(in, canonicalSet)
		if err != nil {
			return model.ActionPlan{}, err
		}
	case model.ModeMoveDuplicates:
		items = planMoveDuplicates(in, canonicalSet)
	case model.ModeDeleteDuplicates:
		items = planDeleteDuplicates(in, canonicalSet)
	default:
		return model.ActionPlan{}, fmt.Errorf("actions: unsupported mode %q", in.Cfg.Mode)
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].SrcPath < items[j].SrcPath })
	for i := range items {
		items[i].Seq = i + 1
	}

	return model.ActionPlan{
		StartedAt: in.Job.StartedAt,
		Items:     items,
	}, nil
}

func planReport(in PlanInput, canonicals map[string]string) []model.ActionPlanItem {
	items := make([]model.ActionPlanItem, 0)
	for _, u := range in.Uniques {
		items = append(items, model.ActionPlanItem{
			Kind: model.ActionKeep, SrcPath: u.AbsolutePath, Rationale: "unique",
		})
	}
	for _, g := range in.Groups {
		canonical := in.Sels[g.BLAKE3Hex].Canonical
		for _, p := range g.Members {
			if p == canonical {
				items = append(items, model.ActionPlanItem{
					Kind: model.ActionKeep, SrcPath: p, Rationale: "canonical",
				})
				continue
			}
			items = append(items, model.ActionPlanItem{
				Kind:      model.ActionIgnore,
				SrcPath:   p,
				Rationale: fmt.Sprintf("duplicate of %s", canonical),
			})
		}
	}
	_ = canonicals
	return items
}

func planCopyUnique(in PlanInput, canonicals map[string]string) ([]model.ActionPlanItem, error) {
	items := make([]model.ActionPlanItem, 0)
	emit := func(srcPath string) error {
		m, ok := in.Mappings[srcPath]
		if !ok {
			return fmt.Errorf("actions: missing OutputMapping for %s", srcPath)
		}
		items = append(items, model.ActionPlanItem{
			Kind:      model.ActionCopy,
			SrcPath:   srcPath,
			DstPath:   m.OutputPath,
			Rationale: "unique-or-canonical",
		})
		return nil
	}

	for _, u := range in.Uniques {
		if err := emit(u.AbsolutePath); err != nil {
			return nil, err
		}
	}
	for _, g := range in.Groups {
		canonical := in.Sels[g.BLAKE3Hex].Canonical
		if err := emit(canonical); err != nil {
			return nil, err
		}
		for _, p := range g.Members {
			if p == canonical {
				continue
			}
			items = append(items, model.ActionPlanItem{
				Kind: model.ActionIgnore, SrcPath: p,
				Rationale: fmt.Sprintf("duplicate of %s", canonical),
			})
		}
	}
	_ = canonicals
	return items, nil
}

func planMoveDuplicates(in PlanInput, canonicals map[string]string) []model.ActionPlanItem {
	items := make([]model.ActionPlanItem, 0)
	for _, u := range in.Uniques {
		items = append(items, model.ActionPlanItem{
			Kind: model.ActionKeep, SrcPath: u.AbsolutePath, Rationale: "unique",
		})
	}
	for _, g := range in.Groups {
		canonical := in.Sels[g.BLAKE3Hex].Canonical
		items = append(items, model.ActionPlanItem{
			Kind: model.ActionKeep, SrcPath: canonical, Rationale: "canonical",
		})
		for _, p := range g.Members {
			if p == canonical {
				continue
			}
			dst := moveDestination(in, p)
			items = append(items, model.ActionPlanItem{
				Kind:      model.ActionMove,
				SrcPath:   p,
				DstPath:   dst,
				Rationale: fmt.Sprintf("duplicate of %s", canonical),
			})
		}
	}
	_ = canonicals
	return items
}

func planDeleteDuplicates(in PlanInput, canonicals map[string]string) []model.ActionPlanItem {
	items := make([]model.ActionPlanItem, 0)
	for _, u := range in.Uniques {
		items = append(items, model.ActionPlanItem{
			Kind: model.ActionKeep, SrcPath: u.AbsolutePath, Rationale: "unique",
		})
	}
	for _, g := range in.Groups {
		canonical := in.Sels[g.BLAKE3Hex].Canonical
		items = append(items, model.ActionPlanItem{
			Kind: model.ActionKeep, SrcPath: canonical, Rationale: "canonical",
		})
		for _, p := range g.Members {
			if p == canonical {
				continue
			}
			items = append(items, model.ActionPlanItem{
				Kind:      model.ActionDelete,
				SrcPath:   p,
				Rationale: fmt.Sprintf("duplicate of %s", canonical),
			})
		}
	}
	_ = canonicals
	return items
}

// moveDestination decides where a duplicate should be moved to.
//
// Strategy:
//   - If config.Destination.Dir is set: <dest>/<root_basename>/<relative_path>.
//   - Otherwise:                        <root>/_duplicates/<relative_path>.
//
// We always preserve the relative path under the root so users can review
// moved files in a structure that mirrors their original layout.
func moveDestination(in PlanInput, srcPath string) string {
	f, ok := in.Files[srcPath]
	if !ok {
		return srcPath
	}
	if in.Cfg.Destination.Dir != "" {
		rootName := filepath.Base(f.InputRoot)
		return filepath.Join(in.Cfg.Destination.Dir, rootName, f.RelativePath)
	}
	return filepath.Join(f.InputRoot, "_duplicates", f.RelativePath)
}
