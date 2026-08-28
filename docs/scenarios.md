# Usage Scenarios

[简体中文](scenarios.zh-CN.md)

This guide shows complete, production-oriented logging scenarios. Copyable
configuration files are stored under [examples](examples/README.md).

## Scenario 1: Development Console

Use this for local development when humans read logs directly.

Config: [examples/console.yml](examples/console.yml)

```yaml
configuration:
  properties:
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %highlight{%-5p} %pid --- [%thread] %c : %m%attrs%n"
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
        disableAnsi: false
  root:
    level: debug
    appenderRefs: [console]
```

Recommended for:

- local command-line development,
- short-running tools,
- debug sessions where colored levels help.

Avoid for:

- high-volume production JSON collection,
- Windows consoles that do not render ANSI well; set `disableAnsi: true`.

## Scenario 2: Container JSON to stdout

Use this for Docker, Kubernetes, Nomad, and platforms where stdout is collected
by the runtime.

Config: [examples/json-stdout.yml](examples/json-stdout.yml)

```yaml
configuration:
  appenders:
    json:
      type: json
      target: stdout
  root:
    level: info
    appenderRefs: [json]
```

Recommended runtime pattern:

```go
handler, _, err := goarklog.ConfigureDefault(context.Background(),
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer handler.Close()
```

Operational notes:

- Prefer `target: stdout` for application events in containers.
- Send internal platform diagnostics to stderr through the service runtime, not
  through a second default logger unless your collector expects it.
- Use direct JSON appender for the lowest overhead fixed event shape.

## Scenario 3: Production Rolling JSON Files

Use this for VM or bare-metal services where local files are the primary log
handoff.

Config: [examples/production-rolling.yml](examples/production-rolling.yml)

Key choices:

- `asyncLogger.enabled: true` decouples business goroutines from disk writes.
- `overflowStrategy: block` preserves logs by applying backpressure.
- `bufferSize: 256KiB` reduces write syscalls.
- `filePattern` contains both `%d` and `%i`, which is required when size rolling
  is active.
- `.gz` archive suffix enables gzip compression.
- `compression.async: true` runs compression and deletion on a single background
  worker.
- `Close` must be called to flush buffers and finish queued rolling actions.

Tune by workload:

| Workload | Recommended change |
| --- | --- |
| Latency-sensitive and logs can be lost | Use `overflowStrategy: drop` or `drop-debug`, then monitor `AsyncDropped`. |
| Audit or billing logs | Keep `overflowStrategy: block` and use `flushOnWrite` only for the audit sink. |
| Slow disk | Increase `queueSize` and `batchSize`, but keep enough memory headroom. |
| Caller fields needed | Enable `includeLocation` only on the target logger or appender ref. |

## Scenario 4: Split Application and Audit Logs

Use this when audit events require different retention, permissions, and
schema.

Config: [examples/split-audit.yml](examples/split-audit.yml)

Logger contract:

```go
audit := slog.New(handler).With("goark.logger", "goark.audit")
audit.InfoContext(ctx, "user permission changed",
	slog.String("principal", "alice"),
	slog.String("action", "grant"),
	slog.String("resource", "project:42"),
)
```

The `goark.audit` logger has `additivity: false`, so audit events are written
only to the audit appender and are not duplicated in the root application log.

Operational notes:

- `filePermissions: "0600"` is appropriate when audit data contains sensitive
  principals or resource identifiers.
- Keep audit layout stable and explicit. JSON Template is preferable to generic
  JSON when downstream compliance tooling expects exact fields.
- `flushOnWrite: true` increases durability but reduces throughput; apply it to
  audit only, not to all app logs.

## Scenario 5: Appender-Level Async

Use Handler-level async when the whole logging pipeline should be decoupled. Use
Appender-level async when only one sink should be asynchronous.

Config: [examples/async-appender.yml](examples/async-appender.yml)

```yaml
appenders:
  jsonFile:
    type: json
    fileName: logs/app.json
    bufferSize: 256KiB
  asyncJson:
    type: async
    appenderRefs: [jsonFile]
    queueSize: 4096
    batchSize: 128
    overflowStrategy: block
    waitStrategy: yield
root:
  level: info
  appenderRefs: [asyncJson]
```

Use this when:

- console output should remain synchronous but file output should be async,
- a slow rolling file should not block every appender path,
- you are composing failover/routing/rewrite appenders and want a specific
  delegate boundary to be async.

Do not wrap every appender blindly. Multiple async layers add queueing and
shutdown complexity.

## Scenario 6: Routing by Tenant and Redacting Attributes

Use routing for a small, bounded set of known output routes.

Config: [examples/rewrite-routing.yml](examples/rewrite-routing.yml)

```go
logger.InfoContext(ctx, "payment accepted",
	slog.String("tenant", "tenant-a"),
	slog.String("order_id", "ord-100"),
	slog.String("token", "secret"),
)
```

The built-in rewrite appender removes `token`, `password`, and `authorization`,
adds `service=billing`, and then delegates to the routing appender. The routing
appender reads `tenant` and writes to a tenant-specific file or stdout fallback.

Guidance:

- Keep route cardinality bounded. Do not route by user ID or request ID.
- Route missing or unknown values to `defaultRoute`.
- Use rewrite as a last-mile guard, not as a substitute for avoiding sensitive
  values at the call site.

## Scenario 7: Config Reload

Use `NewConfiguredLoggerContext` when a service should reload logging config
without restart.

```go
ctx := context.Background()
logging, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer logging.Close()

logger := logging.Logger("goark.service")
logger.Info("logging started", slog.String("source", string(result.Source)))
```

Set `monitorInterval` in the file:

```yaml
configuration:
  monitorInterval: 30s
  appenders:
    console:
      type: console
  root:
    level: info
    appenderRefs: [console]
```

Reload can change log levels, filters, routes, appenders, and layouts. Reload
cannot change Handler-level async runtime settings. Restart the logger context
to change async enablement, queue size, batch size, overflow strategy, wait
strategy, wait options, or async caller location.

## Scenario 8: MDC, Trace IDs, Marker, and Context Stack

Go does not have Java-style thread locals. Use `context.Context`.

```go
ctx := context.Background()
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", "trace-100"),
	slog.String("span_id", "span-200"),
)
ctx = goarklog.WithThreadName(ctx, "http-worker-1")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
ctx = goarklog.WithContextStack(ctx, "tenant-a", "checkout")

logger.InfoContext(ctx, "request done",
	slog.String("method", "GET"),
	slog.Int("status", 200),
)
```

Pattern example:

```text
%d %-5p [%thread] %c trace=%X{trace_id} marker=%marker ndc=%ndc %m%attrs%n
```

JSON Template example:

```json
{
  "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
  "level": {"$resolver": "level"},
  "logger": {"$resolver": "logger"},
  "traceId": {"$resolver": "attr", "key": "trace_id"},
  "spanId": {"$resolver": "attr", "key": "span_id"},
  "marker": {"$resolver": "marker"},
  "stack": {"$resolver": "contextStack"},
  "attrs": {"$resolver": "mdc"}
}
```

## Scenario 9: Caller Location for a Narrow Logger

Caller lookup is not free. Enable it only where needed.

```yaml
appenders:
  file:
    type: file
    fileName: logs/debug.log
    layout:
      type: pattern
      pattern: "%d %-5p %c %F:%L %M - %m%attrs%n"
root:
  level: info
  appenderRefs: [file]
loggers:
  goark.debug:
    level: debug
    includeLocation: true
    appenderRefs: [file]
    additivity: false
```

With the native logger:

```go
debug, err := goarklog.NewNativeLogger(handler, "goark.debug",
	goarklog.WithLoggerCaller(true),
)
if err != nil {
	return err
}
_ = debug.DebugContext(ctx, "debug point")
```

## Scenario 10: Failover to Console

Use failover when the preferred sink may fail but losing the event is
unacceptable.

```yaml
appenders:
  primary:
    type: file
    fileName: /var/log/my-service/app.log
  fallback:
    type: console
    target: stderr
  failover:
    type: failover
    primary: primary
    failovers: [fallback]
root:
  level: info
  appenderRefs: [failover]
```

Keep failovers simple. A failover target that can block indefinitely defeats the
purpose of a fallback path.

## Scenario 11: Properties Config for Legacy Deployment

Use properties when deployment tooling already renders Java-style properties.

Config: [examples/goark-log.properties](examples/goark-log.properties)

```properties
property.LOG_DIR=logs
appender.console.type=console
appender.console.target=stderr
appender.console.layout.type=pattern
appender.console.layout.pattern=%d %5p %pid --- [%thread] %c : %m%attrs%n
rootLogger.level=info
rootLogger.appenderRefs=console
```

Do not use YAML nesting syntax in properties. Use the flat key prefixes
documented in [Configuration](configuration.md).

## Scenario 12: XML Config for Log4j2-Style Migration

Use XML when a team is migrating from Log4j2 and wants a familiar file shape.

Config: [examples/log4j2-style.xml](examples/log4j2-style.xml)

Core differences from Log4j2:

- The runtime is Go-native and explicit; there is no classpath scanning.
- HTTP, Socket, and Syslog network appenders require external plugin modules.
- Script filters require a caller-provided evaluator; no script engine is
  embedded in core.
- Caller location is opt-in for performance.

## Scenario 13: Programmatic Construction

Programmatic construction gives the most explicit ownership and is useful for
tests, embedded services, or frameworks that already have a config system.

```go
layout := goarklog.NewJSONLayout(goarklog.LayoutOptions{EventEOL: true})
file, err := goarklog.NewFileAppender("logs/app.json",
	goarklog.WithFileLayout(layout),
	goarklog.WithFileBufferSize(256*1024),
)
if err != nil {
	return err
}

handler, err := goarklog.NewHandler(goarklog.Options{
	Appenders: []goarklog.Appender{file},
	Root: goarklog.RootLogger{
		Level:        slog.LevelInfo,
		AppenderRefs: []string{"file"},
	},
})
if err != nil {
	_ = file.Close()
	return err
}
defer handler.Close()
```

Rules:

- The caller owns appender construction errors.
- `NewHandler` owns appender shutdown after it succeeds.
- If `NewHandler` fails after appenders were created manually, close them
  yourself.

## Scenario 14: Low-Allocation Native Logging

Use the native logger for hot paths with fixed fields.

```go
logger, err := goarklog.NewNativeLogger(handler, "goark.http")
if err != nil {
	return err
}

if logger.Enabled(ctx, slog.LevelInfo) {
	_ = logger.LogAttrs3(ctx, slog.LevelInfo, "request done",
		slog.String("method", method),
		slog.Int("status", status),
		slog.Duration("elapsed", elapsed),
	)
}
```

Guidance:

- `LogAttrs3` is the fixed three-attribute fast path.
- Use `LogAttrs` when the number of attributes is dynamic.
- Use `AtInfo().WithString(...).Log(...)` when fluent construction is clearer.
- Avoid `slog.Any` in the hottest JSON path unless you need complex payloads.

## Scenario 15: Custom Plugin

Use explicit plugin registration when a service or module adds a custom appender,
layout, filter, lookup, or JSON Template resolver.

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginLookup("tenant", lookupTenant),
	goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
))
if err != nil {
	return err
}

handler, _, err := goarklog.NewConfiguredHandler(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

Plugin factories should validate every required field, keep ownership explicit,
and avoid global mutable state unless the lifecycle is process-wide by design.
