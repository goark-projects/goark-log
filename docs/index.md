# Documentation Index

[简体中文](index.zh-CN.md)

This directory is the reference documentation for `goark.dev/log` v0.0.2
preparation. The root README stays short; detailed configuration, operational
scenarios, and release checks live here.

## Start Here

| Document | Purpose |
| --- | --- |
| [Programmatic API](api.md) | Handler construction, native logger, context attributes, reload, status logger, and close ownership. |
| [Configuration](configuration.md) | Full configuration model, load order, YAML/JSON/XML/properties forms, lookups, levels, reload, and routing. |
| [Appenders](appenders.md) | Console, file, JSON, rolling file, async, failover, routing, rewrite, and plugin appender parameters. |
| [Layouts](layouts.md) | Pattern, JSON, JSON Template, Text, XML, CSV, GELF, RFC5424/Syslog, YAML, HTML, converter tables, and resolver tables. |
| [Filters](filters.md) | Filter chain semantics, decisions, built-in filters, parameters, and examples. |
| [Scenarios](scenarios.md) | Copyable scenarios for development, containers, production rolling logs, audit split, reload, routing, redaction, and extension. |
| [Extensibility](extensibility.md) | Explicit plugin registration for appenders, layouts, filters, lookups, and JSON Template resolvers. |
| [Capability Boundary](capabilities.md) | What the core module supports and what belongs in external modules. |
| [Performance](performance.md) | Performance budgets, benchmark commands, pressure tests, and tuning notes. |
| [v0.0.2 Release Checklist](release-v0.0.2.md) | Local and remote checks before publishing v0.0.2. |

## Copyable Configuration Examples

| File | Scenario |
| --- | --- |
| [examples/README.md](examples/README.md) | Directory guide for copyable configuration examples. |
| [examples/console.yml](examples/console.yml) | Human-readable development console output. |
| [examples/json-stdout.yml](examples/json-stdout.yml) | Container and Kubernetes stdout JSON logs. |
| [examples/production-rolling.yml](examples/production-rolling.yml) | Production JSON rolling file with gzip and retention. |
| [examples/split-audit.yml](examples/split-audit.yml) | Separate application and audit logs. |
| [examples/async-appender.yml](examples/async-appender.yml) | Appender-level async wrapping only selected sinks. |
| [examples/rewrite-routing.yml](examples/rewrite-routing.yml) | Attribute rewrite and routing by tenant. |
| [examples/goark-log.properties](examples/goark-log.properties) | Properties configuration equivalent for simpler deployments. |
| [examples/log4j2-style.xml](examples/log4j2-style.xml) | XML configuration using Log4j2-style element names supported by the parser. |

## Supported Formats

- YAML: recommended for service configuration.
- JSON: supported through the same structured decoder and field names.
- XML: supports Log4j2-style appender, layout, filter, policy, strategy, and logger elements.
- properties: supported with flat keys such as `appender.console.type` and `rootLogger.level`.
- TOML: recognized only to fail fast with an explicit unsupported-format error.

## Non-Core Scope

The core module intentionally does not include HTTP, Socket, Syslog network
output, Kafka, SMTP, database sinks, OpenTelemetry, Prometheus, or embedded
script engines. These integrations require connection lifecycle, credentials,
retry, batching, and failure semantics that should live in dedicated modules
and register plugins explicitly.
