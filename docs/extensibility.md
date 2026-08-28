# Extensibility Guide

`goark-log` uses explicit plugin registration. It does not scan packages,
struct tags, file paths, or registries at runtime. This keeps startup
deterministic and keeps the core hot path dependency-light.

## Extension Points

| Extension point | Factory type | Register with |
| --- | --- | --- |
| Appender | `AppenderFactory` | `RegisterAppender`, `WithPluginAppender` |
| Layout | `LayoutFactory` | `RegisterLayout`, `WithPluginLayout` |
| Filter | `FilterFactory` | `RegisterFilter`, `WithPluginFilter` |
| Lookup | `LookupFunc` | `RegisterLookup`, `WithPluginLookup` |
| JSON Template resolver | `JSONTemplateResolverFactory` | `RegisterJSONTemplateResolver`, `WithPluginJSONTemplateResolver` |

Use the process default registry for simple applications. Use a dedicated
registry when a framework, test, or embedded runtime needs isolated plugin
state.

## Registry Usage

Default registry:

```go
err := goarklog.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginLookup("tenant", lookupTenant),
	goarklog.WithPluginLayout("line", buildLineLayout),
))
```

Isolated registry:

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginAppender("http", buildHTTPAppender),
	goarklog.WithPluginFilter("tenant", buildTenantFilter),
))
if err != nil {
	return err
}

handler, _, err := goarklog.NewConfiguredHandler(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

## Appender Plugin

Factory signature:

```go
type AppenderFactory func(config goarklog.AppenderBuildConfig) (goarklog.Appender, error)
```

Minimal appender:

```go
type discardAppender struct {
	name string
}

func (a *discardAppender) Name() string {
	if a.name == "" {
		return "discard"
	}
	return a.name
}

func (a *discardAppender) Append(ctx context.Context, event goarklog.Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (a *discardAppender) Close() error {
	return nil
}

func buildDiscardAppender(config goarklog.AppenderBuildConfig) (goarklog.Appender, error) {
	if strings.TrimSpace(config.Name) == "" {
		return nil, fmt.Errorf("discard appender name is empty")
	}
	return &discardAppender{name: config.Name}, nil
}
```

Registration:

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginAppender("discard", buildDiscardAppender),
))
```

Configuration:

```yaml
appenders:
  discard:
    type: discard
root:
  level: info
  appenderRefs: [discard]
```

Appender plugin rules:

- Validate the appender name and required fields.
- Respect `ctx.Err()` before expensive work.
- Make `Append` safe for concurrent callers.
- Make `Close` idempotent.
- Own external connection lifecycle inside the appender.
- Do not block forever on network writes; use timeouts, bounded queues, or
  caller-visible errors.
- Keep external dependencies in the plugin module, not in `goark.dev/log`.

## AppenderBuildConfig Fields

`AppenderBuildConfig` receives normalized inputs from configuration:

| Field | Source |
| --- | --- |
| `Name`, `Type` | Appender map key and configured type. |
| `Target` | `target`. |
| `URL`, `Method`, `Address`, `Network`, `Facility`, `AppName` | External appender fields. |
| `ConnectTimeout`, `WriteTimeout` | External timeout strings. |
| `FileName` | `fileName`, `file-name`, or `path`. |
| `Layout` | Built layout object. |
| `AppenderRefs` | Simple appender ref names. |
| `Delegates` | Resolved downstream appenders for composite plugins. |
| `Routes`, `DefaultRoute`, `RouteKey` | Resolved routing config. |
| `QueueSize`, `BatchSize`, `OverflowStrategy`, `WaitStrategy`, `WaitOptions` | Async fields. |
| `BufferSize`, `FlushOnWrite`, `Append`, `CreateOnDemand`, `FilePermissions` | File-style fields. |
| `Rolling` | Rolling build config. |
| `Rewrite` | Built-in rewrite policy config. |

The factory still owns semantic validation. A field being present in
`AppenderBuildConfig` does not mean the core module has a built-in appender for
that transport.

## Layout Plugin

Factory signature:

```go
type LayoutFactory func(config goarklog.LayoutBuildConfig) (goarklog.Layout, error)
```

Example:

```go
type lineLayout struct{}

func (lineLayout) Format(buf *bytes.Buffer, event goarklog.Event) error {
	buf.WriteString(event.Message)
	buf.WriteByte('\n')
	return nil
}

func buildLineLayout(config goarklog.LayoutBuildConfig) (goarklog.Layout, error) {
	return lineLayout{}, nil
}
```

Configuration:

```yaml
appenders:
  console:
    type: console
    layout:
      type: line
```

Layout plugin rules:

- Compile expensive templates or regex values in the factory, not in `Format`.
- Do not retain mutable event slices without copying.
- Write to the provided buffer only.
- Keep `Format` deterministic and free of network or filesystem side effects.

## Filter Plugin

Factory signature:

```go
type FilterFactory func(config goarklog.FilterBuildConfig) (goarklog.Filter, error)
```

Example:

```go
type tenantFilter struct {
	tenant string
}

func (f tenantFilter) Decide(ctx context.Context, event goarklog.Event) goarklog.FilterDecision {
	value, ok := event.Attr("tenant")
	if ok && value.String() == f.tenant {
		return goarklog.FilterNeutral
	}
	return goarklog.FilterDeny
}

func buildTenantFilter(config goarklog.FilterBuildConfig) (goarklog.Filter, error) {
	if strings.TrimSpace(config.Value) == "" {
		return nil, fmt.Errorf("tenant filter value is empty")
	}
	return tenantFilter{tenant: strings.TrimSpace(config.Value)}, nil
}
```

Configuration:

```yaml
filters:
  tenantA:
    type: tenant
    value: tenant-a
root:
  level: info
  filters: [tenantA]
```

Filter plugin rules:

- Return `neutral` for pass-through unless the plugin intentionally accepts.
- Return `deny` for policy rejection.
- Avoid allocations, regex compilation, map construction, and reflection in
  `Decide`.
- Make shared state immutable or lock-protected.

## Lookup Plugin

Lookup signature:

```go
type LookupFunc func(key string) (string, bool)
```

Example:

```go
func lookupTenant(key string) (string, bool) {
	switch key {
	case "id":
		return "tenant-a", true
	default:
		return "", false
	}
}
```

Configuration:

```yaml
properties:
  LOG_DIR: "logs/${tenant:id}"
```

Lookup plugin rules:

- Return `(value, true)` only when the value exists.
- Keep lookups local and deterministic.
- Do not perform unbounded network calls during configuration loading.
- Namespaces `jndi`, `ldap`, and `rmi` are blocked and cannot be registered.

## JSON Template Resolver Plugin

Factory signature:

```go
type JSONTemplateResolverFactory func(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error)
```

Example resolver:

```go
type constantResolver struct {
	value string
}

func (r constantResolver) AppendJSON(buf *bytes.Buffer, event goarklog.Event) {
	_ = event
	buf.WriteString(strconv.Quote(r.value))
}

func buildConstantResolver(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
	raw := config.Options["value"]
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("constant resolver value must be a string")
	}
	return constantResolver{value: value}, nil
}
```

Template:

```json
{
  "service": {"$resolver": "constant", "value": "billing"},
  "message": {"$resolver": "message"}
}
```

Resolver plugin rules:

- Parse and validate options in the factory.
- Append valid JSON values only.
- Avoid allocating in `AppendJSON` on hot paths.
- Do not mutate the event.

## Registrar Generator

The module includes a small generator for registrar boilerplate:

```bash
go run goark.dev/log/cmd/goark-log-plugin-gen \
  -package mylog \
  -appender discard=buildDiscardAppender \
  -layout line=buildLineLayout \
  -filter tenant=buildTenantFilter \
  -lookup tenant=lookupTenant \
  -json-template-resolver constant=buildConstantResolver \
  -out zz_generated_plugins.go
```

Use generated registrars when a plugin package exports several extension
points. Keep generated files committed so consumers do not need to run
generators during normal builds.

## Dependency Boundary

External integrations should live outside the core module:

| Integration | Reason to keep external |
| --- | --- |
| HTTP, Socket, Syslog network output | Connection management, retries, deadlines, TLS, and backpressure differ by deployment. |
| Kafka, Pulsar, RabbitMQ | Client dependencies and delivery semantics are heavy and broker-specific. |
| SMTP | Slow network I/O and credential handling should not enter the core logging path. |
| Database sinks | Transactions, batching, schema, and failure modes vary by database. |
| OpenTelemetry and Prometheus | Observability design should be consistent across Goark modules, not forced into the logging core. |
| Script engines | Runtime and sandbox choices have security implications. |

This boundary keeps `goark.dev/log` small, predictable, and suitable for low
level packages.
