package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type scanCounters struct {
	entriesTotal  atomic.Int64
	dirsTotal     atomic.Int64
	filesTotal    atomic.Int64
	hiddenSkipped atomic.Int64
	matchedVideos atomic.Int64
}

func (c *scanCounters) snapshot() (entries, dirs, files, hidden, matched int) {
	return int(c.entriesTotal.Load()), int(c.dirsTotal.Load()), int(c.filesTotal.Load()),
		int(c.hiddenSkipped.Load()), int(c.matchedVideos.Load())
}

func scanDir(root string, opts Options) ([]string, dirScanStats, error) {
	stats := dirScanStats{root: root}

	infoStart := time.Now()
	info, err := os.Stat(root)
	if err != nil {
		return nil, stats, err
	}
	if !info.IsDir() {
		return nil, stats, fmt.Errorf("不是目录")
	}
	if opts.Perf != nil && opts.Perf.Enabled() {
		opts.Perf.StepSince(fmt.Sprintf("stat:%s", filepath.Base(root)), infoStart, root)
	}

	workers := ResolveScanWorkers(opts.ScanWorkers)

	walkStart := time.Now()
	files, counters, err := walkParallel(root, workers, opts)
	entries, dirs, filesTotal, hidden, matched := counters.snapshot()
	stats.entriesTotal = entries
	stats.dirsTotal = dirs
	stats.filesTotal = filesTotal
	stats.hiddenSkipped = hidden
	stats.matchedVideos = matched

	if opts.Perf != nil && opts.Perf.Enabled() {
		detail := fmt.Sprintf("entries=%d matched=%d workers=%d", stats.entriesTotal, stats.matchedVideos, workers)
		if opts.Cache != nil && opts.Cache.Enabled() {
			hits, misses, stale, _ := opts.Cache.Stats()
			detail += fmt.Sprintf(" cache_hit=%d cache_miss=%d cache_stale=%d", hits, misses, stale)
		}
		opts.Perf.StepSince(fmt.Sprintf("walk:%s", filepath.Base(root)), walkStart, detail)
	}

	return files, stats, err
}

func walkParallel(root string, workers int, opts Options) ([]string, *scanCounters, error) {
	counters := &scanCounters{}
	var (
		files   []string
		filesMu sync.Mutex
	)

	dirs := make(chan string, workers*4)
	var wg sync.WaitGroup

	visit := func(dir string) {
		defer wg.Done()

		if cached, ok := opts.Cache.Lookup(dir); ok {
			counters.entriesTotal.Add(int64(cached.EntryCount))
			counters.filesTotal.Add(int64(cached.FileCount))
			counters.hiddenSkipped.Add(int64(cached.HiddenSkip))
			counters.matchedVideos.Add(int64(len(cached.Files)))

			if len(cached.Files) > 0 {
				filesMu.Lock()
				files = append(files, cached.Files...)
				filesMu.Unlock()
			}

			for _, sub := range cached.Subdirs {
				counters.dirsTotal.Add(1)
				wg.Add(1)
				dirs <- filepath.Join(dir, sub)
			}
			return
		}

		if opts.Cache != nil && opts.Cache.Enabled() {
			opts.Cache.RecordMiss()
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			if opts.Perf != nil && opts.Perf.Enabled() {
				opts.Perf.Note(fmt.Sprintf("readdir 失败 %s: %v", dir, err))
			}
			return
		}

		info, err := os.Stat(dir)
		if err != nil {
			return
		}

		var (
			dirFiles   []string
			subdirs    []string
			entryCount int
			fileCount  int
			hiddenSkip int
		)

		for _, entry := range entries {
			entryCount++
			counters.entriesTotal.Add(1)
			if counters.entriesTotal.Load()%int64(walkProgressInterval) == 0 && opts.Perf != nil && opts.Perf.Enabled() {
				entriesTotal, _, _, _, matched := counters.snapshot()
				opts.Perf.Note(fmt.Sprintf(
					"walk 进度 %s: entries=%d matched=%d workers=%d",
					filepath.Base(root),
					entriesTotal,
					matched,
					workers,
				))
			}

			name := entry.Name()
			isDir := entryIsDir(entry)
			if opts.SkipHidden && isHiddenEntry(name) {
				hiddenSkip++
				counters.hiddenSkipped.Add(1)
				if isDir {
					continue
				}
				continue
			}

			path := filepath.Join(dir, name)
			if isDir {
				counters.dirsTotal.Add(1)
				subdirs = append(subdirs, name)
				wg.Add(1)
				dirs <- path
				continue
			}

			fileCount++
			counters.filesTotal.Add(1)
			ext := strings.ToLower(filepath.Ext(name))
			if !opts.Extensions[ext] {
				continue
			}

			counters.matchedVideos.Add(1)
			dirFiles = append(dirFiles, path)
			filesMu.Lock()
			files = append(files, path)
			filesMu.Unlock()
		}

		if opts.Cache != nil && opts.Cache.Enabled() {
			opts.Cache.Store(dir, dirCacheEntry{
				ModTime:    info.ModTime().UnixNano(),
				Files:      dirFiles,
				Subdirs:    subdirs,
				EntryCount: entryCount,
				FileCount:  fileCount,
				HiddenSkip: hiddenSkip,
			})
		}
	}

	for i := 0; i < workers; i++ {
		go func() {
			for dir := range dirs {
				visit(dir)
			}
		}()
	}

	wg.Add(1)
	dirs <- root
	wg.Wait()
	close(dirs)

	return files, counters, nil
}

func entryIsDir(entry fs.DirEntry) bool {
	if entry.Type() != 0 {
		return entry.Type().IsDir()
	}
	return entry.IsDir()
}
