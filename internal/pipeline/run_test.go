package pipeline

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"deduplicator/internal/config"
	"deduplicator/internal/model"
)

func fixtureDir(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine source file location")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "test", "fixtures", rel)
	return filepath.Clean(root)
}

func TestPipeline_MixedTree_DeterministicAndCorrect(t *testing.T) {
	root := fixtureDir(t, "pipeline/mixed_tree")
	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Mode = model.ModeReport
	cfg.Audit.Dir = t.TempDir()

	in := Input{Cfg: cfg, Version: "test"}

	first, err := Run(context.Background(), in)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(first.Groups) != 2 {
		t.Fatalf("expected 2 duplicate groups (image+pdf), got %d", len(first.Groups))
	}
	if len(first.Uniques) != 1 {
		t.Fatalf("expected 1 unique, got %d: %+v", len(first.Uniques), first.Uniques)
	}

	// Re-run and compare deterministic fields.
	cfg2 := cfg
	cfg2.Audit.Dir = t.TempDir()
	second, err := Run(context.Background(), Input{Cfg: cfg2, Version: "test"})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	if len(second.Groups) != len(first.Groups) {
		t.Fatalf("non-deterministic group count")
	}
	for i := range first.Groups {
		if first.Groups[i].BLAKE3Hex != second.Groups[i].BLAKE3Hex {
			t.Errorf("group[%d] hash changed across runs: %s vs %s", i,
				first.Groups[i].BLAKE3Hex, second.Groups[i].BLAKE3Hex)
		}
	}
	for k, sel := range first.Selections {
		if second.Selections[k].Canonical != sel.Canonical {
			t.Errorf("canonical changed for %s: %s vs %s", k, sel.Canonical, second.Selections[k].Canonical)
		}
	}
}

func TestPipeline_CopyUnique_ProducesMappings(t *testing.T) {
	root := fixtureDir(t, "pipeline/mixed_tree")
	out := t.TempDir()

	cfg := config.Default()
	cfg.Roots = []string{root}
	cfg.Mode = model.ModeCopyUnique
	cfg.Output.Dir = out
	cfg.Audit.Dir = t.TempDir()
	cfg.DryRun = false

	res, err := Run(context.Background(), Input{Cfg: cfg, Version: "test"})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	// 1 unique + 2 canonicals = 3 mappings.
	if len(res.Mappings) != 3 {
		t.Fatalf("expected 3 output mappings, got %d", len(res.Mappings))
	}

	// Every dst must live under `out`.
	for _, m := range res.Mappings {
		if !hasPrefix(m.OutputPath, out) {
			t.Errorf("output path %s not under %s", m.OutputPath, out)
		}
	}
}

func hasPrefix(path, prefix string) bool {
	abs, _ := filepath.Abs(path)
	pabs, _ := filepath.Abs(prefix)
	return len(abs) >= len(pabs) && abs[:len(pabs)] == pabs
}
