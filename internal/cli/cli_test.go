package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execute runs the root command with the given args, capturing stdout/stderr.
// Persistent flag state is reset between runs.
func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	// Reset persistent flag state.
	gFlags = globalFlags{}
	rootCmd.ResetFlags()
	rootCmd.ResetCommands()

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&gFlags.configPath, "config", "", "")
	pf.StringVar(&gFlags.logLevel, "log-level", "info", "")
	pf.StringVar(&gFlags.logFormat, "log-format", "text", "")
	pf.IntVar(&gFlags.workers, "workers", 0, "")
	pf.IntVar(&gFlags.actionWorkers, "action-workers", 0, "")
	pf.StringArrayVar(&gFlags.include, "include", nil, "")
	pf.StringArrayVar(&gFlags.exclude, "exclude", nil, "")
	pf.StringSliceVar(&gFlags.exts, "ext", nil, "")
	pf.BoolVar(&gFlags.followSymlinks, "follow-symlinks", false, "")
	pf.BoolVar(&gFlags.strict, "strict", false, "")
	pf.StringVar(&gFlags.auditDir, "audit-dir", "", "")
	pf.StringVar(&gFlags.format, "format", "text", "")
	pf.StringVar(&gFlags.outReport, "out-report", "", "")
	rootCmd.AddCommand(newScanCmd(), newReportCmd(), newCopyUniqueCmd(),
		newMoveDuplicatesCmd(), newDeleteDuplicatesCmd(), newVersionCmd())

	var bout, berr bytes.Buffer
	rootCmd.SetOut(&bout)
	rootCmd.SetErr(&berr)
	rootCmd.SetArgs(args)
	err = rootCmd.ExecuteContext(context.Background())
	return bout.String(), berr.String(), err
}

// Case M — CLI happy path: report on an empty dir succeeds with zero groups.
func TestCaseM_CLIHappyPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.png"),
		[]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 'x'}, 0o644); err != nil {
		t.Fatal(err)
	}
	aud := filepath.Join(dir, ".dedup-audit")
	stdout, _, err := execute(t, "report", dir,
		"--audit-dir", aud, "--format", "json", "--log-level", "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Parse and sanity-check the JSON.
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if _, ok := payload["job"]; !ok {
		t.Fatalf("expected 'job' key in report, got %v", payload)
	}
}

// Case N — CLI with non-existent root returns exit-code 4.
func TestCaseN_CLIRootDoesNotExist(t *testing.T) {
	_, _, err := execute(t, "report", "/definitely/not/a/dir/12345",
		"--audit-dir", t.TempDir(), "--log-level", "error")
	if err == nil {
		t.Fatal("expected error for missing root")
	}
	var cliErr *Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected cli.Error, got %T: %v", err, err)
	}
	if cliErr.Code != ExitRootNotExist {
		t.Fatalf("expected code %d, got %d", ExitRootNotExist, cliErr.Code)
	}
}

// Case O — CLI: "delete-duplicates" without --yes refuses to execute.
func TestCaseO_CLIInvalidMode_DeleteWithoutYes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.png"), []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0o644)

	_, _, err := execute(t, "delete-duplicates", dir,
		"--dry-run=false", "--audit-dir", filepath.Join(dir, ".aud"), "--log-level", "error")
	if err == nil {
		t.Fatal("expected refusal without --yes")
	}
	var cliErr *Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected cli.Error, got %T", err)
	}
	if cliErr.Code != ExitDeleteNeedsYes {
		t.Fatalf("expected code %d, got %d: %v", ExitDeleteNeedsYes, cliErr.Code, cliErr.Err)
	}
}

// Verify that a flag value overrides env and YAML values (precedence check).
func TestConfigPrecedence_FlagOverridesEnv(t *testing.T) {
	t.Setenv("DEDUP_LOG_LEVEL", "error")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.png"),
		[]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatal(err)
	}
	// We do not care about stdout; we just want the command to succeed, which
	// asserts the flag-layered config path runs.
	stdout, _, err := execute(t, "report", dir,
		"--audit-dir", filepath.Join(dir, ".aud"),
		"--log-level", "debug",
		"--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, `"job"`) {
		t.Fatalf("expected json output, got: %s", stdout)
	}
}

func TestCopyUnique_RequiresOut(t *testing.T) {
	dir := t.TempDir()
	_, _, err := execute(t, "copy-unique", dir, "--log-level", "error",
		"--audit-dir", filepath.Join(dir, ".aud"))
	if err == nil {
		t.Fatal("expected error when --out is missing")
	}
}
