# 配置参考

[English](configuration-reference.md)

这是当前核心模块的完整配置参考。YAML、JSON 和 TOML 共享同一个逻辑模型。
XML 和 properties 会通过下列格式专属键映射到该模型。

## 顶层模型

| 字段 | 别名 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `configuration` | 无 | object | 无 | 可选包装。不能和顶层字段或 `goark.log` 混用。 |
| `goark.log` | 无 | object | 无 | Goark boot 风格可选包装。不能和 `configuration` 混用。 |
| `status` | 无 | string | 无 | 为兼容性解析并执行 lookup。Status logger 行为通过 `NewStatusLogger` 控制。 |
| `monitorInterval` | `monitor-interval` | duration | `0` | 正数时启用 `LoggerContext` 轮询 reload。纯数字按秒解析。 |
| `properties` | 无 | string map | 空 | 文件本地变量，可通过 `${NAME}`、`${prop:NAME}`、`${property:NAME}` 使用。 |
| `customLevels` | `custom-levels` | string 到 int 字符串 map | 空 | 注册进程级自定义级别名。名称必须非空且不能是数字。 |
| `appenders` | 无 | appender map | 默认 console | appender 名称来自 map key。 |
| `filters` | 无 | filter map | 空 | filter 名称来自 map key。 |
| `filterRefs` | `filter-refs` | string array | 空 | 全局 filter 在级别判断前执行。`ACCEPT` 跳过级别过滤；`DENY` 丢弃事件。 |
| `asyncLogger` | `async-logger`、`async` | object | 禁用 | Handler 层异步日志管线。单个文件只能使用一个别名。 |
| `root` | 无 | logger object | `INFO` 到第一个 appender | root logger 规则。 |
| `loggers` | 无 | logger object map | 空 | 命名 logger 规则，使用最长前缀匹配。 |

## 值解析

| 值 | 支持形式 |
| --- | --- |
| Level | `ALL`、`TRACE`、`DEBUG`、`INFO`、`WARN`、`WARNING`、`ERROR`、`FATAL`、`OFF` 或整数。 |
| Duration | Go duration，例如 `500ms`、`2s`、`5m`；`monitorInterval` 也支持纯秒数。 |
| Byte size | `b`、`byte`、`bytes`、`k`、`kb`、`m`、`mb`、`g`、`gb`、`t`、`tb`、`ki`、`kib`、`mi`、`mib`、`gi`、`gib`、`ti`、`tib`。支持小数。 |
| Rolling interval | `off`、`none`、`disabled`、`minute`、`minutely`、`hour`、`hourly`、`day`、`daily`、Go duration，或 `2days` 等 day/minute/hour 文本。 |
| Rolling max age | `off`、`none`、`disabled`、`30d`、`30day`、`30days` 或 Go duration。 |
| File permissions | 八进制字符串，例如 `0644`；或 `rw-r-----` 形式。 |
| Boolean | Go 布尔解析器支持的值，包括 `true` 和 `false`。 |
| Cron | 5、6 或 7 字段。5 字段会补前导秒 `0`；7 字段的 year 必须为 `*` 或 `?`。支持月份和星期名称。 |

## Async Logger

| 字段 | 别名 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `enabled` | 无 | bool pointer | 禁用 | 省略表示禁用 async logger。 |
| `queueSize` | `queue-size` | int | 启用时为 `4096` | 会按 runtime 要求归一化为 ring-buffer 容量。配置值必须非负。 |
| `batchSize` | `batch-size` | int | 启用时为 `64` | 会被限制到不超过 queue size。配置值必须非负。 |
| `overflowStrategy` | `overflow-strategy` | string | `block` | `block`、`blocking`、`drop`、`discard`、`discard-newest`、`drop-debug`、`dropdebug`、`discard-debug`、`discarddebug`、`sync-fallback`、`sync`、`synchronous`、`synchronize`。 |
| `waitStrategy` | `wait-strategy` | string | `block` | `block`、`blocking`、`timeout`、`timeout-block`、`timeoutblocking`、`sleep`、`sleeping`、`yield`、`yielding`、`spin`、`busy-spin`、`busyspin`。 |
| `waitRetries` | `wait-retries` | int | `0` | 必须非负。 |
| `sleepTime` | `sleep-time` | duration | `0` | 供 sleep 类等待策略使用。非法值会校验失败。 |
| `timeout` | 无 | duration | `0` | 可选阻塞超时。非法值会校验失败。 |
| `includeLocation` | `include-location` | bool pointer | false | 为 handler 层 async 事件采集调用位置。reload 时不能改变。 |

## Logger 对象

| 字段 | 别名 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `level` | 无 | level | root 默认 `INFO`；命名 logger 省略时继承 root | 级别阈值。 |
| `appenderRefs` | `appender-refs`、`refs` | string array 或 object array | root 为空时使用第一个配置的 appender | 按名称引用 appender。 |
| `filters` | `filterRefs`、`filter-refs` | string array | 空 | 引用具名 filter。 |
| `additivity` | 无 | bool pointer | 命名 logger 默认为 true | `false` 时命名 logger 必须至少有一个 appender。 |
| `includeLocation` | `include-location` | bool pointer | false | 为需要 location 的 layout 或 appender ref 采集 caller PC。 |

Appender 引用对象：

| 字段 | 别名 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| `ref` | 无 | string | 必填 | 目标 appender 名称。 |
| `level` | 无 | level | 无 | 当前引用独有级别阈值。 |
| `includeLocation` | `include-location` | bool pointer | 继承 route | `false` 清除该引用的 caller PC；`true` 强制采集 location。 |
| `filters` | `filterRefs`、`filter-refs` | string array | 空 | 当前引用独有 filter。 |

## Appender 类型

| 类型 | 别名 | 内置行为 |
| --- | --- | --- |
| Console | `console` | 使用 layout 写 stderr 或 stdout。 |
| File | `file` | 写本地文件，支持 buffer、权限、header、footer。 |
| JSON direct | `json`、`jsonDirect`、`jsonWriter` | 直接编码单行 JSON。设置 `fileName` 时写文件，否则写 stdout/stderr。 |
| Rolling file | `rolling`、`rollingFile` | 本地滚动文件，支持 size/time/cron/startup policy 和归档动作。 |
| Async | `async` | 后台 worker 从队列取事件并写 delegate appender。 |
| Failover | `failover`、`failoverAppender` | 先写 primary，失败后按顺序尝试 failover。 |
| Routing | `routing`、`routingAppender` | 按事件属性选择下游 appender。 |
| Rewrite | `rewrite`、`rewriteAppender` | 委派前增加和删除属性。 |

Appender 通用字段：

| 字段 | 别名 | 内置使用者 | 说明 |
| --- | --- | --- | --- |
| `type` | 无 | 全部 | 配置 appender 必填。类型匹配忽略大小写、连字符和下划线。 |
| `layout` | 无 | console、file、rolling-file | 省略时默认 pattern layout。JSON direct 忽略该字段。 |
| `filters` | `filterRefs`、`filter-refs` | 全部 | 给 appender 包一层 filter chain。 |
| `target` | 无 | console、JSON direct | Console 接受 `stderr`、`stdout`；JSON direct 接受 `stderr`、`stdout` 或省略。 |
| `fileName` | `file-name`、`path` | file、JSON direct file、rolling-file | file 和 rolling-file 必填。JSON direct 可选。 |
| `bufferSize` | `buffer-size` | file、JSON direct file、rolling-file | `0` 禁用应用层 buffer。 |
| `flushOnWrite` | `flush-on-write` | file、JSON direct file、rolling-file | 每条事件后 flush buffered writer。 |
| `append` | 无 | file、rolling-file | 默认 true。`false` 打开时截断。 |
| `createOnDemand` | `create-on-demand` | file、rolling-file | 延迟到第一条事件再创建文件。 |
| `filePermissions` | `file-permissions` | file、rolling-file | 默认 `0644`。 |
| `appenderRefs` | `appender-refs`、`refs` | async、failover、rewrite | delegate 引用。async 支持对象引用。 |
| `primary` | `primary-ref` | failover | 主 delegate。 |
| `failovers` | `failover-refs` | failover | 有序 failover delegate。 |
| `routeKey` | `route-key` | routing | 用作路由键的事件属性。省略时代码默认值为 `route`。 |
| `defaultRoute` | `default-route` | routing | route key 缺失或未命中时使用。 |
| `routes` | 无 | routing | route key 到 appender name 的 map。 |
| `rewrite` | 无 | rewrite | 内置属性 rewrite 策略。 |
| `queueSize` | `queue-size` | async | Appender 队列大小；运行期必须大于零。 |
| `batchSize` | `batch-size` | async | 后台批量大小；运行期必须大于零。 |
| `overflowStrategy` | `overflow-strategy` | async | 取值同 async logger。 |
| `waitStrategy` | `wait-strategy` | async | 取值同 async logger。 |
| `waitRetries` | `wait-retries` | async | 可选等待参数。 |
| `sleepTime` | `sleep-time` | async | 可选等待参数。 |
| `timeout` | 无 | async | 可选等待参数。 |
| `url`、`method`、`address`、`network`、`facility`、`appName`、`app-name`、`connectTimeout`、`connect-timeout`、`writeTimeout`、`write-timeout` | 无 | 外部插件 | 会解析并传给 `AppenderBuildConfig`；当前内置 appender 不实现远程 sink。 |

Rewrite 对象：

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `attrs` | `attributes`、`properties` | string map | 按 key 排序后追加属性。 |
| `remove` | `removeAttrs`、`remove-attrs` | string array | 删除匹配事件属性 key。 |

## Rolling 对象

Rolling 对象只用于 `rolling-file`。

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `filePattern` | `file-pattern` | string | 归档路径 pattern。支持 `%d{...}` 和 `%i`。`.gz` 后缀会启用 gzip。 |
| `maxSize` | `max-size` | byte size | 旧式 size policy。建议使用 `policies.size.size`。 |
| `interval` | 无 | interval | 旧式 time policy。建议使用 `policies.time`。 |
| `cron` | `cronSchedule`、`cron-schedule` | cron | 旧式 cron policy。建议使用 `policies.cron`。 |
| `onStartup` | `on-startup` | bool | 旧式 startup policy。建议使用 `policies.startup.enabled`。 |
| `maxBackups` | `max-backups` | int pointer | 旧式保留数量。建议使用 `strategy.max`。 |
| `maxAge` | `max-age` | duration | 旧式保留时间。建议使用 `strategy.maxAge`。 |
| `gzip` | `compress` | bool | 启用 gzip 压缩。 |
| `directWrite` | `direct-write` | bool | 直接写入 `filePattern`；要求配置 `filePattern`，且拒绝 gzip。 |
| `asyncActions` | `async-actions` | bool | 压缩和删除动作由串行后台 worker 执行。 |
| `actionQueueSize` | `action-queue-size` | int | 异步 rolling action 队列大小；默认 `32`。 |
| `policies` | 无 | object | Log4j2 风格触发策略。 |
| `strategy` | 无 | object | Log4j2 风格滚动策略。 |

Rolling policies：

| 字段 | 别名 | 子字段 |
| --- | --- | --- |
| `policies.size` | `size-based-triggering-policy`、`sizeBasedTriggeringPolicy`、`SizeBasedTriggeringPolicy` | `size`、`maxSize`、`max-size` |
| `policies.time` | `time-based-triggering-policy`、`timeBasedTriggeringPolicy`、`TimeBasedTriggeringPolicy` | `interval`、`every`、`unit`、`modulate` |
| `policies.cron` | `cron-triggering-policy`、`cronTriggeringPolicy`、`CronTriggeringPolicy` | `schedule`、`cron`、`cronSchedule`、`cron-schedule`、`evaluateOnStartup` |
| `policies.startup` | `on-startup-triggering-policy`、`onStartupTriggeringPolicy`、`OnStartupTriggeringPolicy` | `enabled` |

Rolling strategy：

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `type` | 无 | string | `directWrite`、`direct-write`、`directWriteRolloverStrategy` 会启用 direct write。 |
| `max` | `maxBackups`、`max-backups` | int pointer | 归档数量保留。 |
| `maxAge` | `max-age` | duration | 归档时间保留。 |
| `fileIndex` | `file-index` | string | `nomax`、`no-max`、`none`、`max` 或 `min`。 |
| `directWrite` | `direct-write` | bool | 启用 direct write。 |
| `asyncActions` | `async-actions` | bool | 启用后台归档动作。 |
| `actionQueueSize` | `action-queue-size` | int | 后台 action 队列大小。 |
| `compression.gzip` | `compression.compress` | bool | 启用 gzip。 |
| `compression.async` | 无 | bool | 同时启用后台归档动作。 |
| `delete` | 无 | object | 单个删除动作。 |
| `deleteActions` | `delete-actions` | object array | 额外删除动作。 |

Delete action：

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `basePath` | `base-path` | string | 扫描目录。默认归档 pattern 所在目录。 |
| `maxDepth` | `max-depth` | int pointer | 默认 `1`，必须非负。 |
| `maxCount` | `max-count`、`ifAccumulatedFileCount.exceeds`、`if-accumulated-file-count.exceeds` | int pointer | 保留最新文件直到数量上限。 |
| `maxSize` | `max-size`、`ifAccumulatedFileSize.exceeds`、`if-accumulated-file-size.exceeds` | byte size | 保留最新文件直到累计大小超过上限。 |
| `glob` | `ifFileName.glob`、`if-file-name.glob` | glob | 匹配文件名或相对路径。默认 `*`。 |
| `age` | `ifLastModified.age`、`if-last-modified.age` | duration | 删除超过指定时间的文件。 |
| `async` | 无 | bool | 出现在 `delete`、`deleteActions` 或 `delete-actions` 时，会启用后台归档动作。 |

## Layout 类型

| 类型 | 别名 | 输出 |
| --- | --- | --- |
| Pattern | `pattern` | Log4j 风格 converter 文本 pattern。 |
| Text | `text` | 稳定 `key=value` 文本。 |
| JSON | `json` | 事件 JSON object。 |
| JSON Template | `jsonTemplate` | resolver 模板控制的事件 JSON。 |
| XML | `xml`、`xmlLayout` | 单个 `<Event>` XML 片段。 |
| CSV | `csv`、`csvLayout` | 固定字段 CSV 行。 |
| GELF | `gelf`、`gelfLayout` | Graylog Extended Log Format JSON。 |
| RFC5424 | `rfc5424`、`rfc5424Layout` | RFC 5424 syslog 文本行。 |
| Syslog | `syslog`、`syslogLayout` | RFC5424 layout 别名。 |
| YAML | `yaml`、`yamlLayout` | 单个 YAML 文档。 |
| HTML | `html`、`htmlLayout` | HTML 表格行。 |

Layout 字段：

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `type` | 无 | string | layout type 省略时默认 pattern。 |
| `pattern` | 无 | string | 仅 pattern layout 使用。空 pattern 使用 `DefaultSpringBootPattern`。 |
| `eventTemplate` | `event-template` | string | JSON Template 内联 object。 |
| `eventTemplateUri` | `event-template-uri`、`eventTemplatePath`、`event-template-path` | string | JSON Template 文件路径。 |
| `compact` | 无 | bool | 禁用默认事件换行。 |
| `eventEol` | `event-eol` | bool | 即使 compact stream 也追加事件换行。 |
| `complete` | 无 | bool | 写 layout header/footer。JSON 和 JSON Template 会按 appender 隔离 complete-mode 状态。 |
| `includeStacktrace` | `include-stacktrace` | bool | 支持异常结构的 layout 输出结构化 stack。 |
| `stacktraceAsString` | `stacktrace-as-string` | bool | 支持的 layout 输出单字符串 throwable stack。 |
| `propertiesAsList` | `properties-as-list` | bool | 支持的 layout 将 context map 输出为 key/value list。 |
| `includeNullDelimiter` | `include-null-delimiter` | bool | 每条事件后追加 NUL。 |
| `disableAnsi` | `disable-ansi` | bool | 禁用 PatternLayout `highlight` 和 `style` 的 ANSI 输出。 |
| `header` | 无 | string | complete/lifecycle layout 的 header。 |
| `footer` | 无 | string | complete/lifecycle layout 的 footer。 |

## Filter 类型

| 类型 | 别名 | 必填字段 | 行为 |
| --- | --- | --- | --- |
| Threshold | `threshold`、`thresholdFilter` | `level` | 匹配大于等于 level 的事件。 |
| Level | `level`、`levelFilter` | `level` | 精确匹配单个 level。 |
| LevelRange | `levelRange`、`levelRangeFilter` | `minLevel`、`maxLevel` | 闭区间匹配。 |
| Regex | `regex`、`regexFilter` | `pattern`；`field=attr` 时还需要 `key` | 匹配 message、logger 或 attribute。 |
| Attr | `attr`、`attribute`、`attrFilter`、`attributeFilter` | `key` | 匹配属性 key 和精确字符串值。 |
| Deny | `deny`、`denyAll`、`denyFilter`、`denyAllFilter` | 无 | 无条件拒绝。 |
| Composite | `composite`、`compositeFilter` | `filterRefs` | 按顺序执行嵌套 filter。 |
| Marker | `marker`、`markerFilter` | `marker` 或 `value` | 匹配 marker 名称或父级 marker。 |
| NoMarker | `noMarker`、`noMarkerFilter` | 无 | 匹配没有 marker 的事件。 |
| Map | `map`、`mapFilter` | `values`、`key/value` 或 key-value pairs | 按 `and` 或 `or` 匹配事件属性。 |
| ThreadContextMap | `threadContextMap`、`threadContextMapFilter` | map values | MDC 风格数据的 MapFilter 语义别名。 |
| ThreadContextStack | `threadContextStack`、`threadContextStackFilter` | `value`、`text` 或 `pattern` | 匹配 context stack 值。 |
| StructuredData | `structuredData`、`structuredDataFilter` | map values | 结构化数据属性的 MapFilter 语义别名。 |
| Throwable | `throwable`、`throwableFilter` | `pattern`、`text` 或 `value` | 正则匹配 throwable 文本。 |
| StringMatch | `stringMatch`、`stringMatchFilter` | `text`、`value` 或 `pattern` | 对 message 做子串匹配。 |
| Time | `time`、`timeFilter` | 可选 | 匹配一天内时间区间。默认全天。 |
| Burst | `burst`、`burstFilter` | 可选 | 对小于等于指定级别的事件做令牌桶限流。默认 level `warn`、rate `10`、maxBurst `rate*10`。 |
| DynamicThreshold | `dynamicThreshold`、`dynamicThresholdFilter` | `key` | 按事件属性值选择阈值。 |

Filter 字段：

| 字段 | 别名 | 类型 | 说明 |
| --- | --- | --- | --- |
| `type` | 无 | string | 必填。匹配忽略大小写、连字符和下划线。 |
| `level` | 无 | level | Threshold、Level、Burst 默认值或 DynamicThreshold 默认 fallback。 |
| `minLevel` | `min-level` | level | LevelRange 最小值。 |
| `maxLevel` | `max-level` | level | LevelRange 最大值。 |
| `marker` | 无 | string | Marker 名称。 |
| `text` | 无 | string | StringMatch、Throwable 或 ThreadContextStack 值。 |
| `operator` | 无 | string | Map operator：`and` 或 `or`。 |
| `start` | 无 | time-of-day | TimeFilter 开始时间，支持 `HH:MM`、`HH:MM:SS` 或纳秒小数。默认 `00:00:00`。 |
| `end` | 无 | time-of-day | TimeFilter 结束时间。默认 `23:59:59.999999999`。 |
| `timezone` | 无 | IANA zone | TimeFilter 可选时区。 |
| `rate` | 无 | float string | Burst 每秒速率。 |
| `maxBurst` | `max-burst` | int | Burst 最大令牌数。 |
| `field` | 无 | string | Regex 字段：`message`、`msg`、`logger`、`name`、`attr`、`attribute`。 |
| `key` | 无 | string | 属性 key、map 单 key、dynamic threshold key 或 regex attr key。 |
| `value` | 无 | string | 属性值、marker fallback、string fallback、throwable fallback 或 thread stack fallback。 |
| `values` | 无 | string map | Map 类 filter 使用。 |
| `thresholds` | 无 | string 到 level 字符串 map | DynamicThreshold 取值。 |
| `filters` | `filterRefs`、`filter-refs` | string array | Composite 或构建时需要嵌套引用的 filter 使用。 |
| `KeyValuePair` | `keyValuePairs`、`key-value-pairs` | `{key,value}` array | 根据 filter 类型补充 map values 或 dynamic thresholds。 |
| `defaultThreshold` | `default-threshold` | level | DynamicThreshold 默认阈值。 |
| `pattern` | 无 | string | Regex、Throwable、StringMatch 或 ThreadContextStack fallback。 |
| `onMatch` | `on-match` | decision | `neutral`、`accept`、`deny`。默认 `neutral`。 |
| `onMismatch` | `on-mismatch` | decision | `neutral`、`accept`、`deny`。默认 `deny`。 |

## XML 映射

XML root：

| 元素或属性 | 映射 |
| --- | --- |
| `<Configuration status="" monitorInterval="">` | `status`、`monitorInterval` |
| `<Properties><Property name="" value="">text</Property></Properties>` | `properties` |
| `<CustomLevels><CustomLevel name="" intLevel="" value="">text</CustomLevel></CustomLevels>` | `customLevels` |
| `<AsyncLogger .../>` | `asyncLogger` |
| `<Loggers><Root>...</Root><Logger name="">...</Logger></Loggers>` | `root`、`loggers` |

`<Appenders>` 下的 XML appender：

| 元素 | 说明 |
| --- | --- |
| `<Console>` | 内置 console。`target` 支持 `SYSTEM_OUT`、`STDOUT`、`SYSTEM_ERR`、`STDERR`。 |
| `<File>` | 内置 file。 |
| `<RollingFile>` | 内置 rolling file。 |
| `<Async>` | 内置 async appender。 |
| `<Failover>` | 内置 failover。支持 `<Failovers><AppenderRef ref="..."/></Failovers>`。 |
| `<Routing>` | 内置 routing。支持 `<Route key="" ref="">` 或嵌套 `<AppenderRef>`。 |
| `<Rewrite>` | 内置 rewrite。使用 `<KeyValuePair key="" value="">` 和 `<Remove key="">`。 |
| `<Http>`、`<Socket>`、`<Syslog>` | 如果注册了插件会传给插件；核心不提供这些 appender factory。 |

Appender 下的 XML layout：`<PatternLayout>`、`<TextLayout>`、`<JsonLayout>`、
`<JSONLayout>`、`<JsonTemplateLayout>`、`<XmlLayout>`、`<XMLLayout>`、
`<CsvLayout>`、`<CSVLayout>`、`<GelfLayout>`、`<GELFLayout>`、
`<Rfc5424Layout>`、`<RFC5424Layout>`、`<SyslogLayout>`、`<YamlLayout>`、
`<YAMLLayout>`、`<HtmlLayout>`、`<HTMLLayout>` 或通用 `<Layout type="">`。

`<Filters>` 下的 XML filter：`<ThresholdFilter>`、`<LevelFilter>`、
`<LevelRangeFilter>`、`<RegexFilter>`、`<AttrFilter>`、`<AttributeFilter>`、
`<DenyFilter>`、`<DenyAllFilter>`、`<CompositeFilter>`、`<MarkerFilter>`、
`<NoMarkerFilter>`、`<MapFilter>`、`<ThreadContextMapFilter>`、
`<ThreadContextStackFilter>`、`<StructuredDataFilter>`、`<ThrowableFilter>`、
`<StringMatchFilter>`、`<TimeFilter>`、`<BurstFilter>`、`<DynamicThresholdFilter>`。

## Properties 映射

顶层 key：

| Key | 映射 |
| --- | --- |
| `status` | `status` |
| `monitorInterval`、`monitor-interval` | `monitorInterval` |
| `property.NAME` | `properties.NAME` |
| `customLevel.NAME`、`custom-level.NAME` | `customLevels.NAME` |
| `rootLogger.level`、`root.level` | `root.level` |
| `rootLogger.appenderRefs`、`root.appenderRefs` | `root.appenderRefs` |
| `rootLogger.filters`、`root.filters` | `root.filters` |
| `rootLogger.includeLocation`、`rootLogger.include-location`、`root.includeLocation`、`root.include-location` | `root.includeLocation` |
| `rootLogger.appenderRef.ID.FIELD`、`root.appenderRef.ID.FIELD` | 结构化 root appender ref |
| `asyncLogger.FIELD`、`async-logger.FIELD`、`async.FIELD` | async logger |
| `appender.ID.FIELD` | appender |
| `logger.ID.FIELD` | 命名 logger |
| `filter.ID.FIELD` | filter |

Properties 专属 appender 字段包含所有通用 appender 字段，并包括：

| Key pattern | 说明 |
| --- | --- |
| `appender.ID.name` | 将 appender 从逻辑 ID 重命名。 |
| `appender.ID.layout.FIELD` | layout 字段。 |
| `appender.ID.routes.ROUTE_KEY` | routing route map 条目。 |
| `appender.ID.rewrite.attrs.KEY`、`.attributes.KEY`、`.properties.KEY` | rewrite 追加属性。 |
| `appender.ID.rewrite.remove`、`.removeAttrs`、`.remove-attrs` | rewrite 删除属性。 |
| `appender.ID.appenderRef.REF_ID.FIELD` | composite appender 的结构化 appender ref。 |
| `appender.ID.routeKey`、`.route-key`、`.attrKey`、`.attr-key` | routing attr key。 |

Properties 当前映射的 rolling key：

| Key | 映射 |
| --- | --- |
| `rolling.filePattern`、`rolling.file-pattern` | rolling `filePattern` |
| `rolling.maxSize`、`rolling.max-size` | rolling `maxSize` |
| `rolling.interval` | rolling `interval` |
| `rolling.cron`、`rolling.cronSchedule`、`rolling.cron-schedule`、`rolling.policies.cron.schedule`、`rolling.policies.cronTriggeringPolicy.schedule`、`rolling.policies.cron-triggering-policy.schedule` | rolling cron schedule |
| `rolling.strategy.type` | rolling strategy type |
| `rolling.strategy.fileIndex`、`rolling.strategy.file-index` | rolling strategy file index |
| `rolling.directWrite`、`rolling.direct-write`、`rolling.strategy.directWrite`、`rolling.strategy.direct-write` | direct write |
| `rolling.strategy.delete.maxCount`、`.max-count` | delete max count |
| `rolling.strategy.delete.maxSize`、`.max-size` | delete max size |
| `rolling.strategy.delete.ifAccumulatedFileCount.exceeds`、`.if-accumulated-file-count.exceeds` | delete max count |
| `rolling.strategy.delete.ifAccumulatedFileSize.exceeds`、`.if-accumulated-file-size.exceeds` | delete max size |

Properties logger 字段：

| Key | 说明 |
| --- | --- |
| `logger.ID.name` | 将 logger 从逻辑 ID 重命名。 |
| `logger.ID.level` | 级别。 |
| `logger.ID.appenderRefs`、`.appender-refs`、`.refs` | delegate refs。 |
| `logger.ID.filters`、`.filterRefs`、`.filter-refs` | filter refs。 |
| `logger.ID.additivity` | 布尔值。 |
| `logger.ID.includeLocation`、`.include-location` | 布尔值。 |
| `logger.ID.appenderRef.REF_ID.FIELD` | 结构化 appender ref。 |

Properties filter key-value pairs 使用：

```properties
filter.myFilter.keyValuePair.1.key=tenant
filter.myFilter.keyValuePair.1.value=tenant-a
```

pair ID 可以以 `keyValuePair` 或 `kv` 开头。
