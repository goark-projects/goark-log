# 发布流程

[English](RELEASE.md)

本文记录 `goark.dev/log` 的发布流程。日常集成工作在 `dev` 分支进行。不要直接修改 `main`。`dev` 通过验证并按批准流程合入后，才从 `main` 打 release tag。

当前发布准备目标是 `v0.0.2`。详细检查清单见 [docs/release-v0.0.2.zh-CN.md](docs/release-v0.0.2.zh-CN.md)。

## 分支

| 分支 | 目的 |
| --- | --- |
| `dev` | 实现、文档和验证提交的集成分支。 |
| `main` | 发布分支。只应通过发布合并接收已验证的 `dev` 变更。 |

## 本地关卡

Unix shell：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
cd benchmarks/compare
GOWORK=off go test ./...
```

PowerShell：

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
Push-Location benchmarks\compare
go test ./...
Pop-Location
```

长压测关卡：

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

## GitHub Actions

确认 `dev` CI：

```bash
gh run list --branch dev --workflow ci.yml --limit 5
gh run watch <run-id> --exit-status
```

需要时触发 pressure workflow：

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

PowerShell 代理示例：

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## 合并和打 Tag

`dev` 绿灯后执行：

```bash
git switch main
git pull --ff-only origin main
git merge --ff-only dev
git push origin main
git tag -a v0.0.2 -m "release: v0.0.2"
git push origin v0.0.2
```

如果 `main` 无法 fast-forward，停止并检查差异。不要 force-push 共享发布分支。

## 干净模块下载验证

tag 推送后，从干净目录验证模块解析。

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

只有模块验证通过后才创建 GitHub Release。默认英文 release body 使用 `CHANGELOG.md` 的 v0.0.2 部分；中文发布说明使用 `CHANGELOG.zh-CN.md`。
