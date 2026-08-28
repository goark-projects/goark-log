# Performance and Stress Testing

[简体中文](performance.zh-CN.md)

This document records the performance budget, validation commands, and tuning
rules for `goark-log`. Benchmark numbers depend on CPU, Go version, operating
system scheduler, disk cache, antivirus software, and CI runner noise. Always
rerun the commands on the current worktree before making release claims.

## Short Validation

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

Comparison module:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
```

PowerShell:

```powershell
Push-Location benchmarks\compare
$env:GOWORK='off'
go test ./...
Pop-Location
```

## Benchmark Commands

Core benchmarks live in `./benchmarks/core`.

Focused zero-allocation JSON path:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Core pressure benchmarks:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16 ./benchmarks/core
```

Internal data-structure benchmarks:

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./internal/disruptor ./internal/jsoncodec
```

Independent comparison benchmarks:

```bash
cd benchmarks/compare
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

PowerShell equivalent:

```powershell
$env:GOWORK='off'
go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16 ./benchmarks/core
go test -run '^$' -bench . -benchmem ./internal/disruptor ./internal/jsoncodec
Push-Location benchmarks\compare
go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
Pop-Location
```

## Long Stress Tests

Stress tests are skipped by default. Enable them explicitly:

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

PowerShell:

```powershell
$env:GOARK_LOG_STRESS='1'
$env:GOWORK='off'
go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

## CI Layers

Short CI: `.github/workflows/ci.yml`

- Runs root-module tests.
- Runs comparison-module tests.
- Runs a focused race subset for async, rolling, file, and JSON paths.
- Runs a short benchmark smoke to catch broken benchmark entry points.

Long pressure workflow: `.github/workflows/pressure.yml`

- Manual `workflow_dispatch`.
- Daily scheduled run.
- Sets `GOARK_LOG_STRESS=1`.
- Runs `TestStress` under race.
- Uploads core, internal, and comparison benchmark artifacts.

Manual trigger:

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

If an HTTP proxy is required:

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## Hot-Path Design

- `JSONLayout` hand-encodes common `slog.Value` kinds.
- The direct JSON appender writes a fixed JSON event shape without general
  layout dispatch.
- `NewNativeLogger` bypasses the general `slog.Record` facade.
- `LogAttrs3` writes a fixed three-attribute event and avoids variadic slice
  allocation on the common path.
- `JSONTemplateLayout` compiles resolvers once; built-in resolvers append JSON
  directly.
- The internal JSON fallback uses Sonic only on supported Go/architecture
  combinations; Go 1.27+ or unsupported architectures use the standard library
  path to avoid runtime warnings from unsupported Sonic fast paths.
- `LevelName` uses a no-lock path for built-in levels and falls back to the
  registry only after custom levels are registered.
- `AsyncLogger` and `AsyncAppender` use the internal bounded ring buffer.
- Rolling compression and delete actions can be serialized on a background
  worker to avoid concurrent archive mutation.
- File appenders use a mutex around each writer; this preserves line integrity
  under concurrent callers.

## Performance Budget

| Scenario | Budget | Notes |
| --- | --- | --- |
| JSONLayout with common fields | `0 B/op`, `0 allocs/op` | Must not regress to reflection JSON encoding. |
| JSONTemplate default template | `0 B/op`, `0 allocs/op` | Built-in resolvers should append directly. |
| Direct native logger JSON with three attrs | `0 B/op`, `0 allocs/op` | Main API budget for high-throughput paths. |
| Direct native JSON file parallel path | `0 B/op`, `0 allocs/op` | Buffered file write must keep full event lines. |
| Internal ring buffer publish/pop | `0 B/op`, `0 allocs/op` | Must not be replaced by an allocation-heavy queue. |
| `slog.Any` fallback | Faster than stdlib where Sonic fast path is available; no unsupported runtime warnings elsewhere | Platform fallback can vary; keep behavior correct first. |
| Async logger block strategy | no dropped events | `AsyncDropped` should stay zero for `block`. |
| Rolling file parallel writes | complete lines | Active and archive files must contain complete log events. |
| Compare module | no core dependency pollution | zap/zerolog dependencies stay in `benchmarks/compare`. |

## Tuning Guide

| Need | Preferred setting |
| --- | --- |
| Lowest allocation structured output | Direct JSON appender plus native `LogAttrs3`. |
| Human-readable local logs | Console appender with PatternLayout. |
| Container logs | JSON appender to stdout. |
| Durable VM logs | Rolling file, `overflowStrategy: block`, explicit `Close`. |
| High burst tolerance | Increase async `queueSize` and `batchSize`. |
| Better tail latency under queue pressure | Use `drop-debug` or `sync-fallback` after deciding loss semantics. |
| Compliance/audit logs | Avoid lossy overflow; consider `flushOnWrite` only for audit sink. |
| Caller fields | Enable `includeLocation` only on the narrow logger or appender ref that needs it. |
| Expensive dynamic payloads | Prefer typed `slog` values; avoid large `slog.Any` on the hottest path. |

## Sample Numbers

The following sample numbers are historical local regression references from a
Windows i9-11900KF run with Go 1.25 and `GOWORK=off`. They are not release
claims; rerun current-worktree commands for v0.0.2.

| Scenario | Sample result |
| --- | --- |
| `BenchmarkLayout/json` | about `708 ns/op`, `0 B/op`, `0 allocs/op` |
| `BenchmarkNativeLoggerDirectJSON3` | about `770.6 ns/op`, `0 B/op`, `0 allocs/op` |
| `BenchmarkNativeLoggerDirectJSONParallel3` | about `153.4 ns/op`, `0 B/op`, `0 allocs/op` |
| `BenchmarkFileAppenderParallel` | about `286.9 ns/op`, `0 B/op`, `0 allocs/op` |
| `BenchmarkPressureAsyncLoggerQueueMatrix/q8192-b256-block-yield` | about `1303 ns/op`, `257 B/op`, `2 allocs/op`, `0 dropped`, `0 failed` |
| `BenchmarkPressureJSONFileParallel/buffered-256k` | about `270.5 ns/op`, `0 B/op`, `0 allocs/op` |
| `BenchmarkPressureRollingFileParallel/plain` | about `482.5 ns/op`, `140 B/op`, `1 alloc/op` |
| `internal/disruptor` publish/pop | about `18 ns/op`, `0 B/op`, `0 allocs/op` |
| `internal/jsoncodec` goark fallback on Go 1.25 amd64 | about `630 ns/op`, `341 B/op`, `4 allocs/op` |
| `internal/jsoncodec` stdlib comparison on Go 1.25 amd64 | about `1335 ns/op`, `640 B/op`, `17 allocs/op` |
| `benchmarks/compare` goark direct file parallel | about `262.7 ns/op`, `0 B/op`, `0 allocs/op` |
| `benchmarks/compare` zap file parallel | about `317.8 ns/op`, `193 B/op`, `1 alloc/op` |
| `benchmarks/compare` zerolog file parallel | about `197.1 ns/op`, `0 B/op`, `0 allocs/op` |

## Stress Coverage

Stress tests cover:

- multi-producer async logger block-strategy drain,
- concurrent close wakeups when async queues are full,
- rolling file multi-producer writes to real files,
- complete JSON/text line validation,
- async gzip rolling action completion after `Close`.

Pressure benchmarks cover:

- async queue, batch, overflow, and wait-strategy combinations,
- JSON file parallel write with and without buffering,
- rolling file plain, gzip sync, and gzip async,
- caller-location cost,
- direct native logger JSON file paths.
