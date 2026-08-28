# Capabilities

[简体中文](capabilities.zh-CN.md)

This matrix describes the current core module, not planned companion modules.
Use it to decide whether a feature is available directly, available through an
extension boundary, or intentionally outside the core.

## Runtime API

| Capability | Status | Notes |
| --- | --- | --- |
| `slog.Handler` implementation | Built in | Supports standard `slog.Logger`, `WithAttrs`, `WithGroup`, and `LogAttrs`. |
| Named loggers | Built in | `NewLogger`, `WithName`, and `LoggerContext.Logger`. |
| Native logger | Built in | Low-allocation builder, fixed three-attr path, and `slog` interop. |
| Parameterized messages | Built in | `{}` placeholders through `ParameterizedMessageFactory`. |
| Map and structured data messages | Built in | Message attrs are also visible to layouts and filters. |
| Markers | Built in | Supports parent marker matching. |
| MDC-style context attrs | Built in | `WithContextAttrs` and pattern `%X{}` / JSON Template `mdc`. |
| NDC-style context stack | Built in | `WithContextStack`, `%ndc`, and JSON Template `contextStack`. |
| Throwable snapshots | Built in | Go errors with optional stack capture. |
| Status logger | Built in | Internal configuration and reload events. |

## Configuration

| Capability | Status | Notes |
| --- | --- | --- |
| YAML | Built in | Strict known-field structured decoding. |
| JSON | Built in | Same logical model as YAML. |
| TOML | Built in | Same logical model as YAML. |
| XML | Built in | Log4j2-style root and child elements. |
| Java properties | Built in | Practical key mapping; rolling policy coverage is narrower than YAML/TOML/XML. |
| Wrappers | Built in | Top-level, `configuration`, or `goark.log`; cannot be mixed. |
| Config discovery | Built in | Explicit path, env var, boot properties, default files, then defaults. |
| Lookup expansion | Built in and extensible | Built-ins: `env`, `sys`, `go`, `date`, plus file `prop` / `property`. |
| Blocked lookup namespaces | Built in | `jndi`, `ldap`, and `rmi` are blocked. |
| Reload | Built in | Explicit `ConfigReloader` and `LoggerContext` polling through `monitorInterval`. |

## Appenders

| Appender | Status | Notes |
| --- | --- | --- |
| Console | Built in | stdout/stderr with layouts. |
| File | Built in | Buffering, permissions, append/truncate, create-on-demand, headers, footers. |
| JSON direct | Built in | stdout/stderr or file, optimized event JSON path. |
| Rolling file | Built in | Size/time/cron/startup, gzip, retention, delete actions, async archive actions. |
| Async | Built in | Appender-level queue over delegate appenders. |
| Failover | Built in | Primary plus ordered failovers. |
| Routing | Built in | Attribute-key route selection with default route. |
| Rewrite | Built in | Adds and removes attrs before delegation. |
| HTTP | Plugin boundary | Parsed fields can be passed to a registered plugin; no core client. |
| Socket | Plugin boundary | Parsed fields can be passed to a registered plugin; no core client. |
| Network syslog | Plugin boundary | Core has RFC5424/syslog layouts, not a network client. |
| Kafka, Pulsar, RabbitMQ | Plugin boundary | Broker dependencies stay outside core. |
| SMTP, database sinks | Plugin boundary | Implement as external appender modules. |

## Layouts

| Layout | Status | Notes |
| --- | --- | --- |
| Pattern | Built in | Log4j-style converters and ANSI style/highlight support. |
| Text | Built in | Stable text key/value output. |
| JSON | Built in | Structured event JSON with lifecycle options. |
| JSON Template | Built in and extensible | Built-in resolvers plus plugin resolver registry. |
| XML | Built in | Single event XML fragment. |
| CSV | Built in | Fixed event CSV line. |
| GELF | Built in | Graylog Extended Log Format JSON. |
| RFC5424 | Built in | Syslog text line layout. |
| Syslog layout | Built in | Alias of RFC5424 layout. |
| YAML | Built in | Single event YAML document. |
| HTML | Built in | HTML table row. |

## Filters

| Filter family | Status | Notes |
| --- | --- | --- |
| Threshold, level, level range | Built in | Level gates. |
| Regex and string match | Built in | Message/logger/attr or substring matching. |
| Attr, map, thread context map, structured data | Built in | Attribute key/value matching. |
| Marker and no-marker | Built in | Marker presence and hierarchy matching. |
| Thread context stack | Built in | NDC-style stack matching. |
| Throwable | Built in | Throwable and error attribute matching. |
| Time | Built in | Time-of-day intervals with optional IANA time zone. |
| Burst | Built in | Token-bucket limiter for lower-severity events. |
| Dynamic threshold | Built in | Attribute-selected threshold. |
| Deny and composite | Built in | Always-deny and nested chains. |
| Script filter | Go API only | Requires caller-provided evaluator; no configured script runtime in core. |

## Rolling Files

| Capability | Status | Notes |
| --- | --- | --- |
| Size policy | Built in | `%i` required in `filePattern` when enabled. |
| Time policy | Built in | Interval and modulation support. |
| Cron policy | Built in | 5, 6, or 7 fields; year field must be wildcard-like. |
| Startup policy | Built in | Optional rollover on startup. |
| Gzip archives | Built in | For archive files; not allowed with direct write. |
| Max archive count | Built in | `strategy.max` or legacy `maxBackups`. |
| Max archive age | Built in | `strategy.maxAge` or legacy `maxAge`. |
| Delete action | Built in | Path depth, glob, age, accumulated count, and accumulated size. |
| Async archive actions | Built in | Serial background worker for compression and deletion. |
| Direct write strategy | Built in | Requires `filePattern`; rejects gzip. |

## Production Boundaries

| Concern | Current support |
| --- | --- |
| Backpressure | Handler-level async and appender-level async support block, drop, drop-debug, and sync-fallback. |
| Shutdown | Handler and logger context close drains queues and closes appenders. |
| Caller location cost | Captured only when logger/options/routes require it. |
| Hot-path formatting | JSON direct and native logger fast paths exist; claims require current benchmarks. |
| Observability exporters | Plugin boundary; no OpenTelemetry or Prometheus exporter in core. |
| Remote delivery retries | Plugin boundary for remote appenders. |
