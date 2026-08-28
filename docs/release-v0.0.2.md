# v0.0.2 Release Checklist

This checklist is for preparing `goark.dev/log` v0.0.2. Do the implementation
and documentation work on `dev`. Do not edit `main` directly. Use `main` only
for the approved release merge and tag.

## 1. Confirm Branch and Worktree

```bash
git switch dev
git status --short --branch
git log --oneline --decorate -5
```

Expected:

- branch is `dev`,
- there are no unrelated dirty files,
- all intended commits are on `dev`.

If the worktree has unrelated changes, do not reset them. Either keep them out
of the release commit or ask the owner to resolve them.

## 2. Documentation Gate

Check that the public docs describe only implemented core behavior:

```bash
rg -n "HTTP|Socket|Syslog|Kafka|SMTP|Prometheus|OpenTelemetry|script" README.md README.zh-CN.md docs
rg -n "bench 'BenchmarkNativeLoggerDirectJSON3|benchmarks/core|benchmarks/compare" README.md docs
git diff --check
```

Required assertions:

- default README is English,
- Chinese README is separate,
- detailed reference docs are under `docs/`,
- core docs do not claim built-in external appenders,
- benchmark commands use `./benchmarks/core` for core benchmarks,
- compare benchmarks stay in `benchmarks/compare`,
- no trailing whitespace or broken patch whitespace exists.

## 3. Config Example Gate

The copyable files under `docs/examples/` should load with `LoadOptions`.

PowerShell:

```powershell
$env:GOWORK='off'
go test ./...
go test ./internal/integration -run TestDocsExamples_whenLoaded_shouldBuildOptions -count=1
```

The focused `TestDocsExamples_whenLoaded_shouldBuildOptions` test loads every
copyable config file in `docs/examples`.

## 4. Core Test Gate

Unix shell:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
```

PowerShell:

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
```

## 5. Race Gate

Full race gate:

```bash
GOWORK=off go test -race ./...
```

Focused CI-style race gate:

```bash
GOWORK=off go test -race -run 'TestAsync(Logger|Appender)|TestRollingFileAppender|TestFileAppender|TestJSON' -count=1 -timeout=10m ./...
```

If the full race gate is too slow for a local release pass, run the focused gate
locally and run the long pressure workflow remotely.

## 6. Benchmark Gate

Focused hot-path benchmark:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Pressure benchmark:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=5s -count=3 -cpu=1,4,16 ./benchmarks/core
```

Internal benchmark:

```bash
GOWORK=off go test -run '^$' -bench . -benchmem -benchtime=1s -count=1 ./internal/disruptor ./internal/jsoncodec
```

The hard budget for direct native JSON three-attribute paths is:

- `0 B/op`,
- `0 allocs/op`.

## 7. Comparison Module Gate

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=5s -count=3 -cpu=1,4,16
```

The comparison module can depend on zap and zerolog. The core module must not.

## 8. Long Stress Gate

Local:

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

GitHub Actions:

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
gh run list --branch dev --workflow pressure.yml --limit 5
gh run watch <run-id> --exit-status
```

Proxy example for PowerShell:

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## 9. Commit and Push Dev

```bash
git status --short
git add README.md README.zh-CN.md docs examples RELEASE.md CHANGELOG.md .github/workflows internal/integration/docs_examples_test.go
git commit -m "docs: prepare goark-log v0.0.2 documentation"
git push origin dev
```

Adjust the path list if a release only changes a subset of files.

## 10. Merge to Main

After `dev` is green:

```bash
git switch main
git pull --ff-only origin main
git merge --ff-only dev
git push origin main
```

If fast-forward merge is not possible, stop and inspect the difference. Do not
force-push `main`.

## 11. Tag

```bash
git tag -a v0.0.2 -m "release: v0.0.2"
git push origin v0.0.2
```

## 12. Clean Module Download Verification

PowerShell:

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

Unix shell:

```bash
tmp="$(mktemp -d)"
cd "$tmp"
GOWORK=off go mod init verify.local/goark-log
GOWORK=off go get goark.dev/log@v0.0.2
GOWORK=off go test goark.dev/log/...
```

## 13. GitHub Release

Use the `CHANGELOG.md` v0.0.2 section as the release body. The release note
must include:

- user-facing documentation changes,
- behavior or API changes, if any,
- validation commands and results,
- known limitations that remain intentionally out of core.

Do not claim external appender support unless the matching plugin module is
released and tested separately.
