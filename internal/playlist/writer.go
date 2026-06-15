package playlist

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hcm-b0677/usher/internal/perf"
)

const writeProgressInterval = 5_000

func Write(outputPath string, videoFiles []string, log *perf.Logger) error {
	writeStart := time.Now()

	mkdirStart := time.Now()
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录: %w", err)
	}
	if log != nil && log.Enabled() {
		log.StepSince("write:mkdir", mkdirStart, filepath.Dir(outputPath))
	}

	createStart := time.Now()
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件: %w", err)
	}
	defer file.Close()
	if log != nil && log.Enabled() {
		log.StepSince("write:create_file", createStart, outputPath)
	}

	writer := bufio.NewWriterSize(file, 256*1024)

	headerStart := time.Now()
	if _, err := writer.WriteString("#EXTM3U\n"); err != nil {
		return fmt.Errorf("写入 m3u 头: %w", err)
	}
	if log != nil && log.Enabled() {
		log.StepSince("write:header", headerStart, "")
	}

	entriesStart := time.Now()
	var bytesWritten int64
	for i, path := range videoFiles {
		line := fmt.Sprintf("#EXTINF:-1,%s\n%s\n", filepath.Base(path), path)
		n, err := writer.WriteString(line)
		if err != nil {
			return fmt.Errorf("写入条目 %s: %w", path, err)
		}
		bytesWritten += int64(n)

		if log != nil && log.Enabled() && (i+1)%writeProgressInterval == 0 {
			log.Note(fmt.Sprintf(
				"write 进度: entries=%d/%d bytes=%d elapsed=%s",
				i+1,
				len(videoFiles),
				bytesWritten,
				time.Since(entriesStart).Round(time.Millisecond),
			))
		}
	}

	flushStart := time.Now()
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新输出缓冲: %w", err)
	}
	if log != nil && log.Enabled() {
		log.StepSince("write:flush", flushStart, "")
		log.StepSince(
			"write:entries",
			entriesStart,
			fmt.Sprintf("count=%d bytes=%d", len(videoFiles), bytesWritten),
		)
		log.StepSince("write:total", writeStart, outputPath)
	}

	return nil
}
