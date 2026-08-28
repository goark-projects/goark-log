package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goark.dev/log/internal/configvalue"
	configlevel "goark.dev/log/internal/level"
	"goark.dev/log/internal/logfile"
	configlookup "goark.dev/log/internal/lookup"
	internalrouter "goark.dev/log/internal/router"
)

const (
	// EnvConfigPath 是 goark-log 默认配置文件环境变量。
	EnvConfigPath = "GOARK_LOG_CONFIG"
)

var (
	defaultBootConfigKeys = []string{
		"goark.log.config",
		"goark.logging.config",
		"logging.config",
	}
	defaultConfigPaths = []string{
		filepath.Join("conf", "goark-log.yml"),
		filepath.Join("conf", "goark-log.yaml"),
		filepath.Join("conf", "goark-log.json"),
		filepath.Join("conf", "goark-log.xml"),
		filepath.Join("conf", "goark-log.toml"),
		filepath.Join("conf", "goark-log.properties"),
	}
)

// PropertyResolver 是 boot 配置系统需要适配的最小读取接口。
type PropertyResolver interface {
	GetProperty(key string) (string, bool)
}

// PropertyMap 是测试和轻量嵌入场景可直接使用的配置适配器。
type PropertyMap map[string]string

func (m PropertyMap) GetProperty(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

// LookupFunc 根据键解析配置变量。
type LookupFunc = configlookup.Func

// LookupResolver 负责解析配置中的 ${namespace:key} 变量。
type LookupResolver = configlookup.Resolver

// NewLookupResolver 创建带默认 lookup 的解析器。
func NewLookupResolver() *LookupResolver {
	return configlookup.NewResolver()
}

// ConfigSource 标识最终采用的配置来源。
type ConfigSource string

const (
	ConfigSourceExplicit ConfigSource = "explicit"
	ConfigSourceEnv      ConfigSource = "env"
	ConfigSourceBoot     ConfigSource = "boot"
	ConfigSourceFile     ConfigSource = "file"
	ConfigSourceDefault  ConfigSource = "default"
)

const (
	// LevelAll 表示最低阈值，配置为 ALL 时允许所有事件进入日志管线。
	LevelAll   = configlevel.All
	LevelTrace = configlevel.Trace
	// LevelFatal 表示比 ERROR 更高的致命级别。
	LevelFatal = configlevel.Fatal
	// LevelOff 表示最高阈值，配置为 OFF 时关闭普通日志事件。
	LevelOff = configlevel.Off
)

// LevelRegistry 保存日志级别名称和数值的双向映射。
type LevelRegistry = configlevel.Registry

// NewLevelRegistry 创建包含内置级别的注册表。
func NewLevelRegistry() *LevelRegistry {
	return configlevel.NewRegistry()
}

// DefaultLevelRegistry 返回进程默认级别注册表。
func DefaultLevelRegistry() *LevelRegistry {
	return configlevel.DefaultRegistry()
}

// RegisterLevel 向默认注册表注册自定义级别。
func RegisterLevel(name string, level slog.Level) error {
	return configlevel.RegisterDefault(name, level)
}

// ParseLevel 解析日志级别名称。
func ParseLevel(value string) (slog.Level, error) {
	return configlevel.ParseDefault(value)
}

// LevelName 返回级别名称，优先返回已注册的精确名称。
func LevelName(level slog.Level) string {
	return configlevel.NameDefault(level)
}

// ParseByteSize 解析日志滚动大小。
func ParseByteSize(value string) (int64, error) {
	return configvalue.ByteSize(value)
}

// ParseRollingInterval 解析时间滚动间隔。
func ParseRollingInterval(value string) (time.Duration, error) {
	return configvalue.RollingInterval(value)
}

// ParseRollingMaxAge 解析滚动档案最大保留时间。
func ParseRollingMaxAge(value string) (time.Duration, error) {
	return configvalue.RollingMaxAge(value)
}

// ParseMonitorInterval 解析配置监控间隔；纯数字按秒处理。
func ParseMonitorInterval(value string) (time.Duration, error) {
	return configvalue.MonitorInterval(value)
}

func levelName(level slog.Level) string {
	return LevelName(level)
}

func levelPointer(level slog.Level) *slog.Level {
	copied := level
	return &copied
}

// ConfigResult 描述配置解析结果。
type ConfigResult struct {
	Source          ConfigSource
	Path            string
	MonitorInterval time.Duration
}

// ConfigLoadOption 调整配置加载过程。
type ConfigLoadOption func(*configLoadSettings)

type configLoadSettings struct {
	explicitPath string
	envKey       string
	workingDir   string
	boot         PropertyResolver
	defaultPaths []string
	lookups      *LookupResolver
	registry     *PluginRegistry
}

// WithConfigPath 设置显式配置文件路径，优先级最高。
func WithConfigPath(path string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.explicitPath = path
	}
}

// WithConfigEnvKey 设置配置文件路径环境变量名称。
func WithConfigEnvKey(key string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.envKey = key
	}
}

// WithConfigWorkingDir 设置默认配置文件发现的工作目录。
func WithConfigWorkingDir(dir string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.workingDir = dir
	}
}

// WithBootPropertyResolver 接入 boot Environment 或等价配置源。
func WithBootPropertyResolver(resolver PropertyResolver) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.boot = resolver
	}
}

// WithDefaultConfigPaths 覆盖默认配置文件发现路径。
func WithDefaultConfigPaths(paths ...string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.defaultPaths = append([]string(nil), paths...)
	}
}

// WithConfigLookups 设置配置变量解析器。
func WithConfigLookups(resolver *LookupResolver) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.lookups = resolver
	}
}

// WithPluginRegistry 设置配置构建使用的插件注册表。
func WithPluginRegistry(registry *PluginRegistry) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.registry = registry
	}
}

// LoadOptions 按优先级加载并构建 Handler Options。
func LoadOptions(ctx context.Context, options ...ConfigLoadOption) (Options, *ConfigResult, error) {
	if ctx == nil {
		return Options{}, nil, fmt.Errorf("goark-log: context is nil")
	}
	if err := ctx.Err(); err != nil {
		return Options{}, nil, err
	}
	settings, err := newConfigLoadSettings(options...)
	if err != nil {
		return Options{}, nil, err
	}
	path, source, err := settings.resolvePath()
	if err != nil {
		return Options{}, nil, err
	}
	result := &ConfigResult{Source: source, Path: path}
	if path == "" {
		return DefaultOptions(), result, nil
	}
	fileConfig, err := loadConfigFile(ctx, path, settings.lookups)
	if err != nil {
		return Options{}, nil, err
	}
	monitorInterval, err := fileConfig.monitorInterval()
	if err != nil {
		return Options{}, nil, err
	}
	result.MonitorInterval = monitorInterval
	handlerOptions, err := fileConfig.options(settings.registry)
	if err != nil {
		return Options{}, nil, err
	}
	return handlerOptions, result, nil
}

// NewConfiguredHandler 从配置创建 Handler。
func NewConfiguredHandler(ctx context.Context, options ...ConfigLoadOption) (*Handler, *ConfigResult, error) {
	handlerOptions, result, err := LoadOptions(ctx, options...)
	if err != nil {
		return nil, nil, err
	}
	handler, err := NewHandler(handlerOptions)
	if err != nil {
		_ = closeAppenderList(handlerOptions.Appenders)
		return nil, nil, err
	}
	return handler, result, nil
}

// NewConfigured 从配置创建默认命名 logger 和对应 Handler。
func NewConfigured(ctx context.Context, options ...ConfigLoadOption) (*slog.Logger, *Handler, *ConfigResult, error) {
	handler, result, err := NewConfiguredHandler(ctx, options...)
	if err != nil {
		return nil, nil, nil, err
	}
	return NewLogger(handler, defaultLoggerName), handler, result, nil
}

// ConfigureDefault 从配置创建 logger，并安装为 slog 默认 logger。
func ConfigureDefault(ctx context.Context, options ...ConfigLoadOption) (*Handler, *ConfigResult, error) {
	logger, handler, result, err := NewConfigured(ctx, options...)
	if err != nil {
		return nil, nil, err
	}
	slog.SetDefault(logger)
	return handler, result, nil
}

func newConfigLoadSettings(options ...ConfigLoadOption) (*configLoadSettings, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("goark-log: get working directory: %w", err)
	}
	settings := &configLoadSettings{
		envKey:       EnvConfigPath,
		workingDir:   workingDir,
		defaultPaths: append([]string(nil), defaultConfigPaths...),
	}
	for _, option := range options {
		if option != nil {
			option(settings)
		}
	}
	settings.envKey = strings.TrimSpace(settings.envKey)
	if settings.envKey == "" {
		settings.envKey = EnvConfigPath
	}
	if strings.TrimSpace(settings.workingDir) == "" {
		settings.workingDir = workingDir
	}
	if settings.registry == nil {
		settings.registry = DefaultPluginRegistry()
	}
	if settings.lookups == nil {
		settings.lookups = settings.registry.lookupResolver()
	}
	return settings, nil
}

func (s *configLoadSettings) resolvePath() (string, ConfigSource, error) {
	if path := strings.TrimSpace(s.explicitPath); path != "" {
		return s.resolveUserPath(path), ConfigSourceExplicit, nil
	}
	if path := strings.TrimSpace(os.Getenv(s.envKey)); path != "" {
		return s.resolveUserPath(path), ConfigSourceEnv, nil
	}
	if s.boot != nil {
		for _, key := range defaultBootConfigKeys {
			value, ok := s.boot.GetProperty(key)
			if ok && strings.TrimSpace(value) != "" {
				return s.resolveUserPath(value), ConfigSourceBoot, nil
			}
		}
	}
	for _, path := range s.defaultPaths {
		candidate := s.resolveUserPath(path)
		exists, err := logfile.Exists(candidate)
		if err != nil {
			return "", "", fmt.Errorf("goark-log: stat config file %q: %w", candidate, err)
		}
		if exists {
			return candidate, ConfigSourceFile, nil
		}
	}
	return "", ConfigSourceDefault, nil
}

func (s *configLoadSettings) resolveUserPath(path string) string {
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.workingDir, path)
}

func configFormat(path string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepathExt(path), ".")) {
	case "yml", "yaml":
		return "yaml", nil
	case "json":
		return "json", nil
	case "xml":
		return "xml", nil
	case "toml":
		return "toml", nil
	case "properties":
		return "properties", nil
	default:
		return "", fmt.Errorf("goark-log: unsupported config file extension for %q", path)
	}
}

func filepathExt(path string) string {
	index := strings.LastIndexByte(path, '.')
	if index < 0 {
		return ""
	}
	return path[index:]
}

func closeAppenderList(appenders []Appender) error {
	return internalrouter.CloseAppenders(appenders, isAsyncAppender)
}
