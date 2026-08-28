# Log4j2 与 SLF4J 对标

[English](log4j2-slf4j-parity.md)

`goark-log` 以 Go 标准 `log/slog` 作为稳定门面，并补齐 Log4j2 风格的运行时配置、
路由、layout、filter、滚动文件和插件能力。目标是提供熟悉的运维体验，同时避免
Java 风格的运行时 classpath 扫描和代理机制。

## 门面映射

| SLF4J / Log4j 概念 | Goark 对应能力 |
| --- | --- |
| `LoggerFactory.getLogger("a.b")` | `loggerContext.Logger("a.b")`、`NewLogger(handler, "a.b")` 或 `WithName(logger, "a.b")`。 |
| `{}` 参数化消息 | `NewNativeLogger(..., WithLoggerMessageFactory(ParameterizedMessageFactory{}))` 和 `Logf`。 |
| Marker | `NewMarker`、`MarkerAttr`、`WithMarker` 和 marker filter。 |
| MDC | `WithContextAttrs`、`WithContextAttr`、`ContextAttrs`、`%X{key}` 和 JSON Template `mdc`。 |
| NDC / ThreadContext stack | `WithContextStack`、`ContextStack`、`%ndc` 和 JSON Template `contextStack`。 |
| Throwable 日志 | `ThrowableAttr`、`ThrowableWithStackAttr`、`WithError`、`WithErrorStack`、`%ex` 和 JSON Template throwable resolver。 |
| 命名 logger 层级 | 最长前缀 `loggers` 规则和 root 回退。 |
| Additivity | 命名 logger 规则上的 `additivity`。 |
| Appender reference | 字符串或对象形式的 `appenderRefs`，可携带 level、filter 和 location 策略。 |
| 配置重载 | `monitorInterval`、`LoggerContext` 和 `ConfigReloader`。 |

## Log4j2 配置映射

| Log4j2 区域 | 核心支持映射 |
| --- | --- |
| `<Configuration status monitorInterval>` | `status` 和 `monitorInterval`。 |
| `<Properties>` | `properties` 以及 `${NAME}`、`${prop:NAME}`、`${property:NAME}` lookup。 |
| `<Appenders>` | Console、File、RollingFile、Async、Failover、Routing、Rewrite。 |
| `<Loggers>` | `Root` 和命名 `Logger` 元素。 |
| `<AppenderRef>` | 字符串或结构化 appender 引用。 |
| `<Filters>` | [filters](filters.zh-CN.md) 中列出的内置 Log4j 风格 filter。 |
| `<PatternLayout>` | 支持 Log4j 风格 converter 的 Pattern layout。 |
| `<JSONLayout>` | 带生命周期和异常栈选项的 JSON layout。 |
| `<JsonTemplateLayout>` | 带内置和插件 resolver 的 JSON Template layout。 |
| `<Policies>` | size、time、cron、startup 触发策略。 |
| `<DefaultRolloverStrategy>` | 最大数量、最大年龄、索引模式、压缩、异步动作和删除动作。 |

完整示例：[examples/log4j2-service.xml](examples/log4j2-service.xml)。

## Go 化差异

| Java 行为 | Goark 行为 |
| --- | --- |
| 运行时 classpath 扫描 | 通过 `PluginRegistrar` 或生成的 registrar 显式注册插件。 |
| SLF4J facade API | 标准 `log/slog` 门面，加热点路径可选原生 logger。 |
| Java 线程名 | goroutine 没有稳定用户态名称，因此逻辑线程名放在 context 中。 |
| Java exception | Go error 转成 throwable snapshot，可选择捕获 stack frame。 |
| XML class name 插件 | 配置中的插件 kind 通过显式 registry 解析。 |
| Script filter | 仅 Go API 支持调用方提供的 `ScriptEvaluator`；核心不嵌入脚本运行时。 |
| JNDI lookup | `jndi`、`ldap`、`rmi` lookup namespace 被阻断。 |

这些差异是有意设计，用于保持核心确定性、低依赖，并适合 Go 服务。

## Logger 层级

命名 logger 使用最长前缀匹配。

```yaml
configuration:
  appenders:
    console:
      type: console
  root:
    level: info
    appenderRefs: [console]
  loggers:
    goark.demo:
      level: debug
    goark.demo.audit:
      level: info
      appenderRefs: [audit]
      additivity: false
```

`goark.demo.audit.payment` 使用 `goark.demo.audit` 规则。`additivity` 为 true 或省略时，
root appender 也会写入。`additivity: false` 时，该命名 logger 必须提供至少一个
appender。

## Appender 引用语义

Appender 引用可以携带自己的 level、filter 和 caller location 策略。

```yaml
appenderRefs:
  - ref: appRolling
    level: warn
    includeLocation: true
    filterRefs: [businessHours]
```

这对应 Log4j2 的运维模型：同一 logger 可以把事件发送到不同目标，并在每个目标上
使用不同的门控。

## Pattern Layout 覆盖

Pattern layout 包含常见 Log4j 风格 converter：`%d`、`%p`、`%level`、`%pid`、
`%thread`、`%logger`、`%c`、`%msg`、`%m`、`%attrs`、`%kvp`、`%X{key}`、`%mdc`、
`%ex`、`%throwable`、`%marker`、`%ndc`、`%n`、caller converter `%C`、`%M`、`%F`、
`%L`、`%l`、`%uuid`、`%relative`、`%host`、`%sequenceNumber`、`%highlight`、
`%style`、`%notEmpty`、`%replace`、`%enc`、`%equals`、`%equalsIgnoreCase`、
`%maxLen` 和 `%repeat`。

Caller converter 需要捕获调用点，只应对确实需要的路由启用。

## 核心不内置的能力

核心模块不包含 HTTP appender、socket appender、网络 syslog client、Kafka、Pulsar、
RabbitMQ、SMTP、数据库 sink、OpenTelemetry exporter、Prometheus exporter 或嵌入式
脚本运行时。

XML 可以解析若干外部 appender 形态的元素并把字段传给已注册插件，但核心不会自行创建
网络 client。

## 迁移建议

| 现有用法 | 推荐 Goark 用法 |
| --- | --- |
| `logger.info("user {}", user)` | 热点路径使用原生 logger `Logf("user {}", user)`，普通代码使用带属性的 `slog`。 |
| MDC 请求值 | 用 `WithContextAttrs` 存入 `context.Context`。 |
| Log4j2 XML rolling file | 使用 XML 映射，或用 YAML/TOML 的 `rolling.policies` 与 `rolling.strategy`。 |
| Async appenders | 只有部分目标需要异步时使用 appender 级 `async`。 |
| Async loggers | 整个 handler 需要异步时使用 handler 级 `asyncLogger`。 |
| Classpath plugins | 启动时显式注册 `PluginSet` 或生成的 registrar。 |
