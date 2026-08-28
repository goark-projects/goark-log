# Layout Reference

A layout converts an immutable log event snapshot into bytes. Console, file,
and rolling file appenders use layouts. The direct JSON appender has its own
fixed JSON encoder and does not use a configured layout.

## Built-In Layouts

| Type | Aliases | Output |
| --- | --- | --- |
| `pattern` | none | Log4j/Spring Boot style pattern text. |
| `text` | none | Stable key-value text. |
| `json` | none | Structured JSON event. |
| `jsonTemplate` | `json-template` after normalization | JSON event generated from an event template. |
| `xml` | `xmlLayout` | Single XML event. |
| `csv` | `csvLayout` | Single CSV row. |
| `gelf` | `gelfLayout` | Graylog Extended Log Format JSON. |
| `rfc5424` | `rfc5424Layout` | RFC 5424 syslog text event. |
| `syslog` | `syslogLayout` | Alias of the RFC 5424 layout. |
| `yaml` | `yamlLayout` | YAML event document. |
| `html` | `htmlLayout` | HTML table row. |

Layout and plugin kinds are normalized by trimming spaces, lowercasing, and
removing `-` and `_`.

## Common Layout Fields

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

| Field | Aliases | Default | Description |
| --- | --- | --- | --- |
| `type` | none | appender default pattern | Layout kind. |
| `pattern` | none | default Spring Boot pattern | PatternLayout format string. |
| `eventTemplate` | `event-template` | default template | Inline JSON Template event template. |
| `eventTemplateUri` | `event-template-uri`, `eventTemplatePath`, `event-template-path` | empty | Local JSON Template file. |
| `compact` | none | false | Disables the default event newline. |
| `eventEol` | `event-eol` | false | Adds an event newline even when `compact` is true. |
| `complete` | none | false | Enables lifecycle header/footer output. JSON layouts default to array header/footer when complete. |
| `includeStacktrace` | `include-stacktrace` | false | Includes structured stack information where supported. |
| `stacktraceAsString` | `stacktrace-as-string` | false | Emits stacktrace as one string instead of a structured object/list. |
| `propertiesAsList` | `properties-as-list` | false | Emits context attributes as `[{"key":...,"value":...}]` where supported. |
| `includeNullDelimiter` | `include-null-delimiter` | false | Appends NUL after each event. Useful for protocols that require frame delimiters. |
| `disableAnsi` | `disable-ansi` | false | Disables PatternLayout `%highlight` and `%style` ANSI output. |
| `header` | none | empty | Custom lifecycle header when `complete` is true. |
| `footer` | none | empty | Custom lifecycle footer when `complete` is true. |

## PatternLayout

Default pattern:

```text
%d %5level %pid --- [%thread] %logger : %msg%attrs%n
```

Example:

```yaml
layout:
  type: pattern
  pattern: "%d{yyyy-MM-dd HH:mm:ss.SSS} %-5p [%thread] %c{2} trace=%X{trace_id} %m%notEmpty{ %ex{short}}%n"
```

### Width Modifiers

Pattern converters support:

| Form | Meaning |
| --- | --- |
| `%5p` | Minimum width 5, right-aligned. |
| `%-5p` | Minimum width 5, left-aligned. |
| `%.40logger` | Maximum width 40. |
| `%20.40logger` | Minimum width 20, maximum width 40. |

### Converters

| Converter | Aliases | Description |
| --- | --- | --- |
| `%d{format}` | `%date{format}` | Event time. Empty/default uses `2006-01-02T15:04:05.000Z07:00`. |
| `%level` | `%p` | Level name from the level registry. |
| `%pid` | `%processId` | Current process ID. |
| `%thread` | `%t` | Logical thread name from context/attrs, default `main`. |
| `%logger{precision}` | `%c{precision}` | Logger name. Precision keeps the last N dot-separated components. |
| `%msg` | `%message`, `%m` | Event message. |
| `%attrs` | `%kvp`, `%map` | Event attributes as key-value text. |
| `%X{key}` | `%mdc{key}` | Attribute value by key. Empty `%X` or `%mdc` prints all attrs. |
| `%ex{option}` | `%throwable`, `%exception` | Throwable text. Options: empty, `short`, `full`, `none`. |
| `%marker` | none | Marker value. |
| `%ndc` | `%x` | Context stack values. |
| `%C` | `%class` | Caller class/function owner. Requires caller PC. |
| `%M` | `%method` | Caller method/function. Requires caller PC. Case-sensitive: `%M` is method, `%m` is message. |
| `%F` | `%file` | Caller file. Requires caller PC. |
| `%L` | `%line` | Caller line. Requires caller PC. |
| `%l` | `%location` | Caller location string. Requires caller PC. |
| `%n` | none | Newline. |
| `%uuid` | none | Random UUID v4 style value generated per event render. |
| `%relative` | `%r` | Milliseconds since layout package initialization. |
| `%host` | `%hostname` | Host name resolved at process startup. |
| `%sequenceNumber` | `%sn` | Atomic sequence number. |
| `%highlight{pattern}` | none | Applies default ANSI color by level. Disabled by `disableAnsi`. |
| `%style{pattern}{style}` | none | Applies configured ANSI style. Disabled by `disableAnsi`. |
| `%notEmpty{pattern}` | none | Emits nested pattern only when the nested output is not blank after trimming. |
| `%replace{pattern}{regex}{replacement}` | none | Regex replacement on nested output. |
| `%enc{pattern}{mode}` | `%encode` | Escapes nested output. Modes: `json`, `html`, `xml`, `crlf`; unknown mode leaves value unchanged. |
| `%equals{pattern}{test}{substitution}` | none | Replaces nested output when it equals `test`. |
| `%equalsIgnoreCase{pattern}{test}{substitution}` | none | Case-insensitive variant of `%equals`. |
| `%maxLen{pattern}{length}` | `%maxLength` | Truncates nested output to display width. |
| `%repeat{pattern}{count}` | none | Repeats nested output. |
| `%%` | none | Literal percent sign. |

Caller converters resolve `slog.Record.PC`. They are empty unless one of these
is true:

- Handler/root/logger/appender-ref `includeLocation` is true.
- Handler-level async `includeLocation` is true.
- Native logger is created with `WithLoggerCaller(true)`.

Caller capture has measurable cost and should be enabled only for loggers or
appender refs that need it.

### ANSI Styles

`%highlight` uses these level defaults:

| Level | Style |
| --- | --- |
| `FATAL` and above | `red,bold` |
| `ERROR` | `red` |
| `WARN` | `yellow` |
| `INFO` | `green` |
| `DEBUG` | `cyan` |
| lower | `faint` |

`%style` accepts tokens such as `bold`, `faint`, `underline`, `blink`,
`reverse`, foreground colors (`red`, `green`, `yellow`, `blue`, `magenta`,
`cyan`, `white`, `gray`), bright colors, and background forms such as `bgRed` or
`backgroundRed`.

## JSONLayout

```yaml
layout:
  type: json
  eventEol: true
  propertiesAsList: false
  includeStacktrace: true
```

Default fields:

| Field | Description |
| --- | --- |
| `time` | Event time in default layout format. |
| `level` | Level name. |
| `logger` | Logger name. |
| `msg` | Message. |
| event attributes | Emitted as top-level fields unless `propertiesAsList` is true. |
| `contextMap` | Attribute list when `propertiesAsList` is true. |
| `thrown` | Throwable object or string when stacktrace output is enabled and throwable exists. |

Common `slog.Value` kinds are encoded by hand: string, bool, int, uint, float,
duration, time, groups, and errors/stringers. Complex `Any` values use the
internal Sonic-backed JSON codec and fall back to `fmt.Sprint` on marshal error.

With `complete: true`, JSONLayout writes a JSON array stream. Default header is
`[` and default footer is `]`. Commas are inserted between events.

## JSONTemplateLayout

Default event template:

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

Inline template example:

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

Template file example:

```yaml
layout:
  type: jsonTemplate
  eventTemplateUri: conf/log-event-template.json
  stacktraceAsString: true
```

Any JSON field value without `$resolver` is emitted as raw JSON.

### JSON Template Resolvers

| Resolver | Aliases | Options | Output |
| --- | --- | --- | --- |
| `timestamp` | `time` | `format` | Event time using the time-pattern mapper. |
| `level` | none | `field` | Text level by default. `int`, `integer`, or `value` emits slog numeric level. `severity` or `syslogSeverity` emits syslog severity. |
| `logger` | `loggerName` | `precision` | Logger name, optionally shortened to last N components. |
| `message` | `msg` | none | Event message. |
| `thread` | `threadName` | none | Logical thread name. |
| `marker` | none | none | Marker string or `null`. |
| `throwable` | `exception`, `thrown` | `field` | Throwable object by default. Fields: `object`, `type`, `message`, `string`, `formatted`, `rootCause`, `rootCauseMessage`, `stackTrace`, `stackTraceAsString`. |
| `rootCause` | none | none | Throwable root cause object. |
| `stackTrace` | none | none | Stack array, or string when `stacktraceAsString` is enabled. |
| `source` | `location` | none | Caller object with class, method, file, line, and location. Requires caller PC. |
| `process` | none | none | Object containing `pid`. |
| `contextStack` | `ndc` | none | Context stack array. |
| `mdc` | `contextMap`, `attrs` | `flatten`, `propertiesAsList` | Event attributes as an object or list. |
| `attr` | none | `key` required | One attribute value or `null`. |
| `endOfBatch` | none | none | Boolean set by async batching. |

Unknown resolver names are delegated to the configured plugin registry. If no
plugin resolver exists, template compilation fails.

## TextLayout

Text layout emits fixed key-value fields:

```text
time=2026-08-25T10:15:30.123+08:00 level=INFO logger=goark msg="service started" profile=dev
```

It always terminates with a newline.

## XMLLayout

XML layout emits one `<Event>` element per log event. It includes time, level,
logger, thread, message, optional marker, optional throwable, context stack, and
context map entries. `includeStacktrace` adds `<StackTrace>` frames when a
throwable stack is present. `stacktraceAsString` writes the throwable as one
text value.

## CSVLayout

CSV layout emits fields in this order:

```text
time,level,logger,thread,message,attrs
```

The `attrs` column contains key-value text. Standard CSV quoting is applied for
empty fields, commas, quotes, and newlines.

## GELFLayout

GELF layout emits Graylog Extended Log Format JSON:

| Field | Description |
| --- | --- |
| `version` | Always `1.1`. |
| `host` | Process host name. |
| `short_message` | Event message. |
| `full_message` | Throwable text when available. |
| `timestamp` | Unix seconds with fractional microsecond precision. |
| `level` | Syslog severity. |
| `_logger` | Logger name. |
| `_thread` | Logical thread name. |
| `_marker` | Marker when present. |
| `_attr` fields | Event attributes prefixed with `_`, except empty keys, `id`, and keys already starting with `_`. |

`includeNullDelimiter` can be used when the downstream protocol expects NUL
delimited GELF events.

## RFC5424 and Syslog Layout

`rfc5424` and `syslog` use the same layout implementation. The output is a
single RFC 5424 syslog event:

```text
<priority>1 timestamp host appName procid msgid structured-data message
```

Programmatic `RFC5424Layout` exposes `Facility`, `AppName`, and `MessageID`.
Configuration currently builds the default layout instance; appender-level
fields such as `facility` and `appName` are reserved for appender plugins and do
not tune the built-in layout.

## YAMLLayout

YAML layout emits one YAML document per event with time, level, logger, thread,
message, optional marker, optional throwable, context stack, and context map.
When `propertiesAsList` is true, attributes are emitted as key/value entries
instead of a map.

YAML layout uses `gopkg.in/yaml.v3`, so it is not a zero-allocation hot path.
Use JSONLayout or the direct JSON appender for high-throughput structured logs.

## HTMLLayout

HTML layout emits a `<tr>` with cells for time, level, logger, thread, message,
and attributes. It is intended for controlled file snippets or tests, not for
serving untrusted HTML pages directly.
