# 可运行示例

[English](README.md)

这些示例是当前 `goark.dev/log` API 和配置模型的生产形态 smoke demo，不依赖外部服务。

## 全部运行

```bash
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

## Demo 列表

| 目录 | 展示内容 |
| --- | --- |
| [console](console) | `ConfigureDefault`、命名 `slog` logger 和最小控制台配置。 |
| [file](file) | 配置化文件输出和完整 JSON layout 生命周期。 |
| [rolling](rolling) | 原生 logger 快路径通过生产滚动配置写入。 |
| [async](async) | Appender 级异步队列、failover 链和异步计数器。 |
| [reload](reload) | 显式 `ConfigReloader` 改变日志级别。 |
| [extensibility](extensibility) | 隔离插件 registry、自定义 lookup、自定义 JSON Template resolver 和 message factory。 |
| [production](production) | 生产形态服务日志，包含 MDC、NDC、marker、审计、健康检查过滤、throwable stack、滚动文件和 async appender。 |
| [slf4j](slf4j) | SLF4J 风格参数化日志和标准 `slog` 互操作。 |
| [log4j2_config](log4j2_config) | Log4j2 风格 XML 配置，包含 rolling、async fan-out、routing、rewrite、filter 和命名 logger。 |

## 日志目录

写文件 demo 调用 `examples/internal/exampleutil.PrepareLogDir`。

如果设置了 `GOARK_LOG_DIR`，会使用该目录：

```bash
GOARK_LOG_DIR=/tmp/goark-log-demo GOWORK=off go run ./examples/production
```

如果没有设置，demo 会创建临时目录并打印 `logDir=...`。临时目录会在 demo 退出时删除。

## 配置来源

Demo 从 [../docs/examples](../docs/examples) 加载文件。该目录也被集成测试覆盖，因此示例
和文档共用同一个事实来源。

## Smoke Test

模块里没有把所有 `go run` demo 串起来的单一命令。release candidate 需要运行
"全部运行" 中的命令，然后运行：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
```
