# Scenarios

[简体中文](scenarios.zh-CN.md)

This page gives copyable logging scenarios for the current `goark.dev/log`
core. Each scenario uses only built-in appenders, layouts, filters, and
configuration formats.

For field-level detail, use the [configuration reference](configuration-reference.md).

## Container JSON On Stdout

Use the JSON direct appender for stdout collectors. It avoids the generic
layout path and writes one JSON object per event.

```yaml
configuration:
  appenders:
    stdout:
      type: json
      target: stdout
  root:
    level: info
    appenderRefs: [stdout]
```

Runnable demo:

```bash
GOWORK=off go run ./examples/slf4j
```

## Local Service Logs With Retention

Use console output for diagnostics and an async rolling file for service logs.
The archive pattern contains `%d{...}` for time buckets and `%i` for repeated
rollovers within the same bucket.

```yaml
configuration:
  properties:
    LOG_DIR: "${env:GOARK_LOG_DIR:-logs}"
  appenders:
    console:
      type: console
      target: stderr
    appRolling:
      type: rolling-file
      fileName: "${prop:LOG_DIR}/app.log"
      layout:
        type: json
        eventEol: true
        includeStacktrace: true
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%d{yyyyMMdd-HHmmss}-%06i.log.gz"
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
    asyncFile:
      type: async
      appenderRefs: [appRolling]
      queueSize: 8192
      batchSize: 256
      overflowStrategy: block
      waitStrategy: yield
  root:
    level: info
    appenderRefs: [console, asyncFile]
```

Complete example: [production-service.yml](examples/production-service.yml).

## Audit Log Split From Application Log

Create a named logger with `additivity: false` when audit events must not be
duplicated into the root appenders.

```yaml
configuration:
  appenders:
    auditRolling:
      type: rolling-file
      fileName: "${env:GOARK_LOG_DIR:-logs}/audit.log"
      flushOnWrite: true
      filePermissions: "0600"
      layout:
        type: jsonTemplate
        eventTemplate: >
          {
            "timestamp": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
            "level": {"$resolver": "level"},
            "logger": {"$resolver": "logger"},
            "marker": {"$resolver": "marker"},
            "message": {"$resolver": "message"},
            "principal": {"$resolver": "attr", "key": "principal"},
            "action": {"$resolver": "attr", "key": "action"},
            "resource": {"$resolver": "attr", "key": "resource"},
            "contextMap": {"$resolver": "mdc"}
          }
        eventEol: true
  loggers:
    goark.audit:
      level: info
      appenderRefs: [auditRolling]
      additivity: false
```

In Go, use a marker and stable attribute names:

```go
ctx := goarklog.WithMarker(context.Background(), goarklog.NewMarker("AUDIT"))
loggerContext.Logger("goark.audit").InfoContext(ctx, "order approved",
	slog.String("principal", "alice"),
	slog.String("action", "approve"),
	slog.String("resource", "order:1001"),
)
```

## Tenant Routing And Redaction

Use `rewrite` before `routing` when sensitive keys must be removed before an
event reaches any route target.

```yaml
configuration:
  appenders:
    stdout:
      type: json
      target: stdout
    tenantA:
      type: file
      fileName: "${env:GOARK_LOG_DIR:-logs}/tenant-a.log"
      layout:
        type: json
        eventEol: true
    router:
      type: routing
      routeKey: tenant
      defaultRoute: stdout
      routes:
        tenant-a: tenantA
    redacted:
      type: rewrite
      appenderRefs: [router]
      rewrite:
        attrs:
          service: billing
        removeAttrs: [password, token, authorization]
  root:
    level: info
    appenderRefs: [redacted]
```

Complete example: [audit-routing.yml](examples/audit-routing.yml).

## Health Check Noise Control

Use a global or root filter with `onMismatch: neutral` so only the matching
noise is denied.

```yaml
configuration:
  filters:
    dropHealthCheck:
      type: stringMatch
      text: "/health"
      onMatch: deny
      onMismatch: neutral
  root:
    level: info
    filterRefs: [dropHealthCheck]
    appenderRefs: [console]
```

## Dynamic Tenant Thresholds

Use `dynamicThreshold` when one attribute decides the effective level. This is
useful for tenant-specific diagnostics without changing all logger rules.

```yaml
configuration:
  filters:
    tenantThreshold:
      type: dynamicThreshold
      key: tenant
      defaultThreshold: error
      thresholds:
        tenant-a: debug
        tenant-b: info
  filterRefs: [tenantThreshold]
```

## SLF4J-Style Parameterized Logging

The native logger supports `{}` placeholders without forcing attribute
allocation when the level is disabled.

```go
logger, err := goarklog.NewNativeLogger(handler, "goark.demo.slf4j",
	goarklog.WithLoggerMessageFactory(goarklog.ParameterizedMessageFactory{}),
)
if err != nil {
	return err
}
_ = logger.AtInfo().
	WithString("user", "alice").
	Logf("user {} finished request", "alice")
```

Runnable demo:

```bash
GOWORK=off go run ./examples/slf4j
```

## Log4j2-Style XML Configuration

XML configuration supports `Configuration`, `Properties`, `Appenders`,
`Filters`, `Loggers`, `Root`, `AppenderRef`, rolling policies, and rollover
strategy elements used by the core.

```bash
GOWORK=off go run ./examples/log4j2_config
```

Full XML example: [log4j2-service.xml](examples/log4j2-service.xml).

## Configuration Reload

Use `LoggerContext` when you want owned lifecycle and polling reload. Use
`ConfigReloader` for explicit reload.

```go
reloader, err := goarklog.NewConfigReloader(handler,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
if _, err := reloader.Reload(ctx); err != nil {
	return err
}
```

The handler builds the new runtime before it swaps the router. Handler-level
async enablement and queue shape cannot change during reload.

Runnable demo:

```bash
GOWORK=off go run ./examples/reload
```

## Complete JSON Files

`complete: true` writes layout lifecycle headers and footers. JSON and JSON
Template complete mode keep state per appender, so separate files remain valid
streams.

```yaml
configuration:
  appenders:
    jsonFile:
      type: file
      fileName: "${env:GOARK_LOG_DIR:-logs}/events.json"
      bufferSize: 0
      layout:
        type: json
        complete: true
  root:
    level: info
    appenderRefs: [jsonFile]
```

Runnable demo:

```bash
GOWORK=off go run ./examples/file
```

## Explicit Plugin Extension

Plugins are registered explicitly. Use a custom registry when an application or
module needs isolated plugin behavior.

```go
registry := goarklog.NewPluginRegistry()
plugins := goarklog.NewPluginSet(
	goarklog.WithPluginLookup("tenant", tenantLookup),
	goarklog.WithPluginJSONTemplateResolver("constant", buildConstantResolver),
)
if err := registry.RegisterPlugins(plugins); err != nil {
	return err
}
```

Runnable demo:

```bash
GOWORK=off go run ./examples/extensibility
```

## Production Checklist

For services, start with these defaults:

| Area | Recommendation |
| --- | --- |
| API | Use `slog` for ordinary code and native logger builders for hot paths. |
| Output | Use JSON direct for stdout collectors or rolling JSON files for local files. |
| Backpressure | Prefer `block` for audit and required service logs. Use `drop` only for non-critical diagnostics. |
| Retention | Configure both archive count and age when local disks are finite. |
| Redaction | Apply rewrite filters before routing or failover destinations. |
| Shutdown | Call `Close` on handlers or logger contexts. |
| Reload | Keep handler-level async shape stable across reloads. |
