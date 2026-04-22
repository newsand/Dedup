package actions

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

// CheckInvariants enforces the safety rules documented in
// Docs/09-design-actions.md § 9.9. It runs BEFORE any action is applied.
func CheckInvariants(plan model.ActionPlan, canonicals map[string]struct{}, cfg config.Config) error {
	// 1) No destructive action targets a canonical path.
	for _, it := range plan.Items {
		if it.Kind == model.ActionDelete || it.Kind == model.ActionMove {
			if _, isCanonical := canonicals[it.SrcPath]; isCanonical {
				return fmt.Errorf("invariant: %s action on canonical %s", it.Kind, it.SrcPath)
			}
		}
	}

	// 2) For copy-unique, no dst is inside any input root.
	if cfg.Mode == model.ModeCopyUnique {
		absRoots := make([]string, 0, len(cfg.Roots))
		for _, r := range cfg.Roots {
			a, err := filepath.Abs(r)
			if err != nil {
				return err
			}
			absRoots = append(absRoots, a)
		}
		for _, it := range plan.Items {
			if it.Kind != model.ActionCopy {
				continue
			}
			absDst, err := filepath.Abs(it.DstPath)
			if err != nil {
				return err
			}
			for _, r := range absRoots {
				if pathContains(r, absDst) {
					return fmt.Errorf("invariant: copy destination %s is inside root %s", absDst, r)
				}
			}
		}
	}

	// 3) All destructive dst paths unique (naming already resolves collisions).
	dsts := make(map[string]struct{})
	for _, it := range plan.Items {
		if it.DstPath == "" {
			continue
		}
		if _, seen := dsts[it.DstPath]; seen {
			return fmt.Errorf("invariant: duplicate destination %s", it.DstPath)
		}
		dsts[it.DstPath] = struct{}{}
	}

	// 6) Seq is strictly increasing and starts at 1.
	for i, it := range plan.Items {
		if it.Seq != i+1 {
			return errors.New("invariant: plan seq is not contiguous 1..N")
		}
	}
	return nil
}

func pathContains(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if runtime.GOOS == "windows" {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	if parent == child {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(parent, sep) {
		parent += sep
	}
	return strings.HasPrefix(child, parent)
}
