package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCacheRoundTrip(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "movies")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	video := filepath.Join(sub, "a.mp4")
	if err := os.WriteFile(video, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cachePath := filepath.Join(root, "cache.json")
	exts := map[string]bool{".mp4": true}
	extKey := cacheExtKey(exts, true)
	srcKey := sourceKey([]string{sub})
	opts := Options{
		SourceDirs:    []string{sub},
		Extensions:    exts,
		SkipHidden:    true,
		Sort:          true,
		ScanCachePath: cachePath,
		CacheVerify:   "none",
	}

	result1, err := Collect(opts)
	if err != nil {
		t.Fatalf("first Collect() error = %v", err)
	}

	cache, err := LoadScanCache(cachePath, extKey, srcKey, "none", false)
	if err != nil {
		t.Fatal(err)
	}
	if cache.DirCount() == 0 {
		t.Fatal("expected cache entries after first scan")
	}

	result2, err := Collect(opts)
	if err != nil {
		t.Fatalf("second Collect() error = %v", err)
	}

	if len(result1.Videos) != 1 || len(result2.Videos) != 1 {
		t.Fatalf("videos=%#v %#v", result1.Videos, result2.Videos)
	}
	if result1.Videos[0] != video || result2.Videos[0] != video {
		t.Fatalf("unexpected paths: %#v %#v", result1.Videos, result2.Videos)
	}
	if !result2.Change.HasHistory || len(result2.Change.Added) != 0 || len(result2.Change.Removed) != 0 {
		t.Fatalf("second change = %+v", result2.Change)
	}

	result3, err := Collect(Options{
		SourceDirs:    []string{sub},
		Extensions:    exts,
		SkipHidden:    true,
		Sort:          true,
		ScanCachePath: cachePath,
		CacheVerify:   "none",
		FreshScan:     true,
	})
	if err != nil {
		t.Fatalf("fresh Collect() error = %v", err)
	}
	if len(result3.Videos) != 1 {
		t.Fatalf("fresh Collect() = %#v", result3.Videos)
	}
}

func TestCacheLookupNoneSkipsStat(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "movies")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	cache, err := LoadScanCache("", "key", "src", "none", false)
	if err != nil {
		t.Fatal(err)
	}
	cache.disabled = false
	cache.path = filepath.Join(root, "cache.json")
	cache.Store(dir, dirCacheEntry{ModTime: 1, Files: []string{"/fake/a.mp4"}})

	if _, ok := cache.Lookup(dir); !ok {
		t.Fatal("expected cache hit without caring about real mtime")
	}
}

func TestResolveScanWorkers(t *testing.T) {
	if got := ResolveScanWorkers(128); got != maxScanWorkers {
		t.Fatalf("ResolveScanWorkers(128) = %d, want %d", got, maxScanWorkers)
	}
	if got := ResolveScanWorkers(8); got != 8 {
		t.Fatalf("ResolveScanWorkers(8) = %d, want 8", got)
	}
}
