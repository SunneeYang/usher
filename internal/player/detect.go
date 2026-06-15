package player

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type playerDef struct {
	ID          string
	Label       string
	darwinNames []string
	windowsExe  []string
	linuxBins   []string
}

var allPlayers = []playerDef{
	{
		ID:          "iina",
		Label:       "IINA",
		darwinNames: []string{"IINA"},
	},
	{
		ID:          "vlc",
		Label:       "VLC",
		darwinNames: []string{"VLC", "VLC media player"},
		windowsExe: []string{
			`C:\Program Files\VideoLAN\VLC\vlc.exe`,
			`C:\Program Files (x86)\VideoLAN\VLC\vlc.exe`,
		},
		linuxBins: []string{"vlc", "cvlc"},
	},
	{
		ID:          "infuse",
		Label:       "Infuse",
		darwinNames: []string{"Infuse"},
	},
	{
		ID:    "potplayer",
		Label: "PotPlayer",
		windowsExe: []string{
			`C:\Program Files\DAUM\PotPlayer\PotPlayerMini64.exe`,
			`C:\Program Files\DAUM\PotPlayer\PotPlayerMini.exe`,
			`C:\Program Files (x86)\DAUM\PotPlayer\PotPlayerMini64.exe`,
		},
	},
}

// InstalledOptions 返回本机已安装的播放器选项（含系统默认与自定义）。
func InstalledOptions() []Option {
	opts := []Option{{ID: "default", Label: "系统默认"}}
	for _, def := range allPlayers {
		if installed(def) {
			opts = append(opts, Option{ID: def.ID, Label: def.Label})
		}
	}
	opts = append(opts, Option{
		ID:    "custom",
		Label: customOptionLabel(),
	})
	return opts
}

func customOptionLabel() string {
	switch runtime.GOOS {
	case "windows":
		return "自定义程序..."
	default:
		return "自定义应用..."
	}
}

// Options 保留兼容；请优先使用 InstalledOptions。
func Options() []Option {
	return InstalledOptions()
}

func installed(def playerDef) bool {
	switch runtime.GOOS {
	case "darwin":
		for _, name := range def.darwinNames {
			if darwinAppExists(name) {
				return true
			}
		}
	case "windows":
		for _, p := range def.windowsExe {
			if fileExists(p) {
				return true
			}
		}
		if len(def.linuxBins) > 0 {
			for _, bin := range def.linuxBins {
				if commandExists(bin) {
					return true
				}
			}
		}
	case "linux":
		for _, bin := range def.linuxBins {
			if commandExists(bin) {
				return true
			}
		}
	}
	return false
}

func darwinAppExists(name string) bool {
	candidates := []string{
		filepath.Join("/Applications", name+".app"),
		filepath.Join("/System/Applications", name+".app"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Applications", name+".app"))
	}
	for _, path := range candidates {
		if appBundleExists(path) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func appBundleExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() && strings.HasSuffix(path, ".app")
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// IsAvailable 检查播放器 ID 是否可用；custom 需传入应用路径。
func IsAvailable(playerID, customApp string) bool {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	switch playerID {
	case "", "default":
		return true
	case "custom":
		path := strings.TrimSpace(customApp)
		if path == "" {
			return false
		}
		if runtime.GOOS == "darwin" {
			return appBundleExists(path) || fileExists(path)
		}
		return fileExists(path)
	default:
		for _, def := range allPlayers {
			if def.ID == playerID {
				return installed(def)
			}
		}
		return false
	}
}

// SanitizeSelection 将不可用的播放器选择回退到 default。
func SanitizeSelection(playerID, customApp string) (string, string) {
	playerID = strings.ToLower(strings.TrimSpace(playerID))
	if playerID == "" {
		return "default", ""
	}
	if IsAvailable(playerID, customApp) {
		return playerID, customApp
	}
	if playerID == "custom" {
		return "default", ""
	}
	return "default", customApp
}
