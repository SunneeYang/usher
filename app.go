package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/hcm-b0677/usher/internal/app"
	"github.com/hcm-b0677/usher/internal/config"
	"github.com/hcm-b0677/usher/internal/player"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type DesktopApp struct {
	ctx     context.Context
	service *app.Service
}

func NewDesktopApp() (*DesktopApp, error) {
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return &DesktopApp{service: app.NewService(configPath)}, nil
}

func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *DesktopApp) GetVersion() string {
	return a.service.GetVersion()
}

func (a *DesktopApp) GetConfigPath() string {
	return a.service.ConfigPath
}

func (a *DesktopApp) GetConfig() (app.ConfigDTO, error) {
	return a.service.GetConfig()
}

func (a *DesktopApp) SaveConfig(cfg app.ConfigDTO) error {
	return a.service.SaveConfig(cfg)
}

func (a *DesktopApp) RunScan(fresh bool) app.ScanResultDTO {
	return a.service.RunScan(fresh)
}

func (a *DesktopApp) GetPlayerOptions() []player.Option {
	return a.service.PlayerOptions()
}

func (a *DesktopApp) OpenPlaylist(path string) error {
	cfg, err := a.service.GetConfig()
	if err != nil {
		return err
	}
	target := path
	if target == "" {
		target = cfg.OutputFile
	}
	return a.service.OpenPlaylist(target, cfg.Player, cfg.PlayerApp)
}

func (a *DesktopApp) PickPlayerApp() (string, error) {
	filters := []wailsruntime.FileFilter{{DisplayName: "应用程序", Pattern: "*.app"}}
	if runtime.GOOS == "windows" {
		filters = []wailsruntime.FileFilter{{DisplayName: "可执行文件", Pattern: "*.exe"}}
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "选择播放器",
		Filters: filters,
	})
}

func (a *DesktopApp) PickSourceDir() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择视频目录",
	})
}

func (a *DesktopApp) PickOutputFile() (string, error) {
	return wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "保存播放列表",
		DefaultFilename: "playlist.m3u",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "M3U 播放列表", Pattern: "*.m3u"},
		},
	})
}

func (a *DesktopApp) RevealInFinder(path string) error {
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", filepath.Clean(path)).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}
