package goarklog

import (
	"fmt"
)

// Close 关闭所有 appender。
func (h *Handler) Close() error {
	if h == nil || h.router == nil {
		return nil
	}
	if h.async != nil {
		if err := h.async.close(); err != nil {
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
	if options.Async.Enabled != (h.async != nil) {
		return fmt.Errorf("goark-log: async logger enablement cannot be changed by reload")
	}
	if h.async != nil {
		normalized, err := normalizeAsyncLoggerOptions(options.Async)
		if err != nil {
			return err
		}
		if !sameAsyncLoggerRuntimeOptions(normalized, h.async.options) {
			return fmt.Errorf("goark-log: async logger queue settings cannot be changed by reload")
		}
	}
	return h.router.Replace(options)
}

// AsyncDropped 返回 Handler 层异步日志丢弃数量。
func (h *Handler) AsyncDropped() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.droppedCount()
}

// AsyncFailed 返回 Handler 层异步后台写入失败批次数量。
func (h *Handler) AsyncFailed() uint64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.failedCount()
}

// AsyncRemainingCapacity 返回 Handler 层异步队列剩余容量。
func (h *Handler) AsyncRemainingCapacity() int64 {
	if h == nil || h.async == nil {
		return 0
	}
	return h.async.remainingCapacity()
}

func (h *Handler) asyncIncludeLocation() bool {
	return h != nil && h.async != nil && h.async.includeLocation()
}

func (h *Handler) routeIncludeLocation(name string) bool {
	if h == nil || h.router == nil {
		return false
	}
	config := h.router.current.Load()
	if config == nil || !config.includeLocation {
		return false
	}
	return routePlanFromConfig(config, name).route.IncludeLocation
}
