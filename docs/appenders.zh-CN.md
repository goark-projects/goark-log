# Appender 参考

[English](appenders.md)

Appender 是 log event 的最终输出边界。每个 appender 都必须支持并发 `Append` 调用，并在 `Close` 时释放资源。

公开 appender contract：

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

## 内置 Appender Types

| Type | Aliases | 目的 |
| --- | --- | --- |
| `console` | none | 将格式化事件写入 stdout 或 stderr。 |
| `file` | none | 将格式化事件写入一个普通文件。 |
| `json` | `jsonDirect`, `jsonWriter` | 将手写编码的单行 JSON 写入 stdout、stderr 或文件。 |
| `rolling`, `rollingFile` | `rolling-file`, `rolling_file` after normalization | 写入一个 active file，并按 size、time、cron 或 startup 归档。 |
| `async` | none | 用 bounded queue 包装一个或多个 downstream appenders。 |
| `failover`, `failoverAppender` | `failover-appender`, `failover_appender` after normalization | 优先写 primary appender，失败后按顺序写 failover appenders。 |
| `routing`, `routingAppender` | `routing-appender`, `routing_appender` after normalization | 按 route key 选择 downstream appender。 |
| `rewrite`, `rewriteAppender` | `rewrite-appender`, `rewrite_appender` after normalization | 写入 delegate appender 前重写 event attributes。 |

Appender 和 plugin kinds 会先 trim spaces、lowercase，再移除 `-` 和 `_`。

## 通用配置字段

以下字段由 appender config object 接收。只有相关内置 appender 会使用对应字段，外部 appender plugins 也可以通过 `AppenderBuildConfig` 读取它们。

| 字段 | Aliases | Core 使用方 | 说明 |
| --- | --- | --- | --- |
| `type` | none | all | 必填 appender type。 |
| `target` | none | console, json | `stderr`、`stdout`；JSON 在没有 `fileName` 时拒绝 `file`。 |
| `fileName` | `file-name`, `path` | file, json file, rolling | Active log file path。 |
| `layout` | none | console, file, rolling | Layout object。JSON direct appender 忽略 `layout`。 |
| `rolling` | none | rolling | Rolling policy 和 strategy object。 |
| `appenderRefs` | `appender-refs`, `refs` | async, failover, rewrite | Downstream appender references。 |
| `primary` | `primary-ref` | failover | Primary appender reference。 |
| `failovers` | `failover-refs` | failover | Failover appender references。 |
| `routeKey` | `route-key` | routing | 用作 route key 的 event attribute。 |
| `defaultRoute` | `default-route` | routing | Fallback appender reference。 |
| `routes` | none | routing | route value 到 appender name 的 map。 |
| `rewrite` | none | rewrite | Attribute rewrite policy。 |
| `queueSize` | `queue-size` | async | Async appender queue size。 |
| `batchSize` | `batch-size` | async | Async appender batch size。 |
| `overflowStrategy` | `overflow-strategy` | async | Queue-full 行为。 |
| `waitStrategy` | `wait-strategy` | async | Consumer wait 行为。 |
| `waitRetries` | `wait-retries` | async | 可选 wait strategy retries。 |
| `sleepTime` | `sleep-time` | async | 可选 wait strategy sleep duration。 |
| `timeout` | none | async | 可选 blocking timeout。 |
| `bufferSize` | `buffer-size` | file, json file, rolling | 应用层 buffer size。`0` 禁用 buffering。 |
| `flushOnWrite` | `flush-on-write` | file, json file, rolling | 每次事件写入后 flush appender buffer。 |
| `append` | none | file, rolling | 追加而不是 truncate active file。 |
| `createOnDemand` | `create-on-demand` | file, rolling | 延迟到首次写入时创建文件。 |
| `filePermissions` | `file-permissions` | file, rolling | 新建文件权限。支持 octal 或 `rwxr-x---` 风格。 |
| `filters` | `filterRefs`, `filter-refs` | all | Appender wrapper level 的 filter chain。 |
| `url` | none | external only | 为外部 appender plugins 保留。 |
| `method` | none | external only | 为外部 appender plugins 保留。 |
| `address` | none | external only | 为外部 appender plugins 保留。 |
| `network` | none | external only | 为外部 appender plugins 保留。 |
| `facility` | none | external only | 为外部 appender plugins 保留。 |
| `appName` | `app-name` | external only | 为外部 appender plugins 保留。 |
| `connectTimeout` | `connect-timeout` | external only | 为外部 appender plugins 保留。 |
| `writeTimeout` | `write-timeout` | external only | 为外部 appender plugins 保留。 |

## Console Appender

```yaml
appenders:
  console:
    type: console
    target: stderr
    layout:
      type: pattern
      pattern: "%d %5p %pid --- [%thread] %c : %m%attrs%n"
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `type` | required | `console`。 |
| `target` | `stderr` | `stderr` 或 `stdout`。XML 还支持 `SYSTEM_ERR`、`STDERR`、`SYSTEM_OUT`、`STDOUT`。 |
| `layout` | default pattern | 任意内置或已注册 layout。 |
| `filters` | empty | 可选 appender-level filters。 |

Programmatic API：

```go
appender := goarklog.NewConsoleAppender(
	goarklog.WithConsoleName("console"),
	goarklog.WithConsoleWriter(os.Stdout),
	goarklog.WithConsoleLayout(goarklog.TextLayout{}),
)
```

## File Appender

```yaml
appenders:
  file:
    type: file
    fileName: logs/app.log
    bufferSize: 256KiB
    flushOnWrite: false
    append: true
    createOnDemand: false
    filePermissions: "0644"
    layout:
      type: json
      eventEol: true
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `fileName`, `file-name`, `path` | required | 文件路径。Parent directories 使用 `0755` 创建。 |
| `layout` | default pattern | 写入前使用的 layout。 |
| `bufferSize`, `buffer-size` | `256KiB` | 应用层 buffer size。`0` 禁用 buffering。负数失败。 |
| `flushOnWrite`, `flush-on-write` | false | 每个 event 后 flush。更可靠但更慢。 |
| `append` | true | `true` 使用 `O_APPEND`；`false` 打开时 truncate。 |
| `createOnDemand`, `create-on-demand` | false | true 时首次 append 才打开文件。 |
| `filePermissions`, `file-permissions` | `0644` | Octal 如 `0600`，或 symbolic `rw-------`。 |

Programmatic API：

```go
appender, err := goarklog.NewFileAppender("logs/app.log",
	goarklog.WithFileName("file"),
	goarklog.WithFileLayout(goarklog.NewJSONLayout(goarklog.LayoutOptions{EventEOL: true})),
	goarklog.WithFileBufferSize(256*1024),
	goarklog.WithFileAppend(true),
)
```

## JSON Appender

JSON appender 绕过通用 layout dispatch，直接写固定 JSON event shape：

```json
{"time":"2026-08-25T10:15:30.123+08:00","level":"INFO","logger":"goark.http","msg":"request done","status":200}
```

Console JSON：

```yaml
appenders:
  json:
    type: json
    target: stdout
```

File JSON：

```yaml
appenders:
  json:
    type: json
    fileName: logs/app.json
    bufferSize: 256KiB
    flushOnWrite: false
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `target` | `stderr` | `fileName` 为空时可设 `stderr` 或 `stdout`。 |
| `fileName`, `file-name`, `path` | empty | 设置后 JSON 写入该文件，文件规则与 file appender 相同。 |
| `bufferSize`, `buffer-size` | file mode 下 `256KiB` | File buffer size。`0` 禁用 buffering。 |
| `flushOnWrite`, `flush-on-write` | false | 每个 event 后 flush file buffer。 |

没有 `fileName` 时 `target: file` 非法。Programmatic `NewJSONFileAppender` 会拒绝显式 writer，因为 file mode 拥有文件生命周期。

Programmatic API：

```go
stdoutJSON := goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(os.Stdout))

fileJSON, err := goarklog.NewJSONFileAppender("logs/app.json",
	goarklog.WithJSONAppenderBufferSize(256*1024),
)
```

## Rolling File Appender

```yaml
appenders:
  rolling:
    type: rolling-file
    fileName: logs/app.log
    bufferSize: 256KiB
    append: true
    layout:
      type: json
      eventEol: true
    rolling:
      filePattern: logs/archive/app-%d{yyyyMMdd-HHmmss}-%06i.log.gz
      policies:
        size:
          size: 100MiB
        time:
          interval: daily
          modulate: true
        cron:
          schedule: "0 0 0 * * ?"
        startup:
          enabled: true
      strategy:
        max: 30
        maxAge: 30d
        fileIndex: nomax
        compression:
          gzip: true
          async: true
        delete:
          basePath: logs/archive
          maxDepth: 1
          ifFileName:
            glob: "*.log.gz"
          ifLastModified:
            age: 30d
```

Appender-level file fields 与 `file` 相同：`fileName`、`layout`、`bufferSize`、`flushOnWrite`、`append`、`createOnDemand` 和 `filePermissions`。

Rolling defaults：

| Setting | 默认值 |
| --- | --- |
| appender name | `rollingFile` |
| `maxSize` | `10MiB` |
| `maxBackups` | `7` |
| `bufferSize` | `256KiB` |
| `fileIndex` | `nomax` |
| time `modulate` | true |
| action queue size | `32` |

至少启用一个 trigger：size、time、cron 或 startup。

### Rolling Fields

| 字段 | Aliases | 说明 |
| --- | --- | --- |
| `filePattern` | `file-pattern` | Archive path pattern。支持 `%d{layout}`、`%i`、`%0Ni` 和 `%%`。 |
| `maxSize` | `max-size` | size policy 的 legacy shortcut。 |
| `interval` | none | time policy 的 legacy shortcut。 |
| `cron` | `cronSchedule`, `cron-schedule` | cron policy 的 legacy shortcut。 |
| `onStartup` | `on-startup` | startup policy 的 legacy shortcut。 |
| `maxBackups` | `max-backups` | retained archive count 的 legacy shortcut。 |
| `maxAge` | `max-age` | retained archive age 的 legacy shortcut。 |
| `gzip` | `compress` | 启用 gzip archive compression。`filePattern` 以 `.gz` 结尾时也会启用。 |
| `directWrite` | `direct-write` | 直接写入 active pattern file，而不是 rename stable active file。要求 `filePattern`；与 gzip 不兼容。 |
| `asyncActions` | `async-actions` | 使用单个 background worker 执行 compression 和 delete actions。 |
| `actionQueueSize` | `action-queue-size` | Async rolling actions 的 bounded queue。`0` 表示默认 `32`。 |

### Policies

Policy names 支持 concise 和 Log4j2-style names：

| YAML 字段 | Aliases | 字段 |
| --- | --- | --- |
| `policies.size` | `size-based-triggering-policy`, `sizeBasedTriggeringPolicy`, `SizeBasedTriggeringPolicy` | `size`, `maxSize`, `max-size`。 |
| `policies.time` | `time-based-triggering-policy`, `timeBasedTriggeringPolicy`, `TimeBasedTriggeringPolicy` | `interval`, `every`, `unit`, `modulate`。 |
| `policies.cron` | `cron-triggering-policy`, `cronTriggeringPolicy`, `CronTriggeringPolicy` | `schedule`, `cron`, `cronSchedule`, `cron-schedule`。 |
| `policies.startup` | `on-startup-triggering-policy`, `onStartupTriggeringPolicy`, `OnStartupTriggeringPolicy` | `enabled`。 |

Size policy 激活且设置 `filePattern` 时，pattern 必须包含 `%i`，否则配置失败。

### Strategy

| 字段 | Aliases | 说明 |
| --- | --- | --- |
| `type` | none | `directWrite`、`direct-write` 或 `directWriteRolloverStrategy` 启用 direct-write mode。 |
| `max` | none | Retained archive count。覆盖 `maxBackups`。 |
| `maxBackups` | `max-backups` | Retained archive count。 |
| `maxAge` | `max-age` | Retained archive age。 |
| `fileIndex` | `file-index` | `nomax`、`no-max`、`none`、`max` 或 `min`。 |
| `directWrite` | `direct-write` | Direct-write boolean。 |
| `asyncActions` | `async-actions` | Async compression/delete actions。 |
| `actionQueueSize` | `action-queue-size` | Async action queue size。 |
| `compression.gzip` | `compression.compress` | 启用 gzip compression。 |
| `compression.async` | none | 启用 async actions。 |
| `delete` | none | 单个 delete action。 |
| `deleteActions` | `delete-actions` | 多个 delete actions。 |

### Delete Action

| 字段 | Aliases | 默认值 | 说明 |
| --- | --- | --- | --- |
| `basePath` | `base-path` | archive directory 或 active file directory | 要扫描的目录。 |
| `maxDepth` | `max-depth` | `1` | `basePath` 下最大 file depth。 |
| `glob` | `ifFileName.glob` | `*` | 文件名或相对路径 glob。 |
| `age` | `ifLastModified.age` | disabled | 删除早于该 age 的文件。 |
| `maxCount` | `max-count`, `ifAccumulatedFileCount.exceeds` | disabled | 只保留最新文件直到 count 上限。 |
| `maxSize` | `max-size`, `ifAccumulatedFileSize.exceeds` | disabled | 只保留最新文件直到 accumulated size 超过上限。 |
| `async` | none | false | 位于 strategy delete entries 下时启用 rolling async actions。 |

Programmatic API：

```go
appender, err := goarklog.NewRollingFileAppender("logs/app.log",
	goarklog.WithRollingFileLayout(goarklog.NewJSONLayout(goarklog.LayoutOptions{EventEOL: true})),
	goarklog.WithRollingMaxSize(100*1024*1024),
	goarklog.WithRollingInterval(24*time.Hour),
	goarklog.WithRollingFilePattern("logs/archive/app-%d{yyyyMMdd}-%03i.log.gz"),
	goarklog.WithRollingGzip(true),
	goarklog.WithRollingMaxBackups(30),
	goarklog.WithRollingMaxAge(30*24*time.Hour),
	goarklog.WithRollingAsyncActions(true),
)
```

## Async Appender

Async appender 包装 downstream appenders。只希望某个 sink 异步时使用它。

```yaml
appenders:
  file:
    type: file
    fileName: logs/app.log
    layout:
      type: json
      eventEol: true
  asyncFile:
    type: async
    appenderRefs: [file]
    queueSize: 4096
    batchSize: 128
    overflowStrategy: block
    waitStrategy: yield
root:
  level: info
  appenderRefs: [asyncFile]
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `appenderRefs`, `refs` | required | 一个或多个 downstream appenders。 |
| `queueSize` | `1024` | 必须大于 0。会归一化为 ring-buffer capacity。 |
| `batchSize` | `64` | 必须大于 0；上限为 queue size。 |
| `overflowStrategy` | `block` | 与 Handler-level async logger 相同的 aliases。 |
| `waitStrategy` | `block` | 与 Handler-level async logger 相同的 aliases。 |
| `waitRetries` | `0` | 非负。 |
| `sleepTime` | `0` | Go duration。 |
| `timeout` | `0` | Go duration。 |

`Close` 会等待 producers 并 drain queued events。配置构建出来的 composite appenders 中，child appenders 由完整 handler runtime 拥有，不会被 wrapper 重复关闭。

## Failover Appender

```yaml
appenders:
  primary:
    type: file
    fileName: logs/primary.log
  fallback:
    type: console
    target: stderr
  failover:
    type: failover
    primary: primary
    failovers: [fallback]
root:
  level: info
  appenderRefs: [failover]
```

必须配置 `primary` 和至少一个 failover。也支持 shorthand `appenderRefs: [primary, fallback]`；第一个 reference 是 primary。

Failover 行为：

- primary 写成功时，不调用 failovers。
- primary 失败时，按顺序尝试 failovers。
- 第一个成功的 failover 完成写入。
- 所有 delegates 都失败时，返回的 error 会 join 所有 failures。

## Routing Appender

```yaml
appenders:
  defaultJson:
    type: json
    target: stdout
  auditFile:
    type: file
    fileName: logs/audit.log
    layout:
      type: json
      eventEol: true
  router:
    type: routing
    routeKey: channel
    defaultRoute: defaultJson
    routes:
      audit: auditFile
root:
  level: info
  appenderRefs: [router]
```

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `routeKey`, `route-key` | `route` | 读取为 route key 的 event attribute。 |
| `routes` | empty | route value 到 appender name 的 map。 |
| `defaultRoute`, `default-route` | empty | 可选 fallback appender。 |

至少需要一个 route 或 default route。没有 route match 且没有 default route 时，event 会被跳过且不返回 error。

## Rewrite Appender

```yaml
appenders:
  json:
    type: json
    target: stdout
  redacted:
    type: rewrite
    appenderRefs: [json]
    rewrite:
      attrs:
        service: billing
      removeAttrs: [password, token]
root:
  level: info
  appenderRefs: [redacted]
```

| 字段 | Aliases | 说明 |
| --- | --- | --- |
| `appenderRefs` | `refs` | 正好一个 downstream appender。 |
| `rewrite.attrs` | `rewrite.attributes`, `rewrite.properties` | 添加或覆盖 configured string values。 |
| `rewrite.remove` | `rewrite.removeAttrs`, `rewrite.remove-attrs` | 写入前按 key 移除 attributes。 |

内置 rewrite policy 只处理 attributes。更复杂行为应使用 programmatic `NewRewriteAppender` 搭配自定义 `RewritePolicy`，或使用 plugin。

## 不支持的 Core Appenders

XML schema 有 `<Http>`、`<Socket>` 和 `<Syslog>` element slots，generic config object 也暴露 `url`、`address`、`facility` 和 timeouts 等字段。core module 不注册 network appenders。使用这些 types 的配置必须由外部模块注册对应 appender factory，否则会加载失败。
