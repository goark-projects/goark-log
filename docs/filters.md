# Filters

[简体中文](filters.zh-CN.md)

Filters decide whether an event should continue through the logging pipeline.
They are designed to match Log4j2's `ACCEPT`, `DENY`, and `NEUTRAL` model while
remaining explicit Go interfaces.

## Decisions

| Decision | Effect |
| --- | --- |
| `neutral` | The next filter or normal level routing decides. |
| `accept` | In global filters, bypasses the logger level threshold. In route/appender filters, stops filter evaluation and allows the event. |
| `deny` | Drops the event immediately for that filter chain. |

Most configured filters default to `onMatch: neutral` and
`onMismatch: deny`. Set `onMismatch: neutral` when a filter should only deny or
only accept a subset without blocking other events.

## Placement

| Placement | Config field | Execution point |
| --- | --- | --- |
| Global | top-level `filterRefs` | Before logger level checks. |
| Root logger | `root.filterRefs` | After route selection. Additive named loggers inherit root filters. |
| Named logger | `loggers.NAME.filterRefs` | After route selection for matching logger names. |
| Appender | `appenders.NAME.filterRefs` | Wraps that appender. |
| Appender ref | `appenderRefs[].filterRefs` | Applies only to one reference to an appender. |
| Composite filter | `filters.NAME.filterRefs` | Nested filter chain. Cycles are rejected. |

Filter chains run in order. The first `DENY` or `ACCEPT` terminates the chain.

## Built-In Filters

| Type | Aliases | Required fields | Match |
| --- | --- | --- | --- |
| `threshold` | `thresholdFilter` | `level` | Event level is at or above level. |
| `level` | `levelFilter` | `level` | Event level equals level. |
| `levelRange` | `levelRangeFilter` | `minLevel`, `maxLevel` | Event level is in the inclusive range. |
| `regex` | `regexFilter` | `pattern`; `key` when `field=attr` | Regex over message, logger, or attr. |
| `attr` | `attribute`, `attrFilter`, `attributeFilter` | `key` | Attribute exists and string value equals `value`. |
| `deny` | `denyAll`, `denyFilter`, `denyAllFilter` | none | Always denies. |
| `composite` | `compositeFilter` | `filterRefs` | Runs nested filters. |
| `marker` | `markerFilter` | `marker` or `value` | Marker name or parent marker matches. |
| `noMarker` | `noMarkerFilter` | none | Event has no marker. |
| `map` | `mapFilter` | values | Event attrs match all or any configured key/value pairs. |
| `threadContextMap` | `threadContextMapFilter` | values | MDC-style alias of map filter. |
| `threadContextStack` | `threadContextStackFilter` | `value`, `text`, or `pattern` | Context stack contains the value. |
| `structuredData` | `structuredDataFilter` | values | Alias of map filter for structured data attrs. |
| `throwable` | `throwableFilter` | `pattern`, `text`, or `value` | Regex over throwable text or error attrs. |
| `stringMatch` | `stringMatchFilter` | `text`, `value`, or `pattern` | Message contains text. |
| `time` | `timeFilter` | optional | Event time-of-day is in range. |
| `burst` | `burstFilter` | optional | Token bucket for events at or below level. |
| `dynamicThreshold` | `dynamicThresholdFilter` | `key` | Threshold selected by an event attribute. |

`ScriptFilter` exists in the Go API only. The core module does not ship an
embedded script runtime or configured script language.

## Common Fields

| Field | Notes |
| --- | --- |
| `type` | Required for configured filters. Kind matching ignores case, hyphen, and underscore. |
| `onMatch`, `on-match` | `neutral`, `accept`, or `deny`; default `neutral`. |
| `onMismatch`, `on-mismatch` | `neutral`, `accept`, or `deny`; default `deny`. |
| `filters`, `filterRefs`, `filter-refs` | Nested refs, mainly for composite filters. |

## Level Filters

```yaml
filters:
  warnings:
    type: threshold
    level: warn
```

`levelRange` rejects configs where `minLevel > maxLevel`. Level names use the
same parser as logger levels: `ALL`, `TRACE`, `DEBUG`, `INFO`, `WARN`,
`ERROR`, `FATAL`, `OFF`, `WARNING`, or integers.

## Regex And String Filters

```yaml
filters:
  dropHealth:
    type: stringMatch
    text: "/health"
    onMatch: deny
    onMismatch: neutral
  tenantLogger:
    type: regex
    field: logger
    pattern: "^goark\\.tenant\\."
```

Regex field values are `message`, `msg`, `logger`, `name`, `attr`, and
`attribute`. `field: attr` requires `key`.

## Attribute And Map Filters

```yaml
filters:
  tenantA:
    type: map
    operator: and
    values:
      tenant: tenant-a
      region: cn-east
```

`operator` is `and` by default; `or` accepts any matching pair. Values may also
be supplied with a single `key` and `value`, with `KeyValuePair`, with
`keyValuePairs`, or with `key-value-pairs`.

`threadContextMap` and `structuredData` use the same matching behavior.

## Marker And Context Stack

```yaml
filters:
  audit:
    type: marker
    marker: AUDIT
  requestStack:
    type: threadContextStack
    value: request
```

Markers can have parents in Go. A marker filter matches the marker itself or
any parent marker.

## Time Filter

```yaml
filters:
  businessHours:
    type: time
    start: "09:00"
    end: "18:00"
    timezone: Asia/Shanghai
```

Accepted time-of-day formats are `HH:MM`, `HH:MM:SS`, and
`HH:MM:SS.NNNNNNNNN`. When start is later than end, the interval wraps across
midnight. Defaults are full day: `00:00:00` to `23:59:59.999999999`.

## Burst Filter

```yaml
filters:
  debugBurst:
    type: burst
    level: debug
    rate: "10"
    maxBurst: 100
```

Burst applies only to events at or below `level`; higher levels remain neutral.
Default level is `warn`, default rate is `10`, and default max burst is
`rate * 10` with a minimum of 1.

## Dynamic Threshold

```yaml
filters:
  tenantThreshold:
    type: dynamicThreshold
    key: tenant
    defaultThreshold: error
    thresholds:
      tenant-a: debug
      tenant-b: info
```

The event attribute value selects a threshold. If no value matches, the default
threshold is used. `defaultThreshold` falls back to `level`, then `error`.

## Throwable Filter

Throwable filter checks `goark.throwable`, `throwable`, `error`, or `err`
attributes and the event throwable snapshot.

```yaml
filters:
  networkErrors:
    type: throwable
    pattern: "timeout|refused"
```

## Composite Filter

```yaml
filters:
  auditTenantA:
    type: composite
    filterRefs: [auditMarker, tenantA]
```

Composite filters require at least one nested filter. Cyclic references fail
configuration loading.
