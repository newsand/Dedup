package actions

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

func writeTmp(t *testing.T, p string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustPlan(t *testing.T, in PlanInput) model.ActionPlan {
	t.Helper()
	p, err := Plan(in)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func canonicalSet(sels map[string]model.CanonicalSelection) map[string]struct{} {
	out := make(map[string]struct{}, len(sels))
	for _, s := range sels {
		out[s.Canonical] = struct{}{}
	}
	return out
}

// --- Plan tests -----------------------------------------------------------

func TestPlan_Report_UniquesAndCanonicalsAreKeep(t *testing.T) {
	job := model.ScanJob{StartedAt: time.Now()}
	files := map[string]model.DiscoveredFile{
		"/a": {AbsolutePath: "/a"}, "/b": {AbsolutePath: "/b"}, "/c": {AbsolutePath: "/c"},
	}
	sels := map[string]model.CanonicalSelection{
		"H1": {BLAKE3Hex: "H1", Canonical: "/a"},
	}
	cfg := config.Default()
	cfg.Mode = model.ModeReport
	cfg.Roots = []string{"/"}

	plan := mustPlan(t, PlanInput{
		Job:     job,
		Groups:  []model.DuplicateGroup{{BLAKE3Hex: "H1", Members: []string{"/a", "/b"}}},
		Uniques: []model.DiscoveredFile{{AbsolutePath: "/c"}},
		Sels:    sels, Files: files, Cfg: cfg,
	})

	kinds := map[string]model.ActionKind{}
	for _, it := range plan.Items {
		kinds[it.SrcPath] = it.Kind
	}
	if kinds["/a"] != model.ActionKeep || kinds["/c"] != model.ActionKeep || kinds["/b"] != model.ActionIgnore {
		t.Fatalf("unexpected kinds: %+v", kinds)
	}
}

func TestPlan_SeqIsContiguous(t *testing.T) {
	job := model.ScanJob{StartedAt: time.Now()}
	sels := map[string]model.CanonicalSelection{"H": {BLAKE3Hex: "H", Canonical: "/a"}}
	files := map[string]model.DiscoveredFile{
		"/a": {AbsolutePath: "/a"}, "/b": {AbsolutePath: "/b"}, "/c": {AbsolutePath: "/c"},
	}
	cfg := config.Default()
	cfg.Mode = model.ModeReport
	cfg.Roots = []string{"/"}

	plan := mustPlan(t, PlanInput{
		Job: job, Sels: sels, Files: files, Cfg: cfg,
		Groups:  []model.DuplicateGroup{{BLAKE3Hex: "H", Members: []string{"/a", "/b"}}},
		Uniques: []model.DiscoveredFile{{AbsolutePath: "/c"}},
	})

	for i, it := range plan.Items {
		if it.Seq != i+1 {
			t.Fatalf("seq broken at %d: %d", i, it.Seq)
		}
	}
	if !sort.SliceIsSorted(plan.Items, func(i, j int) bool {
		return plan.Items[i].SrcPath < plan.Items[j].SrcPath
	}) {
		t.Fatal("plan not sorted by SrcPath")
	}
}

// Case K — never delete a unique file and never touch the canonical.
func TestCaseK_NeverDeleteUniqueFile(t *testing.T) {
	job := model.ScanJob{StartedAt: time.Now()}
	sels := map[string]model.CanonicalSelection{"H": {BLAKE3Hex: "H", Canonical: "/a/keep.png"}}
	files := map[string]model.DiscoveredFile{
		"/a/keep.png": {AbsolutePath: "/a/keep.png"},
		"/b/dup.png":  {AbsolutePath: "/b/dup.png"},
		"/c/lone.png": {AbsolutePath: "/c/lone.png"},
	}
	cfg := config.Default()
	cfg.Mode = model.ModeDeleteDuplicates
	cfg.Roots = []string{"/"}
	cfg.DryRun = true

	plan := mustPlan(t, PlanInput{
		Job: job, Sels: sels, Files: files, Cfg: cfg,
		Groups:  []model.DuplicateGroup{{BLAKE3Hex: "H", Members: []string{"/a/keep.png", "/b/dup.png"}}},
		Uniques: []model.DiscoveredFile{{AbsolutePath: "/c/lone.png"}},
	})

	for _, it := range plan.Items {
		if it.Kind == model.ActionDelete && it.SrcPath != "/b/dup.png" {
			t.Fatalf("delete targeted something other than the duplicate: %s", it.SrcPath)
		}
		if it.SrcPath == "/a/keep.png" && it.Kind != model.ActionKeep {
			t.Fatalf("canonical must be Keep, got %s", it.Kind)
		}
		if it.SrcPath == "/c/lone.png" && it.Kind != model.ActionKeep {
			t.Fatalf("unique must be Keep, got %s", it.Kind)
		}
	}
}

// --- Invariants ----------------------------------------------------------

func TestInvariants_DeleteCanonicalRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = model.ModeDeleteDuplicates
	cfg.Roots = []string{"/"}

	plan := model.ActionPlan{Items: []model.ActionPlanItem{
		{Seq: 1, Kind: model.ActionDelete, SrcPath: "/a/canon.png"},
	}}
	canon := map[string]struct{}{"/a/canon.png": {}}
	if err := CheckInvariants(plan, canon, cfg); err == nil {
		t.Fatal("expected invariant violation")
	}
}

func TestInvariants_DuplicateDstRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Mode = model.ModeCopyUnique
	cfg.Roots = []string{t.TempDir()}
	cfg.Output.Dir = filepath.Join(t.TempDir(), "out")

	plan := model.ActionPlan{Items: []model.ActionPlanItem{
		{Seq: 1, Kind: model.ActionCopy, SrcPath: "/a/s", DstPath: filepath.Join(cfg.Output.Dir, "x.png")},
		{Seq: 2, Kind: model.ActionCopy, SrcPath: "/b/s", DstPath: filepath.Join(cfg.Output.Dir, "x.png")},
	}}
	if err := CheckInvariants(plan, map[string]struct{}{}, cfg); err == nil {
		t.Fatal("expected duplicate destination invariant")
	}
}

func TestInvariants_CopyDstInsideRootRejected(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Mode = model.ModeCopyUnique
	cfg.Roots = []string{root}
	cfg.Output.Dir = filepath.Join(root, "out")

	plan := model.ActionPlan{Items: []model.ActionPlanItem{
		{Seq: 1, Kind: model.ActionCopy, SrcPath: filepath.Join(root, "a"), DstPath: filepath.Join(root, "out", "x.png")},
	}}
	if err := CheckInvariants(plan, map[string]struct{}{}, cfg); err == nil {
		t.Fatal("expected 'dst inside root' violation")
	}
}

// --- Dry-run Executor ----------------------------------------------------

func TestExecute_DryRunLeavesFSUntouched(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s.png")
	dst := filepath.Join(dir, "out", "s.png")
	writeTmp(t, src, []byte{1, 2, 3})

	cfg := config.Default()
	cfg.Mode = model.ModeCopyUnique
	cfg.Roots = []string{dir}
	cfg.Output.Dir = filepath.Join(dir, "out")
	cfg.DryRun = true

	plan := model.ActionPlan{
		StartedAt: time.Now(),
		Items: []model.ActionPlanItem{{
			Seq: 1, Kind: model.ActionCopy, SrcPath: src, DstPath: dst,
		}},
	}
	logs, err := NewExecutor(cfg, nil, nil).Execute(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Status != model.StatusPlanned {
		t.Fatalf("want planned, got %+v", logs)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not touch fs; dst exists: %v", err)
	}
}

// --- Real Executor -------------------------------------------------------

func TestExecute_CopyUniqueWritesFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s.png")
	dst := filepath.Join(dir, "out", "s.png")
	writeTmp(t, src, []byte("content"))

	cfg := config.Default()
	cfg.Mode = model.ModeCopyUnique
	cfg.Roots = []string{dir}
	cfg.Output.Dir = filepath.Join(dir, "out")
	cfg.DryRun = false

	plan := model.ActionPlan{StartedAt: time.Now(), Items: []model.ActionPlanItem{{
		Seq: 1, Kind: model.ActionCopy, SrcPath: src, DstPath: dst,
	}}}
	logs, err := NewExecutor(cfg, nil, nil).Execute(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if logs[0].Status != model.StatusExecuted {
		t.Fatalf("expected executed, got %v: %s", logs[0].Status, logs[0].Message)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}
}

// --- Canonical stays untouched at execute time ---------------------------

func TestPlan_MoveDuplicates_KeepsCanonical(t *testing.T) {
	sels := map[string]model.CanonicalSelection{"H": {BLAKE3Hex: "H", Canonical: "/r/a.png"}}
	files := map[string]model.DiscoveredFile{
		"/r/a.png": {AbsolutePath: "/r/a.png", InputRoot: "/r", RelativePath: "a.png"},
		"/r/b.png": {AbsolutePath: "/r/b.png", InputRoot: "/r", RelativePath: "b.png"},
	}
	cfg := config.Default()
	cfg.Mode = model.ModeMoveDuplicates
	cfg.Roots = []string{"/r"}

	plan := mustPlan(t, PlanInput{
		Job:    model.ScanJob{StartedAt: time.Now()},
		Groups: []model.DuplicateGroup{{BLAKE3Hex: "H", Members: []string{"/r/a.png", "/r/b.png"}}},
		Sels:   sels, Files: files, Cfg: cfg,
	})
	for _, it := range plan.Items {
		if it.SrcPath == "/r/a.png" && it.Kind != model.ActionKeep {
			t.Fatalf("canonical should be Keep, got %s", it.Kind)
		}
		if it.SrcPath == "/r/b.png" && it.Kind != model.ActionMove {
			t.Fatalf("duplicate should be Move, got %s", it.Kind)
		}
	}
	if err := CheckInvariants(plan, canonicalSet(sels), cfg); err != nil {
		t.Fatalf("invariants failed: %v", err)
	}
}
