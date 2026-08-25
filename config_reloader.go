package goarklog

import (
	"context"
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
