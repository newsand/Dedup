package compare

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"deduplicator/internal/model"
)

// --- helpers ---------------------------------------------------------------

var (
	pngMagic = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	pdfMagic = []byte("%PDF-1.4\n")
)

func writeAt(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func discovered(path string, ft model.FileType, size int64) model.DiscoveredFile {
	return model.DiscoveredFile{
		AbsolutePath: path,
		InputRoot:    filepath.Dir(path),
		RelativePath: filepath.Base(path),
		FileType:     ft,
		Extension:    filepath.Ext(path),
		SizeBytes:    size,
		MTime:        time.Now().UTC().Truncate(time.Second),
	}
}

func readSize(t *testing.T, p string) int64 {
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

func defaultDelegate() *Delegate {
	return NewDelegate(
		NewRegistry(NewImageComparator(), NewPDFComparator()),
		2, false, nil,
	)
}

// --- Case E: Image exact duplicate ----------------------------------------

func TestCaseE_ImageExactDuplicate(t *testing.T) {
	dir := t.TempDir()
	body := append(append([]byte{}, pngMagic...), []byte("payload-identical")...)
	a := filepath.Join(dir, "a", "one.png")
	b := filepath.Join(dir, "b", "two.png")
	writeAt(t, a, body)
	writeAt(t, b, body)

	d := defaultDelegate()
	res, err := d.Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypeImage, readSize(t, a)),
		discovered(b, model.FileTypeImage, readSize(t, b)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(res.Groups))
	}
	if len(res.Groups[0].Members) != 2 {
		t.Fatalf("expected 2 members, got %v", res.Groups[0].Members)
	}
	if len(res.Uniques) != 0 {
		t.Fatalf("expected no uniques, got %d", len(res.Uniques))
	}
}

// --- Case F: Same name, different content ---------------------------------

func TestCaseF_SameNameDifferentContent(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "aaa", "photo.png")
	b := filepath.Join(dir, "bbb", "photo.png")
	writeAt(t, a, append(append([]byte{}, pngMagic...), []byte("alpha")...))
	writeAt(t, b, append(append([]byte{}, pngMagic...), []byte("beta-different")...))

	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypeImage, readSize(t, a)),
		discovered(b, model.FileTypeImage, readSize(t, b)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 0 {
		t.Fatalf("expected no duplicate groups, got %d", len(res.Groups))
	}
	if len(res.Uniques) != 2 {
		t.Fatalf("expected 2 uniques, got %d", len(res.Uniques))
	}
}

// --- Case G: Same size, different hash ------------------------------------

func TestCaseG_SameSizeDifferentHash(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "x.png")
	b := filepath.Join(dir, "y.png")
	// Both 16 bytes but different content.
	writeAt(t, a, []byte{0x89, 'P', 'N', 'G', 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})
	writeAt(t, b, []byte{0x89, 'P', 'N', 'G', 255, 254, 253, 252, 251, 250, 249, 248, 247, 246, 245, 244})

	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypeImage, 16),
		discovered(b, model.FileTypeImage, 16),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 0 {
		t.Fatalf("expected no duplicate groups, got %d: %+v", len(res.Groups), res.Groups)
	}
	if len(res.Uniques) != 2 {
		t.Fatalf("expected 2 uniques, got %d", len(res.Uniques))
	}
}

// --- Case H: PDF exact duplicate ------------------------------------------

func TestCaseH_PDFExactDuplicate(t *testing.T) {
	dir := t.TempDir()
	body := append(append([]byte{}, pdfMagic...), []byte("...content...")...)
	a := filepath.Join(dir, "docs", "a.pdf")
	b := filepath.Join(dir, "other", "b.pdf")
	writeAt(t, a, body)
	writeAt(t, b, body)

	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypePDF, readSize(t, a)),
		discovered(b, model.FileTypePDF, readSize(t, b)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 PDF group, got %d", len(res.Groups))
	}
}

// --- Case I: PDFs with identical bytes but different filesystem metadata ---
// (mtime/owner should never affect duplicity; we verify that here.)

func TestCaseI_PDFSameBytesDifferentMetadata(t *testing.T) {
	dir := t.TempDir()
	body := append(append([]byte{}, pdfMagic...), []byte("same bytes")...)
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	writeAt(t, a, body)
	writeAt(t, b, body)

	// Make mtimes purposefully different.
	if err := os.Chtimes(a, time.Now(), time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(b, time.Now(), time.Date(2024, 6, 6, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypePDF, readSize(t, a)),
		discovered(b, model.FileTypePDF, readSize(t, b)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 1 {
		t.Fatalf("mtime differences must not affect grouping; got %d groups", len(res.Groups))
	}
}

// --- Case J: PDF same name, different bytes -------------------------------

func TestCaseJ_PDFSameNameDifferentBytes(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "folderA", "report.pdf")
	b := filepath.Join(dir, "folderB", "report.pdf")
	writeAt(t, a, append(append([]byte{}, pdfMagic...), []byte("v1")...))
	writeAt(t, b, append(append([]byte{}, pdfMagic...), []byte("version 2 with more content")...))

	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypePDF, readSize(t, a)),
		discovered(b, model.FileTypePDF, readSize(t, b)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) != 0 {
		t.Fatalf("expected no groups, got %+v", res.Groups)
	}
	if len(res.Uniques) != 2 {
		t.Fatalf("expected 2 uniques, got %d", len(res.Uniques))
	}
}

// --- Delegate dispatch ----------------------------------------------------

func TestDelegate_IgnoresUnsupportedTypes(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.xyz")
	writeAt(t, a, []byte("whatever"))
	res, err := defaultDelegate().Run(context.Background(), []model.DiscoveredFile{
		discovered(a, model.FileTypeUnknown, readSize(t, a)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Ignored) != 1 {
		t.Fatalf("expected 1 ignored, got %d", len(res.Ignored))
	}
	if len(res.Groups)+len(res.Uniques) != 0 {
		t.Fatalf("unsupported types must not appear as uniques or dups")
	}
}

// --- PreSize --------------------------------------------------------------

func TestPreSize_FiltersSingletonsBySize(t *testing.T) {
	files := []model.DiscoveredFile{
		{AbsolutePath: "/a", SizeBytes: 10},
		{AbsolutePath: "/b", SizeBytes: 10},
		{AbsolutePath: "/c", SizeBytes: 20}, // singleton by size
		{AbsolutePath: "/d", SizeBytes: 30},
		{AbsolutePath: "/e", SizeBytes: 30},
		{AbsolutePath: "/f", SizeBytes: 30},
	}
	cand, single := PreSize(files)
	if len(cand) != 5 {
		t.Fatalf("expected 5 candidates, got %d", len(cand))
	}
	if len(single) != 1 || single[0].AbsolutePath != "/c" {
		t.Fatalf("expected only /c as singleton, got %+v", single)
	}
}
