# Appender Reference

An appender is the final output boundary for a log event. Every appender must be
safe for concurrent `Append` calls and must release resources on `Close`.

The public appender contract is:

```go
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}
```

## Built-In Appender Types

| Type | Aliases | Purpose |
| --- | --- | --- |
| `console` | none | Writes formatted events to stdout or stderr. |
| `file` | none | Writes formatted events to one regular file. |
| `json` | `jsonDirect`, `jsonWriter` | Writes hand-encoded single-line JSON to stdout, stderr, or a file. |
| `rolling`, `rollingFile` | `rolling-file`, `rolling_file` after normalization | Writes one active file and rolls archives by size, time, cron, or startup. |
| `async` | none | Wraps one or more downstream appenders behind a bounded queue. |
| `failover`, `failoverAppender` | `failover-appender`, `failover_appender` after normalization | Tries a primary appender first, then failover appenders when writes fail. |
| `routing`, `routingAppender` | `routing-appender`, `routing_appender` after normalization | Selects a downstream appender by route key. |
| `rewrite`, `rewriteAppender` | `rewrite-appender`, `rewrite_appender` after normalization | Rewrites event attributes before writing to a delegate appender. |

Appender and plugin kinds are normalized by trimming spaces, lowercasing, and
removing `-` and `_`.

## Common Configuration Fields

These fields are accepted by the appender config object. Only the relevant
built-in appender uses each field. External appender plugins can also read them
through `AppenderBuildConfig`.

| Field | Aliases | Used by core | Description |
| --- | --- | --- | --- |
| `type` | none | all | Required appender type. |
| `target` | none | console, json | `stderr`, `stdout`; JSON also rejects `file` unless `fileName` is set. |
| `fileName` | `file-name`, `path` | file, json file, rolling | Active log file path. |
| `layout` | none | console, file, rolling | Layout object. JSON direct appender ignores `layout`. |
| `rolling` | none | rolling | Rolling policy and strategy object. |
| `appenderRefs` | `appender-refs`, `refs` | async, failover, rewrite | Downstream appender references. |
| `primary` | `primary-ref` | failover | Primary appender reference. |
| `failovers` | `failover-refs` | failover | Failover appender references. |
| `routeKey` | `route-key` | routing | Event attribute used as route key. |
| `defaultRoute` | `default-route` | routing | Fallback appender reference. |
| `routes` | none | routing | Map from route key to appender name. |
| `rewrite` | none | rewrite | Attribute rewrite policy. |
| `queueSize` | `queue-size` | async | Async appender queue size. |
| `batchSize` | `batch-size` | async | Async appender batch size. |
| `overflowStrategy` | `overflow-strategy` | async | Queue-full behavior. |
| `waitStrategy` | `wait-strategy` | async | Consumer wait behavior. |
| `waitRetries` | `wait-retries` | async | Optional wait strategy retries. |
| `sleepTime` | `sleep-time` | async | Optional wait strategy sleep duration. |
| `timeout` | none | async | Optional blocking timeout. |
| `bufferSize` | `buffer-size` | file, json file, rolling | Application-level buffer size. `0` disables buffering. |
| `flushOnWrite` | `flush-on-write` | file, json file, rolling | Flushes the appender buffer after every event. |
| `append` | none | file, rolling | Appends instead of truncating the active file. |
| `createOnDemand` | `create-on-demand` | file, rolling | Delays file creation until the first write. |
| `filePermissions` | `file-permissions` | file, rolling | New file permissions. Accepts octal or `rwxr-x---` style. |
| `filters` | `filterRefs`, `filter-refs` | all | Filter chain applied at the appender wrapper level. |
| `url` | none | external only | Reserved for external appender plugins. |
| `method` | none | external only | Reserved for external appender plugins. |
| `address` | none | external only | Reserved for external appender plugins. |
| `network` | none | external only | Reserved for external appender plugins. |
| `facility` | none | external only | Reserved for external appender plugins. |
| `appName` | `app-name` | external only | Reserved for external appender plugins. |
| `connectTimeout` | `connect-timeout` | external only | Reserved for external appender plugins. |
| `writeTimeout` | `write-timeout` | external only | Reserved for external appender plugins. |

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

| Field | Default | Description |
| --- | --- | --- |
| `type` | required | `console`. |
| `target` | `stderr` | `stderr` or `stdout`. XML also accepts `SYSTEM_ERR`, `STDERR`, `SYSTEM_OUT`, `STDOUT`. |
| `layout` | default pattern | Any built-in or registered layout. |
| `filters` | empty | Optional appender-level filters. |

Programmatic API:

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

| Field | Default | Description |
| --- | --- | --- |
| `fileName`, `file-name`, `path` | required | File path. Parent directories are created with `0755`. |
| `layout` | default pattern | Layout used before writing. |
| `bufferSize`, `buffer-size` | `256KiB` | Application buffer size. `0` disables buffering. Negative values fail. |
| `flushOnWrite`, `flush-on-write` | false | Flush after every event. More durable, slower. |
| `append` | true | `true` uses `O_APPEND`; `false` truncates on open. |
| `createOnDemand`, `create-on-demand` | false | When true, the file opens on first append. |
| `filePermissions`, `file-permissions` | `0644` | Octal such as `0600`, or symbolic `rw-------`. |

Programmatic API:

```go
appender, err := goarklog.NewFileAppender("logs/app.log",
	goarklog.WithFileName("file"),
	goarklog.WithFileLayout(goarklog.NewJSONLayout(goarklog.LayoutOptions{EventEOL: true})),
	goarklog.WithFileBufferSize(256*1024),
	goarklog.WithFileAppend(true),
)
```

## JSON Appender

The JSON appender bypasses general layout dispatch and writes a fixed JSON event
shape directly:

```json
{"time":"2026-08-25T10:15:30.123+08:00","level":"INFO","logger":"goark.http","msg":"request done","status":200}
```

Console JSON:

```yaml
appenders:
  json:
    type: json
    target: stdout
```

File JSON:

```yaml
appenders:
  json:
    type: json
    fileName: logs/app.json
    bufferSize: 256KiB
    flushOnWrite: false
```

| Field | Default | Description |
| --- | --- | --- |
| `target` | `stderr` | `stderr` or `stdout` when `fileName` is empty. |
| `fileName`, `file-name`, `path` | empty | When set, JSON is written to this file with the file appender path rules. |
| `bufferSize`, `buffer-size` | `256KiB` for file mode | File buffer size. `0` disables buffering. |
| `flushOnWrite`, `flush-on-write` | false | Flushes the file buffer after every event. |

`target: file` without `fileName` is invalid. Programmatic `NewJSONFileAppender`
rejects an explicit writer because file mode owns the file lifecycle.

Programmatic API:

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

Appender-level file fields are the same as `file`: `fileName`, `layout`,
`bufferSize`, `flushOnWrite`, `append`, `createOnDemand`, and
`filePermissions`.

Rolling defaults:

| Setting | Default |
| --- | --- |
| appender name | `rollingFile` |
| `maxSize` | `10MiB` |
| `maxBackups` | `7` |
| `bufferSize` | `256KiB` |
| `fileIndex` | `nomax` |
| time `modulate` | true |
| action queue size | `32` |

At least one trigger must be enabled: size, time, cron, or startup.

### Rolling Fields

| Field | Aliases | Description |
| --- | --- | --- |
| `filePattern` | `file-pattern` | Archive path pattern. Supports `%d{layout}`, `%i`, `%0Ni`, and `%%`. |
| `maxSize` | `max-size` | Legacy shortcut for size policy. |
| `interval` | none | Legacy shortcut for time policy. |
| `cron` | `cronSchedule`, `cron-schedule` | Legacy shortcut for cron policy. |
| `onStartup` | `on-startup` | Legacy shortcut for startup policy. |
| `maxBackups` | `max-backups` | Legacy shortcut for retained archive count. |
| `maxAge` | `max-age` | Legacy shortcut for retained archive age. |
| `gzip` | `compress` | Enables gzip archive compression. Also enabled when `filePattern` ends with `.gz`. |
| `directWrite` | `direct-write` | Writes directly to the active pattern file instead of renaming a stable active file. Requires `filePattern`; incompatible with gzip. |
| `asyncActions` | `async-actions` | Runs compression and delete actions on a single background worker. |
| `actionQueueSize` | `action-queue-size` | Bounded queue for async rolling actions. `0` means default `32`. |

### Policies

Policy names accept concise and Log4j2-style names:

| YAML field | Aliases | Fields |
| --- | --- | --- |
| `policies.size` | `size-based-triggering-policy`, `sizeBasedTriggeringPolicy`, `SizeBasedTriggeringPolicy` | `size`, `maxSize`, `max-size`. |
| `policies.time` | `time-based-triggering-policy`, `timeBasedTriggeringPolicy`, `TimeBasedTriggeringPolicy` | `interval`, `every`, `unit`, `modulate`. |
| `policies.cron` | `cron-triggering-policy`, `cronTriggeringPolicy`, `CronTriggeringPolicy` | `schedule`, `cron`, `cronSchedule`, `cron-schedule`. |
| `policies.startup` | `on-startup-triggering-policy`, `onStartupTriggeringPolicy`, `OnStartupTriggeringPolicy` | `enabled`. |

When a size policy is active and `filePattern` is set, the pattern must include
`%i`; otherwise configuration fails.

### Strategy

| Field | Aliases | Description |
| --- | --- | --- |
| `type` | none | `directWrite`, `direct-write`, or `directWriteRolloverStrategy` enables direct-write mode. |
| `max` | none | Retained archive count. Overrides `maxBackups`. |
| `maxBackups` | `max-backups` | Retained archive count. |
| `maxAge` | `max-age` | Retained archive age. |
| `fileIndex` | `file-index` | `nomax`, `no-max`, `none`, `max`, or `min`. |
| `directWrite` | `direct-write` | Direct-write boolean. |
| `asyncActions` | `async-actions` | Async compression/delete actions. |
| `actionQueueSize` | `action-queue-size` | Async action queue size. |
| `compression.gzip` | `compression.compress` | Enables gzip compression. |
| `compression.async` | none | Enables async actions. |
| `delete` | none | Single delete action. |
| `deleteActions` | `delete-actions` | Multiple delete actions. |

### Delete Action

| Field | Aliases | Default | Description |
| --- | --- | --- | --- |
| `basePath` | `base-path` | archive directory or active file directory | Directory to scan. |
| `maxDepth` | `max-depth` | `1` | Maximum file depth under `basePath`. |
| `glob` | `ifFileName.glob` | `*` | File name or relative path glob. |
| `age` | `ifLastModified.age` | disabled | Deletes files older than this age. |
| `maxCount` | `max-count`, `ifAccumulatedFileCount.exceeds` | disabled | Keeps newest files up to this count. |
| `maxSize` | `max-size`, `ifAccumulatedFileSize.exceeds` | disabled | Keeps newest files until accumulated size exceeds this value. |
| `async` | none | false | Enables rolling async actions when present under strategy delete entries. |

Programmatic API:

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

Async appender wraps downstream appenders. It is useful when only a specific
sink should be asynchronous.

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

| Field | Default | Description |
| --- | --- | --- |
| `appenderRefs`, `refs` | required | One or more downstream appenders. |
| `queueSize` | `1024` | Must be greater than zero. Normalized to ring-buffer capacity. |
| `batchSize` | `64` | Must be greater than zero; capped to queue size. |
| `overflowStrategy` | `block` | Same aliases as Handler-level async logger. |
| `waitStrategy` | `block` | Same aliases as Handler-level async logger. |
| `waitRetries` | `0` | Non-negative. |
| `sleepTime` | `0` | Go duration. |
| `timeout` | `0` | Go duration. |

`Close` waits for producers and drains queued events. In configuration-built
composite appenders, child appenders are owned by the full handler runtime and
are not closed twice by the wrapper itself.

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

`primary` plus at least one failover is required. The shorthand
`appenderRefs: [primary, fallback]` is also accepted; the first reference is the
primary.

Failover behavior:

- If the primary write succeeds, failovers are not called.
- If the primary fails, failovers are tried in order.
- The first successful failover completes the write.
- If every delegate fails, the returned error joins all failures.

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

| Field | Default | Description |
| --- | --- | --- |
| `routeKey`, `route-key` | `route` | Event attribute read as route key. |
| `routes` | empty | Map of route value to appender name. |
| `defaultRoute`, `default-route` | empty | Optional fallback appender. |

At least one route or a default route is required. When no route matches and no
default route exists, the event is skipped without error.

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

| Field | Aliases | Description |
| --- | --- | --- |
| `appenderRefs` | `refs` | Exactly one downstream appender. |
| `rewrite.attrs` | `rewrite.attributes`, `rewrite.properties` | Adds or overwrites attributes with configured string values. |
| `rewrite.remove` | `rewrite.removeAttrs`, `rewrite.remove-attrs` | Removes attributes by key before writing. |

The built-in rewrite policy is attribute-only. More complex behavior should use
the programmatic `NewRewriteAppender` with a custom `RewritePolicy` or a plugin.

## Unsupported Core Appenders

The XML schema has `<Http>`, `<Socket>`, and `<Syslog>` element slots, and the
generic config object exposes fields such as `url`, `address`, `facility`, and
timeouts. The core module does not register network appenders. A config using
those types fails unless an external module registers the matching appender
factory.
