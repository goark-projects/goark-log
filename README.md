# goark-log

`goark-log` 是面向 Go 服务的高性能结构化日志库，基于标准库 `log/slog`，提供并发安全的 `Handler`、低分配原生 `Logger`、Appender、Layout、层级路由、配置加载、异步队列和滚动文件能力。

核心目标很明确：

- 保持 Go-native API，优先使用显式构造、显式配置和显式注册。
- 热路径低分配，常见 JSON 和文件写入路径避免反射编码。
- 核心依赖轻量，复杂 JSON fallback 使用 ByteDance Sonic。
- 核心仓库只提供本地输出和组合型输出，不内置外部系统 appender，不内置观测导出。

## 快速开始

```go
package main

import (
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	logger, handler := goarklog.NewDefault()
	defer handler.Close()

	logger = goarklog.WithName(logger, "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

默认输出是可读的单行文本：

```text
2026-08-25T10:15:30.123+08:00  INFO 12345 --- [main] goark.boot : service started profile=dev
```

需要从配置文件加载时：

```go
handler, result, err := goarklog.ConfigureDefault(context.Background(),
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	panic(err)
}
defer handler.Close()

slog.Info("configured", slog.String("source", string(result.Source)))
```

延迟敏感路径可以使用原生 `Logger`，直接把 `slog.Attr` 写入事件管线：

```go
native, err := goarklog.NewNativeLogger(handler, "goark.http")
if err != nil {
	panic(err)
}
_ = native.LogAttrs3(context.Background(), slog.LevelInfo, "request done",
	slog.String("method", "GET"),
	slog.Int("status", 200),
	slog.Duration("elapsed", 8*time.Millisecond),
)
```

## 功能概览

| 范围 | 当前能力 |
| --- | --- |
| 标准库集成 | `slog.Handler`、`slog.Logger`、`LogAttrs`、`WithAttrs`、`WithGroup`。 |
| 原生入口 | `NewNativeLogger`、固定三属性 `LogAttrs3`、fluent builder、消息工厂。 |
| 层级路由 | root、命名 logger、additivity、appenderRef 级别和过滤器。 |
| 输出端 | Console、File、RollingFile、JSONFile、Async、Failover、Routing、Rewrite。 |
| 异步管线 | 有界 ring buffer、批量 drain、block/drop/drop-debug/sync-fallback、wait strategy、关闭 drain。 |
| 滚动文件 | size/time/cron/startup、`%d/%i`、gzip、max/maxAge、delete action、后台串行动作队列。 |
| 布局 | Pattern、Text、JSON、JSONTemplate、XML、CSV、GELF、RFC5424/YAML/HTML。 |
| 过滤器 | Threshold、Level、LevelRange、Regex、Attr、Marker、Map、Throwable、Time、Burst、DynamicThreshold 等。 |
| 配置格式 | YAML、JSON、XML、properties；TOML 明确报错。 |
| Reload | 文件轮询 reload，异步队列结构不热替换。 |
| 扩展点 | PluginRegistry、PluginRegistrar、PluginSet、JSON Template resolver、插件生成器。 |

更完整的能力边界见 [docs/capabilities.md](docs/capabilities.md)。

## 配置

配置加载优先级：

1. 显式路径：`goarklog.WithConfigPath(...)`
2. 环境变量：默认 `GOARK_LOG_CONFIG`
3. boot 配置：`goark.log.config`、`goark.logging.config`、`logging.config`
4. 默认文件：`conf/goark-log.yml`、`conf/goark-log.yaml`、`conf/goark-log.json`、`conf/goark-log.xml`、`conf/goark-log.toml`、`conf/goark-log.properties`
5. 内置默认：`stderr` console，`INFO`

配置支持 YAML、JSON、XML 和 properties。TOML 会明确报错，避免误以为配置已生效。

生产配置建议从 YAML 开始：

```yaml
configuration:
  status: warn
  monitorInterval: 30s
  properties:
    LOG_DIR: logs
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %pid --- [%thread] %c : %m%attrs%n"
  asyncLogger:
    enabled: true
    queueSize: 8192
    batchSize: 256
    overflowStrategy: block
    waitStrategy: yield
    includeLocation: false
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
        disableAnsi: false
    rolling:
      type: rolling-file
      fileName: "${prop:LOG_DIR}/app.log"
      bufferSize: 256KiB
      layout:
        type: json
        eventEol: true
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
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
```

也可以把同样配置放在顶层，或放在 `goark.log` 下方便与 boot 主配置合并。`configuration`、顶层字段、`goark.log` 三种形式只能选一种，避免配置歧义。

## MDC 和调用位置

Go 没有线程局部变量，`goark-log` 用 `context.Context` 承载请求上下文属性：

```go
ctx := goarklog.WithContextAttrs(context.Background(),
	slog.String("trace_id", "trace-1"),
	slog.String("span_id", "span-1"),
)
logger.InfoContext(ctx, "request done")
```

调用位置默认不采集，避免热路径开销。需要 `%class`、`%method`、`%file`、`%line` 或 `%location` 时，显式开启：

```go
native, err := goarklog.NewNativeLogger(handler, "goark.http", goarklog.WithLoggerCaller(true))
```

异步 logger 也可以通过 `asyncLogger.includeLocation: true` 在入队前采集 caller。

## JSON 热路径

JSON 主路径使用手写 `bytes.Buffer` 追加编码，常见 `slog` 基础类型保持低分配。复杂 `slog.Any` fallback 统一通过 `internal/jsoncodec` 调用 ByteDance Sonic：

```go
appender := goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(os.Stdout))
```

文件直写建议使用 `NewJSONFileAppender`，它绕过通用 Layout 调度，适合高吞吐结构化文件输出：

```go
appender, err := goarklog.NewJSONFileAppender("logs/app.json",
	goarklog.WithJSONAppenderBufferSize(256*1024),
)
```

## 异步和滚动文件

`AsyncLogger` 位于 Handler 层，适合把业务 goroutine 和实际写出解耦：

```go
handler, err := goarklog.NewHandler(goarklog.Options{
	Appenders: []goarklog.Appender{goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(os.Stdout))},
	Root:      goarklog.RootLogger{Level: slog.LevelInfo, AppenderRefs: []string{"json"}},
	Async: goarklog.AsyncLoggerOptions{
		Enabled:          true,
		QueueSize:        8192,
		BatchSize:        256,
		OverflowStrategy: goarklog.AsyncOverflowBlock,
		WaitStrategy:     goarklog.AsyncWaitYield,
	},
})
```

`RollingFileAppender` 支持按大小、时间、cron 和启动滚动。压缩和删除动作可以走后台串行队列，`Close` 会等待队列清空。

配置级 `type: async` appender 支持 `queueSize`、`batchSize`、`overflowStrategy` 和 `waitStrategy`，适合只把部分下游输出异步化。

## 过滤器和安全边界

内置过滤器覆盖级别、属性、正则、marker、MDC、异常、时间窗口和突发限流等常见场景。顶层 `filterRefs` 是全局 filter 链，先于 logger level 裁决；root/logger/appenderRef/appender 自身 filter 在对应阶段执行。

配置 lookup 默认只启用 `env`、`sys`、`go`、`date` 和 `property`。`jndi`、`ldap`、`rmi` 这类远程解析 namespace 被拒绝或忽略。

脚本过滤器只保留 `ScriptEvaluator` 契约，核心库不内置脚本引擎。

## 扩展点

核心库提供显式插件注册，不做运行时扫描：

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginLayout("plain", buildPlainLayout),
	goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
))
```

需要生成 registrar 样板时：

```bash
go run goark.dev/log/cmd/goark-log-plugin-gen -package mylog -layout plain=buildPlainLayout -out zz_generated_plugins.go
```

核心当前不内置 HTTP、Socket、Syslog、Kafka、SMTP、Database 等外部系统输出，也不内置 OpenTelemetry、Prometheus 等观测导出。后续如需扩展，应放在独立模块中显式注册。

## 示例

示例位于 [examples/](examples/)，说明见 [examples/README.md](examples/README.md)。

```bash
go test ./examples/...
go run ./examples/console
go run ./examples/rolling
go run ./examples/extensibility
```

## 验证和性能

常规验证：

```bash
go test ./...
go vet ./...
```

长压测默认不进入普通测试，需要显式打开：

```bash
GOARK_LOG_STRESS=1 go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

核心 benchmark：

```bash
go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

独立性能比较模块：

```bash
cd benchmarks/compare
go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

更多性能预算、CI 长测入口和样例数字见 [docs/performance.md](docs/performance.md)。
