# 能力矩阵

[English](capabilities.md)

本矩阵描述当前核心模块，不描述未来配套模块。用它判断功能是核心直接可用、需要扩展边界、
还是有意不放入核心。

## 运行时 API

| 能力 | 状态 | 说明 |
| --- | --- | --- |
| `slog.Handler` 实现 | 内置 | 支持标准 `slog.Logger`、`WithAttrs`、`WithGroup` 和 `LogAttrs`。 |
| 命名 logger | 内置 | `NewLogger`、`WithName` 和 `LoggerContext.Logger`。 |
| 原生 logger | 内置 | 低分配 builder、固定三属性路径和 `slog` 互操作。 |
| 参数化消息 | 内置 | 通过 `ParameterizedMessageFactory` 支持 `{}` 占位符。 |
| Map 与结构化数据消息 | 内置 | 消息属性同时对 layout 和 filter 可见。 |
| Marker | 内置 | 支持父 marker 匹配。 |
| MDC 风格 context 属性 | 内置 | `WithContextAttrs` 和 pattern `%X{}` / JSON Template `mdc`。 |
| NDC 风格 context stack | 内置 | `WithContextStack`、`%ndc` 和 JSON Template `contextStack`。 |
| Throwable snapshot | 内置 | Go error 可选择捕获 stack。 |
| Status logger | 内置 | 内部配置和重载事件。 |

## 配置

| 能力 | 状态 | 说明 |
| --- | --- | --- |
| YAML | 内置 | 结构化解码启用严格字段检查。 |
| JSON | 内置 | 与 YAML 共享逻辑模型。 |
| TOML | 内置 | 与 YAML 共享逻辑模型。 |
| XML | 内置 | Log4j2 风格根元素和子元素。 |
| Java properties | 内置 | 实用 key 映射；rolling policy 覆盖少于 YAML/TOML/XML。 |
| 包装层 | 内置 | 顶层、`configuration` 或 `goark.log`；不能混用。 |
| 配置发现 | 内置 | 显式路径、环境变量、boot property、默认文件、内置默认。 |
| Lookup 展开 | 内置且可扩展 | 内置 `env`、`sys`、`go`、`date`，以及文件 `prop` / `property`。 |
| 阻断 lookup namespace | 内置 | `jndi`、`ldap`、`rmi` 被阻断。 |
| 重载 | 内置 | 显式 `ConfigReloader` 和 `LoggerContext` 基于 `monitorInterval` 的轮询。 |

## Appenders

| Appender | 状态 | 说明 |
| --- | --- | --- |
| Console | 内置 | stdout/stderr，支持 layout。 |
| File | 内置 | 缓冲、权限、追加/截断、按需创建、header、footer。 |
| JSON direct | 内置 | stdout/stderr 或文件，优化事件 JSON 路径。 |
| Rolling file | 内置 | size/time/cron/startup、gzip、保留策略、删除动作、异步归档动作。 |
| Async | 内置 | appender 级队列代理下游 appender。 |
| Failover | 内置 | primary 加有序 failover。 |
| Routing | 内置 | 按事件属性选路，支持默认 route。 |
| Rewrite | 内置 | 委托前添加和删除属性。 |
| HTTP | 插件边界 | 字段可传给已注册插件；核心无 client。 |
| Socket | 插件边界 | 字段可传给已注册插件；核心无 client。 |
| 网络 syslog | 插件边界 | 核心提供 RFC5424/syslog layout，不提供网络 client。 |
| Kafka、Pulsar、RabbitMQ | 插件边界 | Broker 依赖留在核心之外。 |
| SMTP、数据库 sink | 插件边界 | 作为外部 appender 模块实现。 |

## Layouts

| Layout | 状态 | 说明 |
| --- | --- | --- |
| Pattern | 内置 | Log4j 风格 converter 和 ANSI style/highlight。 |
| Text | 内置 | 稳定文本 key/value 输出。 |
| JSON | 内置 | 结构化事件 JSON，带生命周期选项。 |
| JSON Template | 内置且可扩展 | 内置 resolver 加插件 resolver registry。 |
| XML | 内置 | 单事件 XML fragment。 |
| CSV | 内置 | 固定事件 CSV 行。 |
| GELF | 内置 | Graylog Extended Log Format JSON。 |
| RFC5424 | 内置 | Syslog 文本行 layout。 |
| Syslog layout | 内置 | RFC5424 layout 的别名。 |
| YAML | 内置 | 单事件 YAML document。 |
| HTML | 内置 | HTML table row。 |

## Filters

| Filter 家族 | 状态 | 说明 |
| --- | --- | --- |
| Threshold、level、level range | 内置 | 日志级别门控。 |
| Regex 和 string match | 内置 | message/logger/attr 或子串匹配。 |
| Attr、map、thread context map、structured data | 内置 | 属性 key/value 匹配。 |
| Marker 和 no-marker | 内置 | marker 存在性和层级匹配。 |
| Thread context stack | 内置 | NDC 风格 stack 匹配。 |
| Throwable | 内置 | throwable 和 error 属性匹配。 |
| Time | 内置 | 一天内时间区间，可选 IANA 时区。 |
| Burst | 内置 | 低严重度事件的 token bucket 限流。 |
| Dynamic threshold | 内置 | 按属性选择阈值。 |
| Deny 和 composite | 内置 | 恒定拒绝和嵌套链。 |
| Script filter | 仅 Go API | 需要调用方提供 evaluator；核心无配置化脚本运行时。 |

## 滚动文件

| 能力 | 状态 | 说明 |
| --- | --- | --- |
| Size policy | 内置 | 启用时 `filePattern` 必须包含 `%i`。 |
| Time policy | 内置 | 支持 interval 和 modulate。 |
| Cron policy | 内置 | 支持 5、6、7 字段；year 字段必须是通配形式。 |
| Startup policy | 内置 | 可选启动时滚动。 |
| Gzip 归档 | 内置 | 用于归档文件；direct write 不允许 gzip。 |
| 最大归档数量 | 内置 | `strategy.max` 或 legacy `maxBackups`。 |
| 最大归档年龄 | 内置 | `strategy.maxAge` 或 legacy `maxAge`。 |
| 删除动作 | 内置 | 路径深度、glob、年龄、累计数量和累计大小。 |
| 异步归档动作 | 内置 | 压缩和删除使用串行后台 worker。 |
| Direct write strategy | 内置 | 需要 `filePattern`；拒绝 gzip。 |

## 生产边界

| 关注点 | 当前支持 |
| --- | --- |
| 背压 | Handler 级和 appender 级异步支持 block、drop、drop-debug、sync-fallback。 |
| 关闭 | Handler 和 logger context close 会 drain 队列并关闭 appender。 |
| Caller location 成本 | 只有 logger/options/route 需要时才捕获。 |
| 热点格式化 | JSON direct 和原生 logger 快路径已存在；性能结论必须基于当前 benchmark。 |
| Observability exporter | 插件边界；核心没有 OpenTelemetry 或 Prometheus exporter。 |
| 远程投递重试 | 远程 appender 插件边界。 |
