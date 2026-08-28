# 能力边界

[English](capabilities.md)

本文记录 `goark.dev/log` core module 当前提供什么、什么故意不放入 core，以及发布前应该使用哪些验证关卡。

## 设计原则

- Go-native API：显式 construction、interfaces、options 和 plugin registration。
- 低分配热路径：常见 JSON、file、direct native logging 和 ring-buffer paths 避免 reflection-heavy work。
- Core 依赖轻量：comparison dependencies 保持在独立 `benchmarks/compare` module。
- 确定性 shutdown：async logger、async appender、rolling compression 和 delete actions 都在 `Close` 时 drain。
- 安全默认值：core 不提供 remote lookup、不嵌入 script engine、不内置 external-system appender，也不内置 observability exporter。

## Core 已支持

| 范围 | 状态 | 说明 |
| --- | --- | --- |
| `slog.Handler` | supported | 实现 `Enabled`、`Handle`、`WithAttrs` 和 `WithGroup`；可安装为 `slog.Default()`。 |
| Native logger | supported | `NewNativeLogger`、`LogAttrs`、`LogAttrs3`、builder API 和 message factories。 |
| Logger routing | supported | Root logger、named rules、prefix matching、additivity、appender-ref level/filter 和 includeLocation。 |
| Custom levels | supported | 内置 `ALL`、`TRACE`、`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`、`OFF` 和 `RegisterLevel`。 |
| Context attributes | supported | `WithContextAttrs`、`ContextAttrs`、MDC/NDC-style layout output。 |
| Marker and throwable | supported | Context marker、marker attr、throwable attr 和 optional throwable stack snapshots。 |
| Async logger | supported | Bounded ring buffer、batch drain、block/drop/drop-debug/sync-fallback、wait strategies、counters 和 close drain。 |
| Async appender | supported | 带 bounded queue、batch drain、counters、error handler 和 close drain 的 appender wrapper。 |
| File appender | supported | Append/truncate、create-on-demand、permissions、buffering 和 flush-on-write。 |
| JSON appender | supported | Direct single-line JSON 到 stdout/stderr 或 file；file mode 支持 buffering。 |
| Rolling file appender | supported | Size/time/cron/startup triggers、`%d`/`%i`、gzip、max count、max age、delete actions 和 async action worker。 |
| PatternLayout | supported | Time、level、logger、message、attrs、MDC、marker、NDC、caller、throwable、host、sequence、ANSI style 和 nested converters。 |
| Structured layouts | supported | JSON、JSON Template、XML、CSV、GELF、RFC5424/Syslog、YAML 和 HTML。 |
| Filters | supported | Level、range、regex、attrs、marker、MDC、structured data、throwable、time windows、burst limiter 和 dynamic thresholds。 |
| Composite appenders | supported | Async、Failover、Routing 和 Rewrite 可通过配置构建。 |
| Lookups | supported local subset | `env`、`sys`、`go`、`date`、`prop` 和 `property`。 |
| Configuration formats | supported | YAML、JSON、XML、properties；TOML 显式失败。 |
| Reload | supported with constraints | 轮询实际配置文件；async logger queue/runtime shape 不能 hot-replace。 |
| Plugins | supported | `PluginRegistry`、`PluginRegistrar`、`PluginSet`、package helpers、lookup plugins、JSON Template resolvers 和 registrar generator。 |

## 不在 Core

| 范围 | 原因 | 推荐方式 |
| --- | --- | --- |
| HTTP appender | Connection lifecycle、retry、timeout、TLS 和 response handling 因部署而异。 | 构建外部模块并注册 appender plugin。 |
| Socket appender | Framing、reconnect、backpressure 和 protocol choices 差异很大。 | 构建外部模块。 |
| Syslog network appender | Transport、TLS、facility/app name mapping 和 retry policy 与环境强相关。 | 构建外部模块；core 只提供 RFC5424/Syslog layout。 |
| Kafka, Pulsar, RabbitMQ | Broker clients 和 delivery semantics 是重依赖。 | 保持在专门的 Goark integration modules。 |
| SMTP appender | Slow network I/O 和 credential handling 不属于 core hot path。 | 构建带 explicit queue 和 retry behavior 的 plugin module。 |
| Database appender | Schema、transactions、batching 和 failure handling 都是 database-specific。 | 构建 database-specific plugin module。 |
| OpenTelemetry, Prometheus | Observability design 应在 Goark modules 间一致，并保持可选。 | 独立 observability design 后再添加。 |
| Script runtime | JavaScript/Lua/expr/Starlark runtime 和 sandbox 决策有安全影响。 | Core 只暴露 `ScriptEvaluator` API。 |
| Remote lookup namespaces | JNDI/LDAP/RMI-style lookups 对 config-time resolution 不安全。 | 默认阻止。 |
| Runtime plugin scanning | Scanning 增加隐式 startup behavior 和成本。 | 使用 explicit registration 或 generated registrars。 |

## Dependency Boundary

Core `go.mod` 依赖限制为：

- `github.com/bytedance/sonic`
- `gopkg.in/yaml.v3`

Zap 和 zerolog comparison dependencies 位于 `benchmarks/compare/go.mod`，不得移入 core module。

## 验证关卡

短关卡：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./... ./cmd/goark-log-plugin-gen ./internal/disruptor ./internal/jsoncodec
```

Focused hot-path benchmark：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Long stress gate：

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

Independent comparison module：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=5s -count=3 -cpu=1,4,16
```

## 发布边界

发布 tag 前确认：

- `dev` 包含所有预期变更。
- `main` 通过批准的 release flow 更新。
- Core tests 和 compare-module tests 通过。
- Race 和 stress checks 已执行，或明确记录为 deferred。
- Benchmark paths 在 benchmark package split 后对 core benchmark 使用 `./benchmarks/core`。
- README 和 docs 没有声称支持未内置的 external appenders 或 observability exporters。
