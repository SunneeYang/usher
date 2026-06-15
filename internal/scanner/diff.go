package scanner

import (
	"fmt"
	"strings"
)

// VideoChange 描述本次扫描相对上次索引的视频变更。
type VideoChange struct {
	Added      []string
	Removed    []string
	Unchanged  int
	Total      int
	HasHistory bool
}

func (c VideoChange) Summary() string {
	if !c.HasHistory {
		return fmt.Sprintf("首次索引 %d 个视频", c.Total)
	}
	if len(c.Added) == 0 && len(c.Removed) == 0 {
		return fmt.Sprintf("无变化，共 %d 个视频", c.Total)
	}
	return fmt.Sprintf("+%d 新增, -%d 删除, %d 未变 (共 %d)",
		len(c.Added), len(c.Removed), c.Unchanged, c.Total)
}

func (c VideoChange) Detail(maxExamples int) string {
	if maxExamples <= 0 {
		maxExamples = 3
	}
	var parts []string
	if len(c.Added) > 0 {
		parts = append(parts, formatExamples("新增", c.Added, maxExamples))
	}
	if len(c.Removed) > 0 {
		parts = append(parts, formatExamples("删除", c.Removed, maxExamples))
	}
	return strings.Join(parts, " | ")
}

func formatExamples(label string, paths []string, max int) string {
	if len(paths) <= max {
		return fmt.Sprintf("%s: %s", label, strings.Join(shortPaths(paths), ", "))
	}
	shown := shortPaths(paths[:max])
	return fmt.Sprintf("%s: %s ...等 %d 个", label, strings.Join(shown, ", "), len(paths))
}

func shortPaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = shortenPath(p)
	}
	return out
}

func shortenPath(path string) string {
	if len(path) <= 60 {
		return path
	}
	return "..." + path[len(path)-57:]
}

func CompareVideos(previous, current []string) VideoChange {
	if len(previous) == 0 {
		return VideoChange{
			Added:      append([]string(nil), current...),
			Total:      len(current),
			HasHistory: false,
		}
	}

	prevSet := make(map[string]struct{}, len(previous))
	for _, p := range previous {
		prevSet[p] = struct{}{}
	}

	currSet := make(map[string]struct{}, len(current))
	for _, p := range current {
		currSet[p] = struct{}{}
	}

	change := VideoChange{
		Total:      len(current),
		HasHistory: true,
	}

	for p := range currSet {
		if _, ok := prevSet[p]; ok {
			change.Unchanged++
		} else {
			change.Added = append(change.Added, p)
		}
	}
	for p := range prevSet {
		if _, ok := currSet[p]; !ok {
			change.Removed = append(change.Removed, p)
		}
	}

	return change
}
