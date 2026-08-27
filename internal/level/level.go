package level

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// All 表示最低阈值，配置为 ALL 时允许所有事件进入日志管线。
	All   slog.Level = math.MinInt
	Trace slog.Level = -8
	// Fatal 表示比 ERROR 更高的致命级别。
	Fatal slog.Level = 12
	// Off 表示最高阈值，配置为 OFF 时关闭普通日志事件。
	Off slog.Level = math.MaxInt
)

var defaultRegistry = newDefaultRegistry()

// Registry 保存日志级别名称和数值的双向映射。
type Registry struct {
	mu         sync.RWMutex
	byName     map[string]slog.Level
	byValue    map[slog.Level]string
	customized atomic.Bool
}

// NewRegistry 创建包含内置级别的注册表。
func NewRegistry() *Registry {
	return newDefaultRegistry()
}

// DefaultRegistry 返回进程默认级别注册表。
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// RegisterDefault 向默认注册表注册自定义级别。
func RegisterDefault(name string, level slog.Level) error {
	return defaultRegistry.Register(name, level)
}

// ParseDefault 使用默认注册表解析日志级别名称。
func ParseDefault(value string) (slog.Level, error) {
	return defaultRegistry.Parse(value)
}

// NameDefault 使用默认注册表返回级别名称。
func NameDefault(level slog.Level) string {
	return defaultRegistry.Name(level)
}

// Register 注册级别名称和数值。
func (r *Registry) Register(name string, level slog.Level) error {
	if r == nil {
		return fmt.Errorf("goark-log: level registry is nil")
	}
	normalized, err := normalizeName(name)
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
	if !isBuiltIn(normalized, level) {
		r.customized.Store(true)
	}
	return nil
}

// Parse 解析注册表中的级别名称或数字。
func (r *Registry) Parse(value string) (slog.Level, error) {
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
func (r *Registry) Name(level slog.Level) string {
	if r != nil && r.customized.Load() {
		r.mu.RLock()
		name, ok := r.byValue[level]
		r.mu.RUnlock()
		if ok {
			return name
		}
	}
	return defaultName(level)
}

func defaultName(level slog.Level) string {
	switch level {
	case All:
		return "ALL"
	case Off:
		return "OFF"
	case Fatal:
		return "FATAL"
	}
	switch {
	case level <= Trace:
		return "TRACE"
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	case level < Fatal:
		return "ERROR"
	default:
		return "FATAL"
	}
}

func isBuiltIn(name string, level slog.Level) bool {
	switch name {
	case "ALL":
		return level == All
	case "TRACE":
		return level == Trace
	case "DEBUG":
		return level == slog.LevelDebug
	case "INFO":
		return level == slog.LevelInfo
	case "WARN":
		return level == slog.LevelWarn
	case "ERROR":
		return level == slog.LevelError
	case "FATAL":
		return level == Fatal
	case "OFF":
		return level == Off
	default:
		return false
	}
}

func newDefaultRegistry() *Registry {
	registry := &Registry{
		byName:  make(map[string]slog.Level, 8),
		byValue: make(map[slog.Level]string, 8),
	}
	registry.mustRegisterBuiltIn("ALL", All)
	registry.mustRegisterBuiltIn("TRACE", Trace)
	registry.mustRegisterBuiltIn("DEBUG", slog.LevelDebug)
	registry.mustRegisterBuiltIn("INFO", slog.LevelInfo)
	registry.mustRegisterBuiltIn("WARN", slog.LevelWarn)
	registry.mustRegisterBuiltIn("ERROR", slog.LevelError)
	registry.mustRegisterBuiltIn("FATAL", Fatal)
	registry.mustRegisterBuiltIn("OFF", Off)
	return registry
}

func (r *Registry) mustRegisterBuiltIn(name string, level slog.Level) {
	r.byName[name] = level
	r.byValue[level] = name
}

func normalizeName(name string) (string, error) {
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
