package config

import (
	"os"
	"path/filepath"
	"testing"

	"deduplicator/internal/model"
)

func writeYAML(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "dedup.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadYAML_ParsesKnownFields(t *testing.T) {
	dir := t.TempDir()
	p := writeYAML(t, dir, `
roots: ["/a", "/b"]
mode: copy-unique
dry_run: false
output:
  dir: /tmp/out
  preserve_timestamps: true
logging:
  level: debug
  format: json
filters:
  ext: [".png", ".jpg"]
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != model.ModeCopyUnique {
		t.Errorf("mode=%q", cfg.Mode)
	}
	if cfg.DryRun {
		t.Errorf("expected dry_run=false")
	}
	if cfg.Output.Dir != "/tmp/out" {
		t.Errorf("output.dir=%q", cfg.Output.Dir)
	}
	if cfg.Logging.Level != "debug" || cfg.Logging.Format != "json" {
		t.Errorf("logging=%+v", cfg.Logging)
	}
}

func TestApplyEnv_OverridesYAML(t *testing.T) {
	t.Setenv("DEDUP_MODE", "move-duplicates")
	t.Setenv("DEDUP_LOG_LEVEL", "error")
	t.Setenv("DEDUP_WORKERS", "8")

	cfg := Default()
	cfg.Mode = model.ModeReport
	cfg.Logging.Level = "info"
	cfg.Concurrency.Workers = 0

	cfg = ApplyEnv(cfg)
	if cfg.Mode != model.ModeMoveDuplicates {
		t.Errorf("env did not override mode: got %q", cfg.Mode)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("env did not override log level: got %q", cfg.Logging.Level)
	}
	if cfg.Concurrency.Workers != 8 {
		t.Errorf("env did not override workers: got %d", cfg.Concurrency.Workers)
	}
}

func TestValidate_RejectsCopyUniqueWithoutOutput(t *testing.T) {
	cfg := Default()
	cfg.Roots = []string{"/x"}
	cfg.Mode = model.ModeCopyUnique
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when copy-unique has no output.dir")
	}
}

func TestValidate_RejectsOutputInsideRoot(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "out")
	cfg := Default()
	cfg.Roots = []string{root}
	cfg.Mode = model.ModeCopyUnique
	cfg.Output.Dir = out
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when output inside root")
	}
}

func TestValidate_AcceptsValidConfig(t *testing.T) {
	cfg := Default()
	cfg.Roots = []string{t.TempDir()}
	cfg.Mode = model.ModeReport
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
