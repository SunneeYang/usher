package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDirs      []string `yaml:"source_dirs"`
	VideoExtensions []string `yaml:"video_extensions"`
	OutputFile      string   `yaml:"output_file"`
	Shuffle         bool     `yaml:"shuffle"`
	Sort            bool     `yaml:"sort"`
	SkipHidden      bool     `yaml:"skip_hidden"`
	ScanWorkers     int      `yaml:"scan_workers"`
	ScanCache       string   `yaml:"scan_cache"`
	CacheVerify     string   `yaml:"cache_verify"`
	Player          string   `yaml:"player"`
	PlayerApp       string   `yaml:"player_app"`
	OpenAfterScan   bool     `yaml:"open_after_scan"`
}

func Default() Config {
	return Config{
		SourceDirs:      []string{},
		VideoExtensions: append([]string(nil), DefaultVideoExtensions...),
		OutputFile:      "playlist.m3u",
		Sort:            true,
		SkipHidden:      true,
		ScanWorkers:     32,
		ScanCache:       ".usher-scan-cache.json",
		CacheVerify:     "none",
		Player:          "default",
		OpenAfterScan:   false,
	}
}

func normalizeCacheVerify(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none":
		return "none"
	case "mtime":
		return "mtime"
	default:
		return "none"
	}
}

func (c *Config) CacheVerifyMode() string {
	return normalizeCacheVerify(c.CacheVerify)
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置文件: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置文件: %w", err)
	}

	cfg.resolvePaths(path)
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if len(c.SourceDirs) == 0 {
		return fmt.Errorf("source_dirs 不能为空")
	}
	return c.ValidateForSave()
}

func (c Config) ValidateForSave() error {
	if strings.TrimSpace(c.OutputFile) == "" {
		return fmt.Errorf("output_file 不能为空")
	}
	if len(c.VideoExtensions) == 0 {
		return fmt.Errorf("video_extensions 不能为空")
	}
	return nil
}

func (c *Config) resolvePaths(configPath string) {
	if c.ScanCache == "" {
		return
	}
	if filepath.IsAbs(c.ScanCache) {
		return
	}
	c.ScanCache = filepath.Join(filepath.Dir(configPath), c.ScanCache)
}

func (c Config) ExtensionSet() map[string]bool {
	extMap := make(map[string]bool, len(c.VideoExtensions))
	for _, ext := range c.VideoExtensions {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[ext] = true
	}
	return extMap
}
