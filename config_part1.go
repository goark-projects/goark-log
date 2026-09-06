package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"goark.dev/log/internal/configfile"
	"goark.dev/log/internal/logfile"
)

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
	if settings.data != nil {
		return settings.loadData(ctx)
	}
	path, source, err := settings.resolvePath()
	if err != nil {
		return Options{}, nil, err
	}
	result := &ConfigResult{Source: source, Path: path}
	if path == "" {
		return settings.customize(ctx, DefaultOptions(), result)
	}
	fileConfig, err := configfile.Load(ctx, path, settings.lookups)
	if err != nil {
		return Options{}, nil, err
	}
	monitorInterval, err := fileConfig.MonitorIntervalDuration()
	if err != nil {
		return Options{}, nil, err
	}
	result.MonitorInterval = monitorInterval
	handlerOptions, err := fileConfig.Options(settings.registry)
	if err != nil {
		return Options{}, nil, err
	}
	return settings.customize(ctx, Options(handlerOptions), result)
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

func configfileFormat(name string) (string, error) {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) {
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
		return "", fmt.Errorf("goark-log: unsupported config file extension for %q", name)
	}
}

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

// ConfigureDefault 从配置创建 logger，并安装为 slog 默认 logger。
func ConfigureDefault(
	ctx context.Context,
	options ...ConfigLoadOption,
) (*Handler, *ConfigResult, error) {
	logger, handler, result, err := NewConfigured(ctx, options...)
	if err != nil {
		return nil, nil, err
	}
	slog.SetDefault(logger)
	return handler, result, nil
}

// NewConfigured 从配置创建默认命名 logger 和对应 Handler。
func NewConfigured(
	ctx context.Context,
	options ...ConfigLoadOption,
) (*slog.Logger, *Handler, *ConfigResult, error) {
	handler, result, err := NewConfiguredHandler(ctx, options...)
	if err != nil {
		return nil, nil, nil, err
	}
	return NewLogger(handler, defaultLoggerName), handler, result, nil
}

const (
	ConfigSourceExplicit ConfigSource = "explicit"
	ConfigSourceEnv      ConfigSource = "env"
	ConfigSourceBoot     ConfigSource = "boot"
	ConfigSourceFile     ConfigSource = "file"
	ConfigSourceDefault  ConfigSource = "default"
)

func (s *configLoadSettings) resolveUserPath(path string) string {
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.workingDir, path)
}

// WithConfigEnvKey 设置配置文件路径环境变量名称。
func WithConfigEnvKey(key string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.envKey = key
	}
}

// WithBootPropertyResolver 接入 boot Environment 或等价配置源。
func WithBootPropertyResolver(resolver PropertyResolver) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.boot = resolver
	}
}

// WithConfigLookups 设置配置变量解析器。
func WithConfigLookups(resolver *LookupResolver) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.lookups = resolver
	}
}

const (
	// EnvConfigPath 是 goark-log 默认配置文件环境变量。
	EnvConfigPath = "GOARK_LOG_CONFIG"
)

// ConfigSource 标识最终采用的配置来源。
type ConfigSource string

// OptionsCustomizer 在配置文件解析完成后定制日志运行期选项。
type OptionsCustomizer func(context.Context, Options, *ConfigResult) (Options, error)
