package player

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMissingFile(t *testing.T) {
	err := Open(filepath.Join(t.TempDir(), "missing.m3u"), "default", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestOpenExistingFileDefault(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skip opening apps in CI")
	}
	path := filepath.Join(t.TempDir(), "playlist.m3u")
	if err := os.WriteFile(path, []byte("#EXTM3U\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Open(path, "default", ""); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
}
