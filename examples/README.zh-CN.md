# 可运行示例

[English](README.md)

`examples/` 目录包含只依赖 core module 的小型可运行程序，不需要外部服务。

## 命令

```bash
GOWORK=off go test ./examples/...
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
```

PowerShell：

```powershell
$env:GOWORK='off'
go test ./examples/...
go run ./examples/console
go run ./examples/file
go run ./examples/rolling
go run ./examples/async
go run ./examples/reload
go run ./examples/extensibility
```

## 示例程序

| 目录 | 目的 | 输出 |
| --- | --- | --- |
| `console` | 默认 console logger 和 named logger 用法。 | stderr。 |
| `file` | 普通 file appender 和显式 close。 | 系统临时目录下的 `goark-log-example/file.log`。 |
| `rolling` | size rollover、startup rollover、archive pattern 和 gzip compression。 | 系统临时目录下的 `goark-log-example/rolling.log` 及 archive 文件。 |
| `async` | AsyncAppender 包装 rolling appender。 | 系统临时目录下的 `goark-log-example/async-rolling.log` 及 archive 文件。 |
| `reload` | 配置文件加载和运行时 reload。 | 临时配置文件和 console 输出。 |
| `extensibility` | `PluginRegistry`、custom JSON Template resolver 和 message factory。 | stdout JSON。 |

## 配置示例

可复制配置文件位于 [../docs/examples](../docs/examples/README.zh-CN.md)：

- [console.yml](../docs/examples/console.yml)
- [json-stdout.yml](../docs/examples/json-stdout.yml)
- [production-rolling.yml](../docs/examples/production-rolling.yml)
- [split-audit.yml](../docs/examples/split-audit.yml)
- [async-appender.yml](../docs/examples/async-appender.yml)
- [rewrite-routing.yml](../docs/examples/rewrite-routing.yml)
- [goark-log.properties](../docs/examples/goark-log.properties)
- [goark-log.toml](../docs/examples/goark-log.toml)
- [log4j2-style.xml](../docs/examples/log4j2-style.xml)

## 阅读顺序

1. `console`：最小集成。
2. `file`：文件写入生命周期和 close 行为。
3. `rolling`：归档、压缩和保留行为。
4. `async`：异步包装器和 shutdown drain。
5. `reload`：配置 reload 入口。
6. `extensibility`：插件注册和 resolver 扩展。

## 新增示例规则

- 必须能通过 `go test ./examples/...` 编译。
- 文件输出必须使用临时目录，不允许写入仓库目录。
- 只演示 core capability。
- 程序代码保持短小，细节解释放在 `docs/`。
