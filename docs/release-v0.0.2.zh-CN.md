# v0.0.2 发布检查清单

[English](release-v0.0.2.md)

本清单用于下一个 `goark-log` release candidate。必须在将要打 tag 的精确 commit 上运行。

GitHub Release 文案维护在 [github-release-v0.0.2.zh-CN.md](github-release-v0.0.2.zh-CN.md)。

## 需要验证的范围

| 领域 | 必须成立 |
| --- | --- |
| API | `slog` 门面、原生 logger、context attrs、markers、messages、throwable snapshots、status logger 和插件 API 可以编译且已文档化。 |
| 配置 | YAML、JSON、TOML、XML 和 properties 示例都能通过 `LoadOptions` 加载。 |
| Appenders | Console、file、JSON direct、rolling、async、failover、routing、rewrite 有测试或可运行 demo 覆盖。 |
| Layouts | Pattern、text、JSON、JSON Template、XML、CSV、GELF、RFC5424/syslog、YAML、HTML 都已文档化。 |
| Filters | 每个内置 filter 家族都已文档化并有集成覆盖。 |
| Rolling | Size/time/cron/startup 策略、gzip、保留策略、删除动作、direct write、异步归档动作通过测试。 |
| Reload | 显式 reload 和 `monitorInterval` 行为已验证。 |
| 文档 | 公开 Markdown 默认英文且有简体中文副本。 |
| Demo | `examples` 下所有示例不依赖外部服务即可运行。 |

## 正确性门禁

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
```

并发代码变更后运行 race suite：

```bash
GOWORK=off go test -race ./...
```

## Demo Smoke 门禁

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

## 性能门禁

核心 benchmark：

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

对比 benchmark：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

任何性能说明都需要先记录命令、Go 版本、OS、架构、CPU 和 commit。

## 发布步骤

1. 确保工作树除预期发布变更外干净。
2. 确认 `git diff --check` 干净。
3. 运行正确性门禁。
4. 运行 demo smoke 门禁。
5. 如果发布说明提到性能，运行 benchmark。
6. 更新 `CHANGELOG.md`、`CHANGELOG.zh-CN.md`、`RELEASE.md` 和 `RELEASE.zh-CN.md`。
7. 更新 `docs/github-release-v0.0.2.md` 和 `docs/github-release-v0.0.2.zh-CN.md`。
8. 合并到 `main`。
9. 只有同一 commit 通过门禁后，才从 `main` 打 tag。

## 网络代理

只有 Go 需要访问网络时使用代理：

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```

## 不能发布的条件

出现以下任一情况不得打 tag：

| 条件 | 原因 |
| --- | --- |
| 全量测试失败。 | 正确性没有建立。 |
| 公开 Markdown 双语检查失败。 | 文档契约破坏。 |
| 配置示例无法加载。 | 可复制文档不可信。 |
| Demo 失败。 | 面向用户的示例未达到生产级。 |
| 性能结论缺少新 benchmark。 | 结论没有证据。 |
| 核心文档宣称不支持的远程 sink 或 exporter。 | 边界描述错误。 |
