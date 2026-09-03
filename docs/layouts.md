# Layouts

[简体中文](layouts.zh-CN.md)

Layouts encode a snapshot `Event` into bytes. Console, file, and rolling-file
appenders use layouts; JSON direct appender bypasses layouts for a hot JSON
path.

## Layout Options

| Field | Notes |
| --- | --- |
| `compact` | Suppresses the default event newline. |
| `eventEol`, `event-eol` | Adds a newline even when compact is true. |
| `complete` | Enables stream header/footer output. JSON layouts use array-style complete mode by default. |
| `includeStacktrace`, `include-stacktrace` | Emits structured stack details where supported. |
| `stacktraceAsString`, `stacktrace-as-string` | Emits throwable stack as one string where supported. |
| `propertiesAsList`, `properties-as-list` | Emits context attributes as key/value lists where supported. |
| `includeNullDelimiter`, `include-null-delimiter` | Appends NUL after each event. |
| `disableAnsi`, `disable-ansi` | Disables ANSI output from pattern `highlight` and `style`. |
| `header` | Custom complete-mode header. |
| `footer` | Custom complete-mode footer. |

Complete JSON and JSON Template layouts clone lifecycle state per appender so
separate files produce valid independent streams.

## Built-In Layout Types

| Type | Aliases | Output |
| --- | --- | --- |
| `pattern` | none | Log4j-style text pattern. Empty pattern uses `DefaultSpringBootPattern`. |
| `text` | none | Stable `key=value` text. |
| `json` | none | JSON object with event fields and attributes. |
| `jsonTemplate` | none | JSON object defined by a resolver template. |
| `xml` | `xmlLayout` | Single `<Event>` XML fragment. |
| `csv` | `csvLayout` | CSV line: time, level, logger, thread, message, attrs. |
| `gelf` | `gelfLayout` | Graylog Extended Log Format JSON. |
| `rfc5424` | `rfc5424Layout` | RFC 5424 syslog text line. |
| `syslog` | `syslogLayout` | Alias of RFC5424 layout. |
| `yaml` | `yamlLayout` | Single YAML document. |
| `html` | `htmlLayout` | HTML table row. |

## Pattern Layout

Default pattern:

```text
%d %5level %pid --- [%thread] %logger : %msg%attrs%n
```

Field width and truncation follow Log4j-style syntax: `%5p`, `%-5p`, `%.30c`,
and combinations are accepted.

| Converter | Aliases | Output |
| --- | --- | --- |
| `%d{format}` | `%date{format}` | Event timestamp. |
| `%p` | `%level` | Level name. |
| `%pid` | `%processId` | Process ID. |
| `%thread` | `%t` | Logical thread name, default `main`. |
| `%logger{precision}` | `%c{precision}` | Logger name with Log4j2-compatible precision and abbreviation rules. |
| `%msg` | `%message`, `%m` | Message text. |
| `%attrs` | `%kvp`, `%map` | Event attributes as pattern key/value text. |
| `%X{key}` | `%mdc{key}` | Single attribute value. Without a key, emits all attrs. |
| `%ex{option}` | `%throwable{option}`, `%exception{option}` | Throwable or `error`/`err` attr. Options: empty, `none`, `short`, `full`. |
| `%marker` | none | Marker name. |
| `%ndc` | `%x` | Context stack values joined by spaces. |
| `%n` | none | Newline. |
| `%C` | `%class` | Caller class/function owner. |
| `%M` | `%method` | Caller method/function. |
| `%F` | `%file` | Caller file base name. |
| `%L` | `%line` | Caller line. |
| `%l` | `%location` | Combined caller location. |
| `%uuid` | none | Random version 4 UUID. |
| `%relative` | `%r` | Milliseconds since layout package initialization. |
| `%host` | `%hostname` | Host name. |
| `%sequenceNumber` | `%sn` | Global monotonically increasing sequence number. |
| `%highlight{pattern}` | none | ANSI level color unless `disableAnsi` is true. |
| `%style{pattern}{style}` | none | ANSI style unless `disableAnsi` is true. |
| `%notEmpty{pattern}` | none | Emits child pattern only when it is non-blank. |
| `%replace{pattern}{regex}{replacement}` | none | Regex replacement over child output. |
| `%enc{pattern}{mode}` | `%encode{pattern}{mode}` | Escapes child output. |
| `%equals{pattern}{test}{substitution}` | `%equalsIgnoreCase{pattern}{test}{substitution}` | Substitutes when child output matches. |
| `%maxLen{pattern}{length}` | `%maxLength{pattern}{length}` | Truncates child output. |
| `%repeat{pattern}{count}` | none | Repeats child output. |
| `%%` | none | Literal percent sign. |

### Logger Name Precision

Given the logger name `org.apache.commons.Foo`:

| Pattern | Output | Meaning |
| --- | --- | --- |
| `%logger` | `org.apache.commons.Foo` | Full logger name. |
| `%logger{1}` | `Foo` | Retain one rightmost component. |
| `%logger{2}` | `commons.Foo` | Retain two rightmost components. |
| `%logger{-1}` | `apache.commons.Foo` | Drop one leftmost component. |
| `%logger{1.}` | `o.a.c.Foo` | Retain one character from each non-final component. |
| `%logger{1~.2~}` | `o~.ap~.co~.Foo` | Apply per-component lengths and abbreviation markers. |
| `%.8logger` | `org.apac` | Apply the Log4j maximum-width modifier after conversion. |

`%logger{1.2*}` retains the two rightmost components in full and abbreviates
each earlier component to one character. Precision is compiled with the layout;
it is not parsed for every event.

Caller converters require location capture through logger options, logger
config, async config, or appender-ref `includeLocation`.

## Time Formats

`%d{...}`, date lookups, and JSON Template timestamp resolver accept:

| Format | Output |
| --- | --- |
| empty, `DEFAULT`, `ISO8601`, `ISO8601_OFFSET_DATE_TIME` | `2006-01-02T15:04:05.000Z07:00` layout. |
| `RFC3339` | Go `time.RFC3339`. |
| `RFC3339NANO` | Go `time.RFC3339Nano`. |
| `UNIX`, `UNIX_SECONDS` | Unix seconds. |
| `UNIX_MILLIS`, `UNIX_MS` | Unix milliseconds. |
| `UNIX_MICROS`, `UNIX_US` | Unix microseconds. |
| `UNIX_NANOS`, `UNIX_NS` | Unix nanoseconds. |
| Java-style date pattern | Common Java tokens are mapped to Go reference-time layout. |

Supported Java token mappings include `yyyy`, `yy`, `MM`, `dd`, `HH`, `mm`,
`ss`, `SSS`, `SSSSSS`, `XXX`, `XX`, and `X`.

## JSON Layout

JSON layout writes `time`, `level`, `logger`, `msg`, attrs, and optional
`thrown`. With `propertiesAsList`, attrs move under `contextMap` as key/value
entries. With `complete`, the default header/footer are `[` and `]`.

```yaml
layout:
  type: json
  eventEol: true
  includeStacktrace: true
```

## JSON Template Layout

JSON Template uses an object whose fields are either raw JSON constants or
resolver objects. Empty template uses the built-in default event template.

```yaml
layout:
  type: jsonTemplate
  eventTemplate: >
    {
      "ts": {"$resolver": "timestamp", "format": "UNIX_MILLIS"},
      "level": {"$resolver": "level"},
      "logger": {"$resolver": "logger", "precision": 3},
      "message": {"$resolver": "message"},
      "trace_id": {"$resolver": "attr", "key": "trace_id"}
    }
  eventEol: true
```

Template fields:

| Resolver | Aliases | Options | Output |
| --- | --- | --- | --- |
| `timestamp` | `time` | `format` | Timestamp string or Unix number. |
| `level` | none | `field=int`, `integer`, `value`, `severity`, `syslogSeverity` | Level name, integer value, or syslog severity. |
| `logger` | `loggerName` | `precision` | Logger name. |
| `message` | `msg` | none | Message text. |
| `thread` | `threadName` | none | Logical thread name. |
| `marker` | none | none | Marker name or `null`. |
| `throwable` | `exception`, `thrown` | `field` | Throwable object, type, message, string, root cause, stack trace, or stack string. |
| `rootCause` | none | none | Throwable root cause object. |
| `stackTrace` | none | none | Stack frames or stack string when layout option is set. |
| `source` | `location` | none | Caller object with class, method, file, line, and location. |
| `process` | none | none | Process object with `pid`. |
| `contextStack` | `ndc` | none | Context stack array. |
| `mdc` | `contextMap`, `attrs` | `flatten`, `propertiesAsList` | Attribute object or list. |
| `attr` | none | `key` required | One attribute value or `null`. |
| `endOfBatch` | none | none | Boolean set by async batches. |
| custom | none | custom raw options | Resolved through plugin registry. |

Use `eventTemplateUri`, `eventTemplatePath`, or their kebab aliases to load a
template file.

## Structured Layouts

| Layout | Throwable behavior | Attribute behavior |
| --- | --- | --- |
| `xml` | String by default, stack frames when `includeStacktrace`, full string when `stacktraceAsString`. | `<ContextMap><Entry key="">...`. |
| `yaml` | String by default, map when `includeStacktrace`, string when `stacktraceAsString`. | Map by default, list with `propertiesAsList`. |
| `gelf` | `full_message` from throwable or `error` attr. | Additional fields are prefixed with `_`; empty, `id`, and already-underscore keys are skipped. |
| `rfc5424` / `syslog` | Message text only. | Attributes are encoded into `[goark@32473 ...]`. |
| `csv` | Included only when present in attr text. | Attrs are one CSV field. |
| `html` | Included only when present in attr text. | Attrs are escaped in one table cell. |

## Layout Lifecycle

File and console appenders call header on first open/write and footer on close.
Rolling file writes footers before rollover and headers after opening a new
stream. `createOnDemand` delays this lifecycle until the first event.
