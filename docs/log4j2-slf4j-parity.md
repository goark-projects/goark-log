# Log4j2 And SLF4J Parity

[简体中文](log4j2-slf4j-parity.zh-CN.md)

`goark-log` uses Go's `log/slog` as the stable facade and adds Log4j2-style
runtime configuration, routing, layouts, filters, rolling files, and plugins.
The intent is familiar operational behavior without Java-style runtime classpath
scanning or proxy mechanics.

## Facade Mapping

| SLF4J / Log4j concept | Goark equivalent |
| --- | --- |
| `LoggerFactory.getLogger("a.b")` | `loggerContext.Logger("a.b")`, `NewLogger(handler, "a.b")`, or `WithName(logger, "a.b")`. |
| Parameterized `{}` messages | `NewNativeLogger(..., WithLoggerMessageFactory(ParameterizedMessageFactory{}))` and `Logf`. |
| Markers | `NewMarker`, `MarkerAttr`, `WithMarker`, and marker filters. |
| MDC | `WithContextAttrs`, `WithContextAttr`, `ContextAttrs`, `%X{key}`, and JSON Template `mdc`. |
| NDC / ThreadContext stack | `WithContextStack`, `ContextStack`, `%ndc`, and JSON Template `contextStack`. |
| Throwable logging | `ThrowableAttr`, `ThrowableWithStackAttr`, `WithError`, `WithErrorStack`, `%ex`, and JSON Template throwable resolvers. |
| Named logger hierarchy | Longest-prefix `loggers` rules with root fallback. |
| Additivity | `additivity` on named logger rules. |
| Appender references | `appenderRefs` as strings or objects with level, filters, and location policy. |
| Configuration reload | `monitorInterval`, `LoggerContext`, and `ConfigReloader`. |

## Log4j2 Configuration Mapping

| Log4j2 area | Supported core mapping |
| --- | --- |
| `<Configuration status monitorInterval>` | `status` and `monitorInterval`. |
| `<Properties>` | `properties` plus `${NAME}`, `${prop:NAME}`, and `${property:NAME}` lookups. |
| `<Appenders>` | Console, File, RollingFile, Async, Failover, Routing, and Rewrite. |
| `<Loggers>` | `Root` and named `Logger` elements. |
| `<AppenderRef>` | String or structured appender references. |
| `<Filters>` | Built-in Log4j-style filter families listed in [filters](filters.md). |
| `<PatternLayout>` | Pattern layout with Log4j-style converters. |
| `<JSONLayout>` | JSON layout with lifecycle and stacktrace options. |
| `<JsonTemplateLayout>` | JSON Template layout with built-in and plugin resolvers. |
| `<Policies>` | Size, time, cron, and startup triggering policies. |
| `<DefaultRolloverStrategy>` | Max count, max age, file index, compression, async actions, and delete actions. |

Full example: [examples/log4j2-service.xml](examples/log4j2-service.xml).

## Go-Native Differences

| Java behavior | Goark behavior |
| --- | --- |
| Runtime classpath scanning | Explicit plugin registration through `PluginRegistrar` or generated registrars. |
| SLF4J facade API | Standard `log/slog` facade plus optional native logger for hot paths. |
| Java thread name | Logical thread name stored in context because goroutines do not have stable user-facing names. |
| Java exceptions | Go errors are captured as throwable snapshots with optional stack frames. |
| XML plugins by class name | Configured plugin kind names resolve through an explicit registry. |
| Script filters | Go API only through caller-provided `ScriptEvaluator`; no embedded scripting runtime in core. |
| JNDI lookups | `jndi`, `ldap`, and `rmi` lookup namespaces are blocked. |

These differences are deliberate. They keep the core deterministic,
dependency-light, and safe for Go services.

## Logger Hierarchy

Named logger selection uses the longest matching prefix.

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

`goark.demo.audit.payment` uses the `goark.demo.audit` rule. If `additivity` is
true or omitted, root appenders are also used. If `additivity: false`, the
named logger must provide at least one appender.

## Appender Reference Semantics

Appender references can carry their own level, filters, and caller-location
policy.

```yaml
appenderRefs:
  - ref: appRolling
    level: warn
    includeLocation: true
    filterRefs: [businessHours]
```

This maps to the Log4j2 operational model where one logger can send the same
event to different destinations with different gates.

## Pattern Layout Coverage

The pattern layout includes common Log4j-style converters: `%d`, `%p`,
`%level`, `%pid`, `%thread`, `%logger`, `%c`, `%msg`, `%m`, `%attrs`, `%kvp`,
`%X{key}`, `%mdc`, `%ex`, `%throwable`, `%marker`, `%ndc`, `%n`, caller
converters `%C`, `%M`, `%F`, `%L`, `%l`, `%uuid`, `%relative`, `%host`,
`%sequenceNumber`, `%highlight`, `%style`, `%notEmpty`, `%replace`, `%enc`,
`%equals`, `%equalsIgnoreCase`, `%maxLen`, and `%repeat`.

Caller converters require caller capture. Enable it only for the routes that
need it.

## Unsupported In Core

The core module does not include HTTP appenders, socket appenders, network
syslog clients, Kafka, Pulsar, RabbitMQ, SMTP, database sinks, OpenTelemetry
exporters, Prometheus exporters, or an embedded script runtime.

XML can parse several external appender-shaped elements and pass their fields
to a registered plugin, but the core does not create a network client for those
elements by itself.

## Migration Notes

| Existing usage | Recommended Goark usage |
| --- | --- |
| `logger.info("user {}", user)` | Native logger `Logf("user {}", user)` for hot paths, or `slog` with attrs for ordinary code. |
| MDC request values | Store values on `context.Context` with `WithContextAttrs`. |
| Log4j2 XML rolling file | Use the XML mapping or YAML/TOML with `rolling.policies` and `rolling.strategy`. |
| Async appenders | Use appender-level `async` when only one destination needs queueing. |
| Async loggers | Use handler-level `asyncLogger` when the whole handler should be asynchronous. |
| Classpath plugins | Register a `PluginSet` or generated registrar explicitly at startup. |
