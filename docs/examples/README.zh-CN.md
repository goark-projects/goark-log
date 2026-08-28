# 配置示例

[English](README.md)

本目录下每个配置文件都会被 `TestDocsExamples_whenLoaded_shouldBuildOptions` 加载测试。
这些文件是当前可复制示例，不是历史兼容样本。

## 文件

| 文件 | 格式 | 场景 |
| --- | --- | --- |
| [basic-console.yml](basic-console.yml) | YAML | 最小控制台 logger，使用 pattern layout。 |
| [container-json.yml](container-json.yml) | YAML | 容器采集使用 JSON direct stdout。 |
| [complete-json-file.yml](complete-json-file.yml) | YAML | 带 layout 生命周期的完整 JSON 文件流。 |
| [production-service.yml](production-service.yml) | YAML | 生产风格控制台、异步滚动业务日志、审计日志、filter 和重载周期。 |
| [audit-routing.yml](audit-routing.yml) | YAML | 按租户 routing，并用 rewrite 脱敏。 |
| [async-failover.yml](async-failover.yml) | YAML | 带 failover 链的 async appender。 |
| [filters-showcase.yml](filters-showcase.yml) | YAML | 所有内置配置化 filter 家族。 |
| [json-template.yml](json-template.yml) | YAML | JSON Template resolver 字段。 |
| [log4j2-service.xml](log4j2-service.xml) | XML | 使用 rolling、async fan-out、routing、rewrite、filter 和命名 logger 的 Log4j2 风格服务配置。 |
| [goark-log.toml](goark-log.toml) | TOML | 常见本地文件配置的 TOML 示例。 |
| [goark-log.properties](goark-log.properties) | properties | Java properties 映射示例。 |

## 加载文件

```go
loggerContext, result, err := goarklog.NewConfiguredLoggerContext(ctx,
	goarklog.WithConfigPath("docs/examples/production-service.yml"),
)
if err != nil {
	return err
}
defer loggerContext.Close()

loggerContext.Logger("goark.demo").Info("ready", slog.String("source", string(result.Source)))
```

## 环境变量

多数写文件示例使用 `GOARK_LOG_DIR`：

```bash
GOARK_LOG_DIR=/var/log/my-service GOWORK=off go run ./examples/production
```

未设置 `GOARK_LOG_DIR` 时，可运行 demo 会创建临时目录并打印 `logDir=...`。

## 格式说明

| 格式 | 说明 |
| --- | --- |
| YAML / JSON / TOML | 共享同一个逻辑模型。字段见[配置参考](../configuration-reference.zh-CN.md)。 |
| XML | 使用 Log4j2 风格元素。核心支持 `log4j2-service.xml` 中使用的元素。 |
| properties | 使用实用 Java properties 映射。部分高级 rolling 嵌套策略更适合用 YAML、TOML 或 XML 表达。 |

## 验证

```bash
GOWORK=off go test ./internal/integration -run TestDocsExamples -count=1
```
