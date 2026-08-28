# 配置参考

[English](configuration.md)

本文描述当前 worktree 中 `goark.dev/log` 实现的配置契约，覆盖 v0.0.2 准备阶段支持的字段名、默认值、校验规则和 reload 边界。

## 加载顺序

`LoadOptions`、`NewConfigured`、`NewConfiguredHandler`、`NewConfiguredLoggerContext` 和 `ConfigureDefault` 按以下优先级解析配置：

1. `WithConfigPath(path)`。
2. 环境变量 `GOARK_LOG_CONFIG`，或 `WithConfigEnvKey(key)` 指定的 key。
3. Boot property resolver keys：`goark.log.config`、`goark.logging.config`、`logging.config`。
4. 当前工作目录下的默认文件：
   - `conf/goark-log.yml`
   - `conf/goark-log.yaml`
   - `conf/goark-log.json`
   - `conf/goark-log.xml`
   - `conf/goark-log.toml`
   - `conf/goark-log.properties`
5. 内置默认配置：stderr console appender，root level `INFO`。

相对路径默认基于 `os.Getwd()` 解析；使用 `WithConfigWorkingDir(dir)` 可以覆盖默认工作目录。

## 支持格式

| 格式 | 状态 | 说明 |
| --- | --- | --- |
| YAML | 支持 | 推荐默认格式。使用 strict known fields 解析。 |
| JSON | 支持 | 使用与 YAML 相同的 structured schema 和字段名。 |
| TOML | 支持 | TOML 解析后使用同一套 structured schema。 |
| XML | 支持 | 使用 Log4j2-style elements 表达 appenders、layouts、filters、policies 和 loggers。 |
| properties | 支持 | 使用 `appender.console.type` 这类 flat keys。 |

YAML/JSON/TOML 的结构化解码使用严格模式：未知字段会导致解析失败。日志配置错误不应该被静默忽略。

## 包装结构

Structured YAML 或 JSON 文件只能使用以下三种结构之一：

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

不要把 top-level fields 与 `configuration` 或 `goark.log` 混用，也不要在同一文件中同时使用两个 wrapper。

## 顶层字段

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `status` | string | none | 为兼容性解析并做 lookup resolution。当前不会改变 `StatusLogger`；status 行为通过 `NewStatusLogger` API 设置。 |
| `monitorInterval`, `monitor-interval` | duration string | disabled | 使用 `NewConfiguredLoggerContext` 时启用 `LoggerContext` 文件轮询 reload。纯数字按秒处理。 |
| `properties` | map string to string | empty | 可通过 `${prop:key}`、`${property:key}` 和 shorthand `${key}` 读取的本地值。 |
| `customLevels`, `custom-levels` | map string to integer string | empty | 注册进程级 custom log level names。 |
| `appenders` | map | omitted 时 default console | Named appender definitions。 |
| `filters` | map | empty | Named filter definitions。 |
| `filterRefs`, `filter-refs` | string list | empty | Global filter chain，在 logger level checks 前执行。 |
| `asyncLogger`, `async-logger`, `async` | object | disabled | Handler-level asynchronous pipeline。只使用一个 alias。 |
| `root` | object | `INFO` and first appender | Root logger route。 |
| `loggers` | map | empty | Named logger routes。支持 exact name 和 prefix matches。 |

## Levels

内置 levels：

| 名称 | 值 | 语义 |
| --- | --- | --- |
| `ALL` | minimum integer | 允许所有事件通过 level gate。 |
| `TRACE` | `-8` | 低于 `DEBUG`。 |
| `DEBUG` | `-4` | 与 `slog.LevelDebug` 相同。 |
| `INFO` | `0` | 与 `slog.LevelInfo` 相同；默认 root level。 |
| `WARN`, `WARNING` | `4` | 与 `slog.LevelWarn` 相同。 |
| `ERROR` | `8` | 与 `slog.LevelError` 相同。 |
| `FATAL` | `12` | 高于 `ERROR`。 |
| `OFF` | maximum integer | 通过 level 关闭普通事件。 |

支持 numeric levels。Custom level names 必须非空、非数字且不包含 whitespace：

```yaml
configuration:
  customLevels:
    NOTICE: "2"
    SECURITY: "6"
```

Custom levels 会注册到进程默认 level registry。

## 值解析

### Byte Sizes

Byte-size 字段包括 appender `bufferSize`、rolling `maxSize` 和 delete action `maxSize`。

| 形式 | 含义 |
| --- | --- |
| `0` | 0 字节；是否禁用 buffering 或 size limits 取决于字段语义。 |
| `b`, `byte`, `bytes` | 字节。 |
| `k`, `kb`, `m`, `mb`, `g`, `gb`, `t`, `tb` | 十进制单位，base 1000。 |
| `ki`, `kib`, `mi`, `mib`, `gi`, `gib`, `ti`, `tib` | 二进制单位，base 1024。 |

支持小数，例如 `1.5MiB`。值必须非负并且能放入 `int64`。

### Monitor Interval

`monitorInterval` 支持：

- empty、`0`、`off`、`none`、`disabled`、`false`：禁用；
- 纯数字如 `30`：秒；
- Go duration 如 `500ms`、`5s`、`2m`、`1h`。

负数会被拒绝。

### Rolling Interval

Rolling time policy interval 支持：

- empty、`0`、`off`、`none`、`disabled`：禁用；
- `minute`、`minutely`、`hour`、`hourly`、`day`、`daily`；
- Go duration 如 `30s`、`5m`、`1h`；
- day/minute/hour suffix 如 `2days`、`4hours`、`15mins`。

负数会被拒绝。

### Rolling Max Age

Retention age 字段支持：

- empty、`0`、`off`、`none`、`disabled`：禁用；
- Go duration 如 `720h`；
- day forms 如 `30d`、`30day`、`30days`。

负数会被拒绝。

## Lookups

Lookups 在构建配置对象前解析。

| 形式 | 说明 |
| --- | --- |
| `${prop:LOG_DIR}` | 从 `properties` 读取值。 |
| `${property:LOG_DIR}` | 等同于 `prop`。 |
| `${LOG_DIR}` | property lookup shorthand。 |
| `${LOG_DIR:-logs}` | 带 fallback 的 property lookup。 |
| `${env:HOME}` | 读取环境变量。 |
| `${env:LOG_DIR:-logs}` | 带 fallback 的环境变量 lookup。 |
| `${sys:pid}` | Process ID。 |
| `${sys:hostname}` | Host name。 |
| `${sys:cwd}` | 当前工作目录。 |
| `${sys:os}`, `${sys:arch}` | Go runtime OS 和 architecture。 |
| `${go:version}` | Go runtime version。 |
| `${go:os}`, `${go:arch}` | Go runtime OS 和 architecture。 |
| `${date:yyyyMMdd}` | 当前时间，使用 time-pattern mapper 格式化。 |

`$$` 转义为单个美元符号。Lookup expression 必须以 `}` 结束。未知 namespace 或没有 fallback 的缺失值会导致配置加载失败。

不能注册 blocked namespaces：`jndi`、`ldap`、`rmi`。

## Time Patterns

时间格式支持 Java/Log4j 风格子集和 Unix timestamp modes。

| Pattern | 输出行为 |
| --- | --- |
| empty, `DEFAULT`, `ISO8601`, `ISO8601_OFFSET_DATE_TIME` | `2006-01-02T15:04:05.000Z07:00`。 |
| `RFC3339` | Go `time.RFC3339`。 |
| `RFC3339NANO` | Go `time.RFC3339Nano`。 |
| `UNIX`, `UNIX_SECONDS` | Unix seconds。 |
| `UNIX_MILLIS`, `UNIX_MS` | Unix milliseconds。 |
| `UNIX_MICROS`, `UNIX_US` | Unix microseconds。 |
| `UNIX_NANOS`, `UNIX_NS` | Unix nanoseconds。 |
| `yyyy`, `yy`, `MM`, `dd`, `HH`, `mm`, `ss`, `SSS`, `SSSSSS`, `X`, `XX`, `XXX` | 转换为 Go reference-time layout tokens。 |

## Async Logger

Handler-level async 通过 `asyncLogger`、`async-logger` 或 `async` 配置。

| 字段 | 类型 | enabled 时默认值 | 说明 |
| --- | --- | --- | --- |
| `enabled` | bool | false | 启用 Handler-level async pipeline。 |
| `queueSize`, `queue-size` | int | `4096` | Queue capacity。正数会归一化为内部 ring-buffer capacity。 |
| `batchSize`, `batch-size` | int | `64` | 每次 drain loop 最多消费事件数。上限为 queue capacity。 |
| `overflowStrategy`, `overflow-strategy` | string | `block` | Queue-full 行为。 |
| `waitStrategy`, `wait-strategy` | string | `block` | Consumer wait 行为。 |
| `waitRetries`, `wait-retries` | int | `0` | 可选 wait-strategy retry count，必须非负。 |
| `sleepTime`, `sleep-time` | duration | `0` | 可选 sleep duration，必须按 Go duration 解析。 |
| `timeout` | duration | `0` | 可选 blocking timeout，必须按 Go duration 解析。 |
| `includeLocation`, `include-location` | bool | false | 入队前捕获 caller PC。有额外成本。 |

Overflow strategy aliases：

| Canonical | Aliases | 行为 |
| --- | --- | --- |
| `block` | `blocking` | 施加 backpressure，不丢事件。 |
| `drop` | `discard`, `discard-newest` | Queue full 时丢弃事件。 |
| `drop-debug` | `dropdebug`, `discard-debug`, `discarddebug` | Queue full 时丢弃 `DEBUG` 及以下事件。 |
| `sync-fallback` | `sync`, `synchronous`, `synchronize` | Queue full 时同步写出。 |

Wait strategy aliases：

| Canonical | Aliases |
| --- | --- |
| `block` | `blocking`, `timeout`, `timeout-block`, `timeoutblocking` |
| `sleep` | `sleeping` |
| `yield` | `yielding` |
| `spin` | `busy-spin`, `busyspin` |

Reload 不能改变 async enablement、queue size、batch size、overflow strategy、wait strategy、wait options 或 async caller-location 行为。

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

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `level` | string | `INFO` | Global filters 未 force acceptance 时的最低级别。 |
| `appenderRefs`, `appender-refs`, `refs` | list | first configured appender | 绑定到该 route 的 appenders。元素可以是 string 或 object。 |
| `filters`, `filterRefs`, `filter-refs` | string list | empty | Route filter chain。 |
| `includeLocation`, `include-location` | bool | false | 为该 route 启用 caller PC capture。 |

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

匹配方式是 exact 或 prefix-based。`goark.orm` 规则会匹配 `goark.orm` 和 `goark.orm.mapper`。最具体规则优先。

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `level` | string | root level | 当前 logger route 的最低级别。 |
| `appenderRefs`, `appender-refs`, `refs` | list | empty | Route-specific appenders。 |
| `filters`, `filterRefs`, `filter-refs` | string list | empty | Logger filter chain。 |
| `additivity` | bool | true | true 时追加 root appenders 和 root filters。false 时该 logger 必须定义至少一个 appender。 |
| `includeLocation`, `include-location` | bool | root value | 为该 route 启用或禁用 caller capture。 |

Additive routing 会按 appender name 去重重复 appenders。

## Appender References

Appender references 可以是字符串：

```yaml
appenderRefs: [console, rolling]
```

也可以使用结构化控制：

```yaml
appenderRefs:
  - ref: console
  - ref: rolling
    level: warn
    includeLocation: false
    filterRefs: [onlyProd]
```

结构化字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ref` | string | 必填 appender name。 |
| `level` | string | 可选 per-reference minimum level。 |
| `includeLocation`, `include-location` | bool | 可选 per-reference caller-location override。`false` 会在写入该 appender 前去掉 PC。 |
| `filters`, `filterRefs`, `filter-refs` | string list | 只应用到该 appender reference 的 filter chain。 |

## Properties Format

Properties files 使用 flat keys。Properties adapter 会忽略 unknown keys，但已知字段的非法值会导致加载失败。

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

支持的 properties prefixes：

| Prefix | 用途 |
| --- | --- |
| `property.<name>` | Config property lookup value。 |
| `customLevel.<name>`, `custom-level.<name>` | Custom level registration。 |
| `asyncLogger.*`, `async-logger.*`, `async.*` | Handler-level async logger。 |
| `appender.<id>.*` | Appender definition。 |
| `appender.<id>.layout.*` | Appender layout definition。 |
| `appender.<id>.rolling.*` | Properties adapter 支持的 rolling fields。 |
| `appender.<id>.routes.<key>` | Routing appender route mapping。 |
| `appender.<id>.rewrite.*` | Rewrite appender policy。 |
| `appender.<id>.appenderRef.<id>.*` | Structured appender references。 |
| `filter.<id>.*` | Filter definition。 |
| `filter.<id>.values.<key>` | Map-like filter value。 |
| `filter.<id>.thresholds.<value>` | Dynamic-threshold mapping。 |
| `logger.<id>.*` | Named logger definition。 |
| `rootLogger.*`, `root.*` | Root logger definition。 |

`logger.<id>.name=<actual.logger.name>` 和 `appender.<id>.name=<actualName>` 会作为后续 properties 的 alias。

## TOML 格式

TOML 使用与 YAML 和 JSON 相同的结构化模型。使用 dotted tables 表达 appenders、layouts、filters、root 和 named loggers：

```toml
[configuration]
monitorInterval = "30s"

[configuration.properties]
LOG_DIR = "logs"

[configuration.appenders.console]
type = "console"
target = "stderr"

[configuration.appenders.console.layout]
type = "pattern"
pattern = "%d %5p %pid --- [%thread] %c : %m%attrs%n"

[configuration.appenders.json]
type = "json"
fileName = "${LOG_DIR}/app.json"
bufferSize = "256KiB"

[configuration.root]
level = "info"
appenderRefs = ["console", "json"]

[configuration.loggers."goark.orm"]
level = "debug"
appenderRefs = ["json"]
additivity = false
```

Logger 名称包含点号时，table name 必须加引号，例如 `[configuration.loggers."goark.orm"]`。Duration 和 byte-size 值建议写为字符串，例如 `"30s"` 和 `"256KiB"`。

## XML Format

XML 支持 Log4j2-style names：

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

XML console target aliases：`SYSTEM_OUT`、`STDOUT`、`SYSTEM_ERR`、`STDERR`。

XML parser 为 `<Http>`、`<Socket>` 和 `<Syslog>` 保留 element slots，方便外部插件模块复用配置形状。core registry 不注册 HTTP、Socket 或 Syslog network appenders。

## Reload

以下条件同时满足时，`LoggerContext` 会启动 file polling：

- 配置来自实际文件；
- `monitorInterval` 解析为正 duration；
- 使用 `NewConfiguredLoggerContext`。

Reload 会先构建完整的新 runtime snapshot。新配置失败时，旧 runtime 继续工作。

Reload 可以改变：

- levels；
- filters；
- appenders；
- layouts；
- logger routes；
- 新文件中解析出的 properties 和 lookups。

Reload 不能改变 Handler-level async runtime settings。要改变 queue shape 或启用/关闭 async logging，需要重启 logger context。

始终调用 `Handler` 或 `LoggerContext` 的 `Close`，确保 buffers 和 async queues 被 drain。
