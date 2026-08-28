# Runnable Examples

The `examples/` directory contains small runnable programs that compile against
the core module only. They do not require external services.

## Commands

```bash
GOWORK=off go test ./examples/...
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
```

PowerShell:

```powershell
$env:GOWORK='off'
go test ./examples/...
go run ./examples/console
go run ./examples/file
go run ./examples/rolling
go run ./examples/async
go run ./examples/reload
go run ./examples/extensibility
```

## Example Programs

| Directory | Purpose | Output |
| --- | --- | --- |
| `console` | Default console logger and named logger usage. | stderr. |
| `file` | Plain file appender with explicit close. | `goark-log-example/file.log` under the system temp directory. |
| `rolling` | Size rollover, startup rollover, archive pattern, and gzip compression. | `goark-log-example/rolling.log` and archive files under the system temp directory. |
| `async` | AsyncAppender wrapping a rolling appender. | `goark-log-example/async-rolling.log` and archives under the system temp directory. |
| `reload` | Config file loading and runtime reload. | Temporary config plus console output. |
| `extensibility` | `PluginRegistry`, custom JSON Template resolver, and message factory. | stdout JSON. |

## Config Examples

Copyable configuration files live under [../docs/examples](../docs/examples):

- [console.yml](../docs/examples/console.yml)
- [json-stdout.yml](../docs/examples/json-stdout.yml)
- [production-rolling.yml](../docs/examples/production-rolling.yml)
- [split-audit.yml](../docs/examples/split-audit.yml)
- [async-appender.yml](../docs/examples/async-appender.yml)
- [rewrite-routing.yml](../docs/examples/rewrite-routing.yml)
- [goark-log.properties](../docs/examples/goark-log.properties)
- [log4j2-style.xml](../docs/examples/log4j2-style.xml)

## Reading Order

1. `console`: minimal integration.
2. `file`: file write lifecycle and close behavior.
3. `rolling`: archive, compression, and retention behavior.
4. `async`: async wrapper and shutdown drain.
5. `reload`: config reload entry points.
6. `extensibility`: plugin registration and resolver extension.

## Rules for New Examples

- They must compile with `go test ./examples/...`.
- File output must use a temp directory and must not write into the repository.
- They should demonstrate core capabilities only.
- Keep program code short; put detailed explanations in `docs/`.
