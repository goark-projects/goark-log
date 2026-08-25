# goark-log

`goark-log` 是 `log/slog` 的 Goark 日志实现，提供并发安全的 `slog.Handler`、Appender、Layout、Logger 层级和配置加载。

默认输出使用 Spring Boot 风格：

```text
2026-08-25T10:15:30.123+08:00  INFO 12345 --- [main] goark.boot : service started profile=dev
```

## 快速使用

```go
package main

import (
	"context"
	"log/slog"

	goarklog "goark.dev/goark-log"
)

func main() {
	handler, _, err := goarklog.ConfigureDefault(context.Background())
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger := goarklog.WithName(slog.Default(), "goark.boot")
	logger.Info("service started", slog.String("profile", "dev"))
}
```

## 配置优先级

1. 显式路径：`goarklog.WithConfigPath(...)`
2. 环境变量：默认 `GOARK_LOG_CONFIG`
3. boot 配置：`goark.log.config`、`goark.logging.config`、`logging.config`
4. 默认文件：`conf/goark-log.yml`、`conf/goark-log.yaml`、`conf/goark-log.toml`、`conf/goark-log.properties`
5. 内置默认：`stderr` console，`INFO`

第一版只解析 YAML。发现 TOML 或 properties 会明确报错，避免误以为配置已生效。

## YAML 示例

```yaml
configuration:
  status: warn
  properties:
    LOG_DIR: logs
    LOG_PATTERN: "%d{yyyy-MM-dd HH:mm:ss.SSS} %5p %pid --- [%thread] %c : %m%attrs%n"
  asyncLogger:
    enabled: true
    queueSize: 8192
    batchSize: 128
    overflowStrategy: block
  filters:
    keep-info:
      type: threshold
      level: info
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
        type: pattern
        pattern: "${prop:LOG_PATTERN}"
      rolling:
        filePattern: "${prop:LOG_DIR}/archive/app-%d{yyyyMMdd}-%i.log.gz"
        maxSize: 100MiB
        interval: daily
        onStartup: true
        maxBackups: 30
        maxAge: 30d
  root:
    level: info
    appenderRefs: [console, rolling]
    filters: [keep-info]
  loggers:
    goark.orm:
      level: debug
      appenderRefs: [rolling]
      additivity: false
```

也可以使用顶层字段，或放在 `goark.log` 下方便与 boot 主配置合并。`configuration`、顶层字段、`goark.log` 三种形式只能选一种，避免配置歧义。

```yaml
goark:
  log:
    appenders:
      console:
        type: console
    root:
      level: info
      appenderRefs: [console]
```

## Reload

```go
reloader, err := goarklog.NewConfigReloader(handler, goarklog.WithConfigPath("conf/goark-log.yml"))
if err != nil {
	panic(err)
}
_, err = reloader.Reload(context.Background())
```

`Watch` 使用标准库轮询文件修改时间和大小，不依赖 `fsnotify`。

## Examples

示例位于 `examples/`：

- `examples/console`：默认 Spring Boot 风格 console
- `examples/file`：普通文件 appender
- `examples/rolling`：大小滚动、启动滚动和 gzip
- `examples/async`：async 包装 rolling appender
- `examples/reload`：YAML 配置 reload

```bash
go test ./examples/...
go run ./examples/console
```

## Benchmark

```bash
go test -run '^$' -bench . -benchmem ./...
```

高吞吐路径建议优先使用 `slog.Logger.LogAttrs`，它比 `Info(args ...any)` 少一次
variadic 装箱开销。内置 `TextLayout`、`JSONLayout`、默认 `PatternLayout` 对常见
`slog` 基础类型保持低分配编码；文件类 appender 默认启用缓冲写入，延迟敏感场景再打开
`flushOnWrite`。
