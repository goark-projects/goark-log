# Programmatic API Guide

This guide covers the public programmatic API. Configuration files are covered
in [Configuration](configuration.md).

## Default Logger

```go
logger, handler := goarklog.NewDefault()
defer handler.Close()

logger.Info("service started", slog.String("profile", "dev"))
```

`NewDefault` creates:

- one console appender named `console`,
- stderr target,
- default Spring Boot pattern layout,
- root level `INFO`.

`NewDefaultHandler` returns only the handler. `DefaultOptions` returns the
equivalent options object.

## Handler Construction

```go
appender := goarklog.NewConsoleAppender()
handler, err := goarklog.NewHandler(goarklog.Options{
	Appenders: []goarklog.Appender{appender},
	Root: goarklog.RootLogger{
		Level:        slog.LevelInfo,
		AppenderRefs: []string{"console"},
	},
})
if err != nil {
	return err
}
defer handler.Close()
```

`Options` fields:

| Field | Description |
| --- | --- |
| `Appenders` | Required unless using default options. Names must be non-empty and unique. |
| `Filters` | Global filters. |
| `Root` | Root route. If no root appender is set, the first appender is used. |
| `Loggers` | Named logger rules. |
| `Async` | Handler-level async logger options. |

If you construct appenders manually and `NewHandler` returns an error, close
those appenders yourself. Once `NewHandler` succeeds, `Handler.Close` owns the
configured appenders.

## Named slog Logger

```go
logger := goarklog.NewLogger(handler, "goark.http")
logger.InfoContext(ctx, "request done", slog.Int("status", 200))
```

`NewLogger` attaches an internal `goark.logger` attribute that the handler uses
for routing. `WithName` can rename an existing `slog.Logger`:

```go
logger = goarklog.WithName(logger, "goark.orm")
```

## Configure Default slog

```go
handler, result, err := goarklog.ConfigureDefault(context.Background(),
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer handler.Close()

slog.Info("configured", slog.String("source", string(result.Source)))
```

`ConfigureDefault` installs the configured logger with `slog.SetDefault`.

## LoggerContext

`LoggerContext` is the managed runtime for services that need reload and
centralized shutdown.

```go
logging, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer logging.Close()

logger := logging.Logger("goark.service")
logger.Info("ready", slog.String("source", string(result.Source)))
```

Useful methods:

| Method | Description |
| --- | --- |
| `Logger(name)` | Returns a named `slog.Logger`. |
| `Handler()` | Returns the underlying `*Handler`. |
| `StatusLogger()` | Returns the internal status logger. |
| `ConfigResult()` | Returns the last loaded config result snapshot. |
| `Reload(options)` | Reloads from explicit `Options`. |
| `ReloadConfigured(ctx, options...)` | Reloads from config loading options. |
| `Close()` | Stops config monitor, drains async, closes appenders. |

`NewConfiguredLoggerContext` starts file polling only when the config file has a
positive `monitorInterval`.

## Config Reloader

Use `ConfigReloader` when an application has its own lifecycle or file watcher.

```go
reloader, err := goarklog.NewConfigReloader(handler,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}

changed, result, err := reloader.ReloadIfChanged(ctx)
```

`Watch(ctx, interval, onError)` starts a polling loop and returns a channel that
closes when the context is done.

## Native Logger

The native logger avoids parts of the standard `slog.Record` facade and gives a
fixed three-attribute fast path.

```go
logger, err := goarklog.NewNativeLogger(handler, "goark.http")
if err != nil {
	return err
}

_ = logger.LogAttrs3(ctx, slog.LevelInfo, "request done",
	slog.String("method", "GET"),
	slog.Int("status", 200),
	slog.Duration("elapsed", elapsed),
)
```

Native logger methods:

| Method | Description |
| --- | --- |
| `Name()` | Logger name. |
| `Enabled(ctx, level)` | Checks current route/global-filter state. |
| `Slog()` | Returns an equivalent `*slog.Logger`. |
| `WithAttrs(attrs...)` | Returns a logger with bound attrs. |
| `WithGroup(name)` | Returns a logger with a flattened group prefix. |
| `LogAttrs(ctx, level, message, attrs...)` | Writes a dynamic attr slice. |
| `LogAttrs3(ctx, level, message, a0, a1, a2)` | Writes exactly three attrs with the lowest overhead. |
| `Debug`, `Info`, `Warn`, `Error`, `Fatal` | Convenience methods using background context. |
| `DebugContext`, `InfoContext`, `WarnContext`, `ErrorContext`, `FatalContext` | Context-aware convenience methods. |
| `At(level)`, `AtTrace`, `AtDebug`, `AtInfo`, `AtWarn`, `AtError`, `AtFatal` | Creates a builder. |

`WithLoggerCaller(true)` captures caller PC for native events. Avoid it on hot
paths unless layouts need caller converters or JSON Template source resolver.

## Log Builder

```go
err := logger.AtInfo().
	WithContext(ctx).
	WithGroup("http").
	WithString("method", "GET").
	WithInt("status", 200).
	WithBool("cached", false).
	Log("request done")
```

Builder methods:

| Method | Description |
| --- | --- |
| `Enabled()` | Whether the event would be emitted. |
| `WithContext(ctx)` | Sets event context. |
| `WithGroup(name)` | Adds a flattened group prefix for following attrs. |
| `WithAttr(attr)` | Adds one attr. |
| `WithAttrs(attrs...)` | Adds multiple attrs. |
| `WithString`, `WithInt`, `WithBool`, `WithAny` | Typed attr helpers. |
| `WithMarker(marker)` | Adds a marker attr. |
| `WithError(err)`, `WithThrowable(err)` | Adds throwable without stack capture. |
| `WithErrorStack(err)` | Adds throwable with stack capture. |
| `Log(message)` | Writes a simple string message. |
| `Logf(pattern, args...)` | Uses the configured message factory with `{}` placeholders by default. |
| `LogMessage(message)` | Writes a custom message object. |

The builder stores up to eight attrs inline before allocating a backing slice.

## Context Attributes

```go
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", "trace-1"),
	slog.String("span_id", "span-1"),
)
ctx = goarklog.WithThreadName(ctx, "worker-1")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
ctx = goarklog.WithContextStack(ctx, "tenant-a", "checkout")
```

Context helpers:

| Helper | Description |
| --- | --- |
| `WithContextAttrs` | Adds immutable attr snapshot to context. |
| `WithContextAttr` | Adds one context attr. |
| `ContextAttrs` | Returns context attr snapshot. |
| `NewMarker` | Creates a marker with optional parents. |
| `MarkerAttr` | Converts a marker to a slog attr. |
| `WithMarker`, `ContextMarker` | Stores and reads context marker. |
| `ThreadNameAttr` | Converts logical thread name to attr. |
| `WithThreadName`, `ContextThreadName` | Stores and reads logical thread name. |
| `WithContextStack`, `ContextStack` | Stores and reads NDC-style stack values. |

Event attributes are merged in this order:

1. handler-bound attrs,
2. context attrs,
3. record/native attrs.

When the same key appears more than once, `Event.Attr(key)` returns the latest
value.

## Throwable and Message APIs

Throwable helpers:

| Helper | Description |
| --- | --- |
| `NewThrowable(err)` | Captures error type/message/cause chain without stack. |
| `NewThrowableWithStack(err)` | Captures throwable plus current stack. |
| `ThrowableAttr(err)` | Adds standard `goark.throwable` attr. |
| `ThrowableWithStackAttr(err)` | Adds standard throwable attr with stack. |

Message helpers:

| Helper | Description |
| --- | --- |
| `NewSimpleMessage(text)` | Immutable string message. |
| `NewParameterizedMessage(pattern, args...)` | `{}` placeholder message. |
| `NewMapMessage(attrs...)` | Structured message represented by attrs. |
| `NewStructuredDataMessage(id, type, message, attrs...)` | RFC5424-style structured message. |
| `WithLoggerMessageFactory(factory)` | Replaces native logger parameterized message factory. |

## Status Logger

```go
status := goarklog.NewStatusLogger(
	goarklog.WithStatusLevel(slog.LevelWarn),
	goarklog.WithStatusWriter(os.Stderr),
	goarklog.WithStatusBufferSize(128),
)

logging, err := goarklog.NewLoggerContext(options,
	goarklog.WithLoggerContextStatus(status),
)
```

`StatusLogger` records internal config, reload, and close errors. The config
file `status` field is parsed for compatibility but does not currently tune
`StatusLogger`.

## Closing Rules

- Always call `Close` on `Handler` or `LoggerContext`.
- `Close` drains Handler-level async before closing appenders.
- Runtime close order closes async appenders before their delegates.
- File and rolling appenders flush buffers and write layout footers where
  applicable.
- Rolling async actions are drained before `Close` returns.
- Calling `Close` more than once is safe.
