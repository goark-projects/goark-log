# 示例说明

`examples/` 目录提供可直接运行的小示例，用于验证常见接入方式。示例只依赖核心库，不需要外部服务。

## 运行命令

```bash
go test ./examples/...
go run ./examples/console
go run ./examples/file
go run ./examples/rolling
go run ./examples/async
go run ./examples/reload
go run ./examples/extensibility
```

Windows 本地如果父级 `go.work` 干扰独立模块验证，可以显式关闭：

```powershell
$env:GOWORK='off'
& 'D:\Program Files\go\bin\go.exe' test ./examples/...
```

## 示例清单

| 目录 | 说明 | 输出 |
| --- | --- | --- |
| `console` | 默认 console logger 和命名 logger。 | stderr。 |
| `file` | 普通文件 appender，适合最小文件落盘场景。 | 系统临时目录下的 `goark-log-example/file.log`。 |
| `rolling` | 按大小滚动、启动滚动和 gzip 压缩。 | 系统临时目录下的 `goark-log-example/rolling.log` 及归档文件。 |
| `async` | AsyncAppender 包装 rolling appender。 | 系统临时目录下的 `goark-log-example/async-rolling.log`。 |
| `reload` | 配置文件 reload。 | 临时配置和 console 输出。 |
| `extensibility` | `PluginRegistry`、自定义 JSON Template resolver、MessageFactory。 | stdout JSON。 |

## 推荐阅读顺序

1. `console`：确认最小接入方式。
2. `file`：确认普通文件写入和关闭。
3. `rolling`：确认归档、压缩和保留策略。
4. `async`：确认异步包装和关闭 drain。
5. `reload`：确认配置重载入口。
6. `extensibility`：确认插件注册和模板 resolver 扩展。

## 编写新示例的约束

- 示例必须能通过 `go test ./examples/...` 编译。
- 文件输出必须写入临时目录，不能污染仓库。
- 示例只展示核心库能力，不引入外部系统依赖。
- 示例代码保持简短，复杂说明放入 README 或 `docs/`。
