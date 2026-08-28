# Runnable Examples

[简体中文](README.zh-CN.md)

These examples are production-shaped smoke demos for the current
`goark.dev/log` API and configuration model. They do not require external
services.

## Run All

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

## Demos

| Directory | Demonstrates |
| --- | --- |
| [console](console) | `ConfigureDefault`, named `slog` logger, and minimal console config. |
| [file](file) | Configured file output with complete JSON layout lifecycle. |
| [rolling](rolling) | Native logger fast path writing through the production rolling configuration. |
| [async](async) | Appender-level async queue, failover chain, and async counters. |
| [reload](reload) | Explicit `ConfigReloader` level change. |
| [extensibility](extensibility) | Isolated plugin registry, custom lookup, custom JSON Template resolver, and message factory. |
| [production](production) | Production-shaped service logging with MDC, NDC, marker, audit, health filtering, throwable stack, rolling files, and async appender. |
| [slf4j](slf4j) | SLF4J-style parameterized logging plus standard `slog` interop. |
| [log4j2_config](log4j2_config) | Log4j2-style XML configuration with rolling, async fan-out, routing, rewrite, filters, and named loggers. |

## Log Directory

File-writing demos call `examples/internal/exampleutil.PrepareLogDir`.

If `GOARK_LOG_DIR` is set, that directory is used:

```bash
GOARK_LOG_DIR=/tmp/goark-log-demo GOWORK=off go run ./examples/production
```

If it is not set, the demo creates a temporary directory and prints
`logDir=...`. Temporary directories are removed when the demo exits.

## Config Sources

The demos load files from [../docs/examples](../docs/examples). That directory
is also covered by integration tests, so the examples and docs share one source
of truth.

## Smoke Test

There is no single command in the module that runs every `go run` demo. For a
release candidate, run the commands listed in "Run All" and then run:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
