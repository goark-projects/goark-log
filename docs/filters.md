# Filter Reference

[简体中文](filters.zh-CN.md)

Filters decide whether a log event should continue through the pipeline. Every
filter must be concurrency-safe.

## Decisions

| Decision | Meaning |
| --- | --- |
| `neutral` | The filter has no final decision; the next filter or level gate decides. |
| `accept` | The filter explicitly accepts the event. A global `accept` can bypass the route level gate. |
| `deny` | The event is dropped immediately at the current stage. |

Decision parsing accepts empty string as `neutral`.

Most built-in filters default to:

- `onMatch: neutral`
- `onMismatch: deny`

Script filters created through the API additionally default `onError` to
`deny`. The core configuration does not register a script filter plugin or any
script engine.

## Evaluation Order

1. Handler receives a `slog.Record` or native logger event.
2. Global filters from top-level `filterRefs` are evaluated before level checks.
3. Route level is checked unless a global filter returned `accept`.
4. Logger/root route filters are evaluated.
5. Appender-ref controls are evaluated for each referenced appender.
6. Appender wrapper filters are evaluated.
7. The appender writes the event.

Any `deny` stops the event at that stage. `neutral` lets the event continue.

## Config Shape

```yaml
filters:
  onlyWarn:
    type: threshold
    level: warn
    onMatch: neutral
    onMismatch: deny

filterRefs: [onlyWarn]
```

Top-level `filterRefs` defines global filters. `root`, named loggers, appender
refs, and appenders can each define `filters`, `filterRefs`, or `filter-refs`.

## Built-In Filter Types

| Type | Aliases | Purpose |
| --- | --- | --- |
| `threshold` | `thresholdFilter` | Matches events at or above a level. |
| `level` | `levelFilter` | Matches exactly one level. |
| `levelRange` | `levelRangeFilter` | Matches a closed level range. |
| `regex` | `regexFilter` | Matches message, logger, or one attribute by regexp. |
| `attr` | `attribute`, `attrFilter`, `attributeFilter` | Matches one attribute key and value. |
| `deny` | `denyAll`, `denyFilter`, `denyAllFilter` | Always denies. |
| `composite` | `compositeFilter` | Runs a nested filter chain. |
| `marker` | `markerFilter` | Matches marker or parent marker name. |
| `noMarker` | `noMarkerFilter` | Matches events without marker. |
| `map` | `mapFilter` | Matches multiple event attributes. |
| `threadContextMap` | `threadContextMapFilter` | Alias of map filter for MDC use. |
| `threadContextStack` | `threadContextStackFilter` | Matches a context stack value. |
| `structuredData` | `structuredDataFilter` | Alias of map filter for structured data fields. |
| `throwable` | `throwableFilter` | Matches throwable text by regexp. |
| `stringMatch` | `stringMatchFilter` | Matches message substring. |
| `time` | `timeFilter` | Matches a time-of-day window. |
| `burst` | `burstFilter` | Token-bucket limiter for lower-level events. |
| `dynamicThreshold` | `dynamicThresholdFilter` | Selects a threshold from an event attribute value. |

Filter type names are normalized by lowercasing and removing `-` and `_`.

## Common Fields

| Field | Aliases | Description |
| --- | --- | --- |
| `type` | none | Required filter type. |
| `level` | none | Level used by threshold, level, burst, or dynamic threshold fallback. |
| `minLevel` | `min-level` | Lower bound for level range. |
| `maxLevel` | `max-level` | Upper bound for level range. |
| `marker` | none | Marker filter target. |
| `text` | none | String match or stack target. |
| `operator` | none | Map filter operator: `and` or `or`. |
| `start` | none | Time filter start. |
| `end` | none | Time filter end. |
| `timezone` | none | IANA timezone for time filter, for example `Asia/Shanghai`. |
| `rate` | none | Burst tokens per second. |
| `maxBurst` | `max-burst` | Burst token bucket capacity. |
| `field` | none | Regex field: `message`, `logger`, or `attr`. |
| `key` | none | Attribute key. |
| `value` | none | Attribute value. |
| `values` | none | Map of expected attribute key/value pairs. |
| `thresholds` | none | Dynamic-threshold map from attribute value to level. |
| `filters` | `filterRefs`, `filter-refs` | Nested filters for composite. |
| `defaultThreshold` | `default-threshold` | Dynamic threshold fallback. |
| `pattern` | none | Regex pattern for regex/throwable or fallback text for some filters. |
| `onMatch` | `on-match` | Decision when matched. |
| `onMismatch` | `on-mismatch` | Decision when not matched. |
| `KeyValuePair` | `keyValuePairs`, `key-value-pairs` | Log4j-style key/value entries for map-like filters. |

## ThresholdFilter

```yaml
filters:
  warnAndAbove:
    type: threshold
    level: warn
```

Required field: `level`.

Matches when `event.Level >= level`.

## LevelFilter

```yaml
filters:
  onlyError:
    type: level
    level: error
```

Required field: `level`.

Matches when `event.Level == level`.

## LevelRangeFilter

```yaml
filters:
  businessNoise:
    type: levelRange
    minLevel: info
    maxLevel: warn
```

Required fields: `minLevel`, `maxLevel`. `minLevel` must be less than or equal
to `maxLevel`.

## RegexFilter

```yaml
filters:
  routeByLogger:
    type: regex
    field: logger
    pattern: "^goark\\.orm(\\.|$)"
    onMatch: accept
    onMismatch: neutral

  rejectHealthChecks:
    type: regex
    field: attr
    key: path
    pattern: "^/healthz$"
    onMatch: deny
    onMismatch: neutral
```

| Field | Default | Description |
| --- | --- | --- |
| `pattern` | required | Go regular expression. |
| `field` | `message` | `message`, `msg`, `logger`, `name`, `attr`, or `attribute`. |
| `key` | required for `attr` | Attribute key to read. |

## AttrFilter

```yaml
filters:
  auditOnly:
    type: attr
    key: channel
    value: audit
```

Required fields: `key`, `value`.

Matches when the latest event attribute with `key` has the exact string value.

## DenyFilter

```yaml
filters:
  disabled:
    type: deny
```

Always returns `deny`.

## CompositeFilter

```yaml
filters:
  onlyProd:
    type: attr
    key: profile
    value: prod
    onMatch: neutral
    onMismatch: deny
  noHealthChecks:
    type: regex
    field: attr
    key: path
    pattern: "^/healthz$"
    onMatch: deny
    onMismatch: neutral
  prodTraffic:
    type: composite
    filterRefs: [onlyProd, noHealthChecks]
```

Required field: `filters`, `filterRefs`, or `filter-refs`.

Nested filter references are resolved by name. Cycles are rejected.

## MarkerFilter and NoMarkerFilter

```yaml
filters:
  securityMarker:
    type: marker
    marker: SECURITY
    onMatch: accept
    onMismatch: neutral
  unmarkedOnly:
    type: noMarker
```

Marker matching uses `Marker.Contains`, so a child marker can match a configured
parent marker.

Markers can be added through context or attributes:

```go
ctx := goarklog.WithMarker(context.Background(), goarklog.NewMarker("SECURITY"))
logger.InfoContext(ctx, "login failed")
```

## MapFilter

```yaml
filters:
  tenantAudit:
    type: map
    operator: and
    values:
      tenant: acme
      channel: audit
    onMatch: accept
    onMismatch: neutral
```

| Field | Default | Description |
| --- | --- | --- |
| `values` | empty | Map of expected event attributes. |
| `KeyValuePair` | empty | Alternative key/value pair list. |
| `key`, `value` | empty | Single pair shortcut. |
| `operator` | `and` | `and` requires every pair. `or` requires at least one pair. |

At least one value is required.

`threadContextMap` and `structuredData` are aliases backed by the same map
filter behavior.

## ThreadContextStackFilter

```yaml
filters:
  paymentScope:
    type: threadContextStack
    value: payment
```

The target value can be configured by `value`, `text`, or `pattern`.

Context stack values can be added with:

```go
ctx := goarklog.WithContextStack(context.Background(), "payment", "checkout")
```

## ThrowableFilter

```yaml
filters:
  networkError:
    type: throwable
    pattern: "(?i)timeout|connection reset"
```

The pattern is matched against:

1. `event.Throwable`, when present.
2. A throwable extracted from `goark.throwable`, `throwable`, `error`, or `err`.
3. A plain `error` or `err` attribute string.

## StringMatchFilter

```yaml
filters:
  skipHealthText:
    type: stringMatch
    text: health check
    onMatch: deny
    onMismatch: neutral
```

The target text can be configured by `text`, `value`, or `pattern`. Matching is
case-sensitive substring matching against `event.Message`.

## TimeFilter

```yaml
filters:
  officeHours:
    type: time
    start: "09:00"
    end: "18:00:00"
    timezone: Asia/Shanghai
    onMatch: neutral
    onMismatch: deny
```

Supported time-of-day formats:

- `15:04`
- `15:04:05`
- `15:04:05.999999999`

Windows that cross midnight are supported. For example `start: "22:00"` and
`end: "06:00"` matches events from 22:00 to midnight and from midnight to
06:00.

## BurstFilter

```yaml
filters:
  noisyWarnings:
    type: burst
    level: warn
    rate: "10"
    maxBurst: 100
    onMatch: neutral
    onMismatch: deny
```

| Field | Default | Description |
| --- | --- | --- |
| `level` | `warn` | Events at or below this level consume tokens. Higher levels are always neutral. |
| `rate` | `10` | Tokens per second. Must be greater than zero. |
| `maxBurst` | `rate * 10`, minimum `1` | Bucket capacity. Must be greater than zero when explicitly set. |

Use `burst` as an appender or route filter when you want to protect a slow sink
from repeated low-value messages.

## DynamicThresholdFilter

```yaml
filters:
  tenantThreshold:
    type: dynamicThreshold
    key: tenant
    defaultThreshold: error
    thresholds:
      acme: debug
      globex: warn
    onMatch: neutral
    onMismatch: deny
```

Required field: `key`.

For each event, the filter reads attribute `key`. If the value exists in
`thresholds`, that level is used; otherwise `defaultThreshold` is used. If
`defaultThreshold` is omitted, `level` is used; if both are omitted, the fallback
is `error`.

## Properties Examples

```properties
filter.warn.type=threshold
filter.warn.level=warn
filter.warn.onMismatch=deny

filter.audit.type=map
filter.audit.operator=and
filter.audit.values.channel=audit
filter.audit.values.tenant=acme
filter.audit.onMatch=accept
filter.audit.onMismatch=neutral

filter.dynamic.type=dynamicThreshold
filter.dynamic.key=tenant
filter.dynamic.defaultThreshold=error
filter.dynamic.thresholds.acme=debug
filter.dynamic.thresholds.globex=warn
```

For Log4j-style key/value pairs in properties:

```properties
filter.audit.type=map
filter.audit.keyValuePair1.key=channel
filter.audit.keyValuePair1.value=audit
filter.audit.keyValuePair2.key=tenant
filter.audit.keyValuePair2.value=acme
```

## XML Examples

```xml
<Filters>
  <ThresholdFilter name="warn" level="warn" onMismatch="deny"/>
  <MapFilter name="audit" operator="and" onMatch="accept" onMismatch="neutral">
    <KeyValuePair key="channel" value="audit"/>
    <KeyValuePair key="tenant" value="acme"/>
  </MapFilter>
  <DynamicThresholdFilter name="tenantThreshold" key="tenant" defaultThreshold="error">
    <KeyValuePair key="acme" value="debug"/>
    <KeyValuePair key="globex" value="warn"/>
  </DynamicThresholdFilter>
</Filters>
```

## Operational Guidance

- Use level gates for cheap coarse filtering.
- Use global filters only when an `accept` decision must bypass normal levels or
  a `deny` decision must apply to every route.
- Use appender-ref filters for sink-specific behavior, such as sending only
  `WARN+` to rolling files while keeping console at `INFO`.
- Use appender wrapper filters when the appender should carry the filter no
  matter which logger references it.
- Avoid expensive regex filters on the highest-volume path unless a cheaper
  attribute or level filter can reduce the candidate set first.
