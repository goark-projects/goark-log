# Capability Boundary

This document records what the `goark.dev/log` core module currently provides,
what is intentionally out of scope, and which validation gates should be used
before release.

## Design Principles

- Go-native API: explicit construction, interfaces, options, and plugin
  registration.
- Low-allocation hot paths: common JSON, file, direct native logging, and
  ring-buffer paths avoid reflection-heavy work.
- Dependency-light core: comparison dependencies stay in the independent
  `benchmarks/compare` module.
- Deterministic shutdown: async logger, async appender, rolling compression, and
  delete actions drain on `Close`.
- Safe defaults: no remote lookup, no embedded script engine, no built-in
  external-system appender, and no observability exporter in core.

## Supported in Core

| Area | Status | Notes |
| --- | --- | --- |
| `slog.Handler` | supported | Implements `Enabled`, `Handle`, `WithAttrs`, and `WithGroup`; can be installed as `slog.Default()`. |
| Native logger | supported | `NewNativeLogger`, `LogAttrs`, `LogAttrs3`, builder API, and message factories. |
| Logger routing | supported | Root logger, named rules, prefix matching, additivity, appender-ref level/filter, and includeLocation. |
| Custom levels | supported | Built-in `ALL`, `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`, `OFF`, and `RegisterLevel`. |
| Context attributes | supported | `WithContextAttrs`, `ContextAttrs`, MDC/NDC-style layout output. |
| Marker and throwable | supported | Context marker, marker attr, throwable attr, and optional throwable stack snapshots. |
| Async logger | supported | Bounded ring buffer, batch drain, block/drop/drop-debug/sync-fallback, wait strategies, counters, and drain on close. |
| Async appender | supported | Appender wrapper with bounded queue, batch drain, counters, error handler, and drain on close. |
| File appender | supported | Append/truncate, create-on-demand, permissions, buffering, and flush-on-write. |
| JSON appender | supported | Direct single-line JSON to stdout/stderr or file; file mode supports buffering. |
| Rolling file appender | supported | Size/time/cron/startup triggers, `%d`/`%i`, gzip, max count, max age, delete actions, and async action worker. |
| PatternLayout | supported | Time, level, logger, message, attrs, MDC, marker, NDC, caller, throwable, host, sequence, ANSI style, and nested converters. |
| Structured layouts | supported | JSON, JSON Template, XML, CSV, GELF, RFC5424/Syslog, YAML, and HTML. |
| Filters | supported | Level, range, regex, attrs, marker, MDC, structured data, throwable, time windows, burst limiter, and dynamic thresholds. |
| Composite appenders | supported | Async, Failover, Routing, and Rewrite are config-buildable. |
| Lookups | supported local subset | `env`, `sys`, `go`, `date`, `prop`, and `property`. |
| Configuration formats | supported | YAML, JSON, XML, properties; TOML explicitly fails. |
| Reload | supported with constraints | Polls a concrete config file; async logger queue/runtime shape cannot be hot-replaced. |
| Plugins | supported | `PluginRegistry`, `PluginRegistrar`, `PluginSet`, package helpers, lookup plugins, JSON Template resolvers, and registrar generator. |

## Not in Core

| Area | Reason | Recommended approach |
| --- | --- | --- |
| HTTP appender | Connection lifecycle, retry, timeout, TLS, and response handling are deployment-specific. | Build an external module and register an appender plugin. |
| Socket appender | Framing, reconnect, backpressure, and protocol choices vary. | Build an external module. |
| Syslog network appender | Transport, TLS, facility/app name mapping, and retry policy are environment-specific. | Build an external module; core only provides RFC5424/Syslog layout. |
| Kafka, Pulsar, RabbitMQ | Broker clients and delivery semantics are heavy dependencies. | Keep in dedicated Goark integration modules. |
| SMTP appender | Slow network I/O and credential handling do not belong in the core hot path. | Build a plugin module with explicit queue and retry behavior. |
| Database appender | Schema, transactions, batching, and failure handling are database-specific. | Build a database-specific plugin module. |
| OpenTelemetry, Prometheus | Observability design should be shared across Goark modules and remain optional. | Add after a separate observability design. |
| Script runtime | JavaScript/Lua/expr/Starlark runtime and sandbox decisions are security-sensitive. | Core exposes `ScriptEvaluator` API only. |
| Remote lookup namespaces | JNDI/LDAP/RMI-style lookups are unsafe for config-time resolution. | Blocked by default. |
| Runtime plugin scanning | Scanning adds implicit startup behavior and cost. | Use explicit registration or generated registrars. |

## Dependency Boundary

Core `go.mod` dependencies are limited to:

- `github.com/bytedance/sonic`
- `gopkg.in/yaml.v3`

The zap and zerolog comparison dependencies live in `benchmarks/compare/go.mod`
and must not be moved into the core module.

## Validation Gates

Short gates:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./... ./cmd/goark-log-plugin-gen ./internal/disruptor ./internal/jsoncodec
```

Focused hot-path benchmark:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Long stress gate:

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

Independent comparison module:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=5s -count=3 -cpu=1,4,16
```

## Release Boundary

Before publishing a tag, confirm:

- `dev` contains every intended change.
- `main` is updated through the approved release flow.
- Core tests and compare-module tests pass.
- Race and stress checks are either current or explicitly documented as deferred.
- Benchmark paths use `./benchmarks/core` for core benchmarks after the benchmark
  package split.
- README and docs do not claim unsupported external appenders or observability
  exporters.
