package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	defaultScanWorkers = 32
	maxScanWorkers     = 64
)

// ResolveScanWorkers 返回实际 worker 数；配置为 0 时按 CPU 自动估算。
func ResolveScanWorkers(configured int) int {
	if configured > 0 {
		if configured > maxScanWorkers {
			return maxScanWorkers
		}
		return configured
	}

	workers := runtime.NumCPU() * 4
	if workers < defaultScanWorkers {
		workers = defaultScanWorkers
	}
	if workers > maxScanWorkers {
		workers = maxScanWorkers
	}
	return workers
}

func cacheExtKey(extensions map[string]bool, skipHidden bool) string {
	exts := make([]string, 0, len(extensions))
	for ext := range extensions {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	h := sha256.New()
	h.Write([]byte(strings.Join(exts, ",")))
	if skipHidden {
		h.Write([]byte{1})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

type dirCacheEntry struct {
	ModTime    int64    `json:"mod_time"`
	Files      []string `json:"files"`
	Subdirs    []string `json:"subdirs"`
	EntryCount int      `json:"entry_count"`
	FileCount  int      `json:"file_count"`
	HiddenSkip int      `json:"hidden_skip"`
}

type scanCacheFile struct {
	ExtKey     string                   `json:"ext_key"`
	SourceKey  string                   `json:"source_key,omitempty"`
	Dirs       map[string]dirCacheEntry `json:"dirs"`
	LastVideos []string                 `json:"last_videos,omitempty"`
}

type ScanCache struct {
	path       string
	extKey     string
	sourceKey  string
	verify     string
	mu         sync.RWMutex
	dirs       map[string]dirCacheEntry
	lastVideos []string
	hits       atomic.Int64
	misses     atomic.Int64
	stale      atomic.Int64
	updated    atomic.Int64
	disabled   bool
}

func LoadScanCache(path, extKey, sourceKey, verify string, fresh bool) (*ScanCache, error) {
	c := &ScanCache{
		path:      path,
		extKey:    extKey,
		sourceKey: sourceKey,
		verify:    verify,
		dirs:      make(map[string]dirCacheEntry),
	}
	if path == "" {
		c.disabled = true
		return c, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("读取扫描缓存: %w", err)
	}

	var file scanCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("解析扫描缓存: %w", err)
	}
	if file.ExtKey != extKey {
		return c, nil
	}
	if file.SourceKey == sourceKey && len(file.LastVideos) > 0 {
		c.lastVideos = file.LastVideos
	}
	if fresh {
		return c, nil
	}
	if file.Dirs != nil {
		c.dirs = file.Dirs
	}
	return c, nil
}

func (c *ScanCache) Enabled() bool {
	return c != nil && !c.disabled
}

func (c *ScanCache) Lookup(dir string) (dirCacheEntry, bool) {
	if !c.Enabled() {
		return dirCacheEntry{}, false
	}

	c.mu.RLock()
	entry, ok := c.dirs[dir]
	c.mu.RUnlock()
	if !ok {
		return dirCacheEntry{}, false
	}

	if c.verify == "mtime" {
		info, err := os.Stat(dir)
		if err != nil || entry.ModTime != info.ModTime().UnixNano() {
			c.stale.Add(1)
			return dirCacheEntry{}, false
		}
	}

	c.hits.Add(1)
	return entry, true
}

func (c *ScanCache) Store(dir string, entry dirCacheEntry) {
	if !c.Enabled() {
		return
	}
	c.mu.Lock()
	c.dirs[dir] = entry
	c.mu.Unlock()
	c.updated.Add(1)
}

func (c *ScanCache) RecordMiss() {
	if c.Enabled() {
		c.misses.Add(1)
	}
}

func (c *ScanCache) DirCount() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.dirs)
}

func (c *ScanCache) LastVideos() []string {
	if c == nil || len(c.lastVideos) == 0 {
		return nil
	}
	return append([]string(nil), c.lastVideos...)
}

func (c *ScanCache) SetLastVideos(videos []string) {
	if !c.Enabled() {
		return
	}
	c.mu.Lock()
	c.lastVideos = append([]string(nil), videos...)
	c.mu.Unlock()
}

func (c *ScanCache) Stats() (hits, misses, stale, updated int64) {
	if c == nil {
		return 0, 0, 0, 0
	}
	return c.hits.Load(), c.misses.Load(), c.stale.Load(), c.updated.Load()
}

func (c *ScanCache) Save() error {
	if !c.Enabled() {
		return nil
	}

	c.mu.RLock()
	file := scanCacheFile{
		ExtKey:     c.extKey,
		SourceKey:  c.sourceKey,
		Dirs:       c.dirs,
		LastVideos: append([]string(nil), c.lastVideos...),
	}
	c.mu.RUnlock()

	data, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("序列化扫描缓存: %w", err)
	}
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("写入扫描缓存: %w", err)
	}
	return nil
}
