# Extensibility

[简体中文](extensibility.zh-CN.md)

`goark-log` extension points are explicit. Applications and companion modules
register plugins during startup; the core never scans the filesystem,
classpath, module cache, or package graph at runtime.

## Extension Points

| Extension point | Registration API | Config/runtime use |
| --- | --- | --- |
| Appender | `RegisterAppender` or `WithPluginAppender` | Creates configured sinks such as external network or broker appenders. |
| Layout | `RegisterLayout` or `WithPluginLayout` | Creates custom event encoders. |
| Filter | `RegisterFilter` or `WithPluginFilter` | Creates custom event gates. |
| Lookup | `RegisterLookup` or `WithPluginLookup` | Resolves `${namespace:key}` in configuration before runtime build. |
| JSON Template resolver | `RegisterJSONTemplateResolver` or `WithPluginJSONTemplateResolver` | Adds resolver names for JSON Template fields. |

Plugin kind matching ignores case, hyphen, and underscore. Lookup namespaces
are lower-case strings.

## Registry Choices

Use `DefaultPluginRegistry()` when plugins are process-wide and should behave
like built-ins.

Use `NewPluginRegistry()` when tests, demos, or applications need isolated
registrations:

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

Pass the registry to config loading:

```go
loggerContext, _, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

## Appender Plugins

Appender plugins receive `AppenderBuildConfig` and return an `Appender`.
The build config includes common fields, remote-oriented fields, layout,
rolling config, appender references, filters, and the registry.

Use this boundary for HTTP, socket, network syslog, Kafka, Pulsar, RabbitMQ,
SMTP, database, or cloud sink modules. The core parses several fields for these
uses but does not implement the clients.

Appender contract:

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

`Append` must be safe for concurrent callers. `Close` must release all owned
resources and flush buffered data.

## Layout Plugins

Layout plugins receive `LayoutBuildConfig` and return a `Layout`.

```go
type Layout interface {
	Append(buf *bytes.Buffer, event Event) error
}
```

Layout plugins should use caller-owned buffers and avoid retaining event
references. If a layout has complete-mode lifecycle state, keep that state
owned by the appender or clone it per appender.

## Filter Plugins

Filter plugins receive `FilterBuildConfig` and return a `Filter`.

```go
type Filter interface {
	Decide(ctx context.Context, event Event) FilterDecision
}
```

Return `FilterNeutral` for "no opinion", `FilterAccept` to allow within the
current chain, and `FilterDeny` to drop.

## Lookup Plugins

Lookup plugins resolve config text before appenders, layouts, filters, and
logger rules are built.

```go
func tenantLookup(key string) (string, bool) {
	if key == "default" {
		return "tenant-a", true
	}
	return "", false
}
```

Security policy blocks `jndi`, `ldap`, and `rmi` namespaces. Missing lookups
without defaults fail configuration loading. Defaults use the form
`${namespace:key:-fallback}`.

## JSON Template Resolver Plugins

Resolvers append raw JSON into the event output.

```go
type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ goarklog.Event) {
	data, err := json.Marshal(string(r))
	if err != nil {
		buf.WriteString("null")
		return
	}
	buf.Write(data)
}
```

Factory options are raw JSON values from the resolver object:

```go
func buildConstantResolver(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
	var value string
	if err := json.Unmarshal(config.Options["value"], &value); err != nil {
		return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
	}
	return constantResolver(value), nil
}
```

Config:

```yaml
layout:
  type: jsonTemplate
  eventTemplate: >
    {
      "component": {"$resolver": "constant", "value": "billing"},
      "message": {"$resolver": "message"}
    }
```

Runnable demo:

```bash
GOWORK=off go run ./examples/extensibility
```

## Generated Registrars

`cmd/goark-log-plugin-gen` generates a small registrar so extension modules can
avoid hand-written registration glue.

```bash
GOWORK=off go run ./cmd/goark-log-plugin-gen \
  -package mylogplugin \
  -appender kafka=goark.dev/log/contrib/kafka.NewAppender \
  -lookup tenant=goark.dev/myapp/logging.TenantLookup \
  -json-template-resolver build=goark.dev/myapp/logging.BuildResolver \
  -output plugins_gen.go
```

Generated files contain a `PluginRegistrar` compatible with
`RegisterPlugins`.

## Plugin Boundaries

Keep plugin modules narrow:

| Module type | Should contain |
| --- | --- |
| Network sink | Connection lifecycle, retries, timeouts, batching, and appender factory. |
| Broker sink | Producer lifecycle, serialization, backpressure, and appender factory. |
| Cloud exporter | Authentication, transport, resource mapping, and appender factory. |
| Custom layout | Encoding only; do not open files or network connections from a layout. |
| Custom filter | Predicate and optional small state only. |

Do not add heavyweight dependencies to the core for optional destinations.

## Validation Checklist

| Check | Command or expectation |
| --- | --- |
| Registry rejects nil factories and empty kinds. | Unit tests in the plugin module. |
| Config examples load. | `GOWORK=off go test ./internal/integration -run TestDocsExamples -count=1`. |
| Race behavior is clean. | `GOWORK=off go test -race ./...` for the module that owns concurrency. |
| Hot path claims are measured. | Benchmarks in the owning module. |
| Shutdown is deterministic. | `Close` drains and returns transport errors. |
