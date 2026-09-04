# goark-log

[English](README.md)

`goark-log` 是面向生产环境 Go 服务的日志框架。它把标准库 `log/slog`
作为第一入口，同时补齐大型服务通常从 Log4j2 和 SLF4J 期待的运行期能力：
命名 logger 层级、appender 引用、结构化 filter、滚动文件、JSON Template
layout、有界异步队列、配置重载、状态事件和显式插件注册。

模块路径：

```bash
go get goark.dev/log
```

模块要求 Go 1.25 或更高版本。

所有 JSON 编码、解码、流式处理和原始消息处理统一使用字节跳动 Sonic。
新 Go 工具链或次要架构上也不会回退到 `encoding/json`。

## 已提供能力

| 范围 | 当前实现 |
| --- | --- |
| 标准 API | `slog.Handler`、`slog.Logger`、`WithAttrs`、`WithGroup`、`LogAttrs`，以及通过 `WithName` 或 `NewLogger` 使用命名 logger。 |
| 原生 API | 低分配 `Logger`、固定三属性快速路径、链式 `LogBuilder`、参数化消息、Map 消息、结构化数据消息、marker、逻辑线程名、context stack 和 throwable 快照。 |
| 配置 | YAML、JSON、TOML、Log4j2 风格 XML 和 Java properties。支持顶层、`configuration`、`goark.log` 三种包装。 |
| 路由 | root logger、最长前缀命名 logger 规则、additivity、appender-ref 级别阈值、appender-ref filter 和按引用控制调用位置。 |
| Appender | Console、File、JSON direct、RollingFile、Async、Failover、Routing、Rewrite。 |
| Layout | Pattern、Text、JSON、JSON Template、XML、CSV、GELF、RFC5424/Syslog 文本、YAML、HTML 行。 |
| Filter | Threshold、Level、LevelRange、Regex、Attr、Marker、NoMarker、Map、ThreadContextMap、ThreadContextStack、StructuredData、Throwable、StringMatch、Time、Burst、DynamicThreshold、Deny、Composite。 |
| 异步 | Handler 层 async logger 和 appender 层 async 队列，支持有界 ring buffer、批量写出、队列满策略、等待策略、计数器和关闭 drain。 |
| 滚动文件 | size、interval、cron、startup rollover、`%d{...}`/`%i` pattern、索引模式、gzip、保留数量、保留时间、删除动作和异步归档动作。 |
| 扩展 | appender、layout、filter、lookup、JSON Template resolver 的显式插件注册表；生成器位于 `cmd/goark-log-plugin-gen`。 |

核心模块不内置 HTTP appender、socket appender、网络 syslog client、Kafka、
Pulsar、RabbitMQ、SMTP、数据库 sink、OpenTelemetry exporter、Prometheus exporter
或脚本运行时。这些能力应放在独立模块中，通过显式插件注册接入。

## 快速开始

```go
package main

import (
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	logger, handler := goarklog.NewDefault()
	defer handler.Close()

	logger = goarklog.WithName(logger, "goark.demo")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

默认输出写入 stdout，使用 Spring Boot 风格 pattern：

```text
2026-08-28T09:30:00.000+08:00  INFO 12345 --- [main] goark.demo : service started profile=dev
```

## 生产启动

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	loggerContext, result, err := goarklog.NewConfiguredLoggerContext(context.Background(),
		goarklog.WithConfigPath("conf/goark-log.yml"),
	)
	if err != nil {
		panic(err)
	}
	defer loggerContext.Close()

	logger := loggerContext.Logger("goark.http")
	logger.Info("logging configured", slog.String("source", string(result.Source)))
}
```

配置路径解析优先级：

1. `WithConfigPath`。
2. 环境变量 `GOARK_LOG_CONFIG`，或 `WithConfigEnvKey` 设置的键。
3. Boot 属性键 `goark.log.config`、`goark.logging.config`、`logging.config`。
4. `conf/goark-log.yml`、`.yaml`、`.json`、`.xml`、`.toml`、`.properties`。
5. 内置默认配置：stdout console，级别 `INFO`。

建议从 [docs/examples/production-service.yml](docs/examples/production-service.yml)
开始生产配置。该示例覆盖控制台诊断、异步滚动 JSON 文件、审计日志、健康检查过滤、保留策略和配置重载。

## 可运行 Demo

```bash
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

如果外部没有设置 `GOARK_LOG_DIR`，demo 会使用临时目录，不依赖任何外部服务。

## 文档地图

| 文档 | 用途 |
| --- | --- |
| [文档索引](docs/index.zh-CN.md) | 面向使用者、运维人员和插件作者的完整导航。 |
| [生产指南](docs/production-guide.zh-CN.md) | 生产启动、安全默认值、重载、关闭和部署说明。 |
| [配置模型](docs/configuration.zh-CN.md) | 格式规则、包装结构、发现顺序、lookup 语义和 reload 行为。 |
| [配置参考](docs/configuration-reference.zh-CN.md) | 完整字段、别名、类型、默认值和校验规则表。 |
| [编程 API](docs/api.zh-CN.md) | 公共构造函数、运行期类型、原生 logger、消息、context 和 status API。 |
| [Appender](docs/appenders.zh-CN.md) | Appender 行为、配置字段、所有权和关闭语义。 |
| [Layout](docs/layouts.zh-CN.md) | Layout 输出格式、pattern converter、JSON Template resolver 和生命周期标志。 |
| [Filter](docs/filters.zh-CN.md) | Filter 裁决、所有内置 filter、挂载位置和嵌套规则。 |
| [使用场景](docs/scenarios.zh-CN.md) | 常见日志场景的可复制配方。 |
| [Log4j2 与 SLF4J 对齐](docs/log4j2-slf4j-parity.zh-CN.md) | 兼容映射和 Go-native 差异。 |
| [扩展指南](docs/extensibility.zh-CN.md) | 插件注册表、生成 registrar 和外部模块边界。 |
| [能力边界](docs/capabilities.zh-CN.md) | 基于源码的能力矩阵和核心不支持边界。 |
| [性能](docs/performance.zh-CN.md) | Benchmark、热路径规则、压测检查和性能注意事项。 |
| [发布检查清单](docs/release-v0.0.2.zh-CN.md) | 下一版发布 gate。 |
| [GitHub 发版说明](docs/github-release-v0.0.2.zh-CN.md) | 可直接复制到 `v0.0.2` GitHub Release 的说明。 |
| [配置示例](docs/examples/README.zh-CN.md) | 可加载 YAML、TOML、XML 和 properties 示例。 |
| [可运行示例](examples/README.zh-CN.md) | Demo 命令和预期行为。 |

## 验证

发布前在当前 worktree 执行：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

对比 benchmark 位于独立模块：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

`dev` 是集成分支。只有发布检查清单在目标提交上通过后，才应从 `main` 打 release tag。
