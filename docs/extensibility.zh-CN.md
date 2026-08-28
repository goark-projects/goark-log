# 扩展指南

[English](extensibility.md)

`goark-log` 使用显式 plugin registration。它不会在运行时扫描 packages、struct tags、file paths 或 registries。这能保持启动确定性，并让 core hot path 维持轻依赖。

## Extension Points

| Extension point | Factory type | 注册方式 |
| --- | --- | --- |
| Appender | `AppenderFactory` | `RegisterAppender`, `WithPluginAppender` |
| Layout | `LayoutFactory` | `RegisterLayout`, `WithPluginLayout` |
| Filter | `FilterFactory` | `RegisterFilter`, `WithPluginFilter` |
| Lookup | `LookupFunc` | `RegisterLookup`, `WithPluginLookup` |
| JSON Template resolver | `JSONTemplateResolverFactory` | `RegisterJSONTemplateResolver`, `WithPluginJSONTemplateResolver` |

简单应用可以使用 process default registry。Framework、test 或 embedded runtime 需要隔离 plugin state 时，应创建 dedicated registry。

## Registry Usage

Default registry：

```go
err := goarklog.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginLookup("tenant", lookupTenant),
	goarklog.WithPluginLayout("line", buildLineLayout),
))
```

Isolated registry：

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

Factory signature：

```go
type AppenderFactory func(config goarklog.AppenderBuildConfig) (goarklog.Appender, error)
```

Minimal appender：

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

Registration：

```go
registry := goarklog.NewPluginRegistry()
err := registry.RegisterPlugins(goarklog.NewPluginSet(
	goarklog.WithPluginAppender("discard", buildDiscardAppender),
))
```

Configuration：

```yaml
appenders:
  discard:
    type: discard
root:
  level: info
  appenderRefs: [discard]
```

Appender plugin 规则：

- 校验 appender name 和 required fields。
- 执行昂贵工作前检查 `ctx.Err()`。
- `Append` 必须对并发调用安全。
- `Close` 必须 idempotent。
- 外部连接生命周期由 appender 自己拥有。
- 网络写入不能永久阻塞；使用 timeouts、bounded queues 或 caller-visible errors。
- 外部依赖放在 plugin module，不放进 `goark.dev/log`。

## AppenderBuildConfig Fields

`AppenderBuildConfig` 接收配置归一化后的输入：

| 字段 | 来源 |
| --- | --- |
| `Name`, `Type` | Appender map key 和 configured type。 |
| `Target` | `target`。 |
| `URL`, `Method`, `Address`, `Network`, `Facility`, `AppName` | External appender fields。 |
| `ConnectTimeout`, `WriteTimeout` | External timeout strings。 |
| `FileName` | `fileName`、`file-name` 或 `path`。 |
| `Layout` | 构建完成的 layout object。 |
| `AppenderRefs` | Simple appender ref names。 |
| `Delegates` | 已解析 downstream appenders，供 composite plugins 使用。 |
| `Routes`, `DefaultRoute`, `RouteKey` | 已解析 routing config。 |
| `QueueSize`, `BatchSize`, `OverflowStrategy`, `WaitStrategy`, `WaitOptions` | Async fields。 |
| `BufferSize`, `FlushOnWrite`, `Append`, `CreateOnDemand`, `FilePermissions` | File-style fields。 |
| `Rolling` | Rolling build config。 |
| `Rewrite` | Built-in rewrite policy config。 |

Factory 仍然要负责 semantic validation。`AppenderBuildConfig` 中存在某个字段，并不表示 core module 内置了对应 transport 的 appender。

## Layout Plugin

Factory signature：

```go
type LayoutFactory func(config goarklog.LayoutBuildConfig) (goarklog.Layout, error)
```

示例：

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

Configuration：

```yaml
appenders:
  console:
    type: console
    layout:
      type: line
```

Layout plugin 规则：

- 在 factory 中编译昂贵 templates 或 regex values，不要放在 `Format`。
- 不要在未复制的情况下持有 mutable event slices。
- 只写入传入的 buffer。
- `Format` 应保持 deterministic，不做网络或文件系统副作用。

## Filter Plugin

Factory signature：

```go
type FilterFactory func(config goarklog.FilterBuildConfig) (goarklog.Filter, error)
```

示例：

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

Configuration：

```yaml
filters:
  tenantA:
    type: tenant
    value: tenant-a
root:
  level: info
  filters: [tenantA]
```

Filter plugin 规则：

- pass-through 使用 `neutral`，除非 plugin 明确要 accept。
- policy rejection 使用 `deny`。
- `Decide` 中避免 allocations、regex compilation、map construction 和 reflection。
- 共享状态必须 immutable 或受 lock 保护。

## Lookup Plugin

Lookup signature：

```go
type LookupFunc func(key string) (string, bool)
```

示例：

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

Configuration：

```yaml
properties:
  LOG_DIR: "logs/${tenant:id}"
```

Lookup plugin 规则：

- 只有值存在时才返回 `(value, true)`。
- Lookups 应保持 local 和 deterministic。
- 配置加载期间不要做无界网络调用。
- `jndi`、`ldap`、`rmi` namespaces 被阻止，不能注册。

## JSON Template Resolver Plugin

Factory signature：

```go
type JSONTemplateResolverFactory func(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error)
```

示例 resolver：

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

Template：

```json
{
  "service": {"$resolver": "constant", "value": "billing"},
  "message": {"$resolver": "message"}
}
```

Resolver plugin 规则：

- 在 factory 中解析并校验 options。
- 只能 append valid JSON values。
- 热路径 `AppendJSON` 避免分配。
- 不要修改 event。

## Registrar Generator

模块包含一个生成 registrar boilerplate 的小工具：

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

当 plugin package 导出多个 extension points 时，使用 generated registrars。Generated files 应提交到仓库，消费者正常 build 不需要运行 generator。

## Dependency Boundary

外部集成应放在 core module 之外：

| Integration | 保持外部的原因 |
| --- | --- |
| HTTP, Socket, Syslog network output | Connection management、retries、deadlines、TLS 和 backpressure 因部署而异。 |
| Kafka, Pulsar, RabbitMQ | Client dependencies 和 delivery semantics 重且 broker-specific。 |
| SMTP | Slow network I/O 和 credential handling 不应进入 core logging path。 |
| Database sinks | Transactions、batching、schema 和 failure modes 因数据库而异。 |
| OpenTelemetry and Prometheus | Observability design 应在 Goark modules 间统一，而不是被 logging core 强制决定。 |
| Script engines | Runtime 和 sandbox 选择有安全影响。 |

这个边界让 `goark.dev/log` 保持小、可预测，并适合被 low level packages 使用。
