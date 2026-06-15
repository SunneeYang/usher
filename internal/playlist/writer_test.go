package playlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out", "playlist.m3u")
	files := []string{
		filepath.Join(dir, "a.mp4"),
		filepath.Join(dir, "b.mkv"),
	}

	if err := Write(output, files, nil); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "#EXTM3U\n") {
		t.Fatalf("missing header: %q", content)
	}
	for _, file := range files {
		if !strings.Contains(content, file) {
			t.Fatalf("missing path %q in %q", file, content)
		}
	}
}
