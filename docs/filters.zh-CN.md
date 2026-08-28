# Filters

[English](filters.md)

Filter 决定事件是否继续通过日志管线。它对齐 Log4j2 的 `ACCEPT`、`DENY`、
`NEUTRAL` 模型，同时保持明确的 Go 接口。

## 裁决

| 裁决 | 影响 |
| --- | --- |
| `neutral` | 继续交给下一个 filter 或正常级别路由判断。 |
| `accept` | 在全局 filter 中绕过 logger 级别阈值；在 route/appender filter 中结束过滤并允许事件。 |
| `deny` | 在当前过滤链中立即丢弃事件。 |

大多数配置 filter 默认 `onMatch: neutral`、`onMismatch: deny`。当 filter 只应该拒绝或只应该
接受一部分事件，而不能阻断其它事件时，应显式设置 `onMismatch: neutral`。

## 放置位置

| 位置 | 配置字段 | 执行点 |
| --- | --- | --- |
| 全局 | 顶层 `filterRefs` | logger 级别判断之前。 |
| Root logger | `root.filterRefs` | 路由选择之后。加性的命名 logger 会继承 root filter。 |
| 命名 logger | `loggers.NAME.filterRefs` | 命中 logger 名称后执行。 |
| Appender | `appenders.NAME.filterRefs` | 包裹该 appender。 |
| Appender ref | `appenderRefs[].filterRefs` | 只作用于该次 appender 引用。 |
| Composite filter | `filters.NAME.filterRefs` | 嵌套过滤链，循环引用会被拒绝。 |

过滤链按顺序执行。第一个 `DENY` 或 `ACCEPT` 会终止链。

## 内置 Filter

| 类型 | 别名 | 必填字段 | 匹配条件 |
| --- | --- | --- | --- |
| `threshold` | `thresholdFilter` | `level` | 事件级别大于等于 level。 |
| `level` | `levelFilter` | `level` | 事件级别等于 level。 |
| `levelRange` | `levelRangeFilter` | `minLevel`, `maxLevel` | 事件级别在闭区间内。 |
| `regex` | `regexFilter` | `pattern`；`field=attr` 时需要 `key` | 对 message、logger 或 attr 做正则匹配。 |
| `attr` | `attribute`, `attrFilter`, `attributeFilter` | `key` | 属性存在且字符串值等于 `value`。 |
| `deny` | `denyAll`, `denyFilter`, `denyAllFilter` | 无 | 永远拒绝。 |
| `composite` | `compositeFilter` | `filterRefs` | 运行嵌套 filter。 |
| `marker` | `markerFilter` | `marker` 或 `value` | marker 或父 marker 匹配。 |
| `noMarker` | `noMarkerFilter` | 无 | 事件没有 marker。 |
| `map` | `mapFilter` | values | 事件属性匹配全部或任意键值。 |
| `threadContextMap` | `threadContextMapFilter` | values | MDC 风格 map filter 别名。 |
| `threadContextStack` | `threadContextStackFilter` | `value`、`text` 或 `pattern` | Context stack 包含目标值。 |
| `structuredData` | `structuredDataFilter` | values | 面向 structured data attrs 的 map filter 别名。 |
| `throwable` | `throwableFilter` | `pattern`、`text` 或 `value` | 对 throwable 文本或 error attrs 做正则。 |
| `stringMatch` | `stringMatchFilter` | `text`、`value` 或 `pattern` | message 包含文本。 |
| `time` | `timeFilter` | 可选 | 事件一天内时间落在区间内。 |
| `burst` | `burstFilter` | 可选 | 对小于等于指定级别的事件做令牌桶限流。 |
| `dynamicThreshold` | `dynamicThresholdFilter` | `key` | 按事件属性选择级别阈值。 |

`ScriptFilter` 只存在于 Go API。核心模块不内置脚本运行时，也不提供配置式脚本语言。

## 通用字段

| 字段 | 说明 |
| --- | --- |
| `type` | 配置 filter 必填。类型匹配忽略大小写、连字符和下划线。 |
| `onMatch`, `on-match` | `neutral`、`accept` 或 `deny`，默认 `neutral`。 |
| `onMismatch`, `on-mismatch` | `neutral`、`accept` 或 `deny`，默认 `deny`。 |
| `filters`, `filterRefs`, `filter-refs` | 嵌套引用，主要用于 composite filter。 |

## 级别 Filter

```yaml
filters:
  warnings:
    type: threshold
    level: warn
```

`levelRange` 会拒绝 `minLevel > maxLevel` 的配置。级别名称和 logger 级别使用同一个解析器：
`ALL`、`TRACE`、`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`、`OFF`、`WARNING`
或整数。

## Regex 和 String Filter

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

Regex field 可取 `message`、`msg`、`logger`、`name`、`attr`、`attribute`。
`field: attr` 必须设置 `key`。

## Attr 和 Map Filter

```yaml
filters:
  tenantA:
    type: map
    operator: and
    values:
      tenant: tenant-a
      region: cn-east
```

`operator` 默认是 `and`；`or` 表示任意键值匹配即可。值可以来自单个 `key` 和 `value`、
`KeyValuePair`、`keyValuePairs` 或 `key-value-pairs`。

`threadContextMap` 和 `structuredData` 使用相同匹配语义。

## Marker 和 Context Stack

```yaml
filters:
  audit:
    type: marker
    marker: AUDIT
  requestStack:
    type: threadContextStack
    value: request
```

Marker 可在 Go 代码中带父级。Marker filter 会匹配自身或任意父 marker。

## Time Filter

```yaml
filters:
  businessHours:
    type: time
    start: "09:00"
    end: "18:00"
    timezone: Asia/Shanghai
```

支持的 time-of-day 格式是 `HH:MM`、`HH:MM:SS` 和 `HH:MM:SS.NNNNNNNNN`。
当 start 晚于 end 时，区间跨午夜。默认全日：`00:00:00` 到
`23:59:59.999999999`。

## Burst Filter

```yaml
filters:
  debugBurst:
    type: burst
    level: debug
    rate: "10"
    maxBurst: 100
```

Burst 只作用于小于等于 `level` 的事件；更高级别保持 neutral。默认 level 是 `warn`，
默认 rate 是 `10`，默认 max burst 是 `rate * 10`，最小为 1。

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

事件属性值选择阈值。没有匹配值时使用默认阈值。`defaultThreshold` 会回退到 `level`，
再回退到 `error`。

## Throwable Filter

Throwable filter 检查 `goark.throwable`、`throwable`、`error` 或 `err` 属性，以及事件异常
快照。

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

Composite filter 至少需要一个嵌套 filter。循环引用会导致配置加载失败。
