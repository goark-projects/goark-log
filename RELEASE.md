# Release Process

This file records the release process for `goark.dev/log`. Day-to-day
integration work happens on `dev`. Do not edit `main` directly. Release tags are
cut from `main` after `dev` has passed validation and has been merged according
to the approved flow.

The current release preparation target is `v0.0.2`. Use
[docs/release-v0.0.2.md](docs/release-v0.0.2.md) as the detailed checklist.

## Branches

| Branch | Purpose |
| --- | --- |
| `dev` | Integration branch for implementation, docs, and verification commits. |
| `main` | Release branch. It should receive validated `dev` changes only through the release merge. |

## Local Gate

Unix shell:

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
cd benchmarks/compare
GOWORK=off go test ./...
```

PowerShell:

```powershell
$env:GOWORK='off'
go test ./...
go vet ./...
go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
Push-Location benchmarks\compare
go test ./...
Pop-Location
```

Long stress gate:

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

## GitHub Actions

Confirm `dev` CI:

```bash
gh run list --branch dev --workflow ci.yml --limit 5
gh run watch <run-id> --exit-status
```

Trigger pressure workflow when needed:

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

PowerShell proxy example:

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## Merge and Tag

After `dev` is green:

```bash
git switch main
git pull --ff-only origin main
git merge --ff-only dev
git push origin main
git tag -a v0.0.2 -m "release: v0.0.2"
git push origin v0.0.2
```

If `main` cannot fast-forward, stop and inspect the difference. Do not
force-push a shared release branch.

## Clean Module Verification

After the tag is pushed, verify module resolution from a clean directory.

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

Create the GitHub Release only after module verification passes. Use the
`CHANGELOG.md` v0.0.2 section as the release body.
