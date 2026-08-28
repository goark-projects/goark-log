# Production Guide

[简体中文](production-guide.zh-CN.md)

This guide describes the production path that is implemented in the current
core module. It avoids optional external sinks and works without network
services.

## Recommended Startup

Use `LoggerContext` for long-running services. It owns the handler, exposes
named loggers, records internal status events, and starts config monitoring when
`monitorInterval` is greater than zero.

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

logger := loggerContext.Logger("goark.http")
logger.InfoContext(ctx, "logging ready", slog.String("source", string(result.Source)))
```

Use `ConfigureDefault` only when the whole process should route `slog.Default()`
through goark-log.

## Production Configuration

Start with [examples/production-service.yml](examples/production-service.yml).
It uses:

- stderr pattern console output for local diagnosis.
- async appender wrapping a rolling JSON file.
- audit rolling file with JSON Template layout and `flushOnWrite`.
- root string-match filter for health-check noise.
- named logger rule for `goark.audit` with `additivity=false`.
- named logger rule for `goark.demo.sql` at `DEBUG`.
- rolling size, daily time, startup rollover, gzip, max backups, max age, and delete action.
- `monitorInterval: 30s` for config polling reload.

## Async Selection

Use handler-level `asyncLogger` when nearly all loggers should pass through the
same asynchronous queue. It has the lowest application code footprint, but its
runtime shape cannot change during reload.

Use appender-level `type: async` when only specific destinations need a queue or
when a failover/routing/rewrite chain should be queued as one downstream sink.

Overflow strategies:

| Strategy | Use when |
| --- | --- |
| `block` | Logs must not be dropped and caller backpressure is acceptable. |
| `drop` | New events can be lost under pressure. |
| `drop-debug` | Debug and lower events may be lost, higher events block. |
| `sync-fallback` | Queue overflow should synchronously write the event. |

Wait strategies:

| Strategy | Use when |
| --- | --- |
| `block` | General production default. |
| `yield` | Low latency under CPU headroom. |
| `sleep` | Lower CPU pressure with optional `sleepTime`. |
| `spin` | Very low latency, CPU-expensive, use only after measuring. |

## Rolling and Retention

For services writing to files, prefer `rolling-file` with:

- `fileName` for the active file.
- `filePattern` containing `%d{...}` and `%i` when size rollover is enabled.
- `policies.size.size` for size rollover.
- `policies.time.interval` and `modulate` for daily/hourly alignment.
- `strategy.max` or `strategy.maxBackups`.
- `strategy.maxAge`.
- `strategy.delete` for directory cleanup beyond the active pattern.
- `strategy.compression.gzip: true` for archived files.
- `strategy.compression.async: true` or `asyncActions: true` only when archive actions may run in the background.

`directWrite` requires `filePattern` and does not support gzip compression.

## Audit Logs

Use a dedicated named logger such as `goark.audit` with `additivity=false`.
Set `flushOnWrite: true` and a restrictive `filePermissions` such as `0600`
when the audit file contains sensitive data. Use `jsonTemplate` to control the
field names consumed by downstream audit pipelines.

## Context Data

Use context APIs for request-scoped data:

```go
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", traceID),
	slog.String("tenant", tenantID),
)
ctx = goarklog.WithThreadName(ctx, "http-worker-1")
ctx = goarklog.WithContextStack(ctx, "request", "checkout")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
```

Use normal `slog.Attr` arguments for event-specific data.

## Reload Rules

`LoggerContext` starts reload polling only when:

- the loaded config result has a non-empty path, and
- `monitorInterval` parses to a positive duration.

`Reload` atomically replaces the router only after the new configuration builds
successfully. If reload fails, the old runtime remains active. Handler-level
async enablement, queue size, batch size, overflow strategy, wait strategy, wait
options, and include-location setting cannot change during reload.

## Shutdown

Always call `Close`. Close drains handler-level async queues, async appenders,
rolling archive actions, layout footers, file buffers, and appender-owned file
handles. Router close runs async appenders first, then the remaining appenders,
and avoids duplicate appender names.

## Containers

For container platforms, prefer [examples/container-json.yml](examples/container-json.yml)
when stdout is the log transport. It writes direct JSON to stdout and uses
`drop-debug` so high-volume debug traffic can be shed under pressure.

## Security Defaults

- Lookup namespaces `jndi`, `ldap`, and `rmi` are blocked.
- Core has no embedded script runtime. `ScriptFilter` requires a caller-provided evaluator through code.
- Core has no remote appenders. Network sinks should be explicit external modules.
- File appenders validate that the target path is not a directory and create parent directories as needed.
