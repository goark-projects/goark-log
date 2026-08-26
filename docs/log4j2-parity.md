# Log4j2 能力对标矩阵

本文档记录 `goark.dev/log` 核心仓库和 Log4j2 的能力边界。核心原则是 Go-native、显式注册、低分配热路径；不做 Java 式运行时扫描，也不把外部系统 appender 放进核心仓库。

## 核心已支持

| Log4j2 能力 | goark-log 状态 | 说明 |
| --- | --- | --- |
| Logger 层级 | 已支持 | root、命名 logger、additivity、AppenderRef 级别和过滤器。 |
| Fluent API | 已支持 | `Logger.AtInfo()/AtDebug()/AtWarn()/AtError()/AtTrace()`。 |
| Message | 已支持 | `SimpleMessage`、`ParameterizedMessage`、`MapMessage`、`StructuredDataMessage`、`MessageFactory`。 |
| 自定义级别 | 已支持 | `ALL/TRACE/DEBUG/INFO/WARN/ERROR/FATAL/OFF`，`RegisterLevel`、`LevelRegistry`，以及 YAML/JSON/XML/properties `customLevels` 配置。 |
| AsyncLogger | 已支持核心语义 | 有界 ring buffer、批量消费、block/drop/drop-debug/sync-fallback、等待策略参数、消费者错误处理、`includeLocation`。 |
| AsyncAppender | 已支持 | 异步包装本地或组合型 appender，关闭时 drain。 |
| RollingFile | 已支持核心策略 | size/time/cron/startup、`%d/%i`、gzip、max/maxAge、delete action、后台动作队列。 |
| PatternLayout | 已支持核心表达式 | 时间、级别、logger 精度、message、MDC、NDC、marker、异常 none/short/full、caller、attrs/map、uuid、sequence、relative、host、replace、encode、equals、maxLen、repeat、highlight/style ANSI 和 `disableAnsi`、notEmpty。 |
| JSON Template Layout | 已支持主要 resolver 和选项 | timestamp、level、logger、message、thread、marker、throwable/rootCause/stackTrace、source/process、contextStack、mdc、attr、endOfBatch、自定义 resolver；支持 layout 通用选项、level field、logger precision、MDC list。 |
| 结构化布局 | 已支持核心参数矩阵 | JSON、XML、CSV、GELF、RFC5424/Syslog layout、YAML、HTML；JSON/XML/CSV/GELF/YAML/HTML 支持 compact、eventEol、complete、stacktraceAsString、propertiesAsList、null delimiter、header/footer。 |
| Filters | 已支持主干 | Threshold、Level、LevelRange、Regex、Attr、Deny、Composite、Marker、NoMarker、Map、ThreadContextMap/Stack、StructuredData、Throwable、StringMatch、Time、Burst、DynamicThreshold；全局 filter 在 logger level 前执行，`ACCEPT` 可放行低级别事件，`DENY` 可提前短路。 |
| 组合型 Appender | 已支持配置化 | Async、Failover、Routing、Rewrite 均可通过 YAML/JSON/XML/properties 配置；Rewrite 内置静态属性追加和属性移除策略。 |
| Lookups | 已支持安全子集 | env、sys、go、date、property；jndi/ldap/rmi 被安全拒绝。 |
| Plugins | 已支持 Go-native 注册 | `PluginRegistry`、包级 helper、`PluginRegistrar`、`PluginSet` 和 registrar 生成器；外部模块显式注册。 |
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

## P0/P1/P2 完成状态

| 优先级 | 范围 | 状态 |
| --- | --- | --- |
| P0 | JSON 热路径 | 已增加 JSON 直写 appender、固定三属性入口和 Sonic 复杂对象 fallback；默认 JSON/JSONTemplate 基础路径保持 `0 alloc/op`。 |
| P0 | Disruptor 语义 | 已补等待策略参数、超时/睡眠重试、异步错误处理和 race 覆盖；仍保持 Go-native 内部实现。 |
| P1 | Log4j2 级别和 filter 语义 | 已补 ALL/TRACE/FATAL/OFF、自定义级别配置、全局 filter 前置裁决、appenderRef filter/level 语义。 |
| P1 | PatternLayout 表达式 converter | 已补 replace、encode、equals、equalsIgnoreCase、maxLen、repeat、sequenceNumber、relative、host、throwable none/short/full、highlight/style ANSI 和 disableAnsi。 |
| P1 | 结构化 Layout 参数矩阵 | 已补 JSON/JSONTemplate/XML/CSV/GELF/YAML/HTML 通用选项、生命周期 header/footer、配置格式映射和校验。 |
| P1 | 组合 appender 配置体验 | 已补 Failover、Routing、Rewrite 的 YAML/JSON/XML/properties 配置；配置型复合 appender 不拥有子 appender 关闭权，避免重复关闭。 |
| P2 | 外部插件体验 | 核心已提供 `PluginRegistrar`、`PluginSet`、包级 helper、`AppenderBuildConfig` 扩展字段和 `goark-log-plugin-gen` registrar 生成器；具体外部 appender 继续放独立仓库。 |

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
