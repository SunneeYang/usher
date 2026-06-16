package app

import (
	"fmt"
	"time"

	"github.com/hcm-b0677/usher/internal/config"
	"github.com/hcm-b0677/usher/internal/perf"
	"github.com/hcm-b0677/usher/internal/player"
	"github.com/hcm-b0677/usher/internal/playlist"
	"github.com/hcm-b0677/usher/internal/scanner"
)

const Version = "0.2.0"

type Service struct {
	ConfigPath string
}

type ConfigDTO struct {
	SourceDirs  []string `json:"sourceDirs"`
	OutputFile  string   `json:"outputFile"`
	Shuffle     bool     `json:"shuffle"`
	Sort        bool     `json:"sort"`
	SkipHidden  bool     `json:"skipHidden"`
	ScanWorkers int      `json:"scanWorkers"`
	ScanCache     string `json:"scanCache"`
	CacheVerify   string `json:"cacheVerify"`
	Player        string `json:"player"`
	PlayerApp     string `json:"playerApp"`
	OpenAfterScan bool   `json:"openAfterScan"`
}

type FolderDTO struct {
	Name   string `json:"name"`
	Videos int    `json:"videos"`
}

type SourceDTO struct {
	Label   string      `json:"label"`
	Path    string      `json:"path"`
	Videos  int         `json:"videos"`
	Subdirs int         `json:"subdirs"`
	TopDirs []FolderDTO `json:"topDirs"`
}

type ChangeDTO struct {
	Added      []string `json:"added"`
	Removed    []string `json:"removed"`
	Unchanged  int      `json:"unchanged"`
	Total      int      `json:"total"`
	HasHistory bool     `json:"hasHistory"`
	Summary    string   `json:"summary"`
}

type ScanResultDTO struct {
	Success    bool        `json:"success"`
	Error      string      `json:"error,omitempty"`
	VideoCount int         `json:"videoCount"`
	OutputFile string      `json:"outputFile"`
	DurationMs int64       `json:"durationMs"`
	Change     ChangeDTO   `json:"change"`
	Sources    []SourceDTO `json:"sources"`
}

func NewService(configPath string) *Service {
	return &Service{ConfigPath: configPath}
}

func (s *Service) GetVersion() string {
	return Version
}

func (s *Service) GetConfig() (ConfigDTO, error) {
	cfg, err := config.LoadOrDefault(s.ConfigPath)
	if err != nil {
		return ConfigDTO{}, err
	}
	dto := toConfigDTO(cfg)
	dto.Player, dto.PlayerApp = player.SanitizeSelection(dto.Player, dto.PlayerApp)
	return dto, nil
}

func (s *Service) SaveConfig(dto ConfigDTO) error {
	dto.Player, dto.PlayerApp = player.SanitizeSelection(dto.Player, dto.PlayerApp)
	cfg := fromConfigDTO(dto)
	return config.Save(s.ConfigPath, cfg)
}

func (s *Service) RunScan(fresh bool) ScanResultDTO {
	start := time.Now()
	cfg, err := config.LoadOrDefault(s.ConfigPath)
	if err != nil {
		return ScanResultDTO{Success: false, Error: err.Error()}
	}
	if err := cfg.Validate(); err != nil {
		return ScanResultDTO{Success: false, Error: err.Error()}
	}

	result, err := scanner.Collect(scanner.Options{
		SourceDirs:    cfg.SourceDirs,
		Extensions:    cfg.ExtensionSet(),
		SkipHidden:    cfg.SkipHidden,
		Sort:          cfg.Sort,
		Shuffle:       cfg.Shuffle,
		ScanWorkers:   cfg.ScanWorkers,
		ScanCachePath: cfg.ScanCache,
		CacheVerify:   cfg.CacheVerifyMode(),
		FreshScan:     fresh,
		Perf:          perf.New(false),
	})
	if err != nil {
		return ScanResultDTO{Success: false, Error: err.Error()}
	}

	if len(result.Videos) == 0 {
		return ScanResultDTO{
			Success:    true,
			VideoCount: 0,
			OutputFile: cfg.OutputFile,
			DurationMs: time.Since(start).Milliseconds(),
			Change:     toChangeDTO(result.Change),
			Sources:    toSourceDTOs(result.Sources),
		}
	}

	if err := playlist.Write(cfg.OutputFile, result.Videos, perf.New(false)); err != nil {
		return ScanResultDTO{Success: false, Error: err.Error()}
	}

	output := ScanResultDTO{
		Success:    true,
		VideoCount: len(result.Videos),
		OutputFile: cfg.OutputFile,
		DurationMs: time.Since(start).Milliseconds(),
		Change:     toChangeDTO(result.Change),
		Sources:    toSourceDTOs(result.Sources),
	}

	if cfg.OpenAfterScan {
		if !player.IsAvailable(cfg.Player, cfg.PlayerApp) {
			output.Error = "播放列表已生成，但所选播放器未安装"
		} else if err := s.OpenPlaylist(cfg.OutputFile, cfg.Player, cfg.PlayerApp); err != nil {
			output.Error = "播放列表已生成，但打开播放器失败: " + err.Error()
		}
	}

	return output
}

func (s *Service) OpenPlaylist(playlistPath, playerID, customApp string) error {
	playerID, customApp = player.SanitizeSelection(playerID, customApp)
	if !player.IsAvailable(playerID, customApp) {
		return fmt.Errorf("所选播放器未安装")
	}
	return player.Open(playlistPath, playerID, customApp)
}

func (s *Service) PlayerOptions() []player.Option {
	return player.InstalledOptions()
}

func toConfigDTO(cfg config.Config) ConfigDTO {
	return ConfigDTO{
		SourceDirs:  cloneStrings(cfg.SourceDirs),
		OutputFile:  cfg.OutputFile,
		Shuffle:     cfg.Shuffle,
		Sort:        cfg.Sort,
		SkipHidden:  cfg.SkipHidden,
		ScanWorkers: cfg.ScanWorkers,
		ScanCache:     cfg.ScanCache,
		CacheVerify:   cfg.CacheVerifyMode(),
		Player:        cfg.Player,
		PlayerApp:     cfg.PlayerApp,
		OpenAfterScan: cfg.OpenAfterScan,
	}
}

func fromConfigDTO(dto ConfigDTO) config.Config {
	cfg := config.Default()
	cfg.SourceDirs = cloneStrings(dto.SourceDirs)
	if dto.OutputFile != "" {
		cfg.OutputFile = dto.OutputFile
	}
	cfg.Shuffle = dto.Shuffle
	cfg.Sort = dto.Sort
	cfg.SkipHidden = dto.SkipHidden
	if dto.ScanWorkers > 0 {
		cfg.ScanWorkers = dto.ScanWorkers
	}
	if dto.ScanCache != "" {
		cfg.ScanCache = dto.ScanCache
	}
	if dto.CacheVerify != "" {
		cfg.CacheVerify = dto.CacheVerify
	}
	if dto.Player != "" {
		cfg.Player = dto.Player
	}
	cfg.PlayerApp = dto.PlayerApp
	cfg.OpenAfterScan = dto.OpenAfterScan
	return cfg
}

func toChangeDTO(change scanner.VideoChange) ChangeDTO {
	return ChangeDTO{
		Added:      cloneStrings(change.Added),
		Removed:    cloneStrings(change.Removed),
		Unchanged:  change.Unchanged,
		Total:      change.Total,
		HasHistory: change.HasHistory,
		Summary:    change.Summary(),
	}
}

func toSourceDTOs(sources []scanner.SourceStat) []SourceDTO {
	if len(sources) == 0 {
		return []SourceDTO{}
	}
	out := make([]SourceDTO, 0, len(sources))
	for _, s := range sources {
		top := make([]FolderDTO, 0, len(s.TopDirs))
		for _, d := range s.TopDirs {
			name := d.Name
			if name == "." {
				name = "(根目录)"
			}
			top = append(top, FolderDTO{Name: name, Videos: d.Videos})
		}
		out = append(out, SourceDTO{
			Label:   s.Label(),
			Path:    s.Path,
			Videos:  s.Videos,
			Subdirs: s.Subdirs,
			TopDirs: top,
		})
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func (s *Service) ConfigPathDisplay() string {
	return fmt.Sprintf("配置: %s", s.ConfigPath)
}
