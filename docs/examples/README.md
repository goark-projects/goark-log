# Configuration Examples

[简体中文](README.zh-CN.md)

Every file in this directory is load-tested by
`TestDocsExamples_whenLoaded_shouldBuildOptions`. They are intended as current,
copyable examples, not historical compatibility samples.

## Files

| File | Format | Scenario |
| --- | --- | --- |
| [basic-console.yml](basic-console.yml) | YAML | Minimal console logger with pattern layout. |
| [container-json.yml](container-json.yml) | YAML | JSON direct stdout for container collectors. |
| [complete-json-file.yml](complete-json-file.yml) | YAML | Complete JSON file stream with layout lifecycle. |
| [production-service.yml](production-service.yml) | YAML | Production-style console, async rolling app log, audit log, filters, and reload interval. |
| [audit-routing.yml](audit-routing.yml) | YAML | Routing by tenant plus rewrite redaction. |
| [async-failover.yml](async-failover.yml) | YAML | Async appender with failover chain. |
| [filters-showcase.yml](filters-showcase.yml) | YAML | Every built-in configured filter family. |
| [json-template.yml](json-template.yml) | YAML | JSON Template resolver fields. |
| [log4j2-service.xml](log4j2-service.xml) | XML | Log4j2-style service configuration using rolling, async fan-out, routing, rewrite, filters, and named loggers. |
| [goark-log.toml](goark-log.toml) | TOML | TOML equivalent for common local-file configuration. |
| [goark-log.properties](goark-log.properties) | properties | Java properties mapping example. |

## Loading A File

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("docs/examples/production-service.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

loggerContext.Logger("goark.demo").Info("ready", slog.String("source", string(result.Source)))
```

## Environment Variables

Most file-writing examples use `GOARK_LOG_DIR`:

```bash
GOARK_LOG_DIR=/var/log/my-service GOWORK=off go run ./examples/production
```

When the runnable demos are used without `GOARK_LOG_DIR`, they create a
temporary directory and print `logDir=...`.

## Format Notes

| Format | Notes |
| --- | --- |
| YAML / JSON / TOML | Share the same logical model. Use the tables in [configuration reference](../configuration-reference.md). |
| XML | Uses Log4j2-style elements. Core supports the elements used by `log4j2-service.xml`. |
| properties | Uses practical Java properties mapping. Some advanced rolling nested policy fields are better expressed in YAML, TOML, or XML. |

## Validation

```bash
GOWORK=off go test ./internal/integration -run TestDocsExamples -count=1
```
