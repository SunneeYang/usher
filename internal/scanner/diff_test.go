package scanner

import (
	"path/filepath"
	"testing"
)

func TestCompareVideos(t *testing.T) {
	previous := []string{"/a/1.mp4", "/a/2.mp4", "/b/old.mkv"}
	current := []string{"/a/1.mp4", "/a/2.mp4", "/b/new.mkv"}

	change := CompareVideos(previous, current)
	if !change.HasHistory {
		t.Fatal("expected history")
	}
	if len(change.Added) != 1 || change.Added[0] != "/b/new.mkv" {
		t.Fatalf("Added = %#v", change.Added)
	}
	if len(change.Removed) != 1 || change.Removed[0] != "/b/old.mkv" {
		t.Fatalf("Removed = %#v", change.Removed)
	}
	if change.Unchanged != 2 || change.Total != 3 {
		t.Fatalf("Unchanged=%d Total=%d", change.Unchanged, change.Total)
	}
}

func TestCompareVideosFirstRun(t *testing.T) {
	change := CompareVideos(nil, []string{"/a/1.mp4", "/a/2.mp4"})
	if change.HasHistory {
		t.Fatal("expected first run")
	}
	if change.Total != 2 {
		t.Fatalf("Total = %d", change.Total)
	}
}

func TestBuildSourceStat(t *testing.T) {
	root := "/Volumes/library-1"
	files := []string{
		filepath.Join(root, "Unsorted", "a.mp4"),
		filepath.Join(root, "Unsorted", "b.mp4"),
		filepath.Join(root, "Movies", "c.mkv"),
		filepath.Join(root, "root.mp4"),
	}
	stat := buildSourceStat(root, files, dirScanStats{dirsTotal: 3, filesTotal: 4})
	if stat.Videos != 4 || stat.Subdirs != 3 {
		t.Fatalf("stat = %+v", stat)
	}
	if len(stat.TopDirs) != 3 {
		t.Fatalf("TopDirs = %+v", stat.TopDirs)
	}
	if stat.TopDirs[0].Name != "Unsorted" || stat.TopDirs[0].Videos != 2 {
		t.Fatalf("TopDirs[0] = %+v", stat.TopDirs[0])
	}
}
