package scanner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// SourceStat 单个 source_dir 的扫描统计。
type SourceStat struct {
	Path    string
	Videos  int
	Subdirs int
	Files   int
	TopDirs []FolderStat
}

// FolderStat 源目录下一级子目录的视频数量。
type FolderStat struct {
	Name   string
	Videos int
}

func buildSourceStat(root string, files []string, scan dirScanStats) SourceStat {
	stat := SourceStat{
		Path:    root,
		Videos:  len(files),
		Subdirs: scan.dirsTotal,
		Files:   scan.filesTotal,
	}

	counts := make(map[string]int)
	rootClean := filepath.Clean(root)
	for _, file := range files {
		rel, err := filepath.Rel(rootClean, file)
		if err != nil || rel == "." {
			counts["."]++
			continue
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 1 {
			counts["."]++
		} else {
			counts[parts[0]]++
		}
	}

	stat.TopDirs = folderStatsFromCounts(counts)
	return stat
}

func folderStatsFromCounts(counts map[string]int) []FolderStat {
	stats := make([]FolderStat, 0, len(counts))
	for name, count := range counts {
		stats = append(stats, FolderStat{Name: name, Videos: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Videos != stats[j].Videos {
			return stats[i].Videos > stats[j].Videos
		}
		return stats[i].Name < stats[j].Name
	})
	return stats
}

func (s SourceStat) Label() string {
	return filepath.Base(s.Path)
}

func (s SourceStat) Summary() string {
	return fmt.Sprintf("%s: %d 个视频 (%d 子目录)", s.Label(), s.Videos, s.Subdirs)
}

func (s SourceStat) TopDirsDetail(max int) string {
	if len(s.TopDirs) == 0 || max <= 0 {
		return ""
	}
	if max > len(s.TopDirs) {
		max = len(s.TopDirs)
	}
	parts := make([]string, 0, max)
	for _, d := range s.TopDirs[:max] {
		label := d.Name
		if label == "." {
			label = "(根目录)"
		}
		parts = append(parts, fmt.Sprintf("%s=%d", label, d.Videos))
	}
	detail := strings.Join(parts, ", ")
	if len(s.TopDirs) > max {
		detail += fmt.Sprintf(" ...等 %d 个", len(s.TopDirs))
	}
	return detail
}
