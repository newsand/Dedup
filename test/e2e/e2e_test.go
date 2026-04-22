//go:build e2e

// Package e2e runs the compiled `dedup` binary against the fixtures.
// Enable with: go test -tags=e2e ./test/e2e/...
package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func binary(t *testing.T) string {
	t.Helper()
	exe := "dedup"
	if runtime.GOOS == "windows" {
		exe = "dedup.exe"
	}
	// Prefer ./bin/dedup from the repo root.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	bin := filepath.Join(repoRoot, "bin", exe)
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("binary not built at %s (run `make build`): %v", bin, err)
	}
	return bin
}

func fixture(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "test", "fixtures", rel))
}

func runDedup(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	cmd := exec.Command(binary(t), args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		t.Logf("stderr:\n%s", errb.String())
	}
	return out.String(), err
}

func TestE2E_ReportMixedTree(t *testing.T) {
	audit := t.TempDir()
	out, err := runDedup(t,
		"report", fixture(t, "pipeline/mixed_tree"),
		"--audit-dir", audit, "--format", "json", "--log-level", "error",
	)
	if err != nil {
		t.Fatalf("dedup report failed: %v\n%s", err, out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	groups, _ := payload["groups"].([]any)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestE2E_DeleteWithoutYesFailsExit6(t *testing.T) {
	cmd := exec.Command(binary(t),
		"delete-duplicates", fixture(t, "pipeline/mixed_tree"),
		"--dry-run=false", "--audit-dir", t.TempDir(), "--log-level", "error",
	)
	err := cmd.Run()
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if ee.ExitCode() != 6 {
		t.Fatalf("expected exit code 6, got %d", ee.ExitCode())
	}
}
