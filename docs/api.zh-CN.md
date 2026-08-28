# 编程式 API 指南

[English](api.md)

本文覆盖公开编程式 API。配置文件见 [配置参考](configuration.zh-CN.md)。

## 默认 Logger

```go
logger, handler := goarklog.NewDefault()
defer handler.Close()

logger.Info("service started", slog.String("profile", "dev"))
```

`NewDefault` 创建：

- 一个名为 `console` 的 console appender；
- stderr 输出目标；
- 默认 Spring Boot 风格 pattern layout；
- root level 为 `INFO`。

`NewDefaultHandler` 只返回 handler。`DefaultOptions` 返回等价的 options object。

## Handler 构造

```go
appender := goarklog.NewConsoleAppender()
handler, err := goarklog.NewHandler(goarklog.Options{
	Appenders: []goarklog.Appender{appender},
	Root: goarklog.RootLogger{
		Level:        slog.LevelInfo,
		AppenderRefs: []string{"console"},
	},
})
if err != nil {
	return err
}
defer handler.Close()
```

`Options` 字段：

| 字段 | 说明 |
| --- | --- |
| `Appenders` | 使用非默认 options 时必填。名称必须非空且唯一。 |
| `Filters` | Global filters。 |
| `Root` | Root route。未设置 root appender 时使用第一个 appender。 |
| `Loggers` | Named logger rules。 |
| `Async` | Handler-level async logger options。 |

如果手动构造 appenders 且 `NewHandler` 返回错误，调用方需要自行关闭这些 appenders。`NewHandler` 成功后，`Handler.Close` 拥有已配置 appenders 的关闭责任。

## Named slog Logger

```go
logger := goarklog.NewLogger(handler, "goark.http")
logger.InfoContext(ctx, "request done", slog.Int("status", 200))
```

`NewLogger` 会附加内部 `goark.logger` attribute，handler 用它做路由。`WithName` 可以重命名已有 `slog.Logger`：

```go
logger = goarklog.WithName(logger, "goark.orm")
```

## 配置默认 slog

```go
handler, result, err := goarklog.ConfigureDefault(context.Background(),
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer handler.Close()

slog.Info("configured", slog.String("source", string(result.Source)))
```

`ConfigureDefault` 创建配置化 logger 并通过 `slog.SetDefault` 安装为默认 logger。

## LoggerContext

`LoggerContext` 是服务端推荐的托管运行时，适合需要 reload 和集中 shutdown 的应用。

```go
logging, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer logging.Close()

logger := logging.Logger("goark.service")
logger.Info("ready", slog.String("source", string(result.Source)))
```

常用方法：

| 方法 | 说明 |
| --- | --- |
| `Logger(name)` | 返回 named `slog.Logger`。 |
| `Handler()` | 返回底层 `*Handler`。 |
| `StatusLogger()` | 返回内部 status logger。 |
| `ConfigResult()` | 返回最后一次配置加载结果快照。 |
| `Reload(options)` | 从显式 `Options` reload。 |
| `ReloadConfigured(ctx, options...)` | 从 config loading options reload。 |
| `Close()` | 停止 config monitor，drain async，关闭 appenders。 |

只有当配置文件来自实际文件并且 `monitorInterval` 为正数时，`NewConfiguredLoggerContext` 才会启动文件轮询。

## Config Reloader

应用已有自己的生命周期或文件 watcher 时，可以直接使用 `ConfigReloader`。

```go
reloader, err := goarklog.NewConfigReloader(handler,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}

changed, result, err := reloader.ReloadIfChanged(ctx)
```

`Watch(ctx, interval, onError)` 启动 polling loop，并返回一个在 context done 时关闭的 channel。

## Native Logger

native logger 避开部分标准 `slog.Record` facade，并提供固定三属性快速路径。

```go
logger, err := goarklog.NewNativeLogger(handler, "goark.http")
if err != nil {
	return err
}

_ = logger.LogAttrs3(ctx, slog.LevelInfo, "request done",
	slog.String("method", "GET"),
	slog.Int("status", 200),
	slog.Duration("elapsed", elapsed),
)
```

Native logger 方法：

| 方法 | 说明 |
| --- | --- |
| `Name()` | Logger 名称。 |
| `Enabled(ctx, level)` | 检查当前 route/global-filter 状态。 |
| `Slog()` | 返回等价的 `*slog.Logger`。 |
| `WithAttrs(attrs...)` | 返回绑定 attrs 的 logger。 |
| `WithGroup(name)` | 返回带 flattened group prefix 的 logger。 |
| `LogAttrs(ctx, level, message, attrs...)` | 写入动态 attr slice。 |
| `LogAttrs3(ctx, level, message, a0, a1, a2)` | 写入正好三个 attrs，开销最低。 |
| `Debug`, `Info`, `Warn`, `Error`, `Fatal` | 使用 background context 的 convenience methods。 |
| `DebugContext`, `InfoContext`, `WarnContext`, `ErrorContext`, `FatalContext` | context-aware convenience methods。 |
| `At(level)`, `AtTrace`, `AtDebug`, `AtInfo`, `AtWarn`, `AtError`, `AtFatal` | 创建 builder。 |

`WithLoggerCaller(true)` 会为 native events 捕获 caller PC。只有 layouts 需要 caller converters 或 JSON Template source resolver 时才应在热路径启用。

## Log Builder

```go
err := logger.AtInfo().
	WithContext(ctx).
	WithGroup("http").
	WithString("method", "GET").
	WithInt("status", 200).
	WithBool("cached", false).
	Log("request done")
```

Builder 方法：

| 方法 | 说明 |
| --- | --- |
| `Enabled()` | 当前事件是否会被输出。 |
| `WithContext(ctx)` | 设置 event context。 |
| `WithGroup(name)` | 为后续 attrs 添加 flattened group prefix。 |
| `WithAttr(attr)` | 添加一个 attr。 |
| `WithAttrs(attrs...)` | 添加多个 attrs。 |
| `WithString`, `WithInt`, `WithBool`, `WithAny` | Typed attr helpers。 |
| `WithMarker(marker)` | 添加 marker attr。 |
| `WithError(err)`, `WithThrowable(err)` | 添加 throwable，不捕获 stack。 |
| `WithErrorStack(err)` | 添加 throwable 并捕获 stack。 |
| `Log(message)` | 写入普通字符串 message。 |
| `Logf(pattern, args...)` | 默认使用 `{}` placeholder 的 message factory。 |
| `LogMessage(message)` | 写入自定义 message object。 |

builder 在分配 backing slice 前，会把最多 8 个 attrs 存在 inline 空间里。

## Context Attributes

```go
ctx = goarklog.WithContextAttrs(ctx,
	slog.String("trace_id", "trace-1"),
	slog.String("span_id", "span-1"),
)
ctx = goarklog.WithThreadName(ctx, "worker-1")
ctx = goarklog.WithMarker(ctx, goarklog.NewMarker("HTTP"))
ctx = goarklog.WithContextStack(ctx, "tenant-a", "checkout")
```

Context helpers：

| Helper | 说明 |
| --- | --- |
| `WithContextAttrs` | 向 context 添加 immutable attr snapshot。 |
| `WithContextAttr` | 向 context 添加单个 attr。 |
| `ContextAttrs` | 返回 context attr snapshot。 |
| `NewMarker` | 创建 marker，可带 parents。 |
| `MarkerAttr` | 将 marker 转为 slog attr。 |
| `WithMarker`, `ContextMarker` | 存取 context marker。 |
| `ThreadNameAttr` | 将 logical thread name 转为 attr。 |
| `WithThreadName`, `ContextThreadName` | 存取 logical thread name。 |
| `WithContextStack`, `ContextStack` | 存取 NDC-style stack values。 |

Event attributes 按以下顺序合并：

1. handler-bound attrs；
2. context attrs；
3. record/native attrs。

同一个 key 出现多次时，`Event.Attr(key)` 返回最新值。

## Throwable 和 Message APIs

Throwable helpers：

| Helper | 说明 |
| --- | --- |
| `NewThrowable(err)` | 捕获 error type/message/cause chain，不捕获 stack。 |
| `NewThrowableWithStack(err)` | 捕获 throwable 和当前 stack。 |
| `ThrowableAttr(err)` | 添加标准 `goark.throwable` attr。 |
| `ThrowableWithStackAttr(err)` | 添加带 stack 的标准 throwable attr。 |

Message helpers：

| Helper | 说明 |
| --- | --- |
| `NewSimpleMessage(text)` | immutable string message。 |
| `NewParameterizedMessage(pattern, args...)` | `{}` placeholder message。 |
| `NewMapMessage(attrs...)` | attrs 表示的 structured message。 |
| `NewStructuredDataMessage(id, type, message, attrs...)` | RFC5424-style structured message。 |
| `WithLoggerMessageFactory(factory)` | 替换 native logger parameterized message factory。 |

## Status Logger

```go
status := goarklog.NewStatusLogger(
	goarklog.WithStatusLevel(slog.LevelWarn),
	goarklog.WithStatusWriter(os.Stderr),
	goarklog.WithStatusBufferSize(128),
)

logging, err := goarklog.NewLoggerContext(options,
	goarklog.WithLoggerContextStatus(status),
)
```

`StatusLogger` 记录内部 config、reload 和 close errors。配置文件里的 `status` 字段会为兼容性解析，但当前不会调节 `StatusLogger`。

## 关闭规则

- 始终调用 `Handler` 或 `LoggerContext` 的 `Close`。
- `Close` 会先 drain Handler-level async，再关闭 appenders。
- runtime close order 会先关闭 async appenders，再关闭它们的 delegates。
- File 和 rolling appenders 会 flush buffers，并在需要时写 layout footers。
- Rolling async actions 会在 `Close` 返回前 drain 完成。
- 重复调用 `Close` 是安全的。
