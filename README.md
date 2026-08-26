# goark-log

`goark-log` 是 `log/slog` 的 Goark 日志实现，提供并发安全的 `slog.Handler`、Appender、Layout、Logger 层级和配置加载。

默认输出使用 Spring Boot 风格：

```text
2026-08-25T10:15:30.123+08:00  INFO 12345 --- [main] goark.boot : service started profile=dev
```

## 快速使用

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	handler, _, err := goarklog.ConfigureDefault(context.Background())
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger := goarklog.WithName(slog.Default(), "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

延迟敏感路径可以使用原生 `Logger` API，直接把 `slog.Attr` 写入 goark-log 事件管线：

```go
native, err := goarklog.NewNativeLogger(handler, "goark.boot")
if err != nil {
	panic(err)
}
_ = native.Info("service started", slog.String("profile", "dev"))
_ = native.AtInfo().
	WithMarker(goarklog.NewMarker("BOOT")).
	WithString("profile", "dev").
	Log("service started")
```

`Logger.AtInfo()/AtDebug()/AtWarn()/AtError()/AtTrace()` 提供 Log4j2 风格 fluent builder；`SimpleMessage`、`ParameterizedMessage`、`MapMessage`、`StructuredDataMessage` 和 `MessageFactory` 提供 Go-native 消息模型。需要自定义级别时使用 `RegisterLevel("NOTICE", slog.Level(2))`，布局输出会优先使用注册名称。

## 配置优先级

1. 显式路径：`goarklog.WithConfigPath(...)`
2. 环境变量：默认 `GOARK_LOG_CONFIG`
3. boot 配置：`goark.log.config`、`goark.logging.config`、`logging.config`
4. 默认文件：`conf/goark-log.yml`、`conf/goark-log.yaml`、`conf/goark-log.json`、`conf/goark-log.xml`、`conf/goark-log.toml`、`conf/goark-log.properties`
5. 内置默认：`stderr` console，`INFO`

配置支持 YAML、JSON、XML 和 properties。TOML 仍会明确报错，避免误以为配置已生效。

## YAML 示例

```yaml
configuration:
  status: warn
  properties:
    LOG_DIR: logs
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %pid --- [%thread] %c : %m%attrs%n"
  asyncLogger:
    enabled: true
    queueSize: 8192
    batchSize: 128
    overflowStrategy: block
    waitStrategy: yield
    includeLocation: false
  filters:
    keep-info:
      type: threshold
      level: info
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
    rolling:
      type: rolling-file
      fileName: "${prop:LOG_DIR}/app.log"
      bufferSize: 256KiB
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%d{yyyyMMdd}-%i.log.gz"
        policies:
          size:
            size: 100MiB
          time:
            interval: daily
            modulate: true
          startup:
            enabled: true
        strategy:
          max: 30
          maxAge: 30d
          compression:
            gzip: true
            async: true
          delete:
            basePath: "${prop:LOG_DIR}/archive"
            maxDepth: 1
            ifFileName:
              glob: "*.log.gz"
            ifLastModified:
              age: 30d
            async: true
  root:
    level: info
    appenderRefs: [console, rolling]
    filters: [keep-info]
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
```

也可以使用顶层字段，或放在 `goark.log` 下方便与 boot 主配置合并。`configuration`、顶层字段、`goark.log` 三种形式只能选一种，避免配置歧义。

`rolling.maxSize`、`rolling.interval`、`rolling.onStartup`、`rolling.maxBackups`、`rolling.maxAge`、`rolling.gzip` 这些第一版字段继续兼容。新配置建议使用
`rolling.policies` 和 `rolling.strategy`：`policies` 组合大小、时间、启动触发策略，`strategy` 控制保留数量、保留时长、gzip 压缩和删除动作。`strategy.compression.async` 或
`strategy.delete.async` 为 `true` 时，压缩和清理会进入后台串行动作队列，`Close` 会等待队列清空后返回。

```yaml
goark:
  log:
    appenders:
      console:
        type: console
    root:
      level: info
      appenderRefs: [console]
```

## Reload

```go
reloader, err := goarklog.NewConfigReloader(handler, goarklog.WithConfigPath("conf/goark-log.yml"))
if err != nil {
	panic(err)
}
_, err = reloader.Reload(context.Background())
```

`Watch` 使用标准库轮询文件修改时间和大小，不依赖 `fsnotify`。
在 `NewConfiguredLoggerContext` 中使用配置文件时，`monitorInterval` 大于 0 会自动启动轮询 reload；纯数字按 Log4j2 习惯表示秒，带单位值按 Go duration 解析。

## MDC 和调用位置

Go 没有 Java 线程局部变量，`goark-log` 用 `context.Context` 承载 MDC：

```go
ctx := goarklog.WithContextAttrs(context.Background(),
	slog.String("trace_id", "trace-1"),
	slog.String("span_id", "span-1"),
)
logger.InfoContext(ctx, "request done")
```

`PatternLayout` 支持 `%X{trace_id}`、`%mdc{trace_id}`、`%marker`、`%class`、`%method`、`%file`、`%line`、`%location`、`%logger{2}`、`%map`、`%uuid`、`%sn`、`%replace{...}{...}{...}`、`%enc/%encode{...}{json|html|xml|crlf}`、`%equals{...}{...}{...}`、`%equalsIgnoreCase{...}{...}{...}`、`%maxLen{...}{n}`、`%repeat{...}{n}`、`%highlight{...}`、`%style{...}{...}`、`%notEmpty{...}`。调用位置来自 `slog.Record.PC` 或 `NewNativeLogger(..., WithLoggerCaller(true))`；异步 logger 也可以通过 `asyncLogger.includeLocation: true` 在入队前采集 caller。

`JSONTemplateLayout` 可通过 `layout.type: jsonTemplate` 启用，支持 `$resolver` 风格字段：`timestamp`、`level`、`logger`、`message`、`thread`、`threadName`、`marker`、`throwable`、`rootCause`、`stackTrace`、`source`、`location`、`process`、`contextStack`、`mdc`、`attr`、`endOfBatch`。模板可以内联写在 `eventTemplate`，也可以用 `eventTemplateUri`/`eventTemplatePath` 指向本地文件；核心库拒绝远程模板 URI。`level.field` 支持 `name/value/severity`，`logger.precision` 支持名称精度截断，`mdc` 支持 `flatten: true` 和 `propertiesAsList` 列表形态，自定义 resolver 通过 `RegisterJSONTemplateResolver` 注册。

核心库还提供 `XMLLayout`、`CSVLayout`、`GELFLayout`、`RFC5424Layout`、`YAMLLayout` 和 `HTMLLayout`，配置中可使用 `layout.type: xml/csv/gelf/rfc5424/syslog/yaml/html`。JSON、JSONTemplate、XML、CSV、GELF、YAML、HTML 支持通用 `LayoutOptions`：`compact`、`eventEol`、`complete`、`includeStacktrace`、`stacktraceAsString`、`propertiesAsList`、`includeNullDelimiter`、`header`、`footer`。

## 过滤器

内置过滤器支持 `ThresholdFilter`、`LevelFilter`、`LevelRangeFilter`、`RegexFilter`、`AttrFilter`、`DenyAllFilter`、`CompositeFilter`、`MarkerFilter`、`NoMarkerFilter`、`MapFilter`、`ThreadContextMapFilter`、`ThreadContextStackFilter`、`StructuredDataFilter`、`ThrowableFilter`、`StringMatchFilter`、`TimeFilter`、`BurstFilter`、`DynamicThresholdFilter`。配置里可以使用短名，也可以使用 Log4j2 风格的 `*Filter` 类型名。

`MapFilter`、`ThreadContextMapFilter` 和 `DynamicThresholdFilter` 支持 `KeyValuePair` 子项；YAML/JSON 也可以用 `values`、`thresholds` 显式 map，properties 可用 `filter.<name>.values.<key>`、`filter.<name>.thresholds.<value>` 或 `filter.<name>.keyValuePair0.key/value`。
`TimeFilter` 支持 `timezone`，没有配置时使用事件时间自身的时区。
`CompositeFilter` 使用 `filters`/`filterRefs` 引用已命名过滤器，构建时会检测循环引用。

`ScriptFilter` 只提供 `ScriptEvaluator` 契约和安全默认策略，核心库不内置脚本引擎。JavaScript、Lua、expr、Starlark 等执行器应放在独立模块中注册，避免核心库绑定脚本运行时和安全策略。

## 安全边界

核心 lookup 默认只启用 `env`、`sys`、`go`、`date`。出于安全边界考虑，`jndi`、`ldap`、`rmi` 这类远程解析 namespace 会被拒绝或忽略，不做 Log4j2 历史高风险能力的机械复刻。

## 外部 Appender

核心库只提供 Appender 接口、插件注册表和配置构建契约，不内置 HTTP、Socket、Syslog、Kafka、SMTP、Database 等外部系统实现。外部系统连接必须放在独立模块中，按需显式导入并注册：

```go
registry := goarklog.NewPluginRegistry()
_ = externalappender.Register(registry)
handler, _, err := goarklog.NewConfiguredHandler(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

核心包保留 `AppenderBuildConfig` 中的 URL、Address、Network、Timeout 等字段，用于独立外部模块读取配置；这些字段不是核心库自带外部 appender 的承诺。核心包自身只提供 `ConsoleAppender`、`FileAppender`、`RollingFileAppender`、`AsyncAppender`、`FailoverAppender`、`RoutingAppender`、`RewriteAppender` 等本地和组合型 appender。

外部包如果只扩展默认注册表，可以直接使用包级 helper：`RegisterAppender`、`RegisterLayout`、`RegisterFilter`、`RegisterLookup`、`RegisterJSONTemplateResolver`、`RegisterPlugins`。需要隔离测试或多套配置时，继续使用 `NewPluginRegistry()` 显式传入。独立外部模块推荐暴露 `PluginRegistrar`，也可以用 `NewPluginSet` 声明注册项；核心库只负责调用 registrar，不负责发现或加载外部包。

```go
func Registrar() goarklog.PluginRegistrar {
	return goarklog.NewPluginSet(
		goarklog.WithPluginAppender("http", buildHTTPAppender),
		goarklog.WithPluginJSONTemplateResolver("trace", buildTraceResolver),
	)
}
```

外部仓库可以用生成器生成 registrar 样板，factory 函数仍由外部仓库自己实现：

```go
//go:generate go run goark.dev/log/cmd/goark-log-plugin-gen -package httpappender -appender http=buildHTTPAppender -out zz_generated_plugins.go
```

## Examples

示例位于 `examples/`：

- `examples/console`：默认 Spring Boot 风格 console
- `examples/file`：普通文件 appender
- `examples/rolling`：大小滚动、启动滚动和 gzip
- `examples/async`：async 包装 rolling appender
- `examples/reload`：配置 reload
- `examples/extensibility`：`PluginRegistrar`、自定义 JSON Template resolver、MessageFactory

```bash
go test ./examples/...
go run ./examples/console
```

## Benchmark

```bash
go test -run '^$' -bench . -benchmem ./...
```

高吞吐路径建议优先使用 `slog.Logger.LogAttrs`，它比 `Info(args ...any)` 少一次
variadic 装箱开销。内置 `TextLayout`、`JSONLayout`、默认 `PatternLayout` 对常见
`slog` 基础类型保持低分配编码；文件类 appender 默认启用缓冲写入，延迟敏感场景再打开
`flushOnWrite`。

更高吞吐场景可以使用 `NewNativeLogger` 创建原生 logger，绕过 `slog.Record` 构造并直接进入 goark-log `Event` 管线。默认不采集调用位置，需要 `%class`、`%method`、`%file`、`%line` 或 `%location` 时显式传入 `WithLoggerCaller(true)`。

当前 JSON 热路径采用手写 `bytes.Buffer` 编码，常见 `slog` 基础类型保持 `0 alloc/op`。复杂 `slog.Any` 对象 fallback 使用 ByteDance Sonic 编码；基础字段不走反射 JSON，避免破坏主路径分配表现。

和 zap、zerolog 的对标基准放在独立子模块，避免核心库引入额外依赖：

```bash
cd benchmarks/compare
go test -run '^$' -bench . -benchmem
```

更多对标边界和性能预算见：

- `docs/log4j2-parity.md`
- `docs/performance.md`
