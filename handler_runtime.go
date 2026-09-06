package log

import (
	"fmt"
	"log/slog"

	internalrouter "goark.dev/log/internal/router"
)

// Close 关闭异步管线和所有 appender。
func (h *Handler) Close() error {
	if h == nil || h.router == nil {
		return nil
	}
	if h.async != nil {
		if err := h.async.Close(); err != nil {
			return err
		}
	}
	return h.router.Close()
}

// Reload 使用新的运行期配置替换当前路由。
func (h *Handler) Reload(options Options) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	options = defaultRuntimeOptions(options)
	if options.Async.Enabled != (h.async != nil) {
		return fmt.Errorf("goark-log: async logger enablement cannot be changed by reload")
	}
	if h.async != nil {
		normalized, err := normalizeAsyncLoggerOptions(options.Async)
		if err != nil {
			return err
		}
		if !sameAsyncLoggerRuntimeOptions(normalized, h.async.Options()) {
			return fmt.Errorf("goark-log: async logger queue settings cannot be changed by reload")
		}
	}
	return h.router.Replace(toRouterOptions(options))
}

// SetLevel 原子设置 Logger 级别；level 为 nil 时恢复配置文件定义或继承关系。
func (h *Handler) SetLevel(name string, level *slog.Level) error {
	if h == nil || h.router == nil {
		return fmt.Errorf("goark-log: handler is nil")
	}
	return h.router.SetLevel(name, level)
}

// LoggerConfigurations 返回 Root 和命名 Logger 的级别配置快照。
func (h *Handler) LoggerConfigurations() []LoggerConfiguration {
	if h == nil || h.router == nil {
		return nil
	}
	return h.router.Configurations()
}

// AsyncDropped 返回 Handler 层异步日志丢弃数量。
func (h *Handler) AsyncDropped() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.Dropped()
}

// AsyncFailed 返回 Handler 层异步后台写入失败批次数量。
func (h *Handler) AsyncFailed() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.Failed()
}

// AsyncOptions 返回 Handler 层异步管线归一化后的运行期配置。
func (h *Handler) AsyncOptions() AsyncLoggerOptions {
	if h == nil || h.async == nil {
		return AsyncLoggerOptions{}
	}
	return h.async.Options()
}

// AsyncRemainingCapacity 返回 Handler 层异步队列剩余容量。
func (h *Handler) AsyncRemainingCapacity() int64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.RemainingCapacity()
}

func defaultRuntimeOptions(options Options) Options {
	if len(options.Appenders) != 0 {
		return options
	}
	defaults := DefaultOptions()
	options.Appenders = defaults.Appenders
	if len(options.Root.AppenderRefs) == 0 && len(options.Root.AppenderRefControls) == 0 {
		options.Root.AppenderRefs = defaults.Root.AppenderRefs
	}
	return options
}

func toRouterOptions(options Options) internalrouter.Options {
	return internalrouter.Options{
		Appenders:       options.Appenders,
		Filters:         options.Filters,
		Root:            options.Root,
		Loggers:         options.Loggers,
		IsAsyncAppender: isAsyncAppender,
	}
}
