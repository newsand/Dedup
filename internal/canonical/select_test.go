package canonical

import (
	"testing"
	"time"

	"deduplicator/internal/model"
)

func df(path string, mtime time.Time) model.DiscoveredFile {
	return model.DiscoveredFile{AbsolutePath: path, MTime: mtime}
}

// Case D — canonical by oldest mtime.
func TestCaseD_CanonicalByOldestMTime(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC) // older

	in := Input{
		Group: model.DuplicateGroup{
			BLAKE3Hex: "deadbeef",
			Members:   []string{"/a/img.png", "/b/img_copy.png"},
		},
		Files: map[string]model.DiscoveredFile{
			"/a/img.png":      df("/a/img.png", t1),
			"/b/img_copy.png": df("/b/img_copy.png", t2),
		},
	}
	sel, err := Select(in)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Canonical != "/b/img_copy.png" {
		t.Fatalf("expected /b/img_copy.png as canonical, got %s", sel.Canonical)
	}
	if !sel.Reason.ByMTime || sel.Reason.ByLexical {
		t.Fatalf("expected ByMTime=true, ByLexical=false; got %+v", sel.Reason)
	}
}

// Case D2 — tie-break by lexical path.
func TestCaseD2_CanonicalTieByLexical(t *testing.T) {
	tm := time.Date(2024, 5, 5, 12, 0, 0, 0, time.UTC)
	in := Input{
		Group: model.DuplicateGroup{
			BLAKE3Hex: "cafef00d",
			Members:   []string{"/z/last.png", "/a/first.png", "/m/middle.png"},
		},
		Files: map[string]model.DiscoveredFile{
			"/z/last.png":   df("/z/last.png", tm),
			"/a/first.png":  df("/a/first.png", tm),
			"/m/middle.png": df("/m/middle.png", tm),
		},
	}
	sel, err := Select(in)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Canonical != "/a/first.png" {
		t.Fatalf("expected /a/first.png, got %s", sel.Canonical)
	}
	if sel.Reason.ByMTime || !sel.Reason.ByLexical {
		t.Fatalf("expected ByMTime=false, ByLexical=true; got %+v", sel.Reason)
	}
}

func TestSelect_EmptyGroupErrors(t *testing.T) {
	_, err := Select(Input{Group: model.DuplicateGroup{BLAKE3Hex: "x"}})
	if err == nil {
		t.Fatalf("expected error for empty group")
	}
}

func TestSelect_IsDeterministic(t *testing.T) {
	tm := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	in := Input{
		Group: model.DuplicateGroup{
			BLAKE3Hex: "x",
			Members:   []string{"/b", "/a", "/c"},
		},
		Files: map[string]model.DiscoveredFile{
			"/a": df("/a", tm), "/b": df("/b", tm), "/c": df("/c", tm),
		},
	}
	first, _ := Select(in)
	for i := 0; i < 10; i++ {
		s, _ := Select(in)
		if s.Canonical != first.Canonical {
			t.Fatalf("non-deterministic result: %s vs %s", s.Canonical, first.Canonical)
		}
	}
	if first.Canonical != "/a" {
		t.Fatalf("expected /a, got %s", first.Canonical)
	}
}
