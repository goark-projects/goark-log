# 发布流程

本文档记录 `goark.dev/log` 当前发布流程。`v0.0.1` 从 `main` 分支打 tag，`dev` 是发布前集成分支。

## 发布分支

- 日常集成分支：`dev`
- 发布分支：`main`
- 首个版本：`v0.0.1`

发布前必须确认 `dev` 已推送，且所有待发布代码都已经包含在 `dev`。短期工作分支合并后应删除，避免后续误发布旧分支内容。

## 本地门禁

Windows 本地建议显式关闭父级 workspace，并固定 Go 工具链：

```powershell
$env:GOWORK='off'
$env:GOTOOLCHAIN='local'
$env:GOCACHE='G:\opensource\goark\.cache\go-build'
& 'D:\Program Files\go\bin\gofmt.exe' -w .
& 'D:\Program Files\go\bin\go.exe' test ./...
& 'D:\Program Files\go\bin\go.exe' vet ./...
Push-Location benchmarks\compare
& 'D:\Program Files\go\bin\go.exe' test ./...
Pop-Location
```

发布前建议再跑一次短 benchmark smoke：

```powershell
$env:GOWORK='off'
$env:GOTOOLCHAIN='local'
$env:GOCACHE='G:\opensource\goark\.cache\go-build'
& 'D:\Program Files\go\bin\go.exe' test -run '^$' -bench 'BenchmarkLayout/json$|BenchmarkNativeLoggerDirectJSON3$|BenchmarkAsyncLoggerParallel3$|BenchmarkFileAppenderParallel$' -benchmem -benchtime=1s -count=1
& 'D:\Program Files\go\bin\go.exe' test -run '^$' -bench . -benchmem -benchtime=1s -count=1 ./internal/disruptor ./internal/jsoncodec
```

长压测不阻塞每次本地发布操作，建议通过 GitHub Actions 的 `pressure` workflow 执行：

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## GitHub Actions

发布前必须确认 `dev` 的 `ci` workflow 通过：

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh run list --branch dev --workflow ci.yml --limit 5
gh run watch <run-id> --exit-status
```

`pressure` workflow 是长期压测入口，适合手动触发或等待定时任务产物。

## 合并和打 tag

`v0.0.1` 必须从 `main` 打 tag：

```powershell
git switch main
git pull --ff-only origin main
git merge --ff-only dev
git push origin main
git tag -a v0.0.1 -m "release: v0.0.1"
git push origin v0.0.1
```

如果 `main` 不能快进合并，先停止发布，检查差异来源，不要强推共享分支。

## 模块拉取验证

tag 推送后，用干净临时目录验证 Go module 可解析：

```powershell
$tmp = Join-Path $env:TEMP 'goark-log-v0.0.1-verify'
Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $tmp | Out-Null
Push-Location $tmp
$env:GOWORK='off'
$env:GOTOOLCHAIN='local'
& 'D:\Program Files\go\bin\go.exe' mod init verify.local/goark-log
& 'D:\Program Files\go\bin\go.exe' get goark.dev/log@v0.0.1
& 'D:\Program Files\go\bin\go.exe' list -m goark.dev/log
Pop-Location
```

验证完成后再创建 GitHub Release，并使用 `CHANGELOG.md` 中的 `v0.0.1` 内容作为发布说明。
