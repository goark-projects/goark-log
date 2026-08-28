# 使用场景

[English](scenarios.md)

本文给出完整、偏生产的日志使用场景。可复制配置文件位于 [examples](examples/README.zh-CN.md)。

## 场景 1：开发 Console

适合本地开发，日志主要由人直接阅读。

配置：[examples/console.yml](examples/console.yml)

```yaml
configuration:
  properties:
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %highlight{%-5p} %pid --- [%thread] %c : %m%attrs%n"
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
        disableAnsi: false
  root:
    level: debug
    appenderRefs: [console]
```

推荐用于：

- 本地命令行开发；
- 短生命周期工具；
- 需要彩色 level 帮助定位的 debug session。

避免用于：

- 高吞吐生产 JSON 采集；
- ANSI 渲染不好的 Windows console；这类环境设置 `disableAnsi: true`。

## 场景 2：容器 JSON 到 stdout

适合 Docker、Kubernetes、Nomad，以及 stdout 由运行平台采集的环境。

配置：[examples/json-stdout.yml](examples/json-stdout.yml)

```yaml
configuration:
  appenders:
    json:
      type: json
      target: stdout
  root:
    level: info
    appenderRefs: [json]
```

推荐 runtime pattern：

```go
handler, _, err := goarklog.ConfigureDefault(context.Background(),
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer handler.Close()
```

运行说明：

- 容器里 application events 优先使用 `target: stdout`。
- 内部平台诊断通常交给 service runtime 写 stderr，不要再用第二套 default logger，除非采集器要求。
- 最低 overhead 的固定结构化输出使用 direct JSON appender。

## 场景 3：生产 Rolling JSON Files

适合 VM 或 bare-metal 服务，本地文件是主要日志交接边界。

配置：[examples/production-rolling.yml](examples/production-rolling.yml)

关键选择：

- `asyncLogger.enabled: true` 将业务 goroutines 与磁盘写入解耦。
- `overflowStrategy: block` 通过 backpressure 保留日志。
- `bufferSize: 256KiB` 降低 write syscalls。
- 启用 size rolling 时，`filePattern` 必须包含 `%d` 和 `%i`。
- `.gz` archive suffix 启用 gzip compression。
- `compression.async: true` 使用单个 background worker 串行执行 compression 和 deletion。
- 必须调用 `Close` flush buffers 并完成 queued rolling actions。

按 workload 调优：

| Workload | 推荐调整 |
| --- | --- |
| Latency-sensitive 且允许丢日志 | 使用 `overflowStrategy: drop` 或 `drop-debug`，并监控 `AsyncDropped`。 |
| Audit 或 billing logs | 保持 `overflowStrategy: block`，仅对 audit sink 使用 `flushOnWrite`。 |
| Slow disk | 增大 `queueSize` 和 `batchSize`，但保留足够内存余量。 |
| 需要 caller fields | 只在目标 logger 或 appender ref 上启用 `includeLocation`。 |

## 场景 4：拆分应用日志和审计日志

适合 audit events 需要不同 retention、permissions 和 schema 的场景。

配置：[examples/split-audit.yml](examples/split-audit.yml)

Logger contract：

```go
audit := slog.New(handler).With("goark.logger", "goark.audit")
audit.InfoContext(ctx, "user permission changed",
	slog.String("principal", "alice"),
	slog.String("action", "grant"),
	slog.String("resource", "project:42"),
)
```

`goark.audit` logger 设置了 `additivity: false`，因此 audit events 只写入 audit appender，不会重复写到 root application log。

运行说明：

- 当 audit data 包含 sensitive principals 或 resource identifiers 时，`filePermissions: "0600"` 更合适。
- Audit layout 应保持稳定且显式。下游 compliance tooling 期待固定字段时，JSON Template 优先于 generic JSON。
- `flushOnWrite: true` 提升 durability 但降低 throughput；只应用到 audit，不要扩散到所有 app logs。

## 场景 5：Appender-Level Async

整个 logging pipeline 都要异步时使用 Handler-level async。只有某个 sink 需要异步时使用 Appender-level async。

配置：[examples/async-appender.yml](examples/async-appender.yml)

```yaml
appenders:
  jsonFile:
    type: json
    fileName: logs/app.json
    bufferSize: 256KiB
  asyncJson:
    type: async
    appenderRefs: [jsonFile]
    queueSize: 4096
    batchSize: 128
    overflowStrategy: block
    waitStrategy: yield
root:
  level: info
  appenderRefs: [asyncJson]
```

适用场景：

- console output 保持同步，但 file output 要异步；
- slow rolling file 不应阻塞每条 appender path；
- 正在组合 failover/routing/rewrite appenders，并只想让特定 delegate boundary 异步。

不要盲目给每个 appender 包一层 async。多层 async 会增加 queueing 和 shutdown complexity。

## 场景 6：按 Tenant Routing 并脱敏 Attributes

Routing 适合小规模、有界、已知的 output routes。

配置：[examples/rewrite-routing.yml](examples/rewrite-routing.yml)

```go
logger.InfoContext(ctx, "payment accepted",
	slog.String("tenant", "tenant-a"),
	slog.String("order_id", "ord-100"),
	slog.String("token", "secret"),
)
```

内置 rewrite appender 会移除 `token`、`password` 和 `authorization`，添加 `service=billing`，然后委托给 routing appender。Routing appender 读取 `tenant`，写入 tenant-specific file 或 stdout fallback。

建议：

- 保持 route cardinality 有界。不要按 user ID 或 request ID 做 routing。
- 缺失或未知值路由到 `defaultRoute`。
- Rewrite 是最后一层防线，不应替代调用点避免写入敏感值的责任。

## 场景 7：Config Reload

服务需要不重启 reload logging config 时使用 `NewConfiguredLoggerContext`。

```go
ctx := context.Background()
logging, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer logging.Close()

logger := logging.Logger("goark.service")
logger.Info("logging started", slog.String("source", string(result.Source)))
```

在文件中设置 `monitorInterval`：

```yaml
configuration:
  monitorInterval: 30s
  appenders:
    console:
      type: console
  root:
    level: info
    appenderRefs: [console]
```

Reload 可改变 log levels、filters、routes、appenders 和 layouts。Reload 不能改变 Handler-level async runtime settings。要改变 async enablement、queue size、batch size、overflow strategy、wait strategy、wait options 或 async caller location，需要重启 logger context。

## 场景 8：MDC、Trace IDs、Marker 和 Context Stack

Go 没有 Java-style thread locals。使用 `context.Context`。

```go
ctx := context.Background()
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", "trace-100"),
	slog.String("span_id", "span-200"),
)
ctx = goarklog.WithThreadName(ctx, "http-worker-1")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
ctx = goarklog.WithContextStack(ctx, "tenant-a", "checkout")

logger.InfoContext(ctx, "request done",
	slog.String("method", "GET"),
	slog.Int("status", 200),
)
```

Pattern 示例：

```text
%d %-5p [%thread] %c trace=%X{trace_id} marker=%marker ndc=%ndc %m%attrs%n
```

JSON Template 示例：

```json
{
  "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
  "level": {"$resolver": "level"},
  "logger": {"$resolver": "logger"},
  "traceId": {"$resolver": "attr", "key": "trace_id"},
  "spanId": {"$resolver": "attr", "key": "span_id"},
  "marker": {"$resolver": "marker"},
  "stack": {"$resolver": "contextStack"},
  "attrs": {"$resolver": "mdc"}
}
```

## 场景 9：只为窄 Logger 启用 Caller Location

Caller lookup 不是免费操作。只在必要位置启用。

```yaml
appenders:
  file:
    type: file
    fileName: logs/debug.log
    layout:
      type: pattern
      pattern: "%d %-5p %c %F:%L %M - %m%attrs%n"
root:
  level: info
  appenderRefs: [file]
loggers:
  goark.debug:
    level: debug
    includeLocation: true
    appenderRefs: [file]
    additivity: false
```

Native logger：

```go
debug, err := goarklog.NewNativeLogger(handler, "goark.debug",
	goarklog.WithLoggerCaller(true),
)
if err != nil {
	return err
}
_ = debug.DebugContext(ctx, "debug point")
```

## 场景 10：Failover 到 Console

当 preferred sink 可能失败但 event 不能丢时使用 failover。

```yaml
appenders:
  primary:
    type: file
    fileName: /var/log/my-service/app.log
  fallback:
    type: console
    target: stderr
  failover:
    type: failover
    primary: primary
    failovers: [fallback]
root:
  level: info
  appenderRefs: [failover]
```

Failovers 应保持简单。可能无限阻塞的 failover target 会破坏 fallback path 的意义。

## 场景 11：Legacy Deployment 的 Properties Config

当部署工具已经渲染 Java-style properties 时使用 properties。

配置：[examples/goark-log.properties](examples/goark-log.properties)

```properties
property.LOG_DIR=logs
appender.console.type=console
appender.console.target=stderr
appender.console.layout.type=pattern
appender.console.layout.pattern=%d %5p %pid --- [%thread] %c : %m%attrs%n
rootLogger.level=info
rootLogger.appenderRefs=console
```

不要在 properties 中使用 YAML nesting syntax。使用 [配置参考](configuration.zh-CN.md) 中记录的 flat key prefixes。

## 场景 12：Log4j2-Style 迁移的 XML Config

团队从 Log4j2 迁移并希望保留熟悉文件形状时使用 XML。

配置：[examples/log4j2-style.xml](examples/log4j2-style.xml)

与 Log4j2 的 core differences：

- runtime 是 Go-native 且 explicit；没有 classpath scanning。
- HTTP、Socket 和 Syslog network appenders 需要外部插件模块。
- Script filters 需要调用方提供 evaluator；core 不嵌入 script engine。
- Caller location 为性能考虑默认关闭，需要 opt-in。

## 场景 13：Programmatic Construction

Programmatic construction 拥有最显式的 ownership，适合 tests、embedded services，或已经有自身配置系统的 frameworks。

```go
layout := goarklog.NewJSONLayout(goarklog.LayoutOptions{EventEOL: true})
file, err := goarklog.NewFileAppender("logs/app.json",
	goarklog.WithFileLayout(layout),
	goarklog.WithFileBufferSize(256*1024),
)
if err != nil {
	return err
}

handler, err := goarklog.NewHandler(goarklog.Options{
	Appenders: []goarklog.Appender{file},
	Root: goarklog.RootLogger{
		Level:        slog.LevelInfo,
		AppenderRefs: []string{"file"},
	},
})
if err != nil {
	_ = file.Close()
	return err
}
defer handler.Close()
```

规则：

- 调用方负责 appender construction errors。
- `NewHandler` 成功后拥有 appender shutdown。
- 如果 `NewHandler` 在手动创建 appenders 后失败，调用方自行关闭这些 appenders。

## 场景 14：低分配 Native Logging

热路径固定字段使用 native logger。

```go
logger, err := goarklog.NewNativeLogger(handler, "goark.http")
if err != nil {
	return err
}

if logger.Enabled(ctx, slog.LevelInfo) {
	_ = logger.LogAttrs3(ctx, slog.LevelInfo, "request done",
		slog.String("method", method),
		slog.Int("status", status),
		slog.Duration("elapsed", elapsed),
	)
}
```

建议：

- `LogAttrs3` 是固定三属性快速路径。
- Attribute 数量动态时使用 `LogAttrs`。
- Fluent construction 更清晰时使用 `AtInfo().WithString(...).Log(...)`。
- 在最热 JSON 路径避免 `slog.Any`，除非确实需要复杂 payload。

## 场景 15：Custom Plugin

服务或模块增加 custom appender、layout、filter、lookup 或 JSON Template resolver 时，使用显式 plugin registration。

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginLookup("tenant", lookupTenant),
	goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
))
if err != nil {
	return err
}

handler, _, err := goarklog.NewConfiguredHandler(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

Plugin factories 应验证所有 required fields，保持 ownership explicit，并避免 global mutable state，除非 lifecycle 天然就是 process-wide。
