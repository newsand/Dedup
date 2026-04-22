package naming

import (
	"path/filepath"
	"testing"

	"deduplicator/internal/model"
)

// Case A — flatten normal.
func TestCaseA_FlattenNormal(t *testing.T) {
	got := Flatten("clientes/2025/evento A/foto 01.png", Rules{})
	want := "clientes_2025_evento_a_foto_01.png"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestFlatten_LowercasesExtension(t *testing.T) {
	got := Flatten("a/B.JPG", Rules{})
	want := "a_b.jpg"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestFlatten_DropsDiacritics(t *testing.T) {
	got := Flatten("Ção/arquivo-ímpar.png", Rules{})
	want := "cao_arquivo_impar.png"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

// Case B — suppressname.
func TestCaseB_SuppressName(t *testing.T) {
	files := []model.DiscoveredFile{
		{AbsolutePath: "/root/img3/sub pasta/desc 1029.png", RelativePath: "img3/sub pasta/desc 1029.png"},
		{AbsolutePath: "/root/img3/sub pasta/desc 2048.png", RelativePath: "img3/sub pasta/desc 2048.png"},
	}
	out := Build(files, "/out", Rules{Suppress: true})
	if len(out) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(out))
	}
	got1 := out[files[0].AbsolutePath].OutputName
	got2 := out[files[1].AbsolutePath].OutputName
	if got1 != "img3_sub_pasta_1.png" || got2 != "img3_sub_pasta_2.png" {
		t.Fatalf("suppress output wrong: %q, %q", got1, got2)
	}
}

// Case L — collision between two files flattening to the same base.
func TestCaseL_FlattenCollisionSuffix(t *testing.T) {
	files := []model.DiscoveredFile{
		{AbsolutePath: "/root/a-b/c.png", RelativePath: "a-b/c.png"},
		{AbsolutePath: "/root/a/b-c.png", RelativePath: "a/b-c.png"},
	}
	out := Build(files, "/out", Rules{})
	first := out["/root/a-b/c.png"].OutputName
	second := out["/root/a/b-c.png"].OutputName

	// Lexical order of absolute paths: /root/a-b/c.png < /root/a/b-c.png
	// (because '-' < '/' in ASCII)
	if first != "a_b_c.png" {
		t.Fatalf("expected first=a_b_c.png, got %q", first)
	}
	if second != "a_b_c_1.png" {
		t.Fatalf("expected second=a_b_c_1.png, got %q", second)
	}
}

func TestBuild_IsDeterministic(t *testing.T) {
	files := []model.DiscoveredFile{
		{AbsolutePath: "/b.png", RelativePath: "b.png"},
		{AbsolutePath: "/a/B.png", RelativePath: "a/B.png"},
		{AbsolutePath: "/a/b.png", RelativePath: "a/b.png"}, // collides with the one above (lowercased)
	}
	first := Build(files, "/out", Rules{})
	for i := 0; i < 5; i++ {
		again := Build(files, "/out", Rules{})
		for k, v := range first {
			if again[k].OutputName != v.OutputName {
				t.Fatalf("non-deterministic for %s: %s vs %s", k, again[k].OutputName, v.OutputName)
			}
		}
	}
}

func TestBuild_OutputPathJoinedWithDir(t *testing.T) {
	files := []model.DiscoveredFile{
		{AbsolutePath: "/x/y.png", RelativePath: "y.png"},
	}
	out := Build(files, filepath.Join(string(filepath.Separator), "tmp", "out"), Rules{})
	want := filepath.Join(string(filepath.Separator), "tmp", "out", "y.png")
	if got := out["/x/y.png"].OutputPath; got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestToSnake_EdgeCases(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"---":            "",
		" a  b ":         "a_b",
		"NO WAY":         "no_way",
		"ÁÉÍÓÚ":          "aeiou",
		"Mixed-123.ABC":  "mixed_123_abc",
	}
	for in, want := range cases {
		if got := ToSnake(in); got != want {
			t.Errorf("ToSnake(%q) = %q, want %q", in, got, want)
		}
	}
}
