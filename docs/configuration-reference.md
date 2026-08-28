# Configuration Reference

[简体中文](configuration-reference.zh-CN.md)

This is the exhaustive configuration reference for the current core module.
YAML, JSON, and TOML share the same logical model. XML and properties map into
that model with the format-specific keys listed below.

## Top-Level Model

| Field | Alias | Type | Default | Notes |
| --- | --- | --- | --- | --- |
| `configuration` | none | object | none | Optional wrapper. Must not be mixed with top-level fields or `goark.log`. |
| `goark.log` | none | object | none | Optional wrapper for Goark boot-style configuration. Must not be mixed with `configuration`. |
| `status` | none | string | none | Parsed and lookup-resolved for compatibility. Status logger behavior is controlled by `NewStatusLogger`. |
| `monitorInterval` | `monitor-interval` | duration | `0` | Enables `LoggerContext` polling reload when positive. Plain numbers are seconds. |
| `properties` | none | map string to string | empty | File-local values available through `${NAME}`, `${prop:NAME}`, and `${property:NAME}`. |
| `customLevels` | `custom-levels` | map string to int string | empty | Registers process-wide custom level names. Names must be non-empty and non-numeric. |
| `appenders` | none | map string to appender | default console | Appender names are the map keys. |
| `filters` | none | map string to filter | empty | Filter names are the map keys. |
| `filterRefs` | `filter-refs` | string array | empty | Global filters run before level checks. `ACCEPT` bypasses level filtering; `DENY` drops the event. |
| `asyncLogger` | `async-logger`, `async` | object | disabled | Handler-level asynchronous logger pipeline. Use only one alias per file. |
| `root` | none | logger object | `INFO` to first appender | Root logger rule. |
| `loggers` | none | map string to logger object | empty | Named logger rules using longest-prefix matching. |

## Value Parsers

| Value | Accepted forms |
| --- | --- |
| Level | `ALL`, `TRACE`, `DEBUG`, `INFO`, `WARN`, `WARNING`, `ERROR`, `FATAL`, `OFF`, or integer. |
| Duration | Go duration strings such as `500ms`, `2s`, `5m`; `monitorInterval` also accepts plain seconds. |
| Byte size | `b`, `byte`, `bytes`, `k`, `kb`, `m`, `mb`, `g`, `gb`, `t`, `tb`, `ki`, `kib`, `mi`, `mib`, `gi`, `gib`, `ti`, `tib`. Decimal values are accepted. |
| Rolling interval | `off`, `none`, `disabled`, `minute`, `minutely`, `hour`, `hourly`, `day`, `daily`, Go duration, or day/minute/hour words such as `2days`. |
| Rolling max age | `off`, `none`, `disabled`, `30d`, `30day`, `30days`, or Go duration. |
| File permissions | Octal string such as `0644`, or symbolic mode such as `rw-r-----`. |
| Boolean | Go boolean parser values, including `true` and `false`. |
| Cron | 5, 6, or 7 fields. A 5-field expression gets leading seconds `0`; a 7-field expression must use `*` or `?` for year. Month and weekday names are supported. |

## Async Logger

| Field | Alias | Type | Default | Notes |
| --- | --- | --- | --- | --- |
| `enabled` | none | bool pointer | disabled | Omitted means async logger disabled. |
| `queueSize` | `queue-size` | int | `4096` when enabled | Normalized to the ring-buffer capacity required by the runtime. Must be non-negative in config. |
| `batchSize` | `batch-size` | int | `64` when enabled | Clamped to queue size. Must be non-negative in config. |
| `overflowStrategy` | `overflow-strategy` | string | `block` | `block`, `blocking`, `drop`, `discard`, `discard-newest`, `drop-debug`, `dropdebug`, `discard-debug`, `discarddebug`, `sync-fallback`, `sync`, `synchronous`, `synchronize`. |
| `waitStrategy` | `wait-strategy` | string | `block` | `block`, `blocking`, `timeout`, `timeout-block`, `timeoutblocking`, `sleep`, `sleeping`, `yield`, `yielding`, `spin`, `busy-spin`, `busyspin`. |
| `waitRetries` | `wait-retries` | int | `0` | Must be non-negative. |
| `sleepTime` | `sleep-time` | duration | `0` | Used by wait strategies that sleep. Invalid values fail validation. |
| `timeout` | none | duration | `0` | Optional blocking timeout. Invalid values fail validation. |
| `includeLocation` | `include-location` | bool pointer | false | Enables caller capture for handler-level async events. Cannot change during reload. |

## Logger Object

| Field | Alias | Type | Default | Notes |
| --- | --- | --- | --- | --- |
| `level` | none | level | root defaults to `INFO`; named logger inherits root when omitted | Level threshold. |
| `appenderRefs` | `appender-refs`, `refs` | string array or object array | root uses first configured appender when empty | References appenders by name. |
| `filters` | `filterRefs`, `filter-refs` | string array | empty | References named filters. |
| `additivity` | none | bool pointer | true for named loggers | `false` requires at least one appender on the named logger. |
| `includeLocation` | `include-location` | bool pointer | false | Captures caller PC for layouts or appender refs requiring location. |

Appender reference object:

| Field | Alias | Type | Default | Notes |
| --- | --- | --- | --- | --- |
| `ref` | none | string | required | Target appender name. |
| `level` | none | level | none | Per-reference level gate. |
| `includeLocation` | `include-location` | bool pointer | inherits route | `false` clears caller PC for this reference; `true` forces location capture. |
| `filters` | `filterRefs`, `filter-refs` | string array | empty | Per-reference filters. |

## Appender Types

| Type | Aliases | Built-in behavior |
| --- | --- | --- |
| Console | `console` | Writes to stderr or stdout with a layout. |
| File | `file` | Writes to a local file with buffering, permissions, headers, and footers. |
| JSON direct | `json`, `jsonDirect`, `jsonWriter` | Encodes events directly as single-line JSON. Uses stdout/stderr or a file when `fileName` is set. |
| Rolling file | `rolling`, `rollingFile` | Local rolling file appender with size/time/cron/startup policies and archive actions. |
| Async | `async` | Queues events and writes to delegate appenders on one background worker. |
| Failover | `failover`, `failoverAppender` | Tries primary first, then failovers in order. |
| Routing | `routing`, `routingAppender` | Selects a downstream appender by event attribute. |
| Rewrite | `rewrite`, `rewriteAppender` | Adds and removes attributes before delegating. |

Common appender fields:

| Field | Alias | Used by built-ins | Notes |
| --- | --- | --- | --- |
| `type` | none | all | Required for configured appenders. Type matching ignores case, hyphen, and underscore. |
| `layout` | none | console, file, rolling-file | Omitted layout defaults to pattern layout for these appenders. Ignored by JSON direct. |
| `filters` | `filterRefs`, `filter-refs` | all | Wraps the appender with a filter chain. |
| `target` | none | console, JSON direct | Console accepts `stderr`, `stdout`; JSON direct accepts `stderr`, `stdout`, or omitted. |
| `fileName` | `file-name`, `path` | file, JSON direct file, rolling-file | Required for file and rolling-file. Optional for JSON direct. |
| `bufferSize` | `buffer-size` | file, JSON direct file, rolling-file | `0` disables application buffering. |
| `flushOnWrite` | `flush-on-write` | file, JSON direct file, rolling-file | Flushes the buffered writer after each event. |
| `append` | none | file, rolling-file | Defaults to true. `false` truncates on open. |
| `createOnDemand` | `create-on-demand` | file, rolling-file | Delays file creation until the first event. |
| `filePermissions` | `file-permissions` | file, rolling-file | Defaults to `0644` unless explicitly set. |
| `appenderRefs` | `appender-refs`, `refs` | async, failover, rewrite | Delegate references. Objects are supported for async. |
| `primary` | `primary-ref` | failover | Primary delegate. |
| `failovers` | `failover-refs` | failover | Ordered failover delegates. |
| `routeKey` | `route-key` | routing | Event attribute used as the route key. Default in code is `route` when omitted. |
| `defaultRoute` | `default-route` | routing | Used when the route key is missing or unmatched. |
| `routes` | none | routing | Map from route key to appender name. |
| `rewrite` | none | rewrite | Built-in attribute rewrite policy. |
| `queueSize` | `queue-size` | async | Appender queue size; must be greater than zero at runtime. |
| `batchSize` | `batch-size` | async | Background batch size; must be greater than zero at runtime. |
| `overflowStrategy` | `overflow-strategy` | async | Same values as async logger. |
| `waitStrategy` | `wait-strategy` | async | Same values as async logger. |
| `waitRetries` | `wait-retries` | async | Optional wait parameter. |
| `sleepTime` | `sleep-time` | async | Optional wait parameter. |
| `timeout` | none | async | Optional wait parameter. |
| `url`, `method`, `address`, `network`, `facility`, `appName`, `app-name`, `connectTimeout`, `connect-timeout`, `writeTimeout`, `write-timeout` | none | external plugins | Parsed and passed through `AppenderBuildConfig`; current built-ins do not implement remote sinks. |

Rewrite object:

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `attrs` | `attributes`, `properties` | map string to string | Adds attributes sorted by key. |
| `remove` | `removeAttrs`, `remove-attrs` | string array | Removes matching event attribute keys. |

## Rolling Object

The rolling object is used only by `rolling-file`.

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `filePattern` | `file-pattern` | string | Archive path pattern. Supports `%d{...}` and `%i`. `.gz` suffix enables gzip. |
| `maxSize` | `max-size` | byte size | Legacy size policy. Prefer `policies.size.size`. |
| `interval` | none | interval | Legacy time policy. Prefer `policies.time`. |
| `cron` | `cronSchedule`, `cron-schedule` | cron | Legacy cron policy. Prefer `policies.cron`. |
| `onStartup` | `on-startup` | bool | Legacy startup policy. Prefer `policies.startup.enabled`. |
| `maxBackups` | `max-backups` | int pointer | Legacy retention. Prefer `strategy.max`. |
| `maxAge` | `max-age` | duration | Legacy retention. Prefer `strategy.maxAge`. |
| `gzip` | `compress` | bool | Enables gzip compression. |
| `directWrite` | `direct-write` | bool | Writes directly to `filePattern`; requires `filePattern` and rejects gzip. |
| `asyncActions` | `async-actions` | bool | Runs compression and delete actions on a serial background worker. |
| `actionQueueSize` | `action-queue-size` | int | Queue size for asynchronous rolling actions; default is `32`. |
| `policies` | none | object | Log4j2-style triggering policies. |
| `strategy` | none | object | Log4j2-style rollover strategy. |

Rolling policies:

| Field | Aliases | Child fields |
| --- | --- | --- |
| `policies.size` | `size-based-triggering-policy`, `sizeBasedTriggeringPolicy`, `SizeBasedTriggeringPolicy` | `size`, `maxSize`, `max-size` |
| `policies.time` | `time-based-triggering-policy`, `timeBasedTriggeringPolicy`, `TimeBasedTriggeringPolicy` | `interval`, `every`, `unit`, `modulate` |
| `policies.cron` | `cron-triggering-policy`, `cronTriggeringPolicy`, `CronTriggeringPolicy` | `schedule`, `cron`, `cronSchedule`, `cron-schedule`, `evaluateOnStartup` |
| `policies.startup` | `on-startup-triggering-policy`, `onStartupTriggeringPolicy`, `OnStartupTriggeringPolicy` | `enabled` |

Rolling strategy:

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `type` | none | string | `directWrite`, `direct-write`, and `directWriteRolloverStrategy` enable direct write. |
| `max` | `maxBackups`, `max-backups` | int pointer | Archive count retention. |
| `maxAge` | `max-age` | duration | Archive age retention. |
| `fileIndex` | `file-index` | string | `nomax`, `no-max`, `none`, `max`, or `min`. |
| `directWrite` | `direct-write` | bool | Enables direct write. |
| `asyncActions` | `async-actions` | bool | Enables background archive actions. |
| `actionQueueSize` | `action-queue-size` | int | Background action queue size. |
| `compression.gzip` | `compression.compress` | bool | Enables gzip. |
| `compression.async` | none | bool | Also enables background archive actions. |
| `delete` | none | object | Single delete action. |
| `deleteActions` | `delete-actions` | object array | Additional delete actions. |

Delete action:

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `basePath` | `base-path` | string | Directory to scan. Defaults to archive pattern directory. |
| `maxDepth` | `max-depth` | int pointer | Defaults to `1`; must be non-negative. |
| `maxCount` | `max-count`, `ifAccumulatedFileCount.exceeds`, `if-accumulated-file-count.exceeds` | int pointer | Keeps newest files up to the count. |
| `maxSize` | `max-size`, `ifAccumulatedFileSize.exceeds`, `if-accumulated-file-size.exceeds` | byte size | Keeps newest files until accumulated size exceeds the limit. |
| `glob` | `ifFileName.glob`, `if-file-name.glob` | glob | Matches file base name or relative path. Defaults to `*`. |
| `age` | `ifLastModified.age`, `if-last-modified.age` | duration | Deletes files older than age. |
| `async` | none | bool | Enables background archive actions when present on `delete`, `deleteActions`, or `delete-actions`. |

## Layout Types

| Type | Aliases | Output |
| --- | --- | --- |
| Pattern | `pattern` | Text pattern with Log4j-style converters. |
| Text | `text` | Stable `key=value` text. |
| JSON | `json` | Event JSON object. |
| JSON Template | `jsonTemplate` | Event JSON controlled by resolver template. |
| XML | `xml`, `xmlLayout` | Single `<Event>` XML fragment. |
| CSV | `csv`, `csvLayout` | Fixed field CSV line. |
| GELF | `gelf`, `gelfLayout` | Graylog Extended Log Format JSON. |
| RFC5424 | `rfc5424`, `rfc5424Layout` | RFC 5424 syslog text line. |
| Syslog | `syslog`, `syslogLayout` | Alias of RFC5424 layout. |
| YAML | `yaml`, `yamlLayout` | Single YAML document. |
| HTML | `html`, `htmlLayout` | HTML table row. |

Layout fields:

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `type` | none | string | Omitted layout type defaults to pattern. |
| `pattern` | none | string | Pattern layout only. Empty pattern uses `DefaultSpringBootPattern`. |
| `eventTemplate` | `event-template` | string | JSON Template inline object. |
| `eventTemplateUri` | `event-template-uri`, `eventTemplatePath`, `event-template-path` | string | JSON Template file path. |
| `compact` | none | bool | Suppresses default event newline. |
| `eventEol` | `event-eol` | bool | Adds event newline even in compact streams. |
| `complete` | none | bool | Writes layout header/footer for stream formats. JSON and JSON Template isolate complete-mode state per appender. |
| `includeStacktrace` | `include-stacktrace` | bool | Emits structured stack data where supported. |
| `stacktraceAsString` | `stacktrace-as-string` | bool | Emits throwable stack as one string where supported. |
| `propertiesAsList` | `properties-as-list` | bool | Emits context map as key/value list where supported. |
| `includeNullDelimiter` | `include-null-delimiter` | bool | Appends NUL after each event. |
| `disableAnsi` | `disable-ansi` | bool | Disables PatternLayout `highlight` and `style` ANSI output. |
| `header` | none | string | Header for complete/lifecycle layouts. |
| `footer` | none | string | Footer for complete/lifecycle layouts. |

## Filter Types

| Type | Aliases | Required fields | Behavior |
| --- | --- | --- | --- |
| Threshold | `threshold`, `thresholdFilter` | `level` | Matches events at or above level. |
| Level | `level`, `levelFilter` | `level` | Matches exactly one level. |
| LevelRange | `levelRange`, `levelRangeFilter` | `minLevel`, `maxLevel` | Inclusive range. |
| Regex | `regex`, `regexFilter` | `pattern`; `key` if `field=attr` | Matches message, logger, or an attribute. |
| Attr | `attr`, `attribute`, `attrFilter`, `attributeFilter` | `key` | Matches attribute key and exact string value. |
| Deny | `deny`, `denyAll`, `denyFilter`, `denyAllFilter` | none | Always denies. |
| Composite | `composite`, `compositeFilter` | `filterRefs` | Runs nested filters in order. |
| Marker | `marker`, `markerFilter` | `marker` or `value` | Matches marker name or parent marker. |
| NoMarker | `noMarker`, `noMarkerFilter` | none | Matches events without marker. |
| Map | `map`, `mapFilter` | `values`, `key/value`, or key-value pairs | Matches event attributes with `and` or `or`. |
| ThreadContextMap | `threadContextMap`, `threadContextMapFilter` | map values | Alias behavior of MapFilter for MDC-style data. |
| ThreadContextStack | `threadContextStack`, `threadContextStackFilter` | `value`, `text`, or `pattern` | Matches context stack value. |
| StructuredData | `structuredData`, `structuredDataFilter` | map values | Alias behavior of MapFilter for structured data attrs. |
| Throwable | `throwable`, `throwableFilter` | `pattern`, `text`, or `value` | Regex matches throwable text. |
| StringMatch | `stringMatch`, `stringMatchFilter` | `text`, `value`, or `pattern` | Substring match on message. |
| Time | `time`, `timeFilter` | optional | Matches time-of-day range. Defaults to full day. |
| Burst | `burst`, `burstFilter` | optional | Token-bucket limiter for events at or below level. Defaults level `warn`, rate `10`, maxBurst `rate*10`. |
| DynamicThreshold | `dynamicThreshold`, `dynamicThresholdFilter` | `key` | Selects threshold from event attribute value. |

Filter fields:

| Field | Alias | Type | Notes |
| --- | --- | --- | --- |
| `type` | none | string | Required. Matching ignores case, hyphen, and underscore. |
| `level` | none | level | Threshold, Level, Burst default, or DynamicThreshold default fallback. |
| `minLevel` | `min-level` | level | LevelRange minimum. |
| `maxLevel` | `max-level` | level | LevelRange maximum. |
| `marker` | none | string | Marker name. |
| `text` | none | string | StringMatch, Throwable, or ThreadContextStack value. |
| `operator` | none | string | Map operator: `and` or `or`. |
| `start` | none | time-of-day | TimeFilter start, accepts `HH:MM`, `HH:MM:SS`, or nanosecond fraction. Default `00:00:00`. |
| `end` | none | time-of-day | TimeFilter end. Default `23:59:59.999999999`. |
| `timezone` | none | IANA zone | Optional time zone for TimeFilter. |
| `rate` | none | float string | Burst rate per second. |
| `maxBurst` | `max-burst` | int | Burst maximum tokens. |
| `field` | none | string | Regex field: `message`, `msg`, `logger`, `name`, `attr`, `attribute`. |
| `key` | none | string | Attribute key, map single key, dynamic threshold key, or regex attr key. |
| `value` | none | string | Attribute value, marker fallback, string fallback, throwable fallback, or thread stack fallback. |
| `values` | none | map string to string | Map-like filters. |
| `thresholds` | none | map string to level string | DynamicThreshold values. |
| `filters` | `filterRefs`, `filter-refs` | string array | Nested filters for Composite or any filter that needs nested refs during build. |
| `KeyValuePair` | `keyValuePairs`, `key-value-pairs` | array of `{key,value}` | Adds map values or dynamic thresholds depending on filter type. |
| `defaultThreshold` | `default-threshold` | level | DynamicThreshold default. |
| `pattern` | none | string | Regex, Throwable, StringMatch, or ThreadContextStack fallback. |
| `onMatch` | `on-match` | decision | `neutral`, `accept`, or `deny`. Default is `neutral`. |
| `onMismatch` | `on-mismatch` | decision | `neutral`, `accept`, or `deny`. Default is `deny`. |

## XML Mapping

XML root:

| Element or attribute | Maps to |
| --- | --- |
| `<Configuration status="" monitorInterval="">` | `status`, `monitorInterval` |
| `<Properties><Property name="" value="">text</Property></Properties>` | `properties` |
| `<CustomLevels><CustomLevel name="" intLevel="" value="">text</CustomLevel></CustomLevels>` | `customLevels` |
| `<AsyncLogger .../>` | `asyncLogger` |
| `<Loggers><Root>...</Root><Logger name="">...</Logger></Loggers>` | `root`, `loggers` |

XML appenders under `<Appenders>`:

| Element | Notes |
| --- | --- |
| `<Console>` | Built-in console. `target` accepts `SYSTEM_OUT`, `STDOUT`, `SYSTEM_ERR`, `STDERR`. |
| `<File>` | Built-in file. |
| `<RollingFile>` | Built-in rolling file. |
| `<Async>` | Built-in async appender. |
| `<Failover>` | Built-in failover. `<Failovers><AppenderRef ref="..."/></Failovers>` is supported. |
| `<Routing>` | Built-in routing. `<Route key="" ref="">` or nested `<AppenderRef>`. |
| `<Rewrite>` | Built-in rewrite. Uses `<KeyValuePair key="" value="">` and `<Remove key="">`. |
| `<Http>`, `<Socket>`, `<Syslog>` | Parsed and passed to plugins if registered; core does not provide these appender factories. |

XML layouts under appenders: `<PatternLayout>`, `<TextLayout>`, `<JsonLayout>`,
`<JSONLayout>`, `<JsonTemplateLayout>`, `<XmlLayout>`, `<XMLLayout>`,
`<CsvLayout>`, `<CSVLayout>`, `<GelfLayout>`, `<GELFLayout>`,
`<Rfc5424Layout>`, `<RFC5424Layout>`, `<SyslogLayout>`, `<YamlLayout>`,
`<YAMLLayout>`, `<HtmlLayout>`, `<HTMLLayout>`, or generic `<Layout type="">`.

XML filters under `<Filters>`: `<ThresholdFilter>`, `<LevelFilter>`,
`<LevelRangeFilter>`, `<RegexFilter>`, `<AttrFilter>`, `<AttributeFilter>`,
`<DenyFilter>`, `<DenyAllFilter>`, `<CompositeFilter>`, `<MarkerFilter>`,
`<NoMarkerFilter>`, `<MapFilter>`, `<ThreadContextMapFilter>`,
`<ThreadContextStackFilter>`, `<StructuredDataFilter>`, `<ThrowableFilter>`,
`<StringMatchFilter>`, `<TimeFilter>`, `<BurstFilter>`, and
`<DynamicThresholdFilter>`.

## Properties Mapping

Top-level keys:

| Key | Maps to |
| --- | --- |
| `status` | `status` |
| `monitorInterval`, `monitor-interval` | `monitorInterval` |
| `property.NAME` | `properties.NAME` |
| `customLevel.NAME`, `custom-level.NAME` | `customLevels.NAME` |
| `rootLogger.level`, `root.level` | `root.level` |
| `rootLogger.appenderRefs`, `root.appenderRefs` | `root.appenderRefs` |
| `rootLogger.filters`, `root.filters` | `root.filters` |
| `rootLogger.includeLocation`, `rootLogger.include-location`, `root.includeLocation`, `root.include-location` | `root.includeLocation` |
| `rootLogger.appenderRef.ID.FIELD`, `root.appenderRef.ID.FIELD` | structured root appender ref |
| `asyncLogger.FIELD`, `async-logger.FIELD`, `async.FIELD` | async logger |
| `appender.ID.FIELD` | appender |
| `logger.ID.FIELD` | named logger |
| `filter.ID.FIELD` | filter |

Properties-specific appender fields include all common appender fields plus:

| Key pattern | Notes |
| --- | --- |
| `appender.ID.name` | Renames the appender from the logical ID. |
| `appender.ID.layout.FIELD` | Layout field. |
| `appender.ID.routes.ROUTE_KEY` | Routing route map entry. |
| `appender.ID.rewrite.attrs.KEY`, `.attributes.KEY`, `.properties.KEY` | Rewrite added attr. |
| `appender.ID.rewrite.remove`, `.removeAttrs`, `.remove-attrs` | Rewrite removals. |
| `appender.ID.appenderRef.REF_ID.FIELD` | Structured appender ref for composite appenders. |
| `appender.ID.routeKey`, `.route-key`, `.attrKey`, `.attr-key` | Routing attr key. |

Properties rolling keys currently mapped by the core:

| Key | Maps to |
| --- | --- |
| `rolling.filePattern`, `rolling.file-pattern` | rolling `filePattern` |
| `rolling.maxSize`, `rolling.max-size` | rolling `maxSize` |
| `rolling.interval` | rolling `interval` |
| `rolling.cron`, `rolling.cronSchedule`, `rolling.cron-schedule`, `rolling.policies.cron.schedule`, `rolling.policies.cronTriggeringPolicy.schedule`, `rolling.policies.cron-triggering-policy.schedule` | rolling cron schedule |
| `rolling.strategy.type` | rolling strategy type |
| `rolling.strategy.fileIndex`, `rolling.strategy.file-index` | rolling strategy file index |
| `rolling.directWrite`, `rolling.direct-write`, `rolling.strategy.directWrite`, `rolling.strategy.direct-write` | direct write |
| `rolling.strategy.delete.maxCount`, `.max-count` | delete max count |
| `rolling.strategy.delete.maxSize`, `.max-size` | delete max size |
| `rolling.strategy.delete.ifAccumulatedFileCount.exceeds`, `.if-accumulated-file-count.exceeds` | delete max count |
| `rolling.strategy.delete.ifAccumulatedFileSize.exceeds`, `.if-accumulated-file-size.exceeds` | delete max size |

Properties logger fields:

| Key | Notes |
| --- | --- |
| `logger.ID.name` | Renames the logger from the logical ID. |
| `logger.ID.level` | Level. |
| `logger.ID.appenderRefs`, `.appender-refs`, `.refs` | Delegate refs. |
| `logger.ID.filters`, `.filterRefs`, `.filter-refs` | Filter refs. |
| `logger.ID.additivity` | Boolean. |
| `logger.ID.includeLocation`, `.include-location` | Boolean. |
| `logger.ID.appenderRef.REF_ID.FIELD` | Structured appender ref. |

Properties filter key-value pairs use:

```properties
filter.myFilter.keyValuePair.1.key=tenant
filter.myFilter.keyValuePair.1.value=tenant-a
```

The pair ID may start with `keyValuePair` or `kv`.
