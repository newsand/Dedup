package scan

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"deduplicator/internal/filetype"
	"deduplicator/internal/model"
)

func writeFile(t *testing.T, p string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalk_DiscoversAndSortsFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "b", "z.jpg"), []byte{0xFF, 0xD8, 0xFF, 0xE0})
	writeFile(t, filepath.Join(root, "a", "a.png"), []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	writeFile(t, filepath.Join(root, "note.txt"), []byte("hi"))

	files, errs := Walk(context.Background(), Options{
		Roots:  []string{root},
		Detect: filetype.Detect,
		IncludeTypes: map[model.FileType]bool{
			model.FileTypeImage: true,
			model.FileTypePDF:   true,
		},
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 2 {
		t.Fatalf("want 2 files, got %d: %+v", len(files), files)
	}

	// Deterministic sort by absolute path.
	if !sort.SliceIsSorted(files, func(i, j int) bool { return files[i].AbsolutePath < files[j].AbsolutePath }) {
		t.Fatalf("files not sorted deterministically: %+v", files)
	}

	for _, f := range files {
		if f.InputRoot == "" || f.RelativePath == "" {
			t.Errorf("expected input_root/relative_path populated: %+v", f)
		}
		if f.FileType == model.FileTypeUnknown {
			t.Errorf("expected known file type, got unknown: %+v", f)
		}
	}
}

func TestWalk_ExcludeFilter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep.png"), []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	writeFile(t, filepath.Join(root, "skip.png"), []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})

	files, _ := Walk(context.Background(), Options{
		Roots:   []string{root},
		Detect:  filetype.Detect,
		Filters: NewFilters(nil, []string{"skip.png"}),
	})
	if len(files) != 1 {
		t.Fatalf("expected exclude to drop skip.png, got %d files", len(files))
	}
	if filepath.Base(files[0].AbsolutePath) != "keep.png" {
		t.Fatalf("unexpected survivor: %s", files[0].AbsolutePath)
	}
}

func TestWalk_ContextCancelStopsWalk(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(root, "f", string(rune('a'+i))+".png"),
			[]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, errs := Walk(ctx, Options{Roots: []string{root}, Detect: filetype.Detect})
	if len(errs) == 0 {
		t.Fatalf("expected cancellation to surface as an error")
	}
}
