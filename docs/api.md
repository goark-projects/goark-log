# Programmatic API

[简体中文](api.zh-CN.md)

This page documents the public Go API implemented by `goark.dev/log`. The
configuration file structs under `internal/configfile` are not public API.

## Import

```go
import goarklog "goark.dev/log"
```

The module targets Go 1.25 or newer and implements the standard `log/slog`
handler contract.

## Construction

| Function | Use |
| --- | --- |
| `DefaultOptions()` | Returns stdout console logging at `INFO` with the Spring Boot style pattern. |
| `NewHandler(options)` | Builds a `*Handler` from programmatic `Options`. |
| `New(options)` | Builds a default-named `*slog.Logger` and its `*Handler`. |
| `NewDefaultHandler()` | Builds the default stdout handler and panics only if the built-in defaults are invalid. |
| `NewDefault()` | Builds the default `*slog.Logger` and `*Handler`. |
| `LoadOptions(ctx, opts...)` | Resolves and parses configuration into `Options`. |
| `NewConfiguredHandler(ctx, opts...)` | Loads config and builds a handler. |
| `NewConfigured(ctx, opts...)` | Loads config and builds a default-named `*slog.Logger` plus handler. |
| `ConfigureDefault(ctx, opts...)` | Loads config and installs the logger through `slog.SetDefault`. |
| `NewLoggerContext(options, opts...)` | Owns a handler and status logger from explicit options. |
| `NewConfiguredLoggerContext(ctx, opts...)` | Loads config, owns the handler, and starts reload polling when configured. |

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

logger := loggerContext.Logger("goark.http")
logger.InfoContext(ctx, "ready", slog.String("source", string(result.Source)))
```

## Handler Options

```go
type Options struct {
	Appenders []Appender
	Filters   []Filter
	Root      RootLogger
	Loggers   []LoggerRule
	Async     AsyncLoggerOptions
}
```

| Field | Runtime meaning |
| --- | --- |
| `Appenders` | Final sinks. At least one appender is required after defaults are applied. |
| `Filters` | Global filters. They run before route level checks. |
| `Root` | Root logger level, appender refs, filters, and location policy. |
| `Loggers` | Named logger rules. The most specific prefix wins. |
| `Async` | Handler-level bounded async queue. |

`Handler.Close()` drains async work, closes appenders, flushes files, and writes
layout footers. `Handler.Reload(options)` atomically swaps the router after the
new runtime builds successfully.

## Logger Names

`NewLogger(handler, name)` returns a `*slog.Logger` with the internal
`goark.logger` attribute bound. `WithName(logger, name)` does the same for an
existing logger and uses `slog.Default()` when the input is nil.

```go
logger := goarklog.NewLogger(handler, "goark.orm.mapper")
logger.Info("query finished", slog.Int("rows", 12))
```

The default logger name is `goark`.

## Native Logger

`NewNativeLogger(handler, name, opts...)` builds a low-allocation logger for hot
paths while still using the same handler, appenders, filters, and layouts.

| Method | Notes |
| --- | --- |
| `Name()` | Returns the effective logger name. |
| `Enabled(ctx, level)` | Checks the current route level. |
| `WithAttrs(attrs...)` | Returns a new logger with bound attributes. |
| `WithGroup(name)` | Returns a new logger with grouped attributes. |
| `Slog()` | Returns an equivalent `*slog.Logger`. |
| `LogAttrs(ctx, level, message, attrs...)` | Writes a structured event. |
| `LogAttrs3(ctx, level, message, a0, a1, a2)` | Fixed three-attribute fast path. |
| `Debug`, `Info`, `Warn`, `Error`, `Fatal` | Convenience methods. |
| `DebugContext`, `InfoContext`, `WarnContext`, `ErrorContext`, `FatalContext` | Context-aware convenience methods. |
| `At(level)`, `AtTrace`, `AtDebug`, `AtInfo`, `AtWarn`, `AtError`, `AtFatal` | Fluent event builder entry points. |

Native logger options:

| Option | Notes |
| --- | --- |
| `WithLoggerCaller(enabled)` | Captures call site when true. Location-enabled routes also force caller capture. |
| `WithLoggerMessageFactory(factory)` | Replaces the default `{}` parameterized message factory. |

## Fluent Builder

`LogBuilder` skips attribute construction when the level is disabled.

| Method | Notes |
| --- | --- |
| `Enabled()` | Reports whether the event can write. |
| `WithContext(ctx)` | Sets the event context. |
| `WithGroup(name)` | Prefixes later attributes with a group. |
| `WithAttr`, `WithAttrs` | Adds `slog.Attr` values. |
| `WithString`, `WithInt`, `WithBool`, `WithAny` | Typed helpers. |
| `WithMarker(marker)` | Adds a marker. |
| `WithError`, `WithThrowable` | Adds a throwable snapshot without stack capture. |
| `WithErrorStack(err)` | Adds a throwable snapshot with stack capture. |
| `Log(message)` | Writes a simple string message. |
| `Logf(pattern, args...)` | Uses `{}` placeholders. |
| `LogMessage(message)` | Writes a `Message`; attributed messages also add attrs. |

```go
_ = logger.AtInfo().
	WithContext(ctx).
	WithGroup("http").
	WithString("method", "GET").
	WithInt("status", 200).
	Logf("request {} completed", requestID)
```

## Context, Markers, And Throwables

| API | Notes |
| --- | --- |
| `WithContextAttrs`, `WithContextAttr`, `ContextAttrs` | MDC-style request attributes. |
| `NewMarker`, `MarkerAttr`, `WithMarker`, `ContextMarker` | Marker values with parent matching. |
| `ThreadNameAttr`, `WithThreadName`, `ContextThreadName` | Logical thread name for Go goroutines. |
| `WithContextStack`, `ContextStack` | NDC-style stack values. |
| `NewThrowable`, `NewThrowableWithStack` | Converts Go errors to throwable snapshots. |
| `ThrowableAttr`, `ThrowableWithStackAttr` | Adds throwable data to slog events. |

Standard attribute keys are `goark.throwable`, `goark.marker`,
`goark.thread`, `goark.contextStack`, `goark.structuredData.id`, and
`goark.structuredData.type`.

## Messages

| Type | Function | Notes |
| --- | --- | --- |
| `SimpleMessage` | `NewSimpleMessage(text)` | Immutable text. |
| `ParameterizedMessage` | `NewParameterizedMessage(pattern, args...)` | Replaces `{}` in order; `\{}` keeps a literal placeholder. |
| `MapMessage` | `NewMapMessage(attrs...)` | Message text is key/value text and attrs are exposed to layouts/filters. |
| `StructuredDataMessage` | `NewStructuredDataMessage(id, type, message, attrs...)` | RFC5424-style structured fields plus normal attrs. |
| `MessageFactoryFunc` | adapter | Allows custom parameterized message behavior. |

## Levels

Built-in levels are `ALL`, `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`,
and `OFF`. `WARNING` parses as `WARN`; integer levels are accepted.

| API | Notes |
| --- | --- |
| `ParseLevel(value)` | Parses names or integers. |
| `LevelName(level)` | Returns a registered exact name or the nearest built-in bucket. |
| `NewLevelRegistry()` | Creates an independent level registry. |
| `DefaultLevelRegistry()` | Returns the process default registry. |
| `RegisterLevel(name, level)` | Registers a process-wide custom level. |

## Appender API

All appenders implement:

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

Constructors include `NewConsoleAppender`, `NewFileAppender`,
`NewJSONAppender`, `NewJSONFileAppender`, `NewRollingFileAppender`,
`NewAsyncAppender`, `NewFailoverAppender`, `NewRoutingAppender`,
`NewRewriteAppender`, and `NewFilteredAppender`.

`NewAppenderRef` plus `WithAppenderRefLevel`, `WithAppenderRefLocation`, and
`WithAppenderRefFilters` models Log4j2-style appender references in code.

## Layout API

`Layout` formats an event into a caller-owned buffer. Built-in constructors are
`NewDefaultLayout`, `NewPatternLayout`, `NewPatternLayoutWithOptions`,
`NewJSONLayout`, `NewJSONTemplateLayout`, `NewJSONTemplateLayoutFromFile`,
`NewXMLLayout`, `NewYAMLLayout`, `NewCSVLayout`, `NewHTMLLayout`, and
`NewGELFLayout`. `TextLayout`, `RFC5424Layout`, and `SyslogLayout` are direct
types.

`LayoutOptions` contains `Compact`, `EventEOL`, `Complete`,
`IncludeStacktrace`, `StacktraceAsString`, `PropertiesAsList`,
`IncludeNullDelimiter`, `DisableANSI`, `Header`, and `Footer`.

## Filter API

All filters implement `Decide(ctx, event) FilterDecision`. Decisions are
`FilterNeutral`, `FilterAccept`, and `FilterDeny`.

Constructors include threshold, level, level range, regex, attr, marker,
no-marker, map, thread context map, thread context stack, structured data,
throwable, string match, time, burst, dynamic threshold, deny, composite, and
script filters. `ScriptFilter` is code-only and needs a caller-provided
`ScriptEvaluator`.

## Configuration API

Config load options:

| Option | Notes |
| --- | --- |
| `WithConfigPath(path)` | Highest precedence explicit path. |
| `WithConfigEnvKey(key)` | Overrides `GOARK_LOG_CONFIG`. |
| `WithConfigWorkingDir(dir)` | Base directory for relative paths and default discovery. |
| `WithBootPropertyResolver(resolver)` | Reads `goark.log.config`, `goark.logging.config`, and `logging.config`. |
| `WithDefaultConfigPaths(paths...)` | Replaces default discovery paths. |
| `WithConfigLookups(resolver)` | Uses a custom lookup resolver. |
| `WithPluginRegistry(registry)` | Uses an explicit plugin registry. |

`ConfigResult` reports `Source`, `Path`, and `MonitorInterval`.

Parser helpers are `ParseByteSize`, `ParseRollingInterval`,
`ParseRollingMaxAge`, and `ParseMonitorInterval`.

## Reload And Status

`ConfigReloader.Reload(ctx)` always reloads. `ReloadIfChanged(ctx)` checks the
configuration path, mod time, and size. `Watch(ctx, interval, onError)` polls
until the context is canceled.

`StatusLogger` records internal configuration and reload events. Use
`NewStatusLogger`, `WithStatusLevel`, `WithStatusWriter`, and
`WithStatusBufferSize`.

## Plugin API

Use `NewPluginRegistry` for isolated registries or `DefaultPluginRegistry` for
process-wide registration. Register explicit plugins with `RegisterAppender`,
`RegisterLayout`, `RegisterFilter`, `RegisterLookup`,
`RegisterJSONTemplateResolver`, or `RegisterPlugins`.

`NewPluginSet` with `WithPluginAppender`, `WithPluginLayout`,
`WithPluginFilter`, `WithPluginLookup`, and
`WithPluginJSONTemplateResolver` creates a reusable `PluginRegistrar`.
