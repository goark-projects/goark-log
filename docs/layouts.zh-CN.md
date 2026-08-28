# Layout 参考

[English](layouts.md)

Layout 将 immutable log event snapshot 转为 bytes。Console、file 和 rolling file appenders 使用 layouts。Direct JSON appender 使用自己的固定 JSON encoder，不使用配置中的 layout。

## 内置 Layouts

| Type | Aliases | 输出 |
| --- | --- | --- |
| `pattern` | none | Log4j/Spring Boot style pattern text。 |
| `text` | none | 稳定 key-value text。 |
| `json` | none | Structured JSON event。 |
| `jsonTemplate` | `json-template` after normalization | 基于 event template 生成的 JSON event。 |
| `xml` | `xmlLayout` | 单个 XML event。 |
| `csv` | `csvLayout` | 单行 CSV。 |
| `gelf` | `gelfLayout` | Graylog Extended Log Format JSON。 |
| `rfc5424` | `rfc5424Layout` | RFC 5424 syslog text event。 |
| `syslog` | `syslogLayout` | RFC 5424 layout 的 alias。 |
| `yaml` | `yamlLayout` | YAML event document。 |
| `html` | `htmlLayout` | HTML table row。 |

Layout 和 plugin kinds 会先 trim spaces、lowercase，再移除 `-` 和 `_`。

## 通用 Layout 字段

```yaml
layout:
  type: json
  compact: false
  eventEol: true
  complete: false
  includeStacktrace: true
  stacktraceAsString: false
  propertiesAsList: false
  includeNullDelimiter: false
  disableAnsi: false
  header: ""
  footer: ""
```

| 字段 | Aliases | 默认值 | 说明 |
| --- | --- | --- | --- |
| `type` | none | appender default pattern | Layout kind。 |
| `pattern` | none | default Spring Boot pattern | PatternLayout format string。 |
| `eventTemplate` | `event-template` | default template | Inline JSON Template event template。 |
| `eventTemplateUri` | `event-template-uri`, `eventTemplatePath`, `event-template-path` | empty | Local JSON Template file。 |
| `compact` | none | false | 禁用默认 event newline。 |
| `eventEol` | `event-eol` | false | 即使 `compact` 为 true，也添加 event newline。 |
| `complete` | none | false | 启用 lifecycle header/footer output。JSON layouts 在 complete 模式默认使用 array header/footer。 |
| `includeStacktrace` | `include-stacktrace` | false | 在支持的 layout 中输出 structured stack information。 |
| `stacktraceAsString` | `stacktrace-as-string` | false | 将 stacktrace 输出为一个 string，而不是 structured object/list。 |
| `propertiesAsList` | `properties-as-list` | false | 在支持的 layout 中把 context attributes 输出为 `[{"key":...,"value":...}]`。 |
| `includeNullDelimiter` | `include-null-delimiter` | false | 每个 event 后追加 NUL。适合需要 frame delimiter 的协议。 |
| `disableAnsi` | `disable-ansi` | false | 禁用 PatternLayout `%highlight` 和 `%style` ANSI output。 |
| `header` | none | empty | `complete` 为 true 时使用的自定义 lifecycle header。 |
| `footer` | none | empty | `complete` 为 true 时使用的自定义 lifecycle footer。 |

## PatternLayout

默认 pattern：

```text
%d %5level %pid --- [%thread] %logger : %msg%attrs%n
```

示例：

```yaml
layout:
  type: pattern
  pattern: "%d{yyyy-MM-dd HH:mm:ss.SSS} %-5p [%thread] %c{2} trace=%X{trace_id} %m%notEmpty{ %ex{short}}%n"
```

### Width Modifiers

Pattern converters 支持：

| 形式 | 含义 |
| --- | --- |
| `%5p` | 最小宽度 5，右对齐。 |
| `%-5p` | 最小宽度 5，左对齐。 |
| `%.40logger` | 最大宽度 40。 |
| `%20.40logger` | 最小宽度 20，最大宽度 40。 |

### Converters

| Converter | Aliases | 说明 |
| --- | --- | --- |
| `%d{format}` | `%date{format}` | Event time。Empty/default 使用 `2006-01-02T15:04:05.000Z07:00`。 |
| `%level` | `%p` | Level registry 中的 level name。 |
| `%pid` | `%processId` | 当前 process ID。 |
| `%thread` | `%t` | context/attrs 中的 logical thread name，默认 `main`。 |
| `%logger{precision}` | `%c{precision}` | Logger name。Precision 保留最后 N 个 dot-separated components。 |
| `%msg` | `%message`, `%m` | Event message。 |
| `%attrs` | `%kvp`, `%map` | Event attributes，key-value text。 |
| `%X{key}` | `%mdc{key}` | 按 key 输出 attribute value。空 `%X` 或 `%mdc` 输出所有 attrs。 |
| `%ex{option}` | `%throwable`, `%exception` | Throwable text。Options：empty、`short`、`full`、`none`。 |
| `%marker` | none | Marker value。 |
| `%ndc` | `%x` | Context stack values。 |
| `%C` | `%class` | Caller class/function owner。要求 caller PC。 |
| `%M` | `%method` | Caller method/function。要求 caller PC。大小写敏感：`%M` 是 method，`%m` 是 message。 |
| `%F` | `%file` | Caller file。要求 caller PC。 |
| `%L` | `%line` | Caller line。要求 caller PC。 |
| `%l` | `%location` | Caller location string。要求 caller PC。 |
| `%n` | none | Newline。 |
| `%uuid` | none | 每次 event render 生成 random UUID v4 style value。 |
| `%relative` | `%r` | 自 layout package initialization 起的毫秒数。 |
| `%host` | `%hostname` | 进程启动时解析的 host name。 |
| `%sequenceNumber` | `%sn` | Atomic sequence number。 |
| `%highlight{pattern}` | none | 按 level 应用默认 ANSI color。可由 `disableAnsi` 禁用。 |
| `%style{pattern}{style}` | none | 应用指定 ANSI style。可由 `disableAnsi` 禁用。 |
| `%notEmpty{pattern}` | none | Nested output trim 后非空时才输出。 |
| `%replace{pattern}{regex}{replacement}` | none | 对 nested output 做 regex replacement。 |
| `%enc{pattern}{mode}` | `%encode` | 转义 nested output。Modes：`json`、`html`、`xml`、`crlf`；未知 mode 保持原值。 |
| `%equals{pattern}{test}{substitution}` | none | nested output 等于 `test` 时替换。 |
| `%equalsIgnoreCase{pattern}{test}{substitution}` | none | `%equals` 的 case-insensitive 版本。 |
| `%maxLen{pattern}{length}` | `%maxLength` | 按 display width 截断 nested output。 |
| `%repeat{pattern}{count}` | none | 重复 nested output。 |
| `%%` | none | Literal percent sign。 |

Caller converters 读取 `slog.Record.PC`。除非满足以下任一条件，否则为空：

- Handler/root/logger/appender-ref `includeLocation` 为 true。
- Handler-level async `includeLocation` 为 true。
- Native logger 通过 `WithLoggerCaller(true)` 创建。

Caller capture 有可测量成本，只应在需要 caller converters 或 JSON Template source resolver 的 logger/appender ref 上启用。

### ANSI Styles

`%highlight` 默认 level styles：

| Level | Style |
| --- | --- |
| `FATAL` and above | `red,bold` |
| `ERROR` | `red` |
| `WARN` | `yellow` |
| `INFO` | `green` |
| `DEBUG` | `cyan` |
| lower | `faint` |

`%style` 支持 `bold`、`faint`、`underline`、`blink`、`reverse`、foreground colors（`red`、`green`、`yellow`、`blue`、`magenta`、`cyan`、`white`、`gray`）、bright colors，以及 `bgRed` 或 `backgroundRed` 这类 background forms。

## JSONLayout

```yaml
layout:
  type: json
  eventEol: true
  propertiesAsList: false
  includeStacktrace: true
```

默认字段：

| 字段 | 说明 |
| --- | --- |
| `time` | 使用 default layout format 的 event time。 |
| `level` | Level name。 |
| `logger` | Logger name。 |
| `msg` | Message。 |
| event attributes | `propertiesAsList` 为 false 时作为 top-level fields 输出。 |
| `contextMap` | `propertiesAsList` 为 true 时输出 attribute list。 |
| `thrown` | 启用 stacktrace output 且存在 throwable 时输出 throwable object 或 string。 |

常见 `slog.Value` kinds 由手写编码处理：string、bool、int、uint、float、duration、time、groups 和 errors/stringers。复杂 `Any` values 使用内部 JSON fallback codec，marshal error 时 fallback 到 `fmt.Sprint`。Fallback codec 会在受支持的 Go/architecture 组合上使用 Sonic；Go 1.27+ 或不受 Sonic 支持的架构上使用标准库。

`complete: true` 时，JSONLayout 写 JSON array stream。默认 header 为 `[`，默认 footer 为 `]`，events 之间自动插入 commas。

## JSONTemplateLayout

默认 event template：

```json
{
  "timestamp": {"$resolver": "timestamp"},
  "level": {"$resolver": "level"},
  "loggerName": {"$resolver": "logger"},
  "message": {"$resolver": "message"},
  "thread": {"$resolver": "thread"},
  "marker": {"$resolver": "marker"},
  "thrown": {"$resolver": "throwable"},
  "contextStack": {"$resolver": "contextStack"},
  "endOfBatch": {"$resolver": "endOfBatch"},
  "contextMap": {"$resolver": "mdc"}
}
```

Inline template 示例：

```yaml
layout:
  type: jsonTemplate
  eventTemplate: >
    {
      "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
      "lvl": {"$resolver": "level"},
      "logger": {"$resolver": "logger", "precision": 2},
      "msg": {"$resolver": "message"},
      "traceId": {"$resolver": "attr", "key": "trace_id"},
      "attrs": {"$resolver": "mdc", "flatten": true}
    }
  eventEol: true
```

Template file 示例：

```yaml
layout:
  type: jsonTemplate
  eventTemplateUri: conf/log-event-template.json
  stacktraceAsString: true
```

没有 `$resolver` 的 JSON field value 会作为 raw JSON 输出。

### JSON Template Resolvers

| Resolver | Aliases | Options | 输出 |
| --- | --- | --- | --- |
| `timestamp` | `time` | `format` | 使用 time-pattern mapper 的 event time。 |
| `level` | none | `field` | 默认输出 text level。`int`、`integer` 或 `value` 输出 slog numeric level。`severity` 或 `syslogSeverity` 输出 syslog severity。 |
| `logger` | `loggerName` | `precision` | Logger name，可缩短为最后 N 个 components。 |
| `message` | `msg` | none | Event message。 |
| `thread` | `threadName` | none | Logical thread name。 |
| `marker` | none | none | Marker string 或 `null`。 |
| `throwable` | `exception`, `thrown` | `field` | 默认 throwable object。Fields：`object`、`type`、`message`、`string`、`formatted`、`rootCause`、`rootCauseMessage`、`stackTrace`、`stackTraceAsString`。 |
| `rootCause` | none | none | Throwable root cause object。 |
| `stackTrace` | none | none | Stack array；`stacktraceAsString` 启用时输出 string。 |
| `source` | `location` | none | Caller object，包含 class、method、file、line 和 location。要求 caller PC。 |
| `process` | none | none | 包含 `pid` 的 object。 |
| `contextStack` | `ndc` | none | Context stack array。 |
| `mdc` | `contextMap`, `attrs` | `flatten`, `propertiesAsList` | Event attributes object 或 list。 |
| `attr` | none | `key` required | 单个 attribute value 或 `null`。 |
| `endOfBatch` | none | none | Async batching 设置的 boolean。 |

未知 resolver names 会交给配置的 plugin registry。如果没有 plugin resolver，template compilation 失败。

## TextLayout

Text layout 输出固定 key-value fields：

```text
time=2026-08-25T10:15:30.123+08:00 level=INFO logger=goark msg="service started" profile=dev
```

它总是以 newline 结束。

## XMLLayout

XML layout 每个 log event 输出一个 `<Event>` element。包含 time、level、logger、thread、message、optional marker、optional throwable、context stack 和 context map entries。`includeStacktrace` 会在 throwable stack 存在时添加 `<StackTrace>` frames。`stacktraceAsString` 会把 throwable 写成一个 text value。

## CSVLayout

CSV layout 按以下顺序输出字段：

```text
time,level,logger,thread,message,attrs
```

`attrs` column 包含 key-value text。对 empty fields、commas、quotes 和 newlines 使用标准 CSV quoting。

## GELFLayout

GELF layout 输出 Graylog Extended Log Format JSON：

| 字段 | 说明 |
| --- | --- |
| `version` | 固定 `1.1`。 |
| `host` | Process host name。 |
| `short_message` | Event message。 |
| `full_message` | 存在 throwable 时输出 throwable text。 |
| `timestamp` | 带 fractional microsecond precision 的 Unix seconds。 |
| `level` | Syslog severity。 |
| `_logger` | Logger name。 |
| `_thread` | Logical thread name。 |
| `_marker` | Marker 存在时输出。 |
| `_attr` fields | Event attributes 加 `_` 前缀输出，排除 empty keys、`id` 和已经以 `_` 开头的 keys。 |

下游协议需要 NUL delimiter 时可使用 `includeNullDelimiter`。

## RFC5424 和 Syslog Layout

`rfc5424` 和 `syslog` 使用同一实现。输出单条 RFC 5424 syslog event：

```text
<priority>1 timestamp host appName procid msgid structured-data message
```

Programmatic `RFC5424Layout` 暴露 `Facility`、`AppName` 和 `MessageID`。当前配置构建的是默认 layout instance；appender-level 字段如 `facility` 和 `appName` 为 appender plugins 保留，不调节内置 layout。

## YAMLLayout

YAML layout 每个 event 输出一个 YAML document，包含 time、level、logger、thread、message、optional marker、optional throwable、context stack 和 context map。`propertiesAsList` 为 true 时，attributes 以 key/value entries 输出，而不是 map。

YAML layout 使用 `gopkg.in/yaml.v3`，因此不是零分配热路径。高吞吐 structured logs 应优先使用 JSONLayout 或 direct JSON appender。

## HTMLLayout

HTML layout 输出一个 `<tr>`，包含 time、level、logger、thread、message 和 attributes cells。它适合受控文件片段或测试，不适合直接服务 untrusted HTML pages。
