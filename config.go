package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

// ConfigSource 标识最终采用的配置来源。
type ConfigSource string

const (
	ConfigSourceExplicit ConfigSource = "explicit"
	ConfigSourceEnv      ConfigSource = "env"
	ConfigSourceBoot     ConfigSource = "boot"
	ConfigSourceFile     ConfigSource = "file"
	ConfigSourceDefault  ConfigSource = "default"
)

// ConfigResult 描述配置解析结果。
type ConfigResult struct {
	Source ConfigSource
	Path   string
}

// ConfigLoadOption 调整配置加载过程。
type ConfigLoadOption func(*configLoadSettings)

type configLoadSettings struct {
	explicitPath string
	envKey       string
	workingDir   string
	boot         PropertyResolver
	defaultPaths []string
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
	fileConfig, err := loadConfigFile(ctx, path)
	if err != nil {
		return Options{}, nil, err
	}
	handlerOptions, err := fileConfig.options()
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
		exists, err := pathExists(candidate)
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

func closeAppenderList(appenders []Appender) error {
	config := &runtimeConfig{all: appenders}
	return config.close()
}
