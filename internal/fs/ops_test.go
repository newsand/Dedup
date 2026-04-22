package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopyFile_Basic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.bin")
	dst := filepath.Join(dir, "out", "a.bin")
	writeTmp(t, src, []byte("hello world"))

	copied, err := CopyFile(src, dst)
	if err != nil || !copied {
		t.Fatalf("want copied=true err=nil; got %v/%v", copied, err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "hello world" {
		t.Fatalf("content mismatch: %q", got)
	}
}

func TestCopyFile_ExistingIdenticalIsSkipped(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s")
	dst := filepath.Join(dir, "d")
	writeTmp(t, src, []byte("same"))
	writeTmp(t, dst, []byte("same"))

	copied, err := CopyFile(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if copied {
		t.Fatal("expected copied=false when dst already has identical bytes")
	}
}

func TestCopyFile_ExistingDifferentFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s")
	dst := filepath.Join(dir, "d")
	writeTmp(t, src, []byte("hello"))
	writeTmp(t, dst, []byte("world"))

	_, err := CopyFile(src, dst)
	if !errors.Is(err, ErrDestExistsMismatch) {
		t.Fatalf("expected ErrDestExistsMismatch, got %v", err)
	}
}

func TestMove_Basic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "s")
	dst := filepath.Join(dir, "d", "s")
	writeTmp(t, src, []byte("x"))

	moved, err := Move(src, dst)
	if err != nil || !moved {
		t.Fatalf("want moved=true err=nil; got %v/%v", moved, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("src should no longer exist")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal("dst should exist")
	}
}

func TestDelete_Basic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "g")
	writeTmp(t, p, []byte("bye"))
	if err := Delete(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
}

func TestDelete_RefusesDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := Delete(dir); err == nil {
		t.Fatal("expected error when deleting directory")
	}
}
