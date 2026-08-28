# 场景指南

[English](scenarios.md)

本页给出当前 `goark.dev/log` 核心模块支持的可复制日志场景。所有示例只使用内置
appender、layout、filter 和配置格式。

字段级细节见[配置参考](configuration-reference.zh-CN.md)。

## 容器 stdout JSON

容器采集链路优先使用 JSON direct appender。它绕过通用 layout 路径，每条事件输出
一个 JSON 对象。

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

可运行 demo：

```bash
GOWORK=off go run ./examples/slf4j
```

## 本地服务日志与保留策略

控制台用于诊断，本地服务日志使用异步滚动文件。归档模式用 `%d{...}` 划分时间桶，
用 `%i` 支持同一个时间桶内多次滚动。

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

完整示例：[production-service.yml](examples/production-service.yml)。

## 审计日志与业务日志隔离

审计日志不应被 root appender 重复写入时，使用 `additivity: false` 的命名 logger。

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

Go 代码中使用 marker 和稳定属性名：

```go
ctx := goarklog.WithMarker(context.Background(), goarklog.NewMarker("AUDIT"))
loggerContext.Logger("goark.audit").InfoContext(ctx, "order approved",
	slog.String("principal", "alice"),
	slog.String("action", "approve"),
	slog.String("resource", "order:1001"),
)
```

## 租户路由与脱敏

敏感字段需要在进入任意目标前删除时，把 `rewrite` 放在 `routing` 前面。

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

完整示例：[audit-routing.yml](examples/audit-routing.yml)。

## 健康检查降噪

只丢弃命中的噪声时，filter 需要配置 `onMismatch: neutral`，避免误拦截其他事件。

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

## 租户动态阈值

当某个事件属性决定有效日志级别时，使用 `dynamicThreshold`。它适合在不调整所有
logger 规则的情况下开启指定租户诊断。

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

## SLF4J 风格参数化日志

原生 logger 支持 `{}` 占位符。级别关闭时，builder 会跳过属性构造和写入路径。

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

可运行 demo：

```bash
GOWORK=off go run ./examples/slf4j
```

## Log4j2 风格 XML 配置

XML 配置支持核心已实现的 `Configuration`、`Properties`、`Appenders`、`Filters`、
`Loggers`、`Root`、`AppenderRef`、滚动策略和 rollover strategy 元素。

```bash
GOWORK=off go run ./examples/log4j2_config
```

完整 XML 示例：[log4j2-service.xml](examples/log4j2-service.xml)。

## 配置重载

需要托管生命周期和轮询重载时使用 `LoggerContext`。需要显式触发时使用
`ConfigReloader`。

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

handler 会先构建新运行时，再原子替换路由。handler 级异步的启用状态和队列形态
不能在重载期间改变。

可运行 demo：

```bash
GOWORK=off go run ./examples/reload
```

## 完整 JSON 文件

`complete: true` 会写入 layout 生命周期 header/footer。JSON 和 JSON Template 的
complete 模式按 appender 隔离状态，因此多个文件会保持各自有效的流结构。

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

可运行 demo：

```bash
GOWORK=off go run ./examples/file
```

## 显式插件扩展

插件必须显式注册。应用或外部模块需要隔离插件行为时，使用独立 registry。

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

可运行 demo：

```bash
GOWORK=off go run ./examples/extensibility
```

## 生产检查项

服务端建议从这些默认策略开始：

| 领域 | 建议 |
| --- | --- |
| API | 普通业务代码使用 `slog`，热点路径使用原生 logger builder。 |
| 输出 | 容器采集使用 JSON direct stdout，本地文件使用滚动 JSON。 |
| 背压 | 审计和关键服务日志优先 `block`；非关键诊断日志才考虑 `drop`。 |
| 保留 | 本地磁盘有限时同时配置归档数量和归档年龄。 |
| 脱敏 | 在 routing 或 failover 目标前应用 rewrite。 |
| 关闭 | 关闭 handler 或 logger context。 |
| 重载 | handler 级异步形态在重载前后保持稳定。 |
