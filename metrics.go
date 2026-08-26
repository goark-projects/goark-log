package goarklog

import (
	"context"
	"fmt"
	"sync/atomic"
)

// MetricsSnapshot 是 Handler 当前运行指标的不可变快照。
type MetricsSnapshot struct {
	// Events 是通过路由过滤后进入 appender 分发阶段的事件数量。
	Events uint64
	// AppenderWrites 是底层 appender 成功写入的次数。
	AppenderWrites uint64
	// AppenderFailures 是底层 appender 返回错误的次数。
	AppenderFailures uint64
	// Filtered 是路由级过滤器拒绝事件的次数。
	Filtered uint64
	// AsyncDropped 是异步队列按丢弃策略放弃事件的次数。
	AsyncDropped uint64
	// AsyncFailed 是异步后台批量写入失败的批次数。
	AsyncFailed uint64
	// AsyncQueueSize 是异步队列的规整后容量。
	AsyncQueueSize int
	// AsyncQueueRemaining 是异步队列当前剩余容量的近似值。
	AsyncQueueRemaining int64
}

// MetricsExporter 把核心指标导出到外部系统。
type MetricsExporter interface {
	ExportLogMetrics(ctx context.Context, snapshot MetricsSnapshot) error
}

// MetricsExporterFunc 把函数适配为 MetricsExporter。
type MetricsExporterFunc func(ctx context.Context, snapshot MetricsSnapshot) error

// ExportLogMetrics 执行指标导出函数。
func (f MetricsExporterFunc) ExportLogMetrics(ctx context.Context, snapshot MetricsSnapshot) error {
	if f == nil {
		return fmt.Errorf("goark-log: metrics exporter func is nil")
	}
	return f(ctx, snapshot)
}

type handlerMetrics struct {
	events           atomic.Uint64
	appenderWrites   atomic.Uint64
	appenderFailures atomic.Uint64
	filtered         atomic.Uint64
}

func (m *handlerMetrics) snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		Events:           m.events.Load(),
		AppenderWrites:   m.appenderWrites.Load(),
		AppenderFailures: m.appenderFailures.Load(),
		Filtered:         m.filtered.Load(),
	}
}

// Metrics 返回 Handler 当前指标快照。
func (h *Handler) Metrics() MetricsSnapshot {
	if h == nil {
		return MetricsSnapshot{}
	}
	snapshot := h.metrics.snapshot()
	if h.async != nil {
		snapshot.AsyncDropped = h.async.droppedCount()
		snapshot.AsyncFailed = h.async.failedCount()
		snapshot.AsyncQueueSize = h.async.queueSize
		snapshot.AsyncQueueRemaining = h.async.remainingCapacity()
	}
	return snapshot
}

// ExportMetrics 使用调用方提供的 exporter 导出指标快照。
func (h *Handler) ExportMetrics(ctx context.Context, exporter MetricsExporter) error {
	if exporter == nil {
		return fmt.Errorf("goark-log: metrics exporter is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return exporter.ExportLogMetrics(ctx, h.Metrics())
}
