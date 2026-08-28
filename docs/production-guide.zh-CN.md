# 生产指南

[English](production-guide.md)

本指南描述当前核心模块已经实现的生产路径。它不依赖可选外部 sink，也不需要网络服务。

## 推荐启动方式

长生命周期服务优先使用 `LoggerContext`。它拥有 handler，提供命名 logger，记录内部
status event，并在 `monitorInterval` 大于零时启动配置轮询。

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

logger := loggerContext.Logger("goark.http")
logger.InfoContext(ctx, "logging ready", slog.String("source", string(result.Source)))
```

只有整个进程都应通过 goark-log 路由 `slog.Default()` 时，才使用 `ConfigureDefault`。

## 生产配置

从 [examples/production-service.yml](examples/production-service.yml) 开始。该配置使用：

- stderr pattern console 输出，用于本地诊断。
- async appender 包裹 rolling JSON 文件。
- 使用 JSON Template layout 和 `flushOnWrite` 的审计滚动文件。
- root string-match filter 过滤健康检查噪声。
- `goark.audit` 命名 logger，并设置 `additivity=false`。
- `goark.demo.sql` 命名 logger，级别为 `DEBUG`。
- size、daily time、startup rollover、gzip、max backups、max age 和 delete action。
- `monitorInterval: 30s` 进行配置轮询重载。

## 异步选择

当几乎所有 logger 都应该进入同一个异步队列时，使用 handler 层 `asyncLogger`。它对业务代码侵入最低，但 reload 时不能改变运行期队列形态。

当只有特定目标需要队列，或需要把 failover/routing/rewrite 链作为一个下游 sink 排队时，使用 appender 层 `type: async`。

队列满策略：

| 策略 | 适用场景 |
| --- | --- |
| `block` | 日志不能丢，允许调用方承受背压。 |
| `drop` | 压力下允许丢弃新事件。 |
| `drop-debug` | debug 及以下事件可丢弃，更高级别阻塞。 |
| `sync-fallback` | 队列满时同步写出当前事件。 |

等待策略：

| 策略 | 适用场景 |
| --- | --- |
| `block` | 通用生产默认值。 |
| `yield` | CPU 有余量时追求低延迟。 |
| `sleep` | 用可选 `sleepTime` 降低 CPU 压力。 |
| `spin` | 极低延迟但消耗 CPU，只能在实测后使用。 |

## 滚动和保留

写文件的服务建议使用 `rolling-file`：

- `fileName` 是活动文件。
- 启用 size rollover 时，`filePattern` 需要包含 `%d{...}` 和 `%i`。
- `policies.size.size` 控制 size rollover。
- `policies.time.interval` 和 `modulate` 控制 daily/hourly 对齐。
- `strategy.max` 或 `strategy.maxBackups` 控制保留数量。
- `strategy.maxAge` 控制保留时间。
- `strategy.delete` 清理活动 pattern 之外的归档目录。
- `strategy.compression.gzip: true` 压缩归档。
- 只有归档动作允许后台执行时，才启用 `strategy.compression.async: true` 或 `asyncActions: true`。

`directWrite` 必须配置 `filePattern`，且不支持 gzip 压缩。

## 审计日志

审计日志建议使用独立命名 logger，例如 `goark.audit`，并设置 `additivity=false`。
审计文件包含敏感数据时，设置 `flushOnWrite: true` 和类似 `0600` 的严格
`filePermissions`。使用 `jsonTemplate` 控制下游审计流水线消费的字段名。

## Context 数据

请求级数据使用 context API：

```go
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", traceID),
	slog.String("tenant", tenantID),
)
ctx = goarklog.WithThreadName(ctx, "http-worker-1")
ctx = goarklog.WithContextStack(ctx, "request", "checkout")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
```

事件级数据继续使用普通 `slog.Attr` 参数。

## Reload 规则

只有满足以下条件时，`LoggerContext` 才会启动 reload 轮询：

- 配置加载结果有非空路径。
- `monitorInterval` 能解析为正数 duration。

`Reload` 会先完整构建新配置，成功后再原子替换 router。失败时旧运行期继续生效。
Handler 层 async 的 enablement、queue size、batch size、overflow strategy、
wait strategy、wait options 和 include-location 设置不能在 reload 时改变。

## 关闭

必须调用 `Close`。关闭会 drain handler 层异步队列、async appender、rolling
归档动作、layout footer、文件 buffer 和 appender 拥有的文件句柄。Router 关闭时先关
async appender，再关其他 appender，并按 appender name 避免重复关闭。

## 容器

容器平台以 stdout 为日志传输通道时，优先使用
[examples/container-json.yml](examples/container-json.yml)。它将 direct JSON 写入 stdout，并使用
`drop-debug` 在压力下丢弃高频 debug 流量。

## 安全默认值

- lookup namespace `jndi`、`ldap`、`rmi` 被阻止。
- 核心没有内嵌脚本运行时。`ScriptFilter` 需要调用方通过代码提供 evaluator。
- 核心没有远程 appender。网络 sink 应作为显式外部模块实现。
- 文件 appender 会校验目标路径不是目录，并按需创建父目录。
