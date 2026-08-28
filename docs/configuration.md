# Configuration Reference

[简体中文](configuration.zh-CN.md)

This document describes the configuration contract implemented by
`goark.dev/log` in the current worktree. It documents the supported field names,
defaults, validation rules, and reload boundaries for v0.0.2 preparation.

## Load Order

`LoadOptions`, `NewConfigured`, `NewConfiguredHandler`,
`NewConfiguredLoggerContext`, and `ConfigureDefault` resolve configuration in
this order:

1. `WithConfigPath(path)`.
2. Environment variable `GOARK_LOG_CONFIG`, or the key passed to
   `WithConfigEnvKey(key)`.
3. Boot property resolver keys: `goark.log.config`, `goark.logging.config`,
   `logging.config`.
4. Default files under the working directory:
   - `conf/goark-log.yml`
   - `conf/goark-log.yaml`
   - `conf/goark-log.json`
   - `conf/goark-log.xml`
   - `conf/goark-log.toml`
   - `conf/goark-log.properties`
5. Built-in default: stderr console appender and root level `INFO`.

Relative paths are resolved against `os.Getwd()` unless
`WithConfigWorkingDir(dir)` is used.

## Supported Formats

| Format | Status | Notes |
| --- | --- | --- |
| YAML | Supported | Recommended default. Parsed with strict known fields. |
| JSON | Supported | Uses the same structured schema and field names as YAML. |
| XML | Supported | Uses Log4j2-style elements for appenders, layouts, filters, policies, and loggers. |
| properties | Supported | Uses flat keys such as `appender.console.type`. |
| TOML | Rejected | A `.toml` file is detected but loading fails with `unsupported config format "toml"`. |

Structured YAML/JSON decoding is strict: unknown fields fail parsing. This is
intentional, because logging configuration mistakes should not be silently
ignored.

## Wrapper Forms

A structured YAML or JSON file can use exactly one of these forms:

```yaml
configuration:
  root:
    level: info
```

```yaml
goark:
  log:
    root:
      level: info
```

```yaml
root:
  level: info
```

Do not mix top-level fields with `configuration` or `goark.log`. Do not use both
wrappers in the same file.

## Top-Level Fields

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `status` | string | none | Parsed and lookup-resolved for compatibility. It does not currently change `StatusLogger`; set status behavior through the `NewStatusLogger` API. |
| `monitorInterval`, `monitor-interval` | duration string | disabled | Enables `LoggerContext` file polling reload when using `NewConfiguredLoggerContext`. Plain numbers mean seconds. |
| `properties` | map string to string | empty | Local values available through `${prop:key}`, `${property:key}`, and shorthand `${key}`. |
| `customLevels`, `custom-levels` | map string to integer string | empty | Registers process-wide custom log level names. |
| `appenders` | map | default console if omitted | Named appender definitions. |
| `filters` | map | empty | Named filter definitions. |
| `filterRefs`, `filter-refs` | string list | empty | Global filter chain, evaluated before logger level checks. |
| `asyncLogger`, `async-logger`, `async` | object | disabled | Handler-level asynchronous pipeline. Use only one alias. |
| `root` | object | `INFO` and first appender | Root logger route. |
| `loggers` | map | empty | Named logger routes. Exact name and prefix matches are supported. |

## Levels

Built-in levels:

| Name | Value | Meaning |
| --- | --- | --- |
| `ALL` | minimum integer | Allows every event through the level gate. |
| `TRACE` | `-8` | Lower than `DEBUG`. |
| `DEBUG` | `-4` | Same value as `slog.LevelDebug`. |
| `INFO` | `0` | Same value as `slog.LevelInfo`; default root level. |
| `WARN`, `WARNING` | `4` | Same value as `slog.LevelWarn`. |
| `ERROR` | `8` | Same value as `slog.LevelError`. |
| `FATAL` | `12` | Above `ERROR`. |
| `OFF` | maximum integer | Disables ordinary events by level. |

Numeric levels are accepted. Custom level names must be non-empty, non-numeric,
and contain no whitespace:

```yaml
configuration:
  customLevels:
    NOTICE: "2"
    SECURITY: "6"
```

Custom levels are registered in the process default level registry.

## Value Parsing

### Byte Sizes

Byte-size fields include appender `bufferSize`, rolling `maxSize`, and delete
action `maxSize`.

| Form | Meaning |
| --- | --- |
| `0` | Zero bytes; commonly disables buffering or size limits depending on the field. |
| `b`, `byte`, `bytes` | Bytes. |
| `k`, `kb`, `m`, `mb`, `g`, `gb`, `t`, `tb` | Decimal units, base 1000. |
| `ki`, `kib`, `mi`, `mib`, `gi`, `gib`, `ti`, `tib` | Binary units, base 1024. |

Decimal numbers are accepted, for example `1.5MiB`. Values must be
non-negative and fit in `int64`.

### Monitor Interval

`monitorInterval` accepts:

- empty, `0`, `off`, `none`, `disabled`, `false`: disabled.
- a plain number such as `30`: seconds.
- a Go duration such as `500ms`, `5s`, `2m`, `1h`.

Negative values are rejected.

### Rolling Interval

Rolling time policy intervals accept:

- empty, `0`, `off`, `none`, `disabled`: disabled.
- `minute`, `minutely`, `hour`, `hourly`, `day`, `daily`.
- Go durations such as `30s`, `5m`, `1h`.
- day/minute/hour suffixes such as `2days`, `4hours`, `15mins`.

Negative values are rejected.

### Rolling Max Age

Retention age fields accept:

- empty, `0`, `off`, `none`, `disabled`: disabled.
- Go durations such as `720h`.
- day forms such as `30d`, `30day`, `30days`.

Negative values are rejected.

## Lookups

Lookups are resolved before configuration objects are built.

| Form | Description |
| --- | --- |
| `${prop:LOG_DIR}` | Reads a value from `properties`. |
| `${property:LOG_DIR}` | Same as `prop`. |
| `${LOG_DIR}` | Shorthand for property lookup. |
| `${LOG_DIR:-logs}` | Property lookup with fallback. |
| `${env:HOME}` | Reads an environment variable. |
| `${env:LOG_DIR:-logs}` | Environment lookup with fallback. |
| `${sys:pid}` | Process ID. |
| `${sys:hostname}` | Host name. |
| `${sys:cwd}` | Current working directory. |
| `${sys:os}`, `${sys:arch}` | Go runtime OS and architecture. |
| `${go:version}` | Go runtime version. |
| `${go:os}`, `${go:arch}` | Go runtime OS and architecture. |
| `${date:yyyyMMdd}` | Current time formatted with the time-pattern mapper. |

`$$` escapes a single dollar sign. Lookup expressions must be closed with `}`.
Unknown namespaces and missing values without fallback fail configuration
loading.

Blocked namespaces cannot be registered: `jndi`, `ldap`, and `rmi`.

## Time Patterns

Time format values support a Java/Log4j-style subset and Unix timestamp modes.

| Pattern | Output behavior |
| --- | --- |
| empty, `DEFAULT`, `ISO8601`, `ISO8601_OFFSET_DATE_TIME` | `2006-01-02T15:04:05.000Z07:00`. |
| `RFC3339` | Go `time.RFC3339`. |
| `RFC3339NANO` | Go `time.RFC3339Nano`. |
| `UNIX`, `UNIX_SECONDS` | Unix seconds. |
| `UNIX_MILLIS`, `UNIX_MS` | Unix milliseconds. |
| `UNIX_MICROS`, `UNIX_US` | Unix microseconds. |
| `UNIX_NANOS`, `UNIX_NS` | Unix nanoseconds. |
| `yyyy`, `yy`, `MM`, `dd`, `HH`, `mm`, `ss`, `SSS`, `SSSSSS`, `X`, `XX`, `XXX` | Converted to Go reference-time layout tokens. |

## Async Logger

Handler-level async is configured by one of `asyncLogger`, `async-logger`, or
`async`.

| Field | Type | Default when enabled | Description |
| --- | --- | --- | --- |
| `enabled` | bool | false | Enables the Handler-level async pipeline. |
| `queueSize`, `queue-size` | int | `4096` | Queue capacity. Positive values are normalized to the internal ring-buffer capacity. |
| `batchSize`, `batch-size` | int | `64` | Max events consumed per drain loop. Capped to queue capacity. |
| `overflowStrategy`, `overflow-strategy` | string | `block` | Queue-full behavior. |
| `waitStrategy`, `wait-strategy` | string | `block` | Consumer wait behavior. |
| `waitRetries`, `wait-retries` | int | `0` | Optional wait-strategy retry count. Must be non-negative. |
| `sleepTime`, `sleep-time` | duration | `0` | Optional sleep duration. Must parse as Go duration. |
| `timeout` | duration | `0` | Optional blocking timeout. Must parse as Go duration. |
| `includeLocation`, `include-location` | bool | false | Captures caller PC before enqueueing. Adds cost. |

Overflow strategy aliases:

| Canonical | Aliases | Behavior |
| --- | --- | --- |
| `block` | `blocking` | Apply backpressure and do not drop events. |
| `drop` | `discard`, `discard-newest` | Drop events when the queue is full. |
| `drop-debug` | `dropdebug`, `discard-debug`, `discarddebug` | Drop `DEBUG` and lower events when full. |
| `sync-fallback` | `sync`, `synchronous`, `synchronize` | Write synchronously when full. |

Wait strategy aliases:

| Canonical | Aliases |
| --- | --- |
| `block` | `blocking`, `timeout`, `timeout-block`, `timeoutblocking` |
| `sleep` | `sleeping` |
| `yield` | `yielding` |
| `spin` | `busy-spin`, `busyspin` |

Reload cannot change async enablement, queue size, batch size, overflow
strategy, wait strategy, wait options, or async caller-location behavior.

## Root Logger

```yaml
root:
  level: info
  includeLocation: false
  appenderRefs:
    - console
    - ref: rolling
      level: warn
      includeLocation: false
      filters: [onlyErrors]
  filters: [businessHours]
```

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `level` | string | `INFO` | Minimum level when global filters do not force acceptance. |
| `appenderRefs`, `appender-refs`, `refs` | list | first configured appender | Appenders attached to this route. Entries can be strings or objects. |
| `filters`, `filterRefs`, `filter-refs` | string list | empty | Route filter chain. |
| `includeLocation`, `include-location` | bool | false | Enables caller PC capture for this route. |

## Named Loggers

```yaml
loggers:
  goark.orm:
    level: debug
    appenderRefs: [rolling]
    additivity: false
  goark.http:
    level: info
    appenderRefs:
      - ref: json
        level: warn
```

Matching is exact or prefix-based. A rule for `goark.orm` matches
`goark.orm` and `goark.orm.mapper`. The most specific rule wins.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `level` | string | root level | Minimum level for this logger route. |
| `appenderRefs`, `appender-refs`, `refs` | list | empty | Route-specific appenders. |
| `filters`, `filterRefs`, `filter-refs` | string list | empty | Logger filter chain. |
| `additivity` | bool | true | When true, root appenders and root filters are appended. When false, this logger must define at least one appender. |
| `includeLocation`, `include-location` | bool | root value | Enables or disables caller capture for this route. |

Duplicate appenders are de-duplicated by appender name during additive routing.

## Appender References

Appender references can be plain strings:

```yaml
appenderRefs: [console, rolling]
```

Or structured controls:

```yaml
appenderRefs:
  - ref: console
  - ref: rolling
    level: warn
    includeLocation: false
    filterRefs: [onlyProd]
```

Structured fields:

| Field | Type | Description |
| --- | --- | --- |
| `ref` | string | Required appender name. |
| `level` | string | Optional per-reference minimum level. |
| `includeLocation`, `include-location` | bool | Optional per-reference caller-location override. `false` strips PC before writing to that appender. |
| `filters`, `filterRefs`, `filter-refs` | string list | Optional filter chain applied only to this appender reference. |

## Properties Format

Properties files use flat keys. Unknown keys are ignored by the properties
adapter, while invalid known values fail loading.

```properties
property.LOG_DIR=logs
monitorInterval=30s

appender.console.type=console
appender.console.target=stderr
appender.console.layout.type=pattern
appender.console.layout.pattern=%d %5p %pid --- [%thread] %c : %m%attrs%n

appender.json.type=json
appender.json.fileName=${LOG_DIR}/app.json
appender.json.bufferSize=256KiB
appender.json.flushOnWrite=false

rootLogger.level=info
rootLogger.appenderRefs=console,json
logger.orm.name=goark.orm
logger.orm.level=debug
logger.orm.appenderRefs=json
logger.orm.additivity=false
```

Supported properties prefixes:

| Prefix | Purpose |
| --- | --- |
| `property.<name>` | Config property lookup value. |
| `customLevel.<name>`, `custom-level.<name>` | Custom level registration. |
| `asyncLogger.*`, `async-logger.*`, `async.*` | Handler-level async logger. |
| `appender.<id>.*` | Appender definition. |
| `appender.<id>.layout.*` | Appender layout definition. |
| `appender.<id>.rolling.*` | Rolling fields supported by the properties adapter. |
| `appender.<id>.routes.<key>` | Routing appender route mapping. |
| `appender.<id>.rewrite.*` | Rewrite appender policy. |
| `appender.<id>.appenderRef.<id>.*` | Structured appender references. |
| `filter.<id>.*` | Filter definition. |
| `filter.<id>.values.<key>` | Map-like filter value. |
| `filter.<id>.thresholds.<value>` | Dynamic-threshold mapping. |
| `logger.<id>.*` | Named logger definition. |
| `rootLogger.*`, `root.*` | Root logger definition. |

`logger.<id>.name=<actual.logger.name>` and `appender.<id>.name=<actualName>`
act as aliases for subsequent properties.

## XML Format

XML supports Log4j2-style names:

```xml
<Configuration monitorInterval="30s">
  <Properties>
    <Property name="LOG_DIR">logs</Property>
  </Properties>
  <Appenders>
    <Console name="console" target="SYSTEM_ERR">
      <PatternLayout pattern="%d %5p %pid --- [%thread] %c : %m%attrs%n"/>
    </Console>
    <RollingFile name="rolling" fileName="${LOG_DIR}/app.log"
                 filePattern="${LOG_DIR}/archive/app-%d{yyyyMMdd}-%i.log.gz">
      <JSONLayout eventEol="true"/>
      <Policies>
        <SizeBasedTriggeringPolicy size="100MiB"/>
        <TimeBasedTriggeringPolicy interval="daily" modulate="true"/>
        <OnStartupTriggeringPolicy enabled="true"/>
      </Policies>
      <DefaultRolloverStrategy max="30" fileIndex="nomax">
        <Delete basePath="${LOG_DIR}/archive" maxDepth="1">
          <IfFileName glob="*.log.gz"/>
          <IfLastModified age="30d"/>
        </Delete>
      </DefaultRolloverStrategy>
    </RollingFile>
  </Appenders>
  <Loggers>
    <Root level="info">
      <AppenderRef ref="console"/>
      <AppenderRef ref="rolling"/>
    </Root>
  </Loggers>
</Configuration>
```

Console target aliases accepted by XML are `SYSTEM_OUT`, `STDOUT`,
`SYSTEM_ERR`, and `STDERR`.

The XML parser has element slots for `<Http>`, `<Socket>`, and `<Syslog>` so
external plugin modules can reuse the config shape. The core registry does not
register HTTP, Socket, or Syslog network appenders.

## Reload

`LoggerContext` starts file polling when:

- configuration was loaded from an actual file,
- `monitorInterval` resolves to a positive duration,
- `NewConfiguredLoggerContext` is used.

Reload builds a complete new runtime snapshot first. If the new configuration
fails, the old runtime remains active.

Reload can change:

- levels,
- filters,
- appenders,
- layouts,
- logger routes,
- properties and lookups resolved in the new file.

Reload cannot change Handler-level async runtime settings. To change queue
shape or enable/disable async logging, restart the logger context.

Always call `Close` on `Handler` or `LoggerContext` so buffers and async queues
are drained.
