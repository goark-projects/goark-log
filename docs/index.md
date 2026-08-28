# goark-log Documentation

[简体中文](index.zh-CN.md)

This documentation is written from the current `goark.dev/log` source. English
is the default public language; every public Markdown page has a Simplified
Chinese counterpart.

## Read First

| Need | Document |
| --- | --- |
| Install and run a logger in one minute | [README](../README.md) |
| Use a production-ready service configuration | [Production guide](production-guide.md) |
| Understand configuration discovery and wrappers | [Configuration model](configuration.md) |
| Find every supported field and alias | [Configuration reference](configuration-reference.md) |
| Migrate Log4j2 or SLF4J usage | [Log4j2 and SLF4J parity](log4j2-slf4j-parity.md) |

## Reference

| Area | Document |
| --- | --- |
| Public Go API | [Programmatic API](api.md) |
| Appender behavior and fields | [Appenders](appenders.md) |
| Output formats and pattern syntax | [Layouts](layouts.md) |
| Filter decisions and filter types | [Filters](filters.md) |
| Recipes for real services | [Scenarios](scenarios.md) |
| Plugins and generated registrars | [Extensibility](extensibility.md) |
| Implemented and unsupported features | [Capabilities](capabilities.md) |
| Benchmarks and hot-path constraints | [Performance](performance.md) |
| Release validation | [v0.0.2 checklist](release-v0.0.2.md) |

## Examples

| Example set | Contents |
| --- | --- |
| [Configuration examples](examples/README.md) | YAML, TOML, XML, and properties files loaded by tests. |
| [Runnable examples](../examples/README.md) | `go run` demos for console, file, rolling, async, reload, plugins, production, SLF4J-style usage, and Log4j2-style XML. |

## Source-Backed Boundaries

The core module currently implements local file and console outputs, direct JSON
output, rolling files, appender composition, layouts, filters, configuration,
reload, and explicit plugins.

The core module does not implement HTTP appenders, socket appenders, network
syslog clients, Kafka, Pulsar, RabbitMQ, SMTP, database sinks, OpenTelemetry
exporters, Prometheus exporters, or embedded script execution. These are
intentional external module boundaries.

## Validation Commands

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
```

Use the proxy below only when a command needs network access:

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```
