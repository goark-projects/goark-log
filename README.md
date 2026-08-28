# goark-log

[简体中文](README.zh-CN.md)

`goark-log` is a production-oriented logging framework for Go. It keeps the
standard `log/slog` contract as the first-class API and adds the runtime pieces
that large services normally expect from Log4j2 and SLF4J: named logger
hierarchies, appender references, structured filters, rolling files, JSON
Template layouts, bounded asynchronous queues, configuration reload, status
events, and explicit plugin registration.

The module path is:

```bash
go get goark.dev/log
```

The module targets Go 1.25 or newer.

## What It Provides

| Area | Current implementation |
| --- | --- |
| Standard API | `slog.Handler`, `slog.Logger`, `WithAttrs`, `WithGroup`, `LogAttrs`, and named loggers through `WithName` or `NewLogger`. |
| Native API | Low-allocation `Logger`, fixed three-attribute fast path, fluent `LogBuilder`, parameterized messages, map messages, structured data messages, markers, thread names, context stack, and throwable snapshots. |
| Configuration | YAML, JSON, TOML, Log4j2-style XML, and Java properties. Supported wrappers are top-level, `configuration`, and `goark.log`. |
| Routing | Root logger, longest-prefix named logger rules, additivity, appender-ref level gates, appender-ref filters, and per-reference location capture. |
| Appenders | Console, File, JSON direct, RollingFile, Async, Failover, Routing, and Rewrite. |
| Layouts | Pattern, Text, JSON, JSON Template, XML, CSV, GELF, RFC5424/Syslog text, YAML, and HTML row layouts. |
| Filters | Threshold, Level, LevelRange, Regex, Attr, Marker, NoMarker, Map, ThreadContextMap, ThreadContextStack, StructuredData, Throwable, StringMatch, Time, Burst, DynamicThreshold, Deny, and Composite. |
| Async | Handler-level async logger and appender-level async queues with bounded ring buffers, batching, overflow strategies, wait strategies, counters, and deterministic drain on close. |
| Rolling files | Size, interval, cron, startup rollover, `%d{...}` and `%i` patterns, index modes, gzip, max backups, max age, delete actions, and asynchronous archive actions. |
| Extensibility | Explicit plugin registry for appenders, layouts, filters, lookups, and JSON Template resolvers. A generator is available at `cmd/goark-log-plugin-gen`. |

The core module does not include HTTP appenders, socket appenders, network
syslog clients, Kafka, Pulsar, RabbitMQ, SMTP, database sinks, OpenTelemetry
exporters, Prometheus exporters, or an embedded script runtime. Those belong in
separate modules that register explicit plugins.

## Quick Start

```go
package main

import (
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	logger, handler := goarklog.NewDefault()
	defer handler.Close()

	logger = goarklog.WithName(logger, "goark.demo")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

Default output uses the Spring Boot style pattern and writes to stderr:

```text
2026-08-28T09:30:00.000+08:00  INFO 12345 --- [main] goark.demo : service started profile=dev
```

## Production Startup

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	loggerContext, result, err := goarklog.NewConfiguredLoggerContext(context.Background(),
		goarklog.WithConfigPath("conf/goark-log.yml"),
	)
	if err != nil {
		panic(err)
	}
	defer loggerContext.Close()

	logger := loggerContext.Logger("goark.http")
	logger.Info("logging configured", slog.String("source", string(result.Source)))
}
```

Configuration path resolution order:

1. `WithConfigPath`.
2. Environment variable `GOARK_LOG_CONFIG`, or the key set by `WithConfigEnvKey`.
3. Boot property keys `goark.log.config`, `goark.logging.config`, and `logging.config`.
4. Default files under `conf/goark-log.yml`, `.yaml`, `.json`, `.xml`, `.toml`, and `.properties`.
5. Built-in default configuration: stderr console at `INFO`.

Use [docs/examples/production-service.yml](docs/examples/production-service.yml)
as the starting production configuration. It covers console diagnostics,
asynchronous rolling JSON files, audit logs, health-check filtering, retention,
and configuration reload.

## Runnable Demos

```bash
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

The demos set `GOARK_LOG_DIR` to a temporary directory unless it is already set.
They do not require external services.

## Documentation Map

| Document | Purpose |
| --- | --- |
| [Documentation index](docs/index.md) | Full navigation for users, operators, and plugin authors. |
| [Production guide](docs/production-guide.md) | Production bootstrap, safe defaults, reload, shutdown, and deployment notes. |
| [Configuration model](docs/configuration.md) | Format rules, wrappers, discovery, lookup semantics, and reload behavior. |
| [Configuration reference](docs/configuration-reference.md) | Exhaustive field, alias, type, default, and validation tables. |
| [Programmatic API](docs/api.md) | Public constructors, runtime types, native logger, messages, context, and status APIs. |
| [Appenders](docs/appenders.md) | Appender behavior, configuration fields, ownership, and close semantics. |
| [Layouts](docs/layouts.md) | Layout output formats, pattern converters, JSON Template resolvers, and lifecycle flags. |
| [Filters](docs/filters.md) | Filter decisions, all built-in filters, placement, and nesting rules. |
| [Scenarios](docs/scenarios.md) | Copyable recipes for common logging scenarios. |
| [Log4j2 and SLF4J parity](docs/log4j2-slf4j-parity.md) | Compatibility mapping and Go-native differences. |
| [Extensibility](docs/extensibility.md) | Plugin registry, generated registrars, and external module boundaries. |
| [Capabilities](docs/capabilities.md) | Source-backed capability matrix and unsupported core boundaries. |
| [Performance](docs/performance.md) | Benchmarks, hot-path rules, stress checks, and performance caveats. |
| [Release checklist](docs/release-v0.0.2.md) | Validation gates for the next release. |
| [GitHub release notes](docs/github-release-v0.0.2.md) | Copyable notes for the `v0.0.2` GitHub Release. |
| [Configuration examples](docs/examples/README.md) | Loadable YAML, TOML, XML, and properties examples. |
| [Runnable examples](examples/README.md) | Demo commands and expected behavior. |

## Verification

Run the current-worktree validation gates before publishing:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Comparison benchmarks live in a separate module:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

`dev` is the integration branch. Cut release tags from `main` only after the
release checklist has passed on the exact commit being tagged.
