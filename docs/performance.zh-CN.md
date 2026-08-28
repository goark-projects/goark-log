# 性能

[English](performance.md)

本页说明 `goark-log` 的性能设计和测量方式。没有在待发布的精确 commit 上运行 benchmark
并记录输出时，不写发布性能结论。

## 热点路径设计

| 领域 | 当前行为 |
| --- | --- |
| 门面 | 标准 `slog.Handler` 路径用于普通业务代码。 |
| 原生 logger | `NewNativeLogger` 提供更低分配路径和按级别短路的 builder。 |
| 固定属性路径 | `LogAttrs3` 避免常见三属性事件构造变长切片。 |
| JSON direct | `NewJSONAppender` 和配置化 `type: json` 绕过通用 layout。 |
| Caller 数据 | 只有请求 location 时才使用 `slog.Record.PC` 和 source 格式化。 |
| 异步 | Handler 级和 appender 级队列使用有界 ring buffer 和显式溢出策略。 |
| 滚动动作 | 压缩和删除可以放到串行后台 worker。 |
| 插件边界 | 重量级可选依赖不进入核心模块。 |

## Benchmark 套件

核心 benchmark：

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

热点路径 benchmark：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

压力 benchmark：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure' -benchmem ./benchmarks/core
```

对比 benchmark 位于独立模块，避免 zap 和 zerolog 成为核心依赖：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

只有需要下载依赖时使用代理：

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```

## 结果解释

Benchmark 数字只对同一机器、OS、Go 版本、commit 和命令有效。至少记录：

| 字段 | 示例 |
| --- | --- |
| Commit | `git rev-parse HEAD` |
| Go version | `go version` |
| OS 和架构 | `go env GOOS GOARCH` |
| 命令 | 精确 benchmark 命令。 |
| 结果 | `ns/op`、`B/op`、`allocs/op`。 |

除非在精确 release candidate 上运行对比 benchmark，并说明工作负载，否则不要宣称超越
zap、zerolog、slog 或其他 logger。

## 调优指南

| 目标 | 设置 |
| --- | --- |
| 最低 stdout 开销 | 使用 `target: stdout` 的 JSON direct appender。 |
| 最低文件开销 | 使用 JSON direct file，或开启缓冲的 JSON layout。 |
| 关键日志稳定延迟 | 使用 async overflow `block` 并配置足够队列容量。 |
| 非关键 debug 日志保持服务前进 | 使用 async overflow `drop` 或 `drop-debug` 并观察计数器。 |
| 保留审计日志 | 使用 `block` 或同步文件写入、`flushOnWrite: true` 和严格权限。 |
| 降低 caller 开销 | 除非 route 或 layout 需要，否则关闭 `includeLocation`。 |
| 降低 layout 成本 | 热点路径优先 JSON direct 或简单 pattern/text layout。 |
| 减少归档争用 | 使用 rolling `compression.async: true` 或 `asyncActions: true`。 |

## 异步计数器

`Handler` 对 handler 级异步暴露计数器：

| 计数器 | 含义 |
| --- | --- |
| `AsyncDropped()` | 被溢出策略丢弃的事件数。 |
| `AsyncFailed()` | 异步投递失败事件数。 |

Appender 级异步通过程序化选项支持 error handler。配置化 appender 级异步按选定溢出策略
写入，并在关闭时 drain。

## 发布门禁

发布说明需要提到性能时：

1. 先运行正确性测试：`GOWORK=off go test ./...`。
2. 运行 `GOWORK=off go vet ./...`。
3. 在 release candidate 上运行核心 benchmark。
4. 只在 `benchmarks/compare` 运行对比 benchmark。
5. 在发布说明记录精确命令和环境。
