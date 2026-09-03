# Layouts

[English](layouts.md)

Layout 将快照化的 `Event` 编码为字节。Console、file 和 rolling-file appender 使用
layout；JSON direct appender 为热点 JSON 路径绕过 layout。

## Layout Options

| 字段 | 说明 |
| --- | --- |
| `compact` | 禁用默认事件换行。 |
| `eventEol`, `event-eol` | 即使 compact 为 true，也为事件追加换行。 |
| `complete` | 启用流 header/footer。JSON layout 默认用数组式 complete 模式。 |
| `includeStacktrace`, `include-stacktrace` | 在支持的布局中输出结构化栈信息。 |
| `stacktraceAsString`, `stacktrace-as-string` | 在支持的布局中把异常栈输出为字符串。 |
| `propertiesAsList`, `properties-as-list` | 在支持的布局中将上下文属性输出为键值列表。 |
| `includeNullDelimiter`, `include-null-delimiter` | 每条事件后追加 NUL。 |
| `disableAnsi`, `disable-ansi` | 禁用 pattern `highlight` 和 `style` 的 ANSI 输出。 |
| `header` | complete 模式自定义 header。 |
| `footer` | complete 模式自定义 footer。 |

Complete JSON 和 JSON Template layout 会为每个 appender 克隆生命周期状态，因此不同文件会
生成独立有效的流。

## 内置 Layout 类型

| 类型 | 别名 | 输出 |
| --- | --- | --- |
| `pattern` | 无 | Log4j 风格文本 pattern。空 pattern 使用 `DefaultSpringBootPattern`。 |
| `text` | 无 | 稳定 `key=value` 文本。 |
| `json` | 无 | 包含事件字段和属性的 JSON 对象。 |
| `jsonTemplate` | 无 | 由 resolver 模板定义的 JSON 对象。 |
| `xml` | `xmlLayout` | 单个 `<Event>` XML 片段。 |
| `csv` | `csvLayout` | CSV 行：time、level、logger、thread、message、attrs。 |
| `gelf` | `gelfLayout` | Graylog Extended Log Format JSON。 |
| `rfc5424` | `rfc5424Layout` | RFC 5424 syslog 文本行。 |
| `syslog` | `syslogLayout` | RFC5424 layout 别名。 |
| `yaml` | `yamlLayout` | 单个 YAML 文档。 |
| `html` | `htmlLayout` | HTML 表格行。 |

## Pattern Layout

默认 pattern：

```text
%d %5level %pid --- [%thread] %logger : %msg%attrs%n
```

字段宽度和截断使用 Log4j 风格语法：`%5p`、`%-5p`、`%.30c` 以及组合形式。

| 转换器 | 别名 | 输出 |
| --- | --- | --- |
| `%d{format}` | `%date{format}` | 事件时间。 |
| `%p` | `%level` | 级别名称。 |
| `%pid` | `%processId` | 进程 ID。 |
| `%thread` | `%t` | 逻辑线程名，默认 `main`。 |
| `%logger{precision}` | `%c{precision}` | logger 名称，支持与 Log4j2 兼容的精度和缩写规则。 |
| `%msg` | `%message`, `%m` | 消息文本。 |
| `%attrs` | `%kvp`, `%map` | 事件属性的 pattern key/value 文本。 |
| `%X{key}` | `%mdc{key}` | 单个属性值。不带 key 时输出全部 attrs。 |
| `%ex{option}` | `%throwable{option}`, `%exception{option}` | Throwable 或 `error`/`err` 属性。选项：空、`none`、`short`、`full`。 |
| `%marker` | 无 | Marker 名称。 |
| `%ndc` | `%x` | Context stack，以空格连接。 |
| `%n` | 无 | 换行。 |
| `%C` | `%class` | 调用方 class/function owner。 |
| `%M` | `%method` | 调用方 method/function。 |
| `%F` | `%file` | 调用方文件名。 |
| `%L` | `%line` | 调用方行号。 |
| `%l` | `%location` | 组合调用位置。 |
| `%uuid` | 无 | 随机 version 4 UUID。 |
| `%relative` | `%r` | layout 包初始化以来的毫秒数。 |
| `%host` | `%hostname` | 主机名。 |
| `%sequenceNumber` | `%sn` | 全局递增序号。 |
| `%highlight{pattern}` | 无 | 按级别输出 ANSI 颜色，除非 `disableAnsi` 为 true。 |
| `%style{pattern}{style}` | 无 | 输出 ANSI style，除非 `disableAnsi` 为 true。 |
| `%notEmpty{pattern}` | 无 | 子 pattern 非空时输出。 |
| `%replace{pattern}{regex}{replacement}` | 无 | 对子输出做正则替换。 |
| `%enc{pattern}{mode}` | `%encode{pattern}{mode}` | 转义子输出。 |
| `%equals{pattern}{test}{substitution}` | `%equalsIgnoreCase{pattern}{test}{substitution}` | 子输出匹配时替换。 |
| `%maxLen{pattern}{length}` | `%maxLength{pattern}{length}` | 截断子输出。 |
| `%repeat{pattern}{count}` | 无 | 重复子输出。 |
| `%%` | 无 | 字面 `%`。 |

### Logger 名称精度

假设 logger 名称为 `org.apache.commons.Foo`：

| Pattern | 输出 | 含义 |
| --- | --- | --- |
| `%logger` | `org.apache.commons.Foo` | 输出完整 logger 名称。 |
| `%logger{1}` | `Foo` | 保留最右侧一段。 |
| `%logger{2}` | `commons.Foo` | 保留最右侧两段。 |
| `%logger{-1}` | `apache.commons.Foo` | 删除最左侧一段。 |
| `%logger{1.}` | `o.a.c.Foo` | 非末尾段各保留一个字符。 |
| `%logger{1~.2~}` | `o~.ap~.co~.Foo` | 按段应用长度和缩写标记。 |
| `%.8logger` | `org.apac` | 转换后应用 Log4j 最大宽度，保留最前 8 个字符。 |

`%logger{1.2*}` 会完整保留最右侧两段，并把此前各段缩写为一个字符。
精度规则在 layout 编译阶段完成解析，不会在每条日志上重复解析。

调用位置转换器需要通过 logger 选项、logger 配置、async 配置或 appender-ref
`includeLocation` 开启位置采集。

## 时间格式

`%d{...}`、date lookup 和 JSON Template timestamp resolver 支持：

| 格式 | 输出 |
| --- | --- |
| 空、`DEFAULT`、`ISO8601`、`ISO8601_OFFSET_DATE_TIME` | `2006-01-02T15:04:05.000Z07:00` 布局。 |
| `RFC3339` | Go `time.RFC3339`。 |
| `RFC3339NANO` | Go `time.RFC3339Nano`。 |
| `UNIX`, `UNIX_SECONDS` | Unix 秒。 |
| `UNIX_MILLIS`, `UNIX_MS` | Unix 毫秒。 |
| `UNIX_MICROS`, `UNIX_US` | Unix 微秒。 |
| `UNIX_NANOS`, `UNIX_NS` | Unix 纳秒。 |
| Java 风格日期 pattern | 常用 Java token 会转换为 Go reference-time layout。 |

支持的 Java token 包括 `yyyy`、`yy`、`MM`、`dd`、`HH`、`mm`、`ss`、
`SSS`、`SSSSSS`、`XXX`、`XX` 和 `X`。

## JSON Layout

JSON layout 输出 `time`、`level`、`logger`、`msg`、attrs 和可选 `thrown`。
开启 `propertiesAsList` 时，attrs 放入 `contextMap` 键值列表。开启 `complete` 时，
默认 header/footer 是 `[` 和 `]`。

```yaml
layout:
  type: json
  eventEol: true
  includeStacktrace: true
```

## JSON Template Layout

JSON Template 使用对象模板。字段可以是原始 JSON 常量，也可以是 resolver 对象。空模板
使用内置默认事件模板。

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

模板字段：

| Resolver | 别名 | 选项 | 输出 |
| --- | --- | --- | --- |
| `timestamp` | `time` | `format` | 时间字符串或 Unix 数值。 |
| `level` | 无 | `field=int`, `integer`, `value`, `severity`, `syslogSeverity` | 级别名称、整数值或 syslog severity。 |
| `logger` | `loggerName` | `precision` | Logger 名称。 |
| `message` | `msg` | 无 | 消息文本。 |
| `thread` | `threadName` | 无 | 逻辑线程名。 |
| `marker` | 无 | 无 | Marker 名称或 `null`。 |
| `throwable` | `exception`, `thrown` | `field` | Throwable 对象、类型、消息、字符串、root cause、stack trace 或 stack string。 |
| `rootCause` | 无 | 无 | Throwable root cause 对象。 |
| `stackTrace` | 无 | 无 | 栈帧，或开启布局选项后的栈字符串。 |
| `source` | `location` | 无 | 包含 class、method、file、line、location 的调用方对象。 |
| `process` | 无 | 无 | 包含 `pid` 的进程对象。 |
| `contextStack` | `ndc` | 无 | Context stack 数组。 |
| `mdc` | `contextMap`, `attrs` | `flatten`, `propertiesAsList` | 属性对象或列表。 |
| `attr` | 无 | 必填 `key` | 单个属性值或 `null`。 |
| `endOfBatch` | 无 | 无 | async 批次设置的布尔值。 |
| 自定义 | 无 | 自定义 raw options | 通过插件注册表解析。 |

使用 `eventTemplateUri`、`eventTemplatePath` 或 kebab 别名可以加载模板文件。

## 结构化 Layout

| Layout | Throwable 行为 | 属性行为 |
| --- | --- | --- |
| `xml` | 默认字符串，`includeStacktrace` 输出栈帧，`stacktraceAsString` 输出完整字符串。 | `<ContextMap><Entry key="">...`。 |
| `yaml` | 默认字符串，`includeStacktrace` 输出 map，`stacktraceAsString` 输出字符串。 | 默认 map，`propertiesAsList` 时为列表。 |
| `gelf` | `full_message` 来自 throwable 或 `error` 属性。 | 附加字段加 `_` 前缀；空键、`id`、已有下划线前缀会跳过。 |
| `rfc5424` / `syslog` | 只使用消息文本。 | 属性编码进 `[goark@32473 ...]`。 |
| `csv` | 只在 attr 文本中体现。 | attrs 放入一个 CSV 字段。 |
| `html` | 只在 attr 文本中体现。 | attrs 转义后放入一个表格单元格。 |

## Layout 生命周期

File 和 console appender 在第一次打开/写入时调用 header，在关闭时调用 footer。Rolling file
在滚动前写 footer，打开新流后写 header。`createOnDemand` 会将该生命周期推迟到第一条事件。
