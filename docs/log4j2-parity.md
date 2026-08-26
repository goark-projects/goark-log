# Log4j2 能力对标矩阵

本文档记录 `goark.dev/log` 核心仓库和 Log4j2 的能力边界。核心原则是 Go-native、显式注册、低分配热路径；不做 Java 式运行时扫描，也不把外部系统 appender 放进核心仓库。

## 核心已支持

| Log4j2 能力 | goark-log 状态 | 说明 |
| --- | --- | --- |
| Logger 层级 | 已支持 | root、命名 logger、additivity、AppenderRef 级别和过滤器。 |
| Fluent API | 已支持 | `Logger.AtInfo()/AtDebug()/AtWarn()/AtError()/AtTrace()`。 |
| Message | 已支持 | `SimpleMessage`、`ParameterizedMessage`、`MapMessage`、`StructuredDataMessage`、`MessageFactory`。 |
| 自定义级别 | 已支持 | `RegisterLevel` 和 `LevelRegistry`，内置级别热路径无锁。 |
| AsyncLogger | 已支持核心语义 | 有界 ring buffer、批量消费、block/drop/drop-debug/sync-fallback、等待策略、`includeLocation`。 |
| AsyncAppender | 已支持 | 异步包装本地或组合型 appender，关闭时 drain。 |
| RollingFile | 已支持核心策略 | size/time/cron/startup、`%d/%i`、gzip、max/maxAge、delete action、后台动作队列。 |
| PatternLayout | 部分支持 | 时间、级别、logger 精度、message、MDC、NDC、marker、异常、caller、attrs/map、uuid、highlight/style 透传、notEmpty。 |
| JSON Template Layout | 已支持主要 resolver | timestamp、level、logger、message、thread、marker、throwable/rootCause/stackTrace、source/process、contextStack、mdc、attr、endOfBatch、自定义 resolver。 |
| 结构化布局 | 已支持 | JSON、XML、CSV、GELF、RFC5424/Syslog layout、YAML、HTML。 |
| Filters | 已支持主干 | Threshold、Level、LevelRange、Regex、Attr、Deny、Composite、Marker、NoMarker、Map、ThreadContextMap/Stack、StructuredData、Throwable、StringMatch、Time、Burst、DynamicThreshold。 |
| Lookups | 已支持安全子集 | env、sys、go、date、property；jndi/ldap/rmi 被安全拒绝。 |
| Plugins | 已支持 Go-native 注册 | `PluginRegistry`、包级 helper、`PluginRegistrar`；外部模块显式注册。 |
| 配置格式 | 已支持 | YAML、JSON、XML、properties；TOML 明确拒绝。 |
| Reload | 已支持 | 文件轮询 reload；异步队列结构不允许热替换。 |

## 核心刻意不包含

| 能力 | 原因 | 推荐落点 |
| --- | --- | --- |
| HTTP/Socket/Syslog/Kafka/SMTP/Database appender 实现 | 外部系统依赖、连接生命周期和失败策略不应污染核心热路径。 | 独立仓库，例如 `goark-log-http`、`goark-log-kafka`。 |
| 脚本引擎实现 | 脚本运行时和安全沙箱策略差异大。 | 独立模块实现 `ScriptEvaluator`。 |
| OpenTelemetry/Prometheus/观测导出 | 观测体系尚未统一，避免提前绑定。 | 后续独立观测包。 |
| CI/workflow | 当前按本地命令验证，不在核心仓库新增 CI。 | 后续统一工程化方案。 |
| JNDI/LDAP/RMI lookup | 安全边界，不复刻高风险历史能力。 | 不提供。 |

## 仍需持续增强

| 优先级 | 缺口 | 建议 |
| --- | --- | --- |
| P0 | 完整 JSON 链路仍慢于 zerolog/zap | 继续优化 appender/layout 直写路径，减少 `Event.Attrs` 跨接口逃逸。 |
| P0 | Disruptor 语义仍是 Go-native 核心子集 | 后续可补消费者异常策略矩阵和更细的 wait strategy 参数。 |
| P1 | PatternLayout 仍未完整覆盖 `%replace/%enc/%equals` 等表达式 converter | 需要独立 expression/converter 解析器，不建议塞进单文件分支。 |
| P1 | 普通 JSON/XML/YAML/CSV layout 参数矩阵仍较薄 | 在不破坏默认输出的前提下补 compact、eventEol、stacktraceAsString 等配置。 |
| P2 | 外部 appender 仓库矩阵 | 按 `PluginRegistrar` 和 `AppenderBuildConfig` 落独立仓库。 |

## 外部 Appender 结构

外部 appender 包应只依赖核心接口，暴露显式 registrar：

```go
package httpappender

import goarklog "goark.dev/log"

func Registrar() goarklog.PluginRegistrar {
	return goarklog.PluginRegistrarFunc(func(registry *goarklog.PluginRegistry) error {
		return registry.RegisterAppender("http", buildHTTPAppender)
	})
}
```

核心使用方显式注册：

```go
registry := goarklog.NewPluginRegistry()
_ = registry.RegisterPlugins(httpappender.Registrar())
```

这样保持 Log4j2 插件体验，但避免核心仓库绑定外部系统依赖。
