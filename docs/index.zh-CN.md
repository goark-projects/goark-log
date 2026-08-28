# 文档索引

[English](index.md)

本目录是 `goark.dev/log` v0.0.2 准备阶段的参考文档。根 README 保持简洁；详细配置、运行场景和发布检查放在这里。

## 从这里开始

| 文档 | 目的 |
| --- | --- |
| [编程式 API](api.zh-CN.md) | Handler 构造、native logger、context attributes、reload、status logger 和 close ownership。 |
| [配置参考](configuration.zh-CN.md) | 完整配置模型、加载顺序、YAML/JSON/XML/properties 形式、lookups、levels、reload 和 routing。 |
| [Appender 参考](appenders.zh-CN.md) | Console、file、JSON、rolling file、async、failover、routing、rewrite 以及 plugin appender 参数。 |
| [Layout 参考](layouts.zh-CN.md) | Pattern、JSON、JSON Template、Text、XML、CSV、GELF、RFC5424/Syslog、YAML、HTML、converter 表和 resolver 表。 |
| [Filter 参考](filters.zh-CN.md) | Filter chain 语义、decision、内置 filters、参数和示例。 |
| [使用场景](scenarios.zh-CN.md) | 开发、容器、生产滚动日志、审计拆分、reload、routing、脱敏和扩展的可复制场景。 |
| [扩展指南](extensibility.zh-CN.md) | appenders、layouts、filters、lookups 和 JSON Template resolvers 的显式插件注册。 |
| [能力边界](capabilities.zh-CN.md) | core module 支持什么，哪些能力属于外部模块。 |
| [性能](performance.zh-CN.md) | 性能预算、benchmark 命令、pressure tests 和调优说明。 |
| [v0.0.2 发布检查清单](release-v0.0.2.zh-CN.md) | 发布 v0.0.2 前的本地和远程检查。 |

## 可复制配置示例

| 文件 | 场景 |
| --- | --- |
| [examples/README.zh-CN.md](examples/README.zh-CN.md) | 配置示例目录说明。 |
| [examples/console.yml](examples/console.yml) | 开发期人类可读 console 输出。 |
| [examples/json-stdout.yml](examples/json-stdout.yml) | 容器和 Kubernetes stdout JSON 日志。 |
| [examples/production-rolling.yml](examples/production-rolling.yml) | 带 gzip 和保留策略的生产 JSON rolling file。 |
| [examples/split-audit.yml](examples/split-audit.yml) | 应用日志和审计日志拆分。 |
| [examples/async-appender.yml](examples/async-appender.yml) | 只包装指定 sink 的 appender-level async。 |
| [examples/rewrite-routing.yml](examples/rewrite-routing.yml) | 属性 rewrite 和按 tenant routing。 |
| [examples/goark-log.properties](examples/goark-log.properties) | 适合简单部署的 properties 配置。 |
| [examples/goark-log.toml](examples/goark-log.toml) | 使用与 YAML/JSON 相同 structured model 的 TOML 配置。 |
| [examples/log4j2-style.xml](examples/log4j2-style.xml) | 使用 parser 支持的 Log4j2-style element names 的 XML 配置。 |

## 支持的格式

- YAML：推荐的服务配置格式。
- JSON：使用同一套 structured decoder 和字段名。
- XML：支持 Log4j2-style appender、layout、filter、policy、strategy 和 logger element。
- properties：使用 `appender.console.type`、`rootLogger.level` 等 flat keys。
- TOML：通过与 YAML/JSON 相同的 structured schema 支持。

## 非 Core 范围

core module 故意不包含 HTTP、Socket、Syslog network output、Kafka、SMTP、database sinks、OpenTelemetry、Prometheus 或 embedded script engines。这些集成需要 connection lifecycle、credentials、retry、batching 和 failure semantics，应放在独立模块并显式注册插件。
