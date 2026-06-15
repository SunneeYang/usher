package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hcm-b0677/usher/internal/app"
	"github.com/hcm-b0677/usher/internal/config"
	"github.com/hcm-b0677/usher/internal/perf"
	"github.com/hcm-b0677/usher/internal/playlist"
	"github.com/hcm-b0677/usher/internal/scanner"
)

func main() {
	showVersion := flag.Bool("version", false, "显示版本信息")
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	perfLog := flag.Bool("perf", false, "输出详细性能日志")
	freshScan := flag.Bool("fresh", false, "忽略扫描缓存，全量重新扫描")
	flag.Parse()

	if *showVersion {
		fmt.Printf("usher v%s\n", app.Version)
		return
	}

	fmt.Printf("╭────────────────────────────────╮\n")
	fmt.Printf("│  usher v%s                  │\n", app.Version)
	fmt.Printf("│  Your NAS video playlist guide │\n")
	fmt.Printf("╰────────────────────────────────╯\n")

	startTime := time.Now()
	perfLogger := perf.New(*perfLog)

	cfgStart := time.Now()
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
	perfLogger.StepSince("config:load", cfgStart, fmt.Sprintf(
		"dirs=%d extensions=%d shuffle=%v sort=%v skip_hidden=%v scan_workers=%d scan_cache=%q cache_verify=%s",
		len(cfg.SourceDirs),
		len(cfg.VideoExtensions),
		cfg.Shuffle,
		cfg.Sort,
		cfg.SkipHidden,
		cfg.ScanWorkers,
		cfg.ScanCache,
		cfg.CacheVerifyMode(),
	))

	if cfg.ScanCache != "" && !*freshScan {
		fmt.Printf("📦 [usher] 扫描缓存已启用: %s (verify=%s)\n", cfg.ScanCache, cfg.CacheVerifyMode())
	} else if *freshScan {
		fmt.Println("🔄 [usher] 全量扫描模式 (-fresh)，忽略缓存")
	} else {
		fmt.Println("ℹ️  [usher] 扫描缓存未启用，可在 config.yaml 设置 scan_cache")
	}

	if cfg.Shuffle {
		fmt.Println("🎲 [usher] 正在对视频列表进行洗牌打乱...")
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
		FreshScan:     *freshScan,
		Perf:          perfLogger,
	})
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	printSourceStats(result.Sources)
	printVideoChange(result.Change)

	videoFiles := result.Videos
	if len(videoFiles) == 0 {
		fmt.Println("ℹ️  [usher] 未找到匹配的视频文件。")
		return
	}

	fmt.Printf("💾 [usher] 正在写入播放列表 -> %s\n", cfg.OutputFile)
	if err := playlist.Write(cfg.OutputFile, videoFiles, perfLogger); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	perfLogger.Total(fmt.Sprintf("videos=%d output=%s", len(videoFiles), cfg.OutputFile))
	fmt.Printf("✨ [usher] 领位完成！共索引 %d 个视频 | 耗时: %v\n", len(videoFiles), time.Since(startTime))
}

func printSourceStats(sources []scanner.SourceStat) {
	for _, s := range sources {
		fmt.Printf("📁 [usher] %s\n", s.Summary())
		if detail := s.TopDirsDetail(5); detail != "" {
			fmt.Printf("   一级子目录: %s\n", detail)
		}
	}
}

func printVideoChange(change scanner.VideoChange) {
	if !change.HasHistory {
		fmt.Printf("📊 [usher] %s\n", change.Summary())
		return
	}
	fmt.Printf("📊 [usher] 视频变更: %s\n", change.Summary())
	if detail := change.Detail(3); detail != "" {
		fmt.Printf("   %s\n", detail)
	}
}
