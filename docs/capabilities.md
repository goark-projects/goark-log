# 能力边界

本文档记录 `goark.dev/log` 核心仓库当前提供的能力、刻意保留的边界，以及发布前需要关注的验证入口。

## 设计原则

- **Go-native**：公共 API 使用 Go 显式构造、接口和配置，不依赖运行时扫描。
- **低分配热路径**：常见 JSON、文件和异步队列路径优先手写编码和固定容量结构。
- **核心依赖轻量**：核心依赖保持在日志必需范围，性能比较依赖放在独立子模块。
- **关闭可证明**：异步 logger、异步 appender、滚动压缩和删除动作都必须在 `Close` 时 drain。
- **安全默认值**：危险远程 lookup、脚本引擎、外部系统连接和观测导出不进入核心仓库。

## 已支持能力

| 范围 | 状态 | 说明 |
| --- | --- | --- |
| `slog.Handler` | 已支持 | `Enabled`、`Handle`、`WithAttrs`、`WithGroup`，可安装为 `slog.Default()`。 |
| 原生 logger | 已支持 | `NewNativeLogger`、`LogAttrs`、`LogAttrs3`、fluent builder、消息工厂。 |
| Logger 路由 | 已支持 | root、命名 logger、additivity、appenderRef level/filter、includeLocation。 |
| 自定义级别 | 已支持 | `ALL/TRACE/DEBUG/INFO/WARN/ERROR/FATAL/OFF` 和 `RegisterLevel`。 |
| 上下文属性 | 已支持 | `WithContextAttrs`、`ContextAttrs`、MDC/NDC 风格布局输出。 |
| AsyncLogger | 已支持 | 有界 ring buffer、批量消费、block/drop/drop-debug/sync-fallback、等待策略和错误计数。 |
| AsyncAppender | 已支持 | appender 包装型异步输出，关闭时 drain，可配置是否关闭子 appender。 |
| FileAppender | 已支持 | append/truncate、createOnDemand、权限、buffer、flushOnWrite。 |
| JSONFileAppender | 已支持 | 文件 JSON 直写，面向低分配结构化日志主路径。 |
| RollingFileAppender | 已支持 | size/time/cron/startup、`%d/%i`、gzip、max/maxAge、delete action、异步动作队列。 |
| PatternLayout | 已支持 | 时间、级别、logger、message、attrs、MDC、marker、caller、异常、host、sequence、ANSI 样式等。 |
| 结构化 Layout | 已支持 | JSON、JSONTemplate、XML、CSV、GELF、RFC5424、YAML、HTML。 |
| Filters | 已支持 | 级别、范围、正则、属性、marker、map、上下文、结构化数据、异常、时间、突发限流等。 |
| 组合 appender | 已支持 | Async、Failover、Routing、Rewrite 均支持配置化构建。 |
| Lookups | 已支持安全子集 | `env`、`sys`、`go`、`date`、`property`。 |
| 配置格式 | 已支持 | YAML、JSON、XML、properties；TOML 明确拒绝。 |
| Reload | 已支持 | 轮询文件变更；异步队列结构不允许热替换。 |
| 插件注册 | 已支持 | `PluginRegistry`、`PluginRegistrar`、`PluginSet`、包级 helper、registrar 生成器。 |

## 当前不进入核心的范围

| 范围 | 原因 | 推荐处理 |
| --- | --- | --- |
| HTTP、Socket、Syslog、Kafka、SMTP、Database 输出 | 外部连接生命周期、重试、限流、凭证和失败策略会污染核心热路径。 | 后续独立模块显式注册。 |
| OpenTelemetry、Prometheus 等观测导出 | 当前发布范围不包含观测体系绑定。 | 后续统一观测设计后独立模块实现。 |
| JavaScript、Lua、expr、Starlark 等脚本引擎 | 脚本运行时和安全沙箱差异大。 | 核心只保留 `ScriptEvaluator` 契约。 |
| 远程 lookup namespace | 存在安全边界和供应链风险。 | 不提供。 |
| 自动扫描插件 | 运行时扫描增加隐式行为和启动成本。 | 使用显式注册或生成 registrar。 |

## 发布前验证入口

短门禁：

```bash
go test ./...
go vet ./...
go test ./... ./cmd/goark-log-plugin-gen ./internal/disruptor ./internal/jsoncodec
```

长压测：

```bash
GOARK_LOG_STRESS=1 go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

压力 benchmark：

```bash
go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

独立性能比较：

```bash
cd benchmarks/compare
go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```
