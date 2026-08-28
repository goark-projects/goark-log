# goark-log v0.0.2

[简体中文](github-release-v0.0.2.zh-CN.md)

`goark-log` v0.0.2 is a production-readiness release for the Goark logging
core. It keeps `log/slog` as the first-class facade and expands the runtime
around Log4j2- and SLF4J-style service logging: richer configuration, rolling
files, async queues, filters, layouts, explicit plugins, and bilingual
documentation.

## Highlights

- Added TOML configuration support alongside YAML, JSON, Log4j2-style XML, and
  Java properties.
- Expanded Log4j2-style runtime coverage: rolling policies, rollover strategy,
  appender references, composite appenders, routing, rewrite, failover, async,
  and filter chains.
- Added production-shaped docs and demos for service logging, SLF4J-style
  parameterized logging, and Log4j2-style XML configuration.
- Rebuilt the public documentation system in English by default, with
  Simplified Chinese counterparts for every public Markdown page.
- Added load-tested configuration examples for console, container JSON,
  complete JSON streams, production rolling files, audit routing, async
  failover, filter coverage, JSON Template, TOML, properties, and XML.
- Kept optional network and observability sinks outside the core through the
  explicit plugin boundary.

## Fixes

- Fixed composite appender filters so configured async, failover, routing, and
  rewrite appenders apply appender-level filter chains consistently.
- Fixed lazy file lifecycle handling: file and rolling-file appenders no longer
  create or touch a `createOnDemand` target when closed before the first event.
- Kept rolling archive gzip/delete actions serialized under pressure, avoiding
  Windows file-use races.
- Hardened production file logging paths around buffering, permissions,
  close-time layout lifecycle, and directory validation.

## Performance And Validation

The release candidate was validated on Windows/amd64 with Go 1.27.0:

```bash
GOWORK=off go test -count=1 ./...
GOWORK=off go vet ./...
GOWORK=off go test -race -count=1 ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -run '^$' -bench . -benchmem -count=1 ./benchmarks/core
pushd benchmarks/compare
GOWORK=off go test -count=1 ./...
GOWORK=off go test -run '^$' -bench . -benchmem -count=1
popd
```

Runnable examples were smoke-tested:

```bash
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

Core hot-path benchmarks confirmed zero allocations for the native direct JSON
three-attribute path on the validation machine. Comparison benchmarks remain in
the separate `benchmarks/compare` module; this release does not claim universal
throughput superiority over zap or zerolog.

## Upgrade

```bash
go get goark.dev/log@v0.0.2
```

Start with:

- [README](../README.md)
- [Production guide](production-guide.md)
- [Configuration reference](configuration-reference.md)
- [Configuration examples](examples/README.md)
- [Runnable examples](../examples/README.md)

## Boundaries

The core module does not ship HTTP appenders, socket appenders, network syslog
clients, Kafka, Pulsar, RabbitMQ, SMTP, database sinks, OpenTelemetry
exporters, Prometheus exporters, or an embedded script runtime. These belong in
separate modules that register explicit plugins.

**Full Changelog**: `v0.0.1...v0.0.2`
