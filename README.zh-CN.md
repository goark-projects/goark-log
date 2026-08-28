# goark-log

[English](README.md)

`goark-log` 是面向 Go 服务的高性能结构化日志框架。它基于标准库
`log/slog`，提供生产级 Handler 运行时、Appender、Layout、层级路由、
Filter、安全配置加载、有界异步队列、滚动文件和显式插件注册能力。

模块路径：

```bash
go get goark.dev/log
```

模块要求 Go 1.25 或更高版本。

## 设计目标

- Go-native 公共 API：使用显式构造函数、接口、选项和插件注册，不做运行时扫描。
- 热路径低分配：常见 JSON、文件直写、原生 logger 和 ring buffer 路径避免重反射编码。
- 核心依赖轻量：zap、zerolog 只放在独立的 `benchmarks/compare` 子模块。
- 安全默认值：核心不提供远程 lookup namespace，不内置脚本运行时，不内置外部系统 appender。
- 关闭行为可证明：异步 logger、异步 appender、滚动压缩和删除动作都会在 `Close` 时 drain。

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

	logger = goarklog.WithName(logger, "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

默认输出写入 stderr，格式是 Spring Boot 风格单行文本：

```text
2026-08-25T10:15:30.123+08:00  INFO 12345 --- [main] goark.boot : service started profile=dev
```

## 配置化启动

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/log"
)

func main() {
	handler, result, err := goarklog.ConfigureDefault(context.Background(),
		goarklog.WithConfigPath("conf/goark-log.yml"),
	)
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	slog.Info("logging configured", slog.String("source", string(result.Source)))
}
```

配置路径解析优先级：

1. `WithConfigPath` 显式路径。
2. `GOARK_LOG_CONFIG`，或 `WithConfigEnvKey` 指定的自定义环境变量。
3. Boot 属性：`goark.log.config`、`goark.logging.config`、`logging.config`。
4. 默认文件：`conf/goark-log.{yml,yaml,json,xml,toml,properties}`。
5. 内置默认配置：stderr console，`INFO` 级别。

当前支持 YAML、JSON、XML 和 properties。TOML 会明确报错，避免旧文件被误认为已经生效。

## 生产 YAML 示例

```yaml
configuration:
  monitorInterval: 30s
  properties:
    LOG_DIR: logs
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %pid --- [%thread] %c : %m%attrs%n"
  asyncLogger:
    enabled: true
    queueSize: 8192
    batchSize: 256
    overflowStrategy: block
    waitStrategy: yield
    includeLocation: false
  appenders:
    console:
      type: console
      target: stderr
      layout:
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
    rolling:
      type: rolling-file
      fileName: "${prop:LOG_DIR}/app.log"
      bufferSize: 256KiB
      layout:
        type: json
        eventEol: true
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%d{yyyyMMdd}-%i.log.gz"
        policies:
          size:
            size: 100MiB
          time:
            interval: daily
            modulate: true
          startup:
            enabled: true
        strategy:
          max: 30
          maxAge: 30d
          compression:
            gzip: true
            async: true
          delete:
            basePath: "${prop:LOG_DIR}/archive"
            maxDepth: 1
            ifFileName:
              glob: "*.log.gz"
            ifLastModified:
              age: 30d
            async: true
  root:
    level: info
    appenderRefs: [console, rolling]
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
```

同一份配置可以放在顶层、`configuration` 下，或 `goark.log` 下。单个文件只能使用其中一种结构，不能混用。

## 原生 Logger

普通业务优先使用 `slog` 以保持生态兼容。极热路径需要更低分配时，使用原生 logger 直接写入 `slog.Attr`。

```go
package main

import (
	"context"
	"io"
	"log/slog"
	"time"

	goarklog "goark.dev/log"
)

func main() {
	appender := goarklog.NewJSONAppender(goarklog.WithJSONAppenderWriter(io.Discard))
	handler, err := goarklog.NewHandler(goarklog.Options{
		Appenders: []goarklog.Appender{appender},
		Root: goarklog.RootLogger{
			Level:        slog.LevelInfo,
			AppenderRefs: []string{"json"},
		},
	})
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger, err := goarklog.NewNativeLogger(handler, "goark.http")
	if err != nil {
		panic(err)
	}

	_ = logger.LogAttrs3(context.Background(), slog.LevelInfo, "request done",
		slog.String("method", "GET"),
		slog.Int("status", 200),
		slog.Duration("elapsed", 8*time.Millisecond),
	)
}
```

## 能力概览

| 范围 | 核心已支持 |
| --- | --- |
| 标准库集成 | `slog.Handler`、`slog.Logger`、`LogAttrs`、`WithAttrs`、`WithGroup`。 |
| 原生日志 | 命名原生 logger、固定三属性快速路径、builder API、消息工厂。 |
| 路由 | root logger、命名 logger 规则、前缀匹配、additivity、appender-ref 控制。 |
| Appender | Console、File、JSON、RollingFile、Async、Failover、Routing、Rewrite。 |
| Layout | Pattern、Text、JSON、JSON Template、XML、CSV、GELF、RFC5424/Syslog、YAML、HTML。 |
| Filter | Threshold、Level、LevelRange、Regex、Attr、Marker、Map、Throwable、Time、Burst、DynamicThreshold 及相关别名。 |
| 配置 | YAML、JSON、XML、properties、本地 lookup、轮询 reload。 |
| 滚动文件 | size、time、cron、startup、`%d`/`%i`、gzip、保留策略、删除动作。 |
| 异步 | 有界 ring buffer、批量写出、block/drop/drop-debug/sync-fallback、关闭 drain。 |
| 扩展 | 显式插件注册、插件集合、lookup 插件、JSON Template resolver 插件、registrar 生成器。 |

HTTP、Socket、Syslog 网络输出、Kafka、SMTP、数据库输出、OpenTelemetry、Prometheus 和脚本引擎没有内置在核心模块。需要这些能力时，应放在独立模块中显式注册插件。

## 文档

- [文档索引](docs/index.md)
- [编程式 API](docs/api.md)
- [配置参考](docs/configuration.md)
- [Appender 参考](docs/appenders.md)
- [Layout 参考](docs/layouts.md)
- [Filter 参考](docs/filters.md)
- [使用场景](docs/scenarios.md)
- [扩展指南](docs/extensibility.md)
- [能力边界](docs/capabilities.md)
- [性能和压测](docs/performance.md)
- [v0.0.2 发布检查清单](docs/release-v0.0.2.md)
- [可运行示例](examples/README.md)

## 验证

Unix shell：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

PowerShell：

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

对比 benchmark 位于独立子模块：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
```

## 发布说明

`dev` 是集成分支。发布 tag 应在 `dev` 验证完成并按发布流程合入 `main` 后，从 `main` 打出。发布 `v0.0.2` 前请按 [docs/release-v0.0.2.md](docs/release-v0.0.2.md) 执行检查。
