package perf

import (
	"fmt"
	"time"
)

// Logger 输出分阶段性能日志，用于定位耗时瓶颈。
type Logger struct {
	enabled bool
	start   time.Time
}

func New(enabled bool) *Logger {
	return &Logger{
		enabled: enabled,
		start:   time.Now(),
	}
}

func (l *Logger) Enabled() bool {
	return l != nil && l.enabled
}

func (l *Logger) Step(label string, d time.Duration, detail string) {
	if !l.Enabled() {
		return
	}
	if detail == "" {
		fmt.Printf("⏱  [perf] %-28s %12s\n", label, formatDuration(d))
		return
	}
	fmt.Printf("⏱  [perf] %-28s %12s | %s\n", label, formatDuration(d), detail)
}

func (l *Logger) StepSince(label string, since time.Time, detail string) {
	l.Step(label, time.Since(since), detail)
}

func (l *Logger) Note(detail string) {
	if !l.Enabled() {
		return
	}
	fmt.Printf("   [perf] %s\n", detail)
}

func (l *Logger) Total(detail string) {
	l.Step("total", time.Since(l.start), detail)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%4dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%7.2fµs", float64(d.Microseconds()))
	case d < time.Second:
		return fmt.Sprintf("%7.2fms", float64(d.Microseconds())/1000)
	default:
		return fmt.Sprintf("%7.2fs", d.Seconds())
	}
}
