package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMTime_ReturnsUTCTruncated(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Force a known mtime.
	want := time.Date(2024, 1, 2, 3, 4, 5, 123_456_789, time.UTC)
	if err := os.Chtimes(p, want, want); err != nil {
		t.Fatal(err)
	}
	got, err := MTime(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != time.UTC {
		t.Errorf("expected UTC, got %v", got.Location())
	}
	if got != want.Truncate(time.Second) {
		t.Errorf("mtime not truncated to second: got %v, want %v", got, want.Truncate(time.Second))
	}
}

func TestMTime_Missing(t *testing.T) {
	if _, err := MTime(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
