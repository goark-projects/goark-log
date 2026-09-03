# Appenders

[简体中文](appenders.zh-CN.md)

Appenders are the final write targets for an event. They are selected through
root and named logger routes, appender references, routing appenders, and
failover appenders.

## Contract

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

`Append` must be safe for concurrent callers. `Close` is expected to flush
buffers and release owned resources. `Handler.Close` closes async appenders
first, then the remaining appenders, and skips duplicate appender names.

## Common Configuration

| Field | Used by | Notes |
| --- | --- | --- |
| `type` | all configured appenders | Required. Kind matching ignores case, hyphen, and underscore. |
| `layout` | console, file, rolling-file | Omitted layout defaults to the Spring Boot style pattern. |
| `filters`, `filterRefs`, `filter-refs` | all | Appender-level filters wrap the appender before it is used. |
| `target` | console, JSON direct | `stdout` is default; `stderr` is supported. |
| `fileName`, `file-name`, `path` | file sinks | File path. Required for file and rolling-file. Optional for JSON direct file output. |
| `bufferSize`, `buffer-size` | file sinks | Byte size string. `0` disables application buffering. |
| `flushOnWrite`, `flush-on-write` | file sinks | Flushes the buffered writer after each event. |
| `append` | file, rolling-file | Defaults to true. False truncates at open. |
| `createOnDemand`, `create-on-demand` | file, rolling-file | Delays opening the file until the first event. |
| `filePermissions`, `file-permissions` | file, rolling-file | Defaults to `0644`; accepts octal or symbolic forms. |
| `appenderRefs`, `appender-refs`, `refs` | composite appenders | Downstream appender references. |

Remote fields such as `url`, `method`, `address`, `network`, `facility`,
`appName`, `connectTimeout`, and `writeTimeout` are parsed and passed to
plugins. The core module does not implement remote appenders.

## Console

Type: `console`.

Console writes to stdout by default, or stderr when `target: stderr` is set.
It supports every layout and writes layout headers on first event and footers
on close when the layout is in complete mode.

```yaml
appenders:
  console:
    type: console
    target: stderr
    layout:
      type: pattern
      pattern: "%d %5p %c : %m%attrs%n"
```

Programmatic API: `NewConsoleAppender`, `WithConsoleName`,
`WithConsoleWriter`, and `WithConsoleLayout`.

## File

Type: `file`.

File writes to a local path and creates parent directories. It validates that
the target is not an existing directory. The default buffer size is 256 KiB.

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

Programmatic API: `NewFileAppender`, `WithFileName`, `WithFileLayout`,
`WithFileBufferSize`, `WithFileFlushOnWrite`, `WithFileAppend`,
`WithFileCreateOnDemand`, and `WithFilePermissions`.

## JSON Direct

Types: `json`, `jsonDirect`, `jsonWriter`.

JSON direct bypasses the general layout interface and emits a single-line JSON
object with `time`, `level`, `logger`, `msg`, and event attributes. Use this on
hot paths or container stdout pipelines.

```yaml
appenders:
  stdout:
    type: json
    target: stdout
```

When `fileName` is set, it writes to a file with optional `bufferSize` and
`flushOnWrite`.

Programmatic API: `NewJSONAppender`, `NewJSONFileAppender`,
`WithJSONAppenderName`, `WithJSONAppenderWriter`,
`WithJSONAppenderBufferSize`, and `WithJSONAppenderFlushOnWrite`.

## Rolling File

Types: `rolling`, `rollingFile`, `rolling-file`.

Rolling file is a local file appender with size, time, cron, startup rollover,
archive patterns, gzip, retention, and delete actions.

Required fields:

| Field | Notes |
| --- | --- |
| `fileName` | Active log file. |
| `rolling.filePattern` | Archive pattern when custom archive naming is needed. Required for `directWrite`. |
| At least one policy | Size, interval, cron, or startup rollover. The default programmatic constructor has size rollover enabled. |

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

Important validation rules:

| Rule | Reason |
| --- | --- |
| `filePattern` must contain `%i` when size rollover is enabled. | Multiple size rollovers can happen within one timestamp bucket. |
| `.gz` suffix or `gzip: true` enables compression. | Compression applies to archives, not the active file. |
| `directWrite` requires `filePattern`. | There is no separate active file. |
| `directWrite` rejects gzip. | The active stream cannot be gzip-renamed safely. |
| `filePattern` must not resolve to the active `fileName` when not direct write. | Prevents self-rename data loss. |

Programmatic API: `NewRollingFileAppender`, `WithRollingFileName`,
`WithRollingFileLayout`, `WithRollingFileBufferSize`,
`WithRollingFileFlushOnWrite`, `WithRollingFileAppend`,
`WithRollingFileCreateOnDemand`, `WithRollingFilePermissions`,
`WithRollingMaxSize`, `WithRollingInterval`, `WithRollingCronSchedule`,
`WithRollingTimeModulate`, `WithRollingFilePattern`,
`WithRollingFileIndexMode`, `WithRollingDirectWrite`,
`WithRolloverOnStartup`, `WithRollingMaxBackups`, `WithRollingMaxAge`,
`WithRollingGzip`, `WithRollingAsyncActions`,
`WithRollingActionQueueSize`, and `WithRollingDeleteActions`.

## Async Appender

Type: `async`.

Async appender queues events and sends them to one or more delegate appenders on
a single background worker. It is useful when only selected destinations should
be asynchronous.

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

Defaults: queue size 1024, batch size 64, overflow `block`, wait `block`.
Queue size is normalized to the ring-buffer capacity required by the runtime.
Close drains the queue.

Programmatic API: `NewAsyncAppender`, `WithAsyncName`, `WithAsyncQueueSize`,
`WithAsyncBatchSize`, `WithAsyncOverflowStrategy`, `WithAsyncWaitStrategy`,
`WithAsyncWaitOptions`, `WithAsyncErrorHandler`, and
`WithAsyncCloseAppenders`.

## Handler-Level Async

`asyncLogger`, `async-logger`, or `async` config enables a single async queue at
the handler boundary. Use it when most events should pass through the same
queue. Defaults when enabled: queue size 4096, batch size 64, overflow `block`,
wait `block`.

Handler-level async runtime shape cannot change during reload. Enablement,
queue size, batch size, overflow strategy, wait strategy, wait options, and
include-location must remain stable.

## Overflow And Wait Strategies

| Overflow strategy | Behavior |
| --- | --- |
| `block` | Producers wait until capacity is available. |
| `drop` | Drops new events when the queue is full and increments the dropped counter. |
| `drop-debug` | Drops events at DEBUG or lower when full; higher levels block. |
| `sync-fallback` | Writes synchronously when the queue is full. |

| Wait strategy | Behavior |
| --- | --- |
| `block` | General-purpose blocking wait. |
| `sleep` | Sleeps between retries; accepts `sleepTime`, `waitRetries`, and `timeout`. |
| `yield` | Yields the processor while waiting. |
| `spin` | Busy spin. Use only after benchmark evidence. |

## Failover

Types: `failover`, `failoverAppender`.

Failover tries the primary appender first. If it returns an error, failovers are
tried in order until one succeeds. When all fail, the joined error is returned.

```yaml
appenders:
  reliable:
    type: failover
    primary: primaryFile
    failovers: [stderrConsole]
```

Config-built failover appenders do not close child appenders themselves because
the router owns the full appender list. Programmatic failover closes children by
default unless `WithFailoverCloseChildren(false)` is used.

## Routing

Types: `routing`, `routingAppender`.

Routing selects a downstream appender by event attribute. The default route key
is `route`; config can set `routeKey`.

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

If the event has no matching route and no default route, the event is skipped
without error.

## Rewrite

Types: `rewrite`, `rewriteAppender`.

Rewrite applies a policy before delegating. The built-in configured policy adds
attributes from `attrs`, `attributes`, or `properties`, and removes keys from
`remove`, `removeAttrs`, or `remove-attrs`.

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

Programmatic API allows a custom `RewritePolicy`.

## Appender References

Appender refs can be strings or objects.

```yaml
appenderRefs:
  - console
  - ref: rolling
    level: warn
    includeLocation: true
    filterRefs: [auditMarker]
```

Per-reference `level` filters before the appender call. `includeLocation: true`
forces caller capture; `includeLocation: false` clears caller data for that
reference.
