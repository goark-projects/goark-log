package goarklog

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	LevelTrace slog.Level = -8
)

var defaultLevelRegistry = newDefaultLevelRegistry()

// LevelRegistry 保存日志级别名称和数值的双向映射。
type LevelRegistry struct {
	mu      sync.RWMutex
	byName  map[string]slog.Level
	byValue map[slog.Level]string
}

// NewLevelRegistry 创建包含内置级别的注册表。
func NewLevelRegistry() *LevelRegistry {
	return newDefaultLevelRegistry()
}

// DefaultLevelRegistry 返回进程默认级别注册表。
func DefaultLevelRegistry() *LevelRegistry {
	return defaultLevelRegistry
}

// RegisterLevel 向默认注册表注册自定义级别。
func RegisterLevel(name string, level slog.Level) error {
	return defaultLevelRegistry.Register(name, level)
}

// ParseLevel 解析日志级别名称。
func ParseLevel(value string) (slog.Level, error) {
	return defaultLevelRegistry.Parse(value)
}

// LevelName 返回级别名称，优先返回已注册的精确名称。
func LevelName(level slog.Level) string {
	return defaultLevelRegistry.Name(level)
}

func levelName(level slog.Level) string {
	return LevelName(level)
}

// Register 注册级别名称和数值。
func (r *LevelRegistry) Register(name string, level slog.Level) error {
	if r == nil {
		return fmt.Errorf("goark-log: level registry is nil")
	}
	normalized, err := normalizeLevelName(name)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byName[normalized]; ok && existing != level {
		return fmt.Errorf("goark-log: log level %q already maps to %d", normalized, existing)
	}
	r.byName[normalized] = level
	r.byValue[level] = normalized
	return nil
}

// Parse 解析注册表中的级别名称或数字。
func (r *LevelRegistry) Parse(value string) (slog.Level, error) {
	if r == nil {
		return slog.LevelInfo, fmt.Errorf("goark-log: level registry is nil")
	}
	text := strings.TrimSpace(value)
	if text == "" {
		return slog.LevelInfo, nil
	}
	normalized := strings.ToUpper(text)
	if normalized == "WARNING" {
		normalized = "WARN"
	}
	r.mu.RLock()
	level, ok := r.byName[normalized]
	r.mu.RUnlock()
	if ok {
		return level, nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return slog.LevelInfo, fmt.Errorf("goark-log: unsupported log level %q", value)
	}
	return slog.Level(parsed), nil
}

// Name 返回注册表中的精确名称，未注册时按标准区间降级。
func (r *LevelRegistry) Name(level slog.Level) string {
	if r != nil {
		r.mu.RLock()
		name, ok := r.byValue[level]
		r.mu.RUnlock()
		if ok {
			return name
		}
	}
	switch {
	case level <= LevelTrace:
		return "TRACE"
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}

func levelPointer(level slog.Level) *slog.Level {
	copied := level
	return &copied
}

func newDefaultLevelRegistry() *LevelRegistry {
	registry := &LevelRegistry{
		byName:  make(map[string]slog.Level, 5),
		byValue: make(map[slog.Level]string, 5),
	}
	registry.mustRegisterBuiltIn("TRACE", LevelTrace)
	registry.mustRegisterBuiltIn("DEBUG", slog.LevelDebug)
	registry.mustRegisterBuiltIn("INFO", slog.LevelInfo)
	registry.mustRegisterBuiltIn("WARN", slog.LevelWarn)
	registry.mustRegisterBuiltIn("ERROR", slog.LevelError)
	return registry
}

func (r *LevelRegistry) mustRegisterBuiltIn(name string, level slog.Level) {
	r.byName[name] = level
	r.byValue[level] = name
}

func normalizeLevelName(name string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	if normalized == "" {
		return "", fmt.Errorf("goark-log: log level name is empty")
	}
	if strings.ContainsAny(normalized, " \t\r\n") {
		return "", fmt.Errorf("goark-log: log level name %q contains whitespace", name)
	}
	if _, err := strconv.Atoi(normalized); err == nil {
		return "", fmt.Errorf("goark-log: log level name %q must not be numeric", name)
	}
	return normalized, nil
}
