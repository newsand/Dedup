package filetype

import (
	"os"
	"path/filepath"
	"testing"

	"deduplicator/internal/model"
)

func TestClassifyExtension(t *testing.T) {
	cases := map[string]model.FileType{
		".jpg":  model.FileTypeImage,
		".jpeg": model.FileTypeImage,
		".png":  model.FileTypeImage,
		".gif":  model.FileTypeImage,
		".webp": model.FileTypeImage,
		".tif":  model.FileTypeImage,
		".pdf":  model.FileTypePDF,
		".txt":  model.FileTypeUnknown,
		"":      model.FileTypeUnknown,
	}
	for ext, want := range cases {
		if got := ClassifyExtension(ext); got != want {
			t.Errorf("ClassifyExtension(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestDetect_SniffPNG(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mislabelled.dat")
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 'r', 'e', 's', 't'}
	if err := os.WriteFile(p, png, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(p, ".dat"); got != model.FileTypeImage {
		t.Fatalf("expected Image via sniff, got %q", got)
	}
}

func TestDetect_SniffPDF(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.bin")
	if err := os.WriteFile(p, []byte("%PDF-1.4\nrest"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(p, ".bin"); got != model.FileTypePDF {
		t.Fatalf("expected PDF via sniff, got %q", got)
	}
}

func TestDetect_UnknownContentStaysUnknown(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "weird")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(p, ""); got != model.FileTypeUnknown {
		t.Fatalf("expected Unknown, got %q", got)
	}
}

func TestDetect_ExtensionWinsWhenKnown(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fake.jpg")
	if err := os.WriteFile(p, []byte("not really a jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Detect(p, ".jpg"); got != model.FileTypeImage {
		t.Fatalf("extension should short-circuit the sniff; got %q", got)
	}
}
