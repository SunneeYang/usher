package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `source_dirs:
  - /media/movies
video_extensions:
  - .mp4
  - mkv
output_file: out.m3u
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	exts := cfg.ExtensionSet()
	if !exts[".mp4"] || !exts[".mkv"] {
		t.Fatalf("ExtensionSet() = %#v", exts)
	}
}

func TestDefaultVideoExtensions(t *testing.T) {
	if len(DefaultVideoExtensions) < 20 {
		t.Fatalf("DefaultVideoExtensions too short: %d", len(DefaultVideoExtensions))
	}

	required := []string{".mp4", ".mkv", ".avi", ".ts", ".m2ts", ".iso", ".webm"}
	set := Default().ExtensionSet()
	for _, ext := range required {
		if !set[ext] {
			t.Fatalf("missing default extension %q", ext)
		}
	}
}

func TestLoadOrDefaultCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("LoadOrDefault() error = %v", err)
	}
	if cfg.OutputFile != "playlist.m3u" {
		t.Fatalf("cfg = %#v", cfg)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "empty source dirs", cfg: Config{OutputFile: "a.m3u", VideoExtensions: []string{".mp4"}}},
		{name: "empty output", cfg: Config{SourceDirs: []string{"/a"}, VideoExtensions: []string{".mp4"}}},
		{name: "empty extensions", cfg: Config{SourceDirs: []string{"/a"}, OutputFile: "a.m3u"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
