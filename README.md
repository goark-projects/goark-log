# goark-log

[简体中文](README.zh-CN.md)

`goark-log` is a high-performance structured logging framework for Go services.
It builds on the standard `log/slog` API while adding a production-oriented
handler runtime, appenders, layouts, hierarchical routing, filters, safe
configuration loading, bounded asynchronous queues, rolling files, and explicit
plugin registration.

The module path is:

```bash
go get goark.dev/log
```

The module targets Go 1.25 or newer.

## Design Goals

- Go-native public APIs: explicit constructors, interfaces, options, and plugin
  registration instead of runtime scanning.
- Low allocation on hot paths: common JSON, file, direct native logging, and
  ring-buffer paths avoid reflection-heavy encoding.
- Dependency-light core: zap and zerolog are used only by the independent
  `benchmarks/compare` module.
- Safe defaults: no remote lookup namespaces, no embedded script runtime, and no
  built-in external-system appenders in the core package.
- Deterministic shutdown: async loggers, async appenders, rolling compression,
  and delete actions drain on `Close`.

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

	logger = goarklog.WithName(logger, "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

Default output is a Spring Boot style single line written to stderr:

```text
2026-08-25T10:15:30.123+08:00  INFO 12345 --- [main] goark.boot : service started profile=dev
```

## Configured Startup

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	handler, result, err := goarklog.ConfigureDefault(context.Background(),
		goarklog.WithConfigPath("conf/goark-log.yml"),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	slog.Info("logging configured", slog.String("source", string(result.Source)))
}
```

Configuration is resolved in this order:

1. `WithConfigPath`.
2. `GOARK_LOG_CONFIG`, or a custom key from `WithConfigEnvKey`.
3. Boot property keys: `goark.log.config`, `goark.logging.config`,
   `logging.config`.
4. Default files under `conf/goark-log.{yml,yaml,json,xml,toml,properties}`.
5. Built-in default: stderr console, `INFO`.

YAML, JSON, XML, and properties are supported. TOML is intentionally rejected
with an explicit error so a stale file cannot be mistaken for an active
configuration.

## Production YAML

```yaml
configuration:
  monitorInterval: 30s
  properties:
    LOG_DIR: logs
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %pid --- [%thread] %c : %m%attrs%n"
  asyncLogger:
    enabled: true
    queueSize: 8192
    batchSize: 256
    overflowStrategy: block
    waitStrategy: yield
    includeLocation: false
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
    rolling:
      type: rolling-file
      fileName: "${prop:LOG_DIR}/app.log"
      bufferSize: 256KiB
      layout:
        type: json
        eventEol: true
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%d{yyyyMMdd}-%i.log.gz"
        policies:
          size:
            size: 100MiB
          time:
            interval: daily
            modulate: true
          startup:
            enabled: true
        strategy:
          max: 30
          maxAge: 30d
          compression:
            gzip: true
            async: true
          delete:
            basePath: "${prop:LOG_DIR}/archive"
            maxDepth: 1
            ifFileName:
              glob: "*.log.gz"
            ifLastModified:
              age: 30d
            async: true
  root:
    level: info
    appenderRefs: [console, rolling]
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
```

The same model can be placed at the top level, under `configuration`, or under
`goark.log`. Use only one form per file.

## Native Logger

Use `slog` for ecosystem compatibility. Use the native logger when a hot path
needs fewer allocations and direct `slog.Attr` handling.

```go
package main

import (
	"context"
	"io"
	"log/slog"
	"time"

	goarklog "goark.dev/log"
)

func main() {
	appender := goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard))
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{appender},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"json"},
		},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := goarklog.NewNativeLogger(handler, "goark.http")
	if err != nil {
		panic(err)
	}

	_ = logger.LogAttrs3(context.Background(), slog.LevelInfo, "request done",
		slog.String("method", "GET"),
		slog.Int("status", 200),
		slog.Duration("elapsed", 8*time.Millisecond),
	)
}
```

## Capability Summary

| Area | Supported in core |
| --- | --- |
| Standard library integration | `slog.Handler`, `slog.Logger`, `LogAttrs`, `WithAttrs`, `WithGroup`. |
| Native logging | Named native logger, fixed three-attribute fast path, builder API, message factories. |
| Routing | Root logger, named logger rules, prefix matching, additivity, appender-ref controls. |
| Appenders | Console, File, JSON, RollingFile, Async, Failover, Routing, Rewrite. |
| Layouts | Pattern, Text, JSON, JSON Template, XML, CSV, GELF, RFC5424/Syslog, YAML, HTML. |
| Filters | Threshold, Level, LevelRange, Regex, Attr, Marker, Map, Throwable, Time, Burst, DynamicThreshold, and related aliases. |
| Configuration | YAML, JSON, XML, properties, local lookups, reload polling. |
| Rolling files | Size, time, cron, startup rollover, `%d`/`%i`, gzip, retention, delete actions. |
| Async | Bounded ring buffer, batching, block/drop/drop-debug/sync-fallback, shutdown drain. |
| Extensibility | Explicit plugin registry, plugin set, lookup plugins, JSON Template resolver plugins, registrar generator. |

HTTP, Socket, Syslog network output, Kafka, SMTP, database sinks, OpenTelemetry,
Prometheus, and script engines are not built into the core module. Add them as
separate modules that register explicit plugins.

## Documentation

- [Documentation index](docs/index.md)
- [Programmatic API](docs/api.md)
- [Configuration reference](docs/configuration.md)
- [Appender reference](docs/appenders.md)
- [Layout reference](docs/layouts.md)
- [Filter reference](docs/filters.md)
- [Usage scenarios](docs/scenarios.md)
- [Extensibility guide](docs/extensibility.md)
- [Capability boundary](docs/capabilities.md)
- [Performance and stress testing](docs/performance.md)
- [v0.0.2 release checklist](docs/release-v0.0.2.md)
- [Runnable examples](examples/README.md)

## Verification

Unix shell:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

PowerShell:

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

The comparison benchmarks live in a separate module:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
```

## Release Notes

`dev` is the integration branch. Release tags should be cut from `main` after
`dev` has been validated and fast-forwarded or merged according to the release
process. Use [docs/release-v0.0.2.md](docs/release-v0.0.2.md) before publishing
`v0.0.2`.
