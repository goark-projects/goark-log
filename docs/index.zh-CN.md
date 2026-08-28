# goark-log 文档

[English](index.md)

本文档基于当前 `goark.dev/log` 源码编写。公共默认语言为英文；每个公开
Markdown 页面都提供简体中文版本。

## 先读这里

| 需求 | 文档 |
| --- | --- |
| 一分钟内安装并跑通 logger | [README](../README.zh-CN.md) |
| 使用生产级服务配置 | [生产指南](production-guide.zh-CN.md) |
| 理解配置发现和包装结构 | [配置模型](configuration.zh-CN.md) |
| 查询每个支持字段和别名 | [配置参考](configuration-reference.zh-CN.md) |
| 迁移 Log4j2 或 SLF4J 用法 | [Log4j2 与 SLF4J 对齐](log4j2-slf4j-parity.zh-CN.md) |

## 参考文档

| 范围 | 文档 |
| --- | --- |
| 公共 Go API | [编程 API](api.zh-CN.md) |
| Appender 行为和字段 | [Appender](appenders.zh-CN.md) |
| 输出格式和 pattern 语法 | [Layout](layouts.zh-CN.md) |
| Filter 裁决和类型 | [Filter](filters.zh-CN.md) |
| 真实服务配方 | [使用场景](scenarios.zh-CN.md) |
| 插件和生成 registrar | [扩展](extensibility.zh-CN.md) |
| 已实现和不支持能力 | [能力边界](capabilities.zh-CN.md) |
| Benchmark 和热路径约束 | [性能](performance.zh-CN.md) |
| 发布验证 | [v0.0.2 检查清单](release-v0.0.2.zh-CN.md) |
| GitHub Release 文案 | [v0.0.2 发版说明](github-release-v0.0.2.zh-CN.md) |

## 示例

| 示例集合 | 内容 |
| --- | --- |
| [配置示例](examples/README.zh-CN.md) | 由测试加载的 YAML、TOML、XML 和 properties 文件。 |
| [可运行示例](../examples/README.zh-CN.md) | console、file、rolling、async、reload、插件、生产、SLF4J 风格和 Log4j2 风格 XML demo。 |

## 基于源码的边界

核心模块当前实现本地文件和控制台输出、JSON 直写、滚动文件、appender 组合、
layout、filter、配置、reload 和显式插件。

核心模块不实现 HTTP appender、socket appender、网络 syslog client、Kafka、
Pulsar、RabbitMQ、SMTP、数据库 sink、OpenTelemetry exporter、Prometheus exporter
或内嵌脚本执行。这些是有意保留给外部模块的边界。

## 验证命令

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
```

只有命令需要网络访问时才使用代理：

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```
