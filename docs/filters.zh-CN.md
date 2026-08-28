# Filter 参考

[English](filters.md)

Filters 决定 log event 是否继续通过 pipeline。每个 filter 都必须 concurrency-safe。

## Decisions

| Decision | 含义 |
| --- | --- |
| `neutral` | filter 没有最终决策，由下一个 filter 或 level gate 决定。 |
| `accept` | filter 明确接受 event。Global `accept` 可以绕过 route level gate。 |
| `deny` | event 在当前阶段立即丢弃。 |

Decision parsing 接受空字符串作为 `neutral`。

大多数内置 filters 默认：

- `onMatch: neutral`
- `onMismatch: deny`

通过 API 创建的 script filters 额外默认 `onError` 为 `deny`。Core configuration 不注册 script filter plugin，也不内置任何 script engine。

## 执行顺序

1. Handler 接收 `slog.Record` 或 native logger event。
2. Top-level `filterRefs` 中的 global filters 在 level checks 前执行。
3. 除非 global filter 返回 `accept`，否则检查 route level。
4. 执行 logger/root route filters。
5. 对每个 referenced appender 执行 appender-ref controls。
6. 执行 appender wrapper filters。
7. Appender 写入 event。

任意 `deny` 都会在当前阶段停止 event。`neutral` 让 event 继续。

## 配置形状

```yaml
filters:
  onlyWarn:
    type: threshold
    level: warn
    onMatch: neutral
    onMismatch: deny

filterRefs: [onlyWarn]
```

Top-level `filterRefs` 定义 global filters。`root`、named loggers、appender refs 和 appenders 都可以定义 `filters`、`filterRefs` 或 `filter-refs`。

## 内置 Filter Types

| Type | Aliases | 目的 |
| --- | --- | --- |
| `threshold` | `thresholdFilter` | 匹配大于等于某 level 的 events。 |
| `level` | `levelFilter` | 精确匹配一个 level。 |
| `levelRange` | `levelRangeFilter` | 匹配闭区间 level range。 |
| `regex` | `regexFilter` | 按 regexp 匹配 message、logger 或一个 attribute。 |
| `attr` | `attribute`, `attrFilter`, `attributeFilter` | 匹配一个 attribute key 和 value。 |
| `deny` | `denyAll`, `denyFilter`, `denyAllFilter` | 永远 deny。 |
| `composite` | `compositeFilter` | 执行嵌套 filter chain。 |
| `marker` | `markerFilter` | 匹配 marker 或 parent marker name。 |
| `noMarker` | `noMarkerFilter` | 匹配没有 marker 的 events。 |
| `map` | `mapFilter` | 匹配多个 event attributes。 |
| `threadContextMap` | `threadContextMapFilter` | MDC 使用的 map filter alias。 |
| `threadContextStack` | `threadContextStackFilter` | 匹配 context stack value。 |
| `structuredData` | `structuredDataFilter` | structured data fields 使用的 map filter alias。 |
| `throwable` | `throwableFilter` | 用 regexp 匹配 throwable text。 |
| `stringMatch` | `stringMatchFilter` | 匹配 message substring。 |
| `time` | `timeFilter` | 匹配一天内的时间窗口。 |
| `burst` | `burstFilter` | 用 token-bucket 限制低级别 events。 |
| `dynamicThreshold` | `dynamicThresholdFilter` | 从 event attribute value 选择 threshold。 |

Filter type names 会 lowercase 并移除 `-` 和 `_`。

## 通用字段

| 字段 | Aliases | 说明 |
| --- | --- | --- |
| `type` | none | 必填 filter type。 |
| `level` | none | threshold、level、burst 或 dynamic threshold fallback 使用的 level。 |
| `minLevel` | `min-level` | Level range 下界。 |
| `maxLevel` | `max-level` | Level range 上界。 |
| `marker` | none | Marker filter target。 |
| `text` | none | String match 或 stack target。 |
| `operator` | none | Map filter operator：`and` 或 `or`。 |
| `start` | none | Time filter start。 |
| `end` | none | Time filter end。 |
| `timezone` | none | IANA timezone，例如 `Asia/Shanghai`。 |
| `rate` | none | Burst tokens per second。 |
| `maxBurst` | `max-burst` | Burst token bucket capacity。 |
| `field` | none | Regex field：`message`、`logger` 或 `attr`。 |
| `key` | none | Attribute key。 |
| `value` | none | Attribute value。 |
| `values` | none | Expected attribute key/value pairs。 |
| `thresholds` | none | Dynamic-threshold map，attribute value 到 level。 |
| `filters` | `filterRefs`, `filter-refs` | Composite 的 nested filters。 |
| `defaultThreshold` | `default-threshold` | Dynamic threshold fallback。 |
| `pattern` | none | regex/throwable pattern，或部分 filters 的 fallback text。 |
| `onMatch` | `on-match` | 匹配时 decision。 |
| `onMismatch` | `on-mismatch` | 不匹配时 decision。 |
| `KeyValuePair` | `keyValuePairs`, `key-value-pairs` | Log4j-style key/value entries，用于 map-like filters。 |

## ThresholdFilter

```yaml
filters:
  warnAndAbove:
    type: threshold
    level: warn
```

必填字段：`level`。

当 `event.Level >= level` 时匹配。

## LevelFilter

```yaml
filters:
  onlyError:
    type: level
    level: error
```

必填字段：`level`。

当 `event.Level == level` 时匹配。

## LevelRangeFilter

```yaml
filters:
  businessNoise:
    type: levelRange
    minLevel: info
    maxLevel: warn
```

必填字段：`minLevel`、`maxLevel`。`minLevel` 必须小于等于 `maxLevel`。

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

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `pattern` | required | Go regular expression。 |
| `field` | `message` | `message`、`msg`、`logger`、`name`、`attr` 或 `attribute`。 |
| `key` | attr field 时 required | 要读取的 attribute key。 |

## AttrFilter

```yaml
filters:
  auditOnly:
    type: attr
    key: channel
    value: audit
```

必填字段：`key`、`value`。

当最新 event attribute 中 `key` 的 string value 精确等于配置值时匹配。

## DenyFilter

```yaml
filters:
  disabled:
    type: deny
```

始终返回 `deny`。

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

必填字段：`filters`、`filterRefs` 或 `filter-refs`。

Nested filter references 按名称解析。Cycles 会被拒绝。

## MarkerFilter 和 NoMarkerFilter

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

Marker matching 使用 `Marker.Contains`，因此 child marker 可以匹配配置的 parent marker。

Marker 可以通过 context 或 attributes 添加：

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

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `values` | empty | Expected event attributes map。 |
| `KeyValuePair` | empty | 可替代 map 的 key/value pair list。 |
| `key`, `value` | empty | 单个 pair shortcut。 |
| `operator` | `and` | `and` 要求所有 pair 匹配；`or` 要求至少一个 pair 匹配。 |

至少需要一个 value。

`threadContextMap` 和 `structuredData` 是由同一个 map filter 行为支撑的 aliases。

## ThreadContextStackFilter

```yaml
filters:
  paymentScope:
    type: threadContextStack
    value: payment
```

目标值可以通过 `value`、`text` 或 `pattern` 配置。

Context stack values 可以这样添加：

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

Pattern 会匹配：

1. `event.Throwable`，如果存在；
2. 从 `goark.throwable`、`throwable`、`error` 或 `err` 提取的 throwable；
3. 普通 `error` 或 `err` attribute string。

## StringMatchFilter

```yaml
filters:
  skipHealthText:
    type: stringMatch
    text: health check
    onMatch: deny
    onMismatch: neutral
```

目标 text 可以通过 `text`、`value` 或 `pattern` 配置。匹配方式是对 `event.Message` 做 case-sensitive substring matching。

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

支持的 time-of-day formats：

- `15:04`
- `15:04:05`
- `15:04:05.999999999`

支持跨午夜窗口。例如 `start: "22:00"` 且 `end: "06:00"` 会匹配 22:00 到午夜，以及午夜到 06:00。

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

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `level` | `warn` | 小于等于该 level 的 events 消耗 tokens。更高级别始终 neutral。 |
| `rate` | `10` | Tokens per second。必须大于 0。 |
| `maxBurst` | `rate * 10`, minimum `1` | Bucket capacity。显式设置时必须大于 0。 |

当需要保护慢 sink 不被重复低价值消息打满时，可以把 `burst` 用作 appender 或 route filter。

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

必填字段：`key`。

每个 event 中，filter 读取 attribute `key`。如果值存在于 `thresholds`，使用对应 level；否则使用 `defaultThreshold`。如果省略 `defaultThreshold`，则使用 `level`；二者都省略时 fallback 为 `error`。

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

Properties 中使用 Log4j-style key/value pairs：

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

## 运行建议

- 使用 level gates 做便宜的粗粒度过滤。
- 只有当 `accept` decision 必须绕过 normal levels，或 `deny` decision 必须应用到每条 route 时，才使用 global filters。
- 使用 appender-ref filters 处理 sink-specific 行为，例如 rolling file 只接收 `WARN+`，console 仍保留 `INFO`。
- 当 appender 无论被哪个 logger 引用都应带同一 filter 时，使用 appender wrapper filters。
- 避免在最高吞吐路径直接使用昂贵 regex filters，除非前面已经用更便宜的 attribute 或 level filter 缩小候选集。
