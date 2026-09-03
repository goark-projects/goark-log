# 编程 API

[English](api.md)

本文档说明 `goark.dev/log` 当前源码实现的公开 Go API。`internal/configfile`
下的配置结构不是公开 API。

## 引入

```go
import goarklog "goark.dev/log"
```

模块要求 Go 1.25 或更新版本，并实现标准 `log/slog` handler 契约。

## 构造入口

| 函数 | 用途 |
| --- | --- |
| `DefaultOptions()` | 返回 stdout、`INFO`、Spring Boot 风格 pattern 的默认配置。 |
| `NewHandler(options)` | 从编程式 `Options` 构建 `*Handler`。 |
| `New(options)` | 构建默认命名的 `*slog.Logger` 和对应 `*Handler`。 |
| `NewDefaultHandler()` | 构建默认 stdout handler；仅在内置默认值非法时 panic。 |
| `NewDefault()` | 构建默认 `*slog.Logger` 和 `*Handler`。 |
| `LoadOptions(ctx, opts...)` | 解析配置并构建 `Options`。 |
| `NewConfiguredHandler(ctx, opts...)` | 从配置构建 handler。 |
| `NewConfigured(ctx, opts...)` | 从配置构建默认命名 logger 和 handler。 |
| `ConfigureDefault(ctx, opts...)` | 从配置构建 logger，并通过 `slog.SetDefault` 安装。 |
| `NewLoggerContext(options, opts...)` | 从显式 options 创建可关闭、可重载的上下文。 |
| `NewConfiguredLoggerContext(ctx, opts...)` | 从配置创建上下文，并在配置启用时启动轮询重载。 |

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("conf/goark-log.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

logger := loggerContext.Logger("goark.http")
logger.InfoContext(ctx, "ready", slog.String("source", string(result.Source)))
```

## Handler Options

```go
type Options struct {
	Appenders []Appender
	Filters   []Filter
	Root      RootLogger
	Loggers   []LoggerRule
	Async     AsyncLoggerOptions
}
```

| 字段 | 运行期含义 |
| --- | --- |
| `Appenders` | 最终输出端。应用默认值后至少需要一个 appender。 |
| `Filters` | 全局过滤器，先于 logger 级别判断执行。 |
| `Root` | 根 logger 的级别、appender 引用、过滤器和位置采集策略。 |
| `Loggers` | 命名 logger 规则，最长前缀优先。 |
| `Async` | Handler 层有界异步队列。 |

`Handler.Close()` 会排空异步工作、关闭 appender、刷新文件并写 layout
footer。`Handler.Reload(options)` 在新运行期构建成功后原子替换路由。

## Logger 命名

`NewLogger(handler, name)` 返回绑定内部 `goark.logger` 属性的
`*slog.Logger`。`WithName(logger, name)` 对已有 logger 做同样处理；输入为
nil 时使用 `slog.Default()`。

```go
logger := goarklog.NewLogger(handler, "goark.orm.mapper")
logger.Info("query finished", slog.Int("rows", 12))
```

默认 logger 名称是 `goark`。

## 原生 Logger

`NewNativeLogger(handler, name, opts...)` 构建低分配 logger，仍然复用同一个
handler、appender、filter 和 layout。

| 方法 | 说明 |
| --- | --- |
| `Name()` | 返回有效 logger 名称。 |
| `Enabled(ctx, level)` | 检查当前路由级别。 |
| `WithAttrs(attrs...)` | 返回绑定属性的新 logger。 |
| `WithGroup(name)` | 返回绑定属性分组的新 logger。 |
| `Slog()` | 返回等价的 `*slog.Logger`。 |
| `LogAttrs(ctx, level, message, attrs...)` | 写结构化事件。 |
| `LogAttrs3(ctx, level, message, a0, a1, a2)` | 固定三个属性的极热路径。 |
| `Debug`, `Info`, `Warn`, `Error`, `Fatal` | 便捷级别方法。 |
| `DebugContext`, `InfoContext`, `WarnContext`, `ErrorContext`, `FatalContext` | 带 context 的便捷方法。 |
| `At(level)`, `AtTrace`, `AtDebug`, `AtInfo`, `AtWarn`, `AtError`, `AtFatal` | 链式事件构造器入口。 |

原生 logger 选项：

| 选项 | 说明 |
| --- | --- |
| `WithLoggerCaller(enabled)` | 为 true 时采集调用位置；路由需要位置时也会强制采集。 |
| `WithLoggerMessageFactory(factory)` | 替换默认 `{}` 参数化消息工厂。 |

## 链式构造器

`LogBuilder` 在级别关闭时跳过属性构建。

| 方法 | 说明 |
| --- | --- |
| `Enabled()` | 当前事件是否会写出。 |
| `WithContext(ctx)` | 设置事件 context。 |
| `WithGroup(name)` | 给后续属性增加分组前缀。 |
| `WithAttr`, `WithAttrs` | 添加 `slog.Attr`。 |
| `WithString`, `WithInt`, `WithBool`, `WithAny` | 类型化属性辅助方法。 |
| `WithMarker(marker)` | 添加 marker。 |
| `WithError`, `WithThrowable` | 添加不采集栈的异常快照。 |
| `WithErrorStack(err)` | 添加带调用栈的异常快照。 |
| `Log(message)` | 写普通字符串消息。 |
| `Logf(pattern, args...)` | 使用 `{}` 占位符。 |
| `LogMessage(message)` | 写 `Message`；带属性的消息会追加 attrs。 |

```go
_ = logger.AtInfo().
	WithContext(ctx).
	WithGroup("http").
	WithString("method", "GET").
	WithInt("status", 200).
	Logf("request {} completed", requestID)
```

## Context、Marker 与 Throwable

| API | 说明 |
| --- | --- |
| `WithContextAttrs`, `WithContextAttr`, `ContextAttrs` | MDC 风格请求属性。 |
| `NewMarker`, `MarkerAttr`, `WithMarker`, `ContextMarker` | 支持父级匹配的 marker。 |
| `ThreadNameAttr`, `WithThreadName`, `ContextThreadName` | Go goroutine 的逻辑线程名。 |
| `WithContextStack`, `ContextStack` | NDC 风格栈值。 |
| `NewThrowable`, `NewThrowableWithStack` | 将 Go error 转换为异常快照。 |
| `ThrowableAttr`, `ThrowableWithStackAttr` | 给 slog 事件添加异常数据。 |

标准属性键包括 `goark.throwable`、`goark.marker`、`goark.thread`、
`goark.contextStack`、`goark.structuredData.id` 和
`goark.structuredData.type`。

## 消息

| 类型 | 函数 | 说明 |
| --- | --- | --- |
| `SimpleMessage` | `NewSimpleMessage(text)` | 不可变文本。 |
| `ParameterizedMessage` | `NewParameterizedMessage(pattern, args...)` | 按顺序替换 `{}`；`\{}` 保留字面占位符。 |
| `MapMessage` | `NewMapMessage(attrs...)` | 文本是 key/value，同时向 layout/filter 暴露 attrs。 |
| `StructuredDataMessage` | `NewStructuredDataMessage(id, type, message, attrs...)` | RFC5424 风格结构化字段和普通 attrs。 |
| `MessageFactoryFunc` | 适配器 | 自定义参数化消息行为。 |

## 级别

内置级别是 `ALL`、`TRACE`、`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`
和 `OFF`。`WARNING` 按 `WARN` 解析，也接受整数级别。

| API | 说明 |
| --- | --- |
| `ParseLevel(value)` | 解析名称或整数。 |
| `LevelName(level)` | 返回已注册精确名称，或返回最接近的内置区间名称。 |
| `NewLevelRegistry()` | 创建独立级别注册表。 |
| `DefaultLevelRegistry()` | 返回进程默认级别注册表。 |
| `RegisterLevel(name, level)` | 注册进程级自定义级别。 |

## Appender API

所有 appender 实现：

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

构造器包括 `NewConsoleAppender`、`NewFileAppender`、`NewJSONAppender`、
`NewJSONFileAppender`、`NewRollingFileAppender`、`NewAsyncAppender`、
`NewFailoverAppender`、`NewRoutingAppender`、`NewRewriteAppender` 和
`NewFilteredAppender`。

`NewAppenderRef` 配合 `WithAppenderRefLevel`、`WithAppenderRefLocation`、
`WithAppenderRefFilters`，在代码中表达 Log4j2 风格 appender 引用。

## Layout API

`Layout` 将事件格式化到调用方持有的 buffer。内置构造器包括
`NewDefaultLayout`、`NewPatternLayout`、`NewPatternLayoutWithOptions`、
`NewJSONLayout`、`NewJSONTemplateLayout`、
`NewJSONTemplateLayoutFromFile`、`NewXMLLayout`、`NewYAMLLayout`、
`NewCSVLayout`、`NewHTMLLayout` 和 `NewGELFLayout`。`TextLayout`、
`RFC5424Layout` 和 `SyslogLayout` 是可直接使用的类型。

`LayoutOptions` 包含 `Compact`、`EventEOL`、`Complete`、
`IncludeStacktrace`、`StacktraceAsString`、`PropertiesAsList`、
`IncludeNullDelimiter`、`DisableANSI`、`Header` 和 `Footer`。

## Filter API

所有 filter 实现 `Decide(ctx, event) FilterDecision`。裁决包括
`FilterNeutral`、`FilterAccept` 和 `FilterDeny`。

构造器覆盖 threshold、level、level range、regex、attr、marker、
no-marker、map、thread context map、thread context stack、structured data、
throwable、string match、time、burst、dynamic threshold、deny、composite 和
script filter。`ScriptFilter` 只在代码中使用，必须由调用方提供
`ScriptEvaluator`。

## 配置 API

配置加载选项：

| 选项 | 说明 |
| --- | --- |
| `WithConfigPath(path)` | 最高优先级显式路径。 |
| `WithConfigEnvKey(key)` | 覆盖 `GOARK_LOG_CONFIG` 环境变量名。 |
| `WithConfigWorkingDir(dir)` | 相对路径和默认发现路径的基准目录。 |
| `WithBootPropertyResolver(resolver)` | 读取 `goark.log.config`、`goark.logging.config` 和 `logging.config`。 |
| `WithDefaultConfigPaths(paths...)` | 替换默认发现路径。 |
| `WithConfigLookups(resolver)` | 使用自定义 lookup resolver。 |
| `WithPluginRegistry(registry)` | 使用显式插件注册表。 |

`ConfigResult` 返回 `Source`、`Path` 和 `MonitorInterval`。

解析辅助函数包括 `ParseByteSize`、`ParseRollingInterval`、
`ParseRollingMaxAge` 和 `ParseMonitorInterval`。

## 重载与状态日志

`ConfigReloader.Reload(ctx)` 总是重新加载。`ReloadIfChanged(ctx)` 检查配置
路径、修改时间和大小。`Watch(ctx, interval, onError)` 轮询到 context 取消。

`StatusLogger` 记录内部配置与重载事件。使用 `NewStatusLogger`、
`WithStatusLevel`、`WithStatusWriter` 和 `WithStatusBufferSize`。

## 插件 API

使用 `NewPluginRegistry` 创建隔离注册表，或使用 `DefaultPluginRegistry`
做进程级注册。通过 `RegisterAppender`、`RegisterLayout`、`RegisterFilter`、
`RegisterLookup`、`RegisterJSONTemplateResolver` 或 `RegisterPlugins` 注册。

`NewPluginSet` 配合 `WithPluginAppender`、`WithPluginLayout`、
`WithPluginFilter`、`WithPluginLookup` 和
`WithPluginJSONTemplateResolver` 可以创建可复用的 `PluginRegistrar`。
