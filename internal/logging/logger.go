package logging

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

var (
	mu           sync.RWMutex
	currentLevel = levelInfo
)

func Init(level string) {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	currentLevel = parseLevel(level)
}

func Debug(msg string, kv ...any) {
	logAt(levelDebug, "DEBUG", msg, kv...)
}

func Info(msg string, kv ...any) {
	logAt(levelInfo, "INFO", msg, kv...)
}

func Warn(msg string, kv ...any) {
	logAt(levelWarn, "WARN", msg, kv...)
}

func Error(msg string, kv ...any) {
	logAt(levelError, "ERROR", msg, kv...)
}

func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel <= levelDebug
}

func parseLevel(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return levelDebug
	case "warn":
		return levelWarn
	case "error":
		return levelError
	case "", "info":
		return levelInfo
	default:
		log.Printf("WARN invalid log level %q, defaulting to info", level)
		return levelInfo
	}
}

func logAt(level int, label, msg string, kv ...any) {
	mu.RLock()
	enabled := level >= currentLevel
	mu.RUnlock()
	if !enabled {
		return
	}

	log.Printf("%s %s%s", label, msg, formatKV(kv...))
}

func formatKV(kv ...any) string {
	if len(kv) == 0 {
		return ""
	}

	var b strings.Builder
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		var value any = "<missing>"
		if i+1 < len(kv) {
			value = kv[i+1]
		}
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(fmt.Sprintf("%v", value))
	}

	return b.String()
}
