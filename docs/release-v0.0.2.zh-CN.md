# v0.0.2 发布检查清单

[English](release-v0.0.2.md)

本检查清单用于准备 `goark.dev/log` v0.0.2。实现和文档工作都在 `dev` 上做。不要直接修改 `main`。`main` 只用于批准后的 release merge 和 tag。

## 1. 确认分支和工作区

```bash
git switch dev
git status --short --branch
git log --oneline --decorate -5
```

期望：

- 当前分支是 `dev`；
- 没有无关 dirty files；
- 所有预期提交都在 `dev`。

如果工作区有无关变更，不要 reset。要么把它们排除在 release commit 外，要么让 owner 先处理。

## 2. 文档关卡

检查公开文档只描述已实现的 core behavior：

```bash
rg -n "HTTP|Socket|Syslog|Kafka|SMTP|Prometheus|OpenTelemetry|script" README.md README.zh-CN.md docs
rg -n "bench 'BenchmarkNativeLoggerDirectJSON3|benchmarks/core|benchmarks/compare" README.md docs
git diff --check
```

必需断言：

- 默认 README 是英文；
- 中文 README 独立存在；
- 所有公开文档都有英文默认文件和 `.zh-CN.md` 中文文件；
- 详细参考文档位于 `docs/`；
- core docs 不声称内置 external appenders；
- core benchmark 使用 `./benchmarks/core`；
- compare benchmark 留在 `benchmarks/compare`；
- 没有 trailing whitespace 或 broken patch whitespace。

## 3. Config Example Gate

`docs/examples/` 下的可复制文件应能通过 `LoadOptions` 加载。

PowerShell：

```powershell
$env:GOWORK='off'
go test ./...
go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
```

`TestDocsExamples_whenLoaded_shouldBuildOptions` 会加载 `docs/examples` 中每一个可复制配置文件。`TestDocsLocalization_whenPublicMarkdownExists_shouldHaveChineseCounterpart` 会检查公开 Markdown 文档都有中文 counterpart。

## 4. Core Test Gate

Unix shell：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
```

PowerShell：

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
```

## 5. Race Gate

Full race gate：

```bash
GOWORK=off go test -race ./...
```

Focused CI-style race gate：

```bash
GOWORK=off go test -race -run 'TestAsync(Logger|Appender)|TestRollingFileAppender|TestFileAppender|TestJSON' -count=1 -timeout=10m ./...
```

如果 full race gate 本地太慢，至少本地跑 focused gate，并远程跑 long pressure workflow。

## 6. Benchmark Gate

Focused hot-path benchmark：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Pressure benchmark：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=5s -count=3 -cpu=1,4,16 ./benchmarks/core
```

Internal benchmark：

```bash
GOWORK=off go test -run '^$' -bench . -benchmem -benchtime=1s -count=1 ./internal/disruptor ./internal/jsoncodec
```

Direct native JSON three-attribute paths 的硬预算：

- `0 B/op`；
- `0 allocs/op`。

## 7. Comparison Module Gate

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=5s -count=3 -cpu=1,4,16
```

Comparison module 可以依赖 zap 和 zerolog。Core module 不允许。

## 8. Long Stress Gate

Local：

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

GitHub Actions：

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
gh run list --branch dev --workflow pressure.yml --limit 5
gh run watch <run-id> --exit-status
```

PowerShell 代理示例：

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## 9. Commit and Push Dev

```bash
git status --short
git add README.md README.zh-CN.md CHANGELOG.md CHANGELOG.zh-CN.md RELEASE.md RELEASE.zh-CN.md docs examples .github/workflows internal/integration/docs_examples_test.go internal/integration/docs_localization_test.go
git commit -m "docs: add bilingual documentation"
git push origin dev
```

如果 release 只改部分文件，按实际 path list 调整。

## 10. Merge to Main

`dev` green 后：

```bash
git switch main
git pull --ff-only origin main
git merge --ff-only dev
git push origin main
```

如果不能 fast-forward merge，停止并检查差异。不要 force-push `main`。

## 11. Tag

```bash
git tag -a v0.0.2 -m "release: v0.0.2"
git push origin v0.0.2
```

## 12. 干净模块下载验证

PowerShell：

```powershell
$tmp = Join-Path $env:TEMP 'goark-log-v0.0.2-verify'
Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $tmp | Out-Null
Push-Location $tmp
$env:GOWORK='off'
go mod init verify.local/goark-log
go get goark.dev/log@v0.0.2
go test goark.dev/log/...
Pop-Location
```

Unix shell：

```bash
tmp="$(mktemp -d)"
cd "$tmp"
GOWORK=off go mod init verify.local/goark-log
GOWORK=off go get goark.dev/log@v0.0.2
GOWORK=off go test goark.dev/log/...
```

## 13. GitHub Release

使用 `CHANGELOG.md` 的 v0.0.2 section 作为默认英文 release body。中文 release notes 使用 `CHANGELOG.zh-CN.md` 的 v0.0.2 section。

Release note 必须包含：

- 用户可见文档变更；
- 行为或 API 变更，如果存在；
- 验证命令和结果；
- 已知限制，尤其是仍故意留在 core 外部的能力。

不要声称 external appender support，除非对应 plugin module 已单独发布并测试。
