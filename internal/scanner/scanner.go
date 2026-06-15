package scanner

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/hcm-b0677/usher/internal/perf"
)

const walkProgressInterval = 10_000

type Options struct {
	SourceDirs    []string
	Extensions    map[string]bool
	SkipHidden    bool
	Sort          bool
	Shuffle       bool
	ShuffleSeed   int64
	ScanWorkers   int
	ScanCachePath string
	CacheVerify   string
	FreshScan     bool
	Cache         *ScanCache
	Perf          *perf.Logger
}

type Result struct {
	Videos  []string
	Change  VideoChange
	Sources []SourceStat
}

type dirScanStats struct {
	root          string
	entriesTotal  int
	dirsTotal     int
	filesTotal    int
	hiddenSkipped int
	matchedVideos int
	duration      time.Duration
}

func Collect(opts Options) (Result, error) {
	if len(opts.SourceDirs) == 0 {
		return Result{}, fmt.Errorf("source dirs 不能为空")
	}

	srcKey := sourceKey(opts.SourceDirs)

	if opts.ScanCachePath != "" && opts.Cache == nil {
		cacheStart := time.Now()
		cache, err := LoadScanCache(
			opts.ScanCachePath,
			cacheExtKey(opts.Extensions, opts.SkipHidden),
			srcKey,
			opts.CacheVerify,
			opts.FreshScan,
		)
		if err != nil {
			return Result{}, err
		}
		opts.Cache = cache
		if opts.Perf != nil && opts.Perf.Enabled() {
			opts.Perf.StepSince(
				"cache:load",
				cacheStart,
				fmt.Sprintf("dirs=%d videos=%d path=%s verify=%s fresh=%v",
					cache.DirCount(), len(cache.LastVideos()), opts.ScanCachePath, opts.CacheVerify, opts.FreshScan),
			)
		}
		defer func() {
			if opts.Cache == nil || !opts.Cache.Enabled() {
				return
			}
			saveStart := time.Now()
			if err := opts.Cache.Save(); err != nil {
				fmt.Printf("⚠️  [usher] 保存扫描缓存失败: %v\n", err)
				return
			}
			if opts.Perf != nil && opts.Perf.Enabled() {
				hits, misses, stale, updated := opts.Cache.Stats()
				opts.Perf.StepSince(
					"cache:save",
					saveStart,
					fmt.Sprintf("hit=%d miss=%d stale=%d updated=%d path=%s", hits, misses, stale, updated, opts.ScanCachePath),
				)
			}
		}()
	}

	collectStart := time.Now()
	if opts.Perf != nil && opts.Perf.Enabled() {
		workers := ResolveScanWorkers(opts.ScanWorkers)
		cacheNote := "cache=off"
		if opts.Cache != nil && opts.Cache.Enabled() {
			cacheNote = fmt.Sprintf("cache=%s verify=%s", opts.ScanCachePath, opts.CacheVerify)
		}
		opts.Perf.Note(fmt.Sprintf(
			"开始并发扫描 %d 个源目录 (scan_workers=%d %s)",
			len(opts.SourceDirs),
			workers,
			cacheNote,
		))
	}

	var (
		files      []string
		sourceStats []SourceStat
		mu         sync.Mutex
		wg         sync.WaitGroup
	)

	for _, dir := range opts.SourceDirs {
		wg.Add(1)
		go func(targetDir string) {
			defer wg.Done()

			dirStart := time.Now()
			localFiles, stats, err := scanDir(targetDir, opts)
			stats.duration = time.Since(dirStart)

			if err != nil {
				if opts.Perf != nil && opts.Perf.Enabled() {
					opts.Perf.Note(fmt.Sprintf("目录 %s 扫描失败: %v (耗时 %s)", targetDir, err, stats.duration))
				}
				fmt.Printf("⚠️  [usher] 扫描目录失败 %s: %v\n", targetDir, err)
				return
			}

			sourceStat := buildSourceStat(targetDir, localFiles, stats)

			if opts.Perf != nil && opts.Perf.Enabled() {
				detail := fmt.Sprintf(
					"videos=%d subdirs=%d files=%d path=%s",
					sourceStat.Videos,
					sourceStat.Subdirs,
					sourceStat.Files,
					targetDir,
				)
				if top := sourceStat.TopDirsDetail(8); top != "" {
					detail += " | top: " + top
				}
				opts.Perf.Step(
					fmt.Sprintf("source:%s", filepath.Base(targetDir)),
					stats.duration,
					detail,
				)
				opts.Perf.Step(
					fmt.Sprintf("scan:%s", filepath.Base(targetDir)),
					stats.duration,
					fmt.Sprintf(
						"entries=%d dirs=%d files=%d hidden_skip=%d matched=%d path=%s",
						stats.entriesTotal,
						stats.dirsTotal,
						stats.filesTotal,
						stats.hiddenSkipped,
						stats.matchedVideos,
						targetDir,
					),
				)
			}

			mu.Lock()
			files = append(files, localFiles...)
			sourceStats = append(sourceStats, sourceStat)
			mu.Unlock()
		}(dir)
	}

	wg.Wait()

	sort.Slice(sourceStats, func(i, j int) bool {
		return sourceStats[i].Path < sourceStats[j].Path
	})

	if opts.Perf != nil && opts.Perf.Enabled() {
		opts.Perf.StepSince("scan:wait_all_dirs", collectStart, fmt.Sprintf("raw_matches=%d", len(files)))
	}

	dedupeStart := time.Now()
	beforeDedupe := len(files)
	files = dedupe(files)
	if opts.Perf != nil && opts.Perf.Enabled() {
		opts.Perf.StepSince(
			"dedupe",
			dedupeStart,
			fmt.Sprintf("before=%d after=%d removed=%d", beforeDedupe, len(files), beforeDedupe-len(files)),
		)
	}

	if opts.Sort && !opts.Shuffle {
		sortStart := time.Now()
		sort.Strings(files)
		if opts.Perf != nil && opts.Perf.Enabled() {
			opts.Perf.StepSince("sort", sortStart, fmt.Sprintf("count=%d", len(files)))
		}
	}

	if opts.Shuffle && len(files) > 1 {
		shuffleStart := time.Now()
		shuffleFiles(files, opts.ShuffleSeed)
		if opts.Perf != nil && opts.Perf.Enabled() {
			opts.Perf.StepSince("shuffle", shuffleStart, fmt.Sprintf("count=%d", len(files)))
		}
	}

	var previous []string
	if opts.Cache != nil && opts.Cache.Enabled() {
		previous = opts.Cache.LastVideos()
	}
	change := CompareVideos(previous, files)
	sort.Strings(change.Added)
	sort.Strings(change.Removed)

	if opts.Perf != nil && opts.Perf.Enabled() {
		opts.Perf.StepSince("videos:diff", collectStart, change.Summary())
		if detail := change.Detail(3); detail != "" {
			opts.Perf.Note(detail)
		}
	}

	if opts.Cache != nil && opts.Cache.Enabled() {
		opts.Cache.SetLastVideos(files)
	}

	if opts.Perf != nil && opts.Perf.Enabled() {
		opts.Perf.StepSince("collect_total", collectStart, fmt.Sprintf("final_count=%d", len(files)))
	}

	return Result{Videos: files, Change: change, Sources: sourceStats}, nil
}

func isHiddenEntry(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

func dedupe(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}
