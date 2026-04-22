package hash

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashReader_EmptyInputHasKnownDigest(t *testing.T) {
	// BLAKE3 of the empty message is a well-known constant.
	const empty = "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
	got, n, err := HashReader(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0 bytes, got %d", n)
	}
	if got != empty {
		t.Fatalf("digest mismatch: got %s", got)
	}
}

func TestFileBLAKE3_Determinism(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.bin")
	data := []byte(strings.Repeat("deduplicator\n", 5000))
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}

	h1, n1, err := FileBLAKE3(p)
	if err != nil {
		t.Fatal(err)
	}
	h2, n2, err := FileBLAKE3(p)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash not deterministic: %s vs %s", h1, h2)
	}
	if n1 != int64(len(data)) || n2 != n1 {
		t.Fatalf("size mismatch: want %d, got %d / %d", len(data), n1, n2)
	}
}

func TestHashReader_MatchesFileBLAKE3(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.bin")
	data := []byte("the quick brown fox jumps over the lazy dog")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}

	hf, _, err := FileBLAKE3(p)
	if err != nil {
		t.Fatal(err)
	}
	hr, _, err := HashReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if hf != hr {
		t.Fatalf("FileBLAKE3 and HashReader disagree: %s vs %s", hf, hr)
	}
}
