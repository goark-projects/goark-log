# Appenders

[English](appenders.md)

Appender 是日志事件的最终输出端。事件通过 root logger、命名 logger、
appender 引用、routing appender 和 failover appender 选择输出目标。

## 契约

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

`Append` 必须支持并发调用。`Close` 应刷新缓冲并释放自己持有的资源。
`Handler.Close` 会先关闭 async appender，再关闭其它 appender，并按名称跳过重复
appender。

## 通用配置

| 字段 | 适用对象 | 说明 |
| --- | --- | --- |
| `type` | 所有配置 appender | 必填。类型匹配忽略大小写、连字符和下划线。 |
| `layout` | console、file、rolling-file | 省略时使用 Spring Boot 风格默认 pattern。 |
| `filters`, `filterRefs`, `filter-refs` | 所有 | 在 appender 被使用前包裹过滤器链。 |
| `target` | console、JSON direct | 默认 `stderr`，支持 `stdout`。 |
| `fileName`, `file-name`, `path` | 文件输出 | 文件路径。file 和 rolling-file 必填，JSON direct 可选。 |
| `bufferSize`, `buffer-size` | 文件输出 | 字节大小字符串。`0` 禁用应用层缓冲。 |
| `flushOnWrite`, `flush-on-write` | 文件输出 | 每条事件后刷新 buffered writer。 |
| `append` | file、rolling-file | 默认 true。false 表示打开时截断。 |
| `createOnDemand`, `create-on-demand` | file、rolling-file | 延迟到第一条事件才打开文件。 |
| `filePermissions`, `file-permissions` | file、rolling-file | 默认 `0644`，支持八进制和符号形式。 |
| `appenderRefs`, `appender-refs`, `refs` | 组合 appender | 下游 appender 引用。 |

`url`、`method`、`address`、`network`、`facility`、`appName`、
`connectTimeout`、`writeTimeout` 等远程字段会解析并传给插件。核心模块不实现远程
appender。

## Console

类型：`console`。

Console 默认写 stderr，`target: stdout` 时写 stdout。支持所有 layout。layout
处于 complete 模式时，第一次事件写 header，关闭时写 footer。

```yaml
appenders:
  console:
    type: console
    target: stderr
    layout:
      type: pattern
      pattern: "%d %5p %c : %m%attrs%n"
```

编程 API：`NewConsoleAppender`、`WithConsoleName`、`WithConsoleWriter`、
`WithConsoleLayout`。

## File

类型：`file`。

File 写本地路径并创建父目录。目标路径如果已存在且是目录，会直接拒绝。默认缓冲大小
是 256 KiB。

```yaml
appenders:
  app:
    type: file
    fileName: "${env:GOARK_LOG_DIR:-logs}/app.log"
    bufferSize: 256KiB
    append: true
    createOnDemand: true
    filePermissions: "0644"
    layout:
      type: json
      eventEol: true
```

编程 API：`NewFileAppender`、`WithFileName`、`WithFileLayout`、
`WithFileBufferSize`、`WithFileFlushOnWrite`、`WithFileAppend`、
`WithFileCreateOnDemand`、`WithFilePermissions`。

## JSON Direct

类型：`json`、`jsonDirect`、`jsonWriter`。

JSON direct 绕过通用 layout 接口，直接输出单行 JSON，对象包含 `time`、`level`、
`logger`、`msg` 和事件属性。适合热点路径和容器 stdout 管道。

```yaml
appenders:
  stdout:
    type: json
    target: stdout
```

设置 `fileName` 后会写文件，并支持 `bufferSize` 和 `flushOnWrite`。

编程 API：`NewJSONAppender`、`NewJSONFileAppender`、
`WithJSONAppenderName`、`WithJSONAppenderWriter`、
`WithJSONAppenderBufferSize`、`WithJSONAppenderFlushOnWrite`。

## Rolling File

类型：`rolling`、`rollingFile`、`rolling-file`。

Rolling file 是本地滚动文件 appender，支持大小、时间、cron、启动滚动、归档模式、
gzip、保留策略和删除动作。

必需条件：

| 字段 | 说明 |
| --- | --- |
| `fileName` | 活动日志文件。 |
| `rolling.filePattern` | 需要自定义归档命名时使用；`directWrite` 必填。 |
| 至少一个触发策略 | 大小、间隔、cron 或启动滚动。编程式构造器默认启用大小滚动。 |

```yaml
appenders:
  appRolling:
    type: rolling-file
    fileName: "${LOG_DIR}/app.log"
    layout:
      type: json
      eventEol: true
      includeStacktrace: true
    rolling:
      filePattern: "${LOG_DIR}/archive/app-%d{yyyyMMdd-HHmmss}-%06i.log.gz"
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
        fileIndex: nomax
        compression:
          gzip: true
          async: true
```

关键校验规则：

| 规则 | 原因 |
| --- | --- |
| 开启大小滚动时 `filePattern` 必须包含 `%i`。 | 同一时间桶内可能多次按大小滚动。 |
| `.gz` 后缀或 `gzip: true` 启用压缩。 | 压缩作用于归档文件，不作用于活动文件。 |
| `directWrite` 必须设置 `filePattern`。 | 没有单独的活动文件。 |
| `directWrite` 不支持 gzip。 | 活动流不能安全地作为 gzip 归档重命名。 |
| 非 direct write 下 `filePattern` 不能解析成活动 `fileName`。 | 防止文件自重命名导致数据风险。 |

编程 API：`NewRollingFileAppender`、`WithRollingFileName`、
`WithRollingFileLayout`、`WithRollingFileBufferSize`、
`WithRollingFileFlushOnWrite`、`WithRollingFileAppend`、
`WithRollingFileCreateOnDemand`、`WithRollingFilePermissions`、
`WithRollingMaxSize`、`WithRollingInterval`、`WithRollingCronSchedule`、
`WithRollingTimeModulate`、`WithRollingFilePattern`、
`WithRollingFileIndexMode`、`WithRollingDirectWrite`、
`WithRolloverOnStartup`、`WithRollingMaxBackups`、`WithRollingMaxAge`、
`WithRollingGzip`、`WithRollingAsyncActions`、
`WithRollingActionQueueSize`、`WithRollingDeleteActions`。

## Async Appender

类型：`async`。

Async appender 将事件放入队列，由单个后台 worker 写到一个或多个下游 appender。它适合
只让特定输出端异步化。

```yaml
appenders:
  asyncFile:
    type: async
    appenderRefs:
      - ref: appRolling
        level: info
    queueSize: 8192
    batchSize: 256
    overflowStrategy: block
    waitStrategy: yield
```

默认值：队列 1024，批量 64，溢出策略 `block`，等待策略 `block`。队列大小会规范化到
ring buffer 需要的容量。关闭时会排空队列。

编程 API：`NewAsyncAppender`、`WithAsyncName`、`WithAsyncQueueSize`、
`WithAsyncBatchSize`、`WithAsyncOverflowStrategy`、`WithAsyncWaitStrategy`、
`WithAsyncWaitOptions`、`WithAsyncErrorHandler`、`WithAsyncCloseAppenders`。

## Handler 层异步

配置 `asyncLogger`、`async-logger` 或 `async` 会在 handler 边界启用一个统一异步队列。
适合绝大多数事件都走同一个队列的服务。启用后的默认值：队列 4096，批量 64，溢出
`block`，等待 `block`。

Handler 层异步运行期形态不能在 reload 时变化。enablement、queue size、batch size、
overflow strategy、wait strategy、wait options 和 include-location 都必须保持稳定。

## 溢出和等待策略

| 溢出策略 | 行为 |
| --- | --- |
| `block` | 生产者等待容量。 |
| `drop` | 队列满时丢弃新事件并增加 dropped 计数。 |
| `drop-debug` | 队列满时丢弃 DEBUG 及以下事件，高级别事件阻塞。 |
| `sync-fallback` | 队列满时同步写出。 |

| 等待策略 | 行为 |
| --- | --- |
| `block` | 通用生产默认。 |
| `sleep` | 重试间 sleep，支持 `sleepTime`、`waitRetries` 和 `timeout`。 |
| `yield` | 等待时让出处理器。 |
| `spin` | 忙等，只能在有基准证据后使用。 |

## Failover

类型：`failover`、`failoverAppender`。

Failover 先写 primary。primary 返回错误后按顺序尝试 failover，直到成功。全部失败时返回
合并错误。

```yaml
appenders:
  reliable:
    type: failover
    primary: primaryFile
    failovers: [stderrConsole]
```

配置构建的 failover 不自行关闭子 appender，因为 router 持有完整 appender 列表。编程式
failover 默认关闭子 appender，除非使用 `WithFailoverCloseChildren(false)`。

## Routing

类型：`routing`、`routingAppender`。

Routing 按事件属性选择下游 appender。默认路由键是 `route`；配置中可设置
`routeKey`。

```yaml
appenders:
  tenantRouter:
    type: routing
    routeKey: tenant
    defaultRoute: stdout
    routes:
      tenant-a: tenantA
      tenant-b: tenantB
```

事件没有命中路由且没有默认路由时，会被跳过且不返回错误。

## Rewrite

类型：`rewrite`、`rewriteAppender`。

Rewrite 在委托写出前修改事件。内置配置策略从 `attrs`、`attributes` 或 `properties` 添加
属性，并从 `remove`、`removeAttrs` 或 `remove-attrs` 删除属性键。

```yaml
appenders:
  redacted:
    type: rewrite
    appenderRefs: [tenantRouter]
    rewrite:
      attrs:
        service: billing
      removeAttrs: [password, token, authorization]
```

编程 API 支持自定义 `RewritePolicy`。

## Appender 引用

Appender ref 可以是字符串或对象。

```yaml
appenderRefs:
  - console
  - ref: rolling
    level: warn
    includeLocation: true
    filterRefs: [auditMarker]
```

引用级 `level` 会先于 appender 调用过滤。`includeLocation: true` 强制采集调用位置；
`includeLocation: false` 会清空该引用上的调用位置数据。
