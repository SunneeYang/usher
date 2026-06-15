package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Option 可选播放器。
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Open 使用指定播放器打开 M3U 播放列表。
func Open(playlistPath, playerID, customApp string) error {
	playlistPath = strings.TrimSpace(playlistPath)
	if playlistPath == "" {
		return fmt.Errorf("播放列表路径为空")
	}
	if _, err := os.Stat(playlistPath); err != nil {
		return fmt.Errorf("播放列表不存在: %w", err)
	}

	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" {
		playerID = "default"
	}

	switch runtime.GOOS {
	case "darwin":
		return openDarwin(playlistPath, playerID, customApp)
	case "windows":
		return openWindows(playlistPath, playerID, customApp)
	default:
		return openLinux(playlistPath, playerID, customApp)
	}
}

func openDarwin(path, playerID, customApp string) error {
	switch playerID {
	case "default":
		return exec.Command("open", path).Start()
	case "custom":
		app := strings.TrimSpace(customApp)
		if app == "" {
			return fmt.Errorf("请先选择自定义播放器")
		}
		return exec.Command("open", "-a", app, path).Start()
	default:
		appName, ok := map[string]string{
			"iina":   "IINA",
			"vlc":    "VLC",
			"infuse": "Infuse",
		}[playerID]
		if !ok {
			return fmt.Errorf("不支持的播放器: %s", playerID)
		}
		return exec.Command("open", "-a", appName, path).Start()
	}
}

func openWindows(path, playerID, customApp string) error {
	path = filepath.Clean(path)
	switch playerID {
	case "default":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	case "custom":
		app := strings.TrimSpace(customApp)
		if app == "" {
			return fmt.Errorf("请先选择自定义播放器")
		}
		return exec.Command(app, path).Start()
	case "vlc":
		return exec.Command("cmd", "/c", "start", "", "vlc", path).Start()
	case "potplayer":
		return exec.Command("cmd", "/c", "start", "", "PotPlayerMini64", path).Start()
	default:
		return fmt.Errorf("不支持的播放器: %s", playerID)
	}
}

func openLinux(path, playerID, customApp string) error {
	switch playerID {
	case "default":
		return exec.Command("xdg-open", path).Start()
	case "custom":
		app := strings.TrimSpace(customApp)
		if app == "" {
			return fmt.Errorf("请先选择自定义播放器")
		}
		return exec.Command(app, path).Start()
	case "vlc":
		return exec.Command("vlc", path).Start()
	default:
		return fmt.Errorf("不支持的播放器: %s", playerID)
	}
}
