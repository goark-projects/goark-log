package log

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	// DefaultReloadInterval 是 Watch 的默认轮询间隔。
	DefaultReloadInterval = 5 * time.Second
)

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
	digest  [sha256.Size]byte
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
	content, err := os.ReadFile(path)
	if err != nil {
		return configSignature{}, fmt.Errorf("goark-log: read config file %q for signature: %w", path, err)
	}
	signature.digest = sha256.Sum256(content)
	return signature, nil
}
