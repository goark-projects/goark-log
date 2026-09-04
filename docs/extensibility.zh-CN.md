# 扩展机制

[English](extensibility.md)

`goark-log` 的扩展点全部是显式的。应用和配套模块在启动阶段注册插件；核心不会在运行时
扫描文件系统、classpath、module cache 或包图。

## 扩展点

| 扩展点 | 注册 API | 配置/运行时用途 |
| --- | --- | --- |
| Appender | `RegisterAppender` 或 `WithPluginAppender` | 创建外部网络或消息队列等配置化 sink。 |
| Layout | `RegisterLayout` 或 `WithPluginLayout` | 创建自定义事件编码器。 |
| Filter | `RegisterFilter` 或 `WithPluginFilter` | 创建自定义事件门控。 |
| Lookup | `RegisterLookup` 或 `WithPluginLookup` | 在运行时构建前解析配置中的 `${namespace:key}`。 |
| JSON Template resolver | `RegisterJSONTemplateResolver` 或 `WithPluginJSONTemplateResolver` | 为 JSON Template 字段增加 resolver 名称。 |

插件 kind 匹配会忽略大小写、连字符和下划线。Lookup namespace 使用小写字符串。

## Registry 选择

插件是进程级并且应当表现得像内置插件时，使用 `DefaultPluginRegistry()`。

测试、demo 或应用需要隔离注册时，使用 `NewPluginRegistry()`：

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

把 registry 传入配置加载：

```go
loggerContext, _, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
	goarklog.WithPluginRegistry(registry),
)
```

## Appender 插件

Appender 插件接收 `AppenderBuildConfig` 并返回 `Appender`。build config 包含公共字段、
远程目标字段、layout、rolling 配置、appender 引用、filter 和 registry。

HTTP、socket、网络 syslog、Kafka、Pulsar、RabbitMQ、SMTP、数据库或云 sink 模块都应
放在这个边界后面。核心会解析其中若干字段，但不会实现对应客户端。

Appender 契约：

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

`Append` 必须支持并发调用。`Close` 必须释放自有资源并刷出缓冲数据。

## Layout 插件

Layout 插件接收 `LayoutBuildConfig` 并返回 `Layout`。

```go
type Layout interface {
	Append(buf *bytes.Buffer, event Event) error
}
```

Layout 插件应使用调用方提供的 buffer，不要保留 event 引用。带 complete 生命周期状态的
layout，应由 appender 拥有状态，或按 appender 克隆状态。

## Filter 插件

Filter 插件接收 `FilterBuildConfig` 并返回 `Filter`。

```go
type Filter interface {
	Decide(ctx context.Context, event Event) FilterDecision
}
```

`FilterNeutral` 表示无意见，`FilterAccept` 表示在当前链路允许，`FilterDeny` 表示丢弃。

## Lookup 插件

Lookup 插件在 appender、layout、filter 和 logger rule 构建前解析配置文本。

```go
func tenantLookup(key string) (string, bool) {
	if key == "default" {
		return "tenant-a", true
	}
	return "", false
}
```

安全策略会阻断 `jndi`、`ldap`、`rmi` namespace。没有默认值的缺失 lookup 会导致配置加载
失败。默认值形式为 `${namespace:key:-fallback}`。

## JSON Template Resolver 插件

Resolver 向事件输出中追加原始 JSON。

```go
type constantResolver string

func (r constantResolver) AppendJSON(buf *bytes.Buffer, _ goarklog.Event) {
	data, err := sonic.ConfigFastest.Marshal(string(r))
	if err != nil {
		buf.WriteString("null")
		return
	}
	buf.Write(data)
}
```

工厂选项来自 resolver 对象中的原始 JSON 值：

```go
func buildConstantResolver(config goarklog.JSONTemplateResolverBuildConfig) (goarklog.JSONTemplateResolver, error) {
	var value string
	if err := sonic.ConfigFastest.Unmarshal(config.Options["value"], &value); err != nil {
		return nil, fmt.Errorf("constant resolver value is invalid: %w", err)
	}
	return constantResolver(value), nil
}
```

配置：

```yaml
layout:
  type: jsonTemplate
  eventTemplate: >
    {
      "component": {"$resolver": "constant", "value": "billing"},
      "message": {"$resolver": "message"}
    }
```

可运行 demo：

```bash
GOWORK=off go run ./examples/extensibility
```

## 生成 Registrar

`cmd/goark-log-plugin-gen` 生成小型 registrar，外部扩展模块不需要手写注册胶水代码。

```bash
GOWORK=off go run ./cmd/goark-log-plugin-gen \
  -package mylogplugin \
  -appender kafka=goark.dev/log/contrib/kafka.NewAppender \
  -lookup tenant=goark.dev/myapp/logging.TenantLookup \
  -json-template-resolver build=goark.dev/myapp/logging.BuildResolver \
  -output plugins_gen.go
```

生成文件包含兼容 `RegisterPlugins` 的 `PluginRegistrar`。

## 插件边界

插件模块保持职责窄化：

| 模块类型 | 应包含内容 |
| --- | --- |
| 网络 sink | 连接生命周期、重试、超时、批处理和 appender factory。 |
| Broker sink | Producer 生命周期、序列化、背压和 appender factory。 |
| 云 exporter | 认证、传输、资源映射和 appender factory。 |
| 自定义 layout | 只做编码；不要从 layout 打开文件或网络连接。 |
| 自定义 filter | 谓词和必要的小型状态。 |

不要为了可选目标把重量级依赖放入核心。

## 验证清单

| 检查 | 命令或期望 |
| --- | --- |
| Registry 拒绝 nil factory 和空 kind。 | 插件模块单元测试。 |
| 配置示例可加载。 | `GOWORK=off go test ./internal/integration -run TestDocsExamples -count=1`。 |
| 并发行为无数据竞争。 | 并发模块运行 `GOWORK=off go test -race ./...`。 |
| 热点性能结论有测量依据。 | 在所属模块提供 benchmark。 |
| 关闭行为确定。 | `Close` 完成 drain 并返回传输错误。 |
