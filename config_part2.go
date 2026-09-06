package log

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"goark.dev/log/internal/configfile"
	internalrouter "goark.dev/log/internal/router"
)

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
		settings.lookups = settings.registry.LookupResolver()
	}
	if settings.boot != nil {
		settings.lookups = settings.lookups.Clone()
		settings.lookups.Register("prop", settings.boot.GetProperty)
		settings.lookups.Register("property", settings.boot.GetProperty)
	}
	return settings, nil
}

func (s *configLoadSettings) loadData(ctx context.Context) (Options, *ConfigResult, error) {
	if s.dataName == "" {
		return Options{}, nil, fmt.Errorf("goark-log: config data name is empty")
	}
	format, err := configfileFormat(s.dataName)
	if err != nil {
		return Options{}, nil, err
	}
	fileConfig, err := configfile.Decode(bytes.NewReader(s.data), format, s.lookups)
	if err != nil {
		return Options{}, nil, err
	}
	handlerOptions, err := fileConfig.Options(s.registry)
	if err != nil {
		return Options{}, nil, err
	}
	monitorInterval, err := fileConfig.MonitorIntervalDuration()
	if err != nil {
		return Options{}, nil, err
	}
	result := &ConfigResult{
		Source:          ConfigSourceExplicit,
		Path:            s.dataName,
		MonitorInterval: monitorInterval,
	}
	return s.customize(ctx, Options(handlerOptions), result)
}

func (s *configLoadSettings) customize(
	ctx context.Context,
	options Options,
	result *ConfigResult,
) (Options, *ConfigResult, error) {
	current := options
	for _, customizer := range s.customizers {
		updated, err := customizer(ctx, current, result)
		if err != nil {
			_ = closeAppenderList(current.Appenders)
			_ = closeAppenderList(updated.Appenders)
			return Options{}, nil, fmt.Errorf("goark-log: customize loaded options: %w", err)
		}
		current = updated
	}
	return current, result, nil
}

// NewConfiguredHandler 从配置创建 Handler。
func NewConfiguredHandler(
	ctx context.Context,
	options ...ConfigLoadOption,
) (*Handler, *ConfigResult, error) {
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

type configLoadSettings struct {
	explicitPath string
	dataName     string
	data         []byte
	envKey       string
	workingDir   string
	boot         PropertyResolver
	defaultPaths []string
	lookups      *LookupResolver
	registry     *PluginRegistry
	customizers  []OptionsCustomizer
}

// WithOptionsCustomizer 注册配置加载后的日志选项定制器。
func WithOptionsCustomizer(customizers ...OptionsCustomizer) ConfigLoadOption {
	copied := append([]OptionsCustomizer(nil), customizers...)
	return func(settings *configLoadSettings) {
		for _, customizer := range copied {
			if customizer != nil {
				settings.customizers = append(settings.customizers, customizer)
			}
		}
	}
}

// WithConfigData 设置内存中的显式配置内容，name 的扩展名决定配置格式。
func WithConfigData(name string, data []byte) ConfigLoadOption {
	copied := append([]byte(nil), data...)
	return func(settings *configLoadSettings) {
		settings.dataName = strings.TrimSpace(name)
		settings.data = copied
	}
}

// ConfigResult 描述配置解析结果。
type ConfigResult struct {
	Source          ConfigSource
	Path            string
	MonitorInterval time.Duration
}

// WithConfigPath 设置显式配置文件路径，优先级最高。
func WithConfigPath(path string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.explicitPath = path
	}
}

// WithConfigWorkingDir 设置默认配置文件发现的工作目录。
func WithConfigWorkingDir(dir string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.workingDir = dir
	}
}

// WithDefaultConfigPaths 覆盖默认配置文件发现路径。
func WithDefaultConfigPaths(paths ...string) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.defaultPaths = append([]string(nil), paths...)
	}
}

// WithPluginRegistry 设置配置构建使用的插件注册表。
func WithPluginRegistry(registry *PluginRegistry) ConfigLoadOption {
	return func(settings *configLoadSettings) {
		settings.registry = registry
	}
}

// ConfigLoadOption 调整配置加载过程。
type ConfigLoadOption func(*configLoadSettings)

func closeAppenderList(appenders []Appender) error {
	return internalrouter.CloseAppenders(appenders, isAsyncAppender)
}
