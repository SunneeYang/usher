package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollect(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "movies")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	files := []string{
		filepath.Join(sub, "a.mp4"),
		filepath.Join(sub, "b.mkv"),
		filepath.Join(sub, "ignore.txt"),
		filepath.Join(sub, ".hidden.mp4"),
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Collect(Options{
		SourceDirs: []string{sub},
		Extensions: map[string]bool{".mp4": true, ".mkv": true},
		SkipHidden: true,
		Sort:       true,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	got := result.Videos
	if len(got) != 2 {
		t.Fatalf("Collect() len = %d, want 2", len(got))
	}
	if got[0] != files[0] || got[1] != files[1] {
		t.Fatalf("Collect() = %#v", got)
	}
	if len(result.Sources) != 1 || result.Sources[0].Videos != 2 {
		t.Fatalf("Sources = %#v", result.Sources)
	}
}

func TestDedupe(t *testing.T) {
	input := []string{"/a/b.mp4", "/a/b.mp4", "/a/c.mp4"}
	got := dedupe(input)
	if len(got) != 2 {
		t.Fatalf("dedupe() len = %d, want 2", len(got))
	}
}
