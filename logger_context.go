package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	// DefaultReloadInterval 是 Watch 的默认轮询间隔。
	DefaultReloadInterval = 5 * time.Second
)

// LoggerContext 管理一个可关闭、可重载的日志运行期。
type LoggerContext struct {
	handler     *Handler
	status      *StatusLogger
	mu          sync.RWMutex
	result      *ConfigResult
	watchCancel context.CancelFunc
	watchDone   <-chan struct{}
}

type loggerContextSettings struct {
	status *StatusLogger
}

// ConfigReloader 负责把配置重新加载到已有 Handler。
type ConfigReloader struct {
	handler *Handler
	options []ConfigLoadOption
	mu      sync.Mutex
	last    configSignature
}

type configSignature struct {
	source  ConfigSource
	path    string
	modTime time.Time
	size    int64
}

// LoggerContextOption 调整 LoggerContext。
type LoggerContextOption func(*loggerContextSettings)

// WithLoggerContextStatus 设置 LoggerContext 使用的 StatusLogger。
func WithLoggerContextStatus(status *StatusLogger) LoggerContextOption {
	return func(settings *loggerContextSettings) {
		settings.status = status
	}
}

// NewLoggerContext 基于显式 Options 创建日志上下文。
func NewLoggerContext(options Options, contextOptions ...LoggerContextOption) (*LoggerContext, error) {
	settings := newLoggerContextSettings(contextOptions...)
	handler, err := NewHandler(options)
	if err != nil {
		settings.status.Error(context.Background(), "build logger context failed", err)
		return nil, err
	}
	settings.status.Info(context.Background(), "logger context started")
	return &LoggerContext{
		handler: handler,
		status:  settings.status,
	}, nil
}

// NewConfiguredLoggerContext 从配置创建日志上下文。
func NewConfiguredLoggerContext(ctx context.Context, configOptions ...ConfigLoadOption) (*LoggerContext, *ConfigResult, error) {
	handlerOptions, result, err := LoadOptions(ctx, configOptions...)
	status := NewStatusLogger()
	if err != nil {
		status.Error(ctx, "load logger context config failed", err)
		return nil, nil, err
	}
	context, err := NewLoggerContext(handlerOptions, WithLoggerContextStatus(status))
	if err != nil {
		_ = closeAppenderList(handlerOptions.Appenders)
		return nil, nil, err
	}
	context.mu.Lock()
	context.result = result
	context.mu.Unlock()
	if err := context.startConfigMonitor(result.MonitorInterval, configOptions...); err != nil {
		_ = context.Close()
		return nil, nil, err
	}
	status.Info(ctx, fmt.Sprintf("logger context config loaded from %s", result.Source))
	return context, result, nil
}

// NewConfigReloader 创建配置重载器。
func NewConfigReloader(handler *Handler, options ...ConfigLoadOption) (*ConfigReloader, error) {
	if handler == nil {
		return nil, fmt.Errorf("goark-log: handler is nil")
	}
	return &ConfigReloader{
		handler: handler,
		options: append([]ConfigLoadOption(nil), options...),
	}, nil
}

// Logger 返回指定名称的 slog.Logger。
func (c *LoggerContext) Logger(name string) *slog.Logger {
	if c == nil || c.handler == nil {
		return slog.Default()
	}
	return NewLogger(c.handler, name)
}

// Handler 返回底层 slog.Handler。
func (c *LoggerContext) Handler() *Handler {
	if c == nil {
		return nil
	}
	return c.handler
}

// StatusLogger 返回内部状态日志器。
func (c *LoggerContext) StatusLogger() *StatusLogger {
	if c == nil {
		return nil
	}
	return c.status
}

// ConfigResult 返回最近一次配置加载结果。
func (c *LoggerContext) ConfigResult() *ConfigResult {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.result == nil {
		return nil
	}
	copied := *c.result
	return &copied
}

// Reload 用显式 Options 重载日志上下文。
func (c *LoggerContext) Reload(options Options) error {
	if c == nil || c.handler == nil {
		return fmt.Errorf("goark-log: logger context is nil")
	}
	if err := c.handler.Reload(options); err != nil {
		c.status.Error(context.Background(), "reload logger context failed", err)
		return err
	}
	c.status.Info(context.Background(), "logger context reloaded")
	return nil
}

// ReloadConfigured 从配置重新加载日志上下文。
func (c *LoggerContext) ReloadConfigured(ctx context.Context, options ...ConfigLoadOption) (*ConfigResult, error) {
	if c == nil || c.handler == nil {
		return nil, fmt.Errorf("goark-log: logger context is nil")
	}
	handlerOptions, result, err := LoadOptions(ctx, options...)
	if err != nil {
		c.status.Error(ctx, "reload logger context config failed", err)
		return nil, err
	}
	if err := c.handler.Reload(handlerOptions); err != nil {
		_ = closeAppenderList(handlerOptions.Appenders)
		c.status.Error(ctx, "reload logger context failed", err)
		return nil, err
	}
	c.mu.Lock()
	c.result = result
	c.mu.Unlock()
	c.status.Info(ctx, fmt.Sprintf("logger context config reloaded from %s", result.Source))
	return result, nil
}

// Close 关闭日志上下文。
func (c *LoggerContext) Close() error {
	if c == nil || c.handler == nil {
		return nil
	}
	c.stopConfigMonitor()
	if err := c.handler.Close(); err != nil {
		c.status.Error(context.Background(), "close logger context failed", err)
		return err
	}
	c.status.Info(context.Background(), "logger context closed")
	return nil
}

// Reload 立即重新加载配置。
func (r *ConfigReloader) Reload(ctx context.Context) (*ConfigResult, error) {
	if r == nil || r.handler == nil {
		return nil, fmt.Errorf("goark-log: config reloader is nil")
	}
	handlerOptions, result, err := LoadOptions(ctx, r.options...)
	if err != nil {
		return nil, err
	}
	if err := r.handler.Reload(handlerOptions); err != nil {
		_ = closeAppenderList(handlerOptions.Appenders)
		return nil, err
	}
	signature, err := r.currentSignature()
	if err == nil {
		r.mu.Lock()
		r.last = signature
		r.mu.Unlock()
	}
	return result, nil
}

// ReloadIfChanged 在配置文件变化后重新加载。
func (r *ConfigReloader) ReloadIfChanged(ctx context.Context) (bool, *ConfigResult, error) {
	if r == nil || r.handler == nil {
		return false, nil, fmt.Errorf("goark-log: config reloader is nil")
	}
	signature, err := r.currentSignature()
	if err != nil {
		return false, nil, err
	}
	r.mu.Lock()
	unchanged := r.last == signature
	r.mu.Unlock()
	if unchanged {
		return false, &ConfigResult{Source: signature.source, Path: signature.path}, nil
	}
	result, err := r.Reload(ctx)
	if err != nil {
		return false, nil, err
	}
	return true, result, nil
}

// Watch 轮询配置文件并在变化时 reload，返回的 channel 会在 ctx 结束后关闭。
func (r *ConfigReloader) Watch(ctx context.Context, interval time.Duration, onError func(error)) <-chan struct{} {
	done := make(chan struct{})
	if interval <= 0 {
		interval = DefaultReloadInterval
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, _, err := r.ReloadIfChanged(ctx); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()
	return done
}

func (c *LoggerContext) startConfigMonitor(interval time.Duration, options ...ConfigLoadOption) error {
	if c == nil || c.handler == nil || interval <= 0 {
		return nil
	}
	c.mu.RLock()
	result := c.result
	c.mu.RUnlock()
	if result == nil || result.Path == "" {
		return nil
	}
	reloader, err := NewConfigReloader(c.handler, options...)
	if err != nil {
		return err
	}
	if signature, err := reloader.currentSignature(); err == nil {
		reloader.mu.Lock()
		reloader.last = signature
		reloader.mu.Unlock()
	}
	watchCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.mu.Lock()
	c.watchCancel = cancel
	c.watchDone = done
	c.mu.Unlock()
	go c.watchConfig(watchCtx, done, reloader, interval)
	return nil
}

func (c *LoggerContext) watchConfig(ctx context.Context, done chan<- struct{}, reloader *ConfigReloader, interval time.Duration) {
	defer close(done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, result, err := reloader.ReloadIfChanged(ctx)
			if err != nil {
				c.status.Error(ctx, "reload logger context config failed", err)
				continue
			}
			if !changed {
				continue
			}
			c.mu.Lock()
			c.result = result
			c.mu.Unlock()
			c.status.Info(ctx, fmt.Sprintf("logger context config reloaded from %s", result.Source))
		}
	}
}

func (c *LoggerContext) stopConfigMonitor() {
	c.mu.Lock()
	cancel := c.watchCancel
	done := c.watchDone
	c.watchCancel = nil
	c.watchDone = nil
	c.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

func (r *ConfigReloader) currentSignature() (configSignature, error) {
	settings, err := newConfigLoadSettings(r.options...)
	if err != nil {
		return configSignature{}, err
	}
	path, source, err := settings.resolvePath()
	if err != nil {
		return configSignature{}, err
	}
	signature := configSignature{
		source: source,
		path:   path,
	}
	if path == "" {
		return signature, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return configSignature{}, fmt.Errorf("goark-log: stat config file %q: %w", path, err)
	}
	signature.modTime = info.ModTime()
	signature.size = info.Size()
	return signature, nil
}

func newLoggerContextSettings(options ...LoggerContextOption) loggerContextSettings {
	settings := loggerContextSettings{
		status: NewStatusLogger(),
	}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	if settings.status == nil {
		settings.status = NewStatusLogger(WithStatusWriter(nil))
	}
	return settings
}
