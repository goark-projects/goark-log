# 性能和压力测试

[English](performance.md)

本文记录 `goark-log` 的性能预算、验证命令和调优规则。Benchmark 数字受 CPU、Go version、operating system scheduler、disk cache、antivirus software 和 CI runner noise 影响。发布性能结论前，必须在当前 worktree 重新运行命令。

## 短验证

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

Comparison module：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
```

PowerShell：

```powershell
Push-Location benchmarks\compare
$env:GOWORK='off'
go test ./...
Pop-Location
```

## Benchmark Commands

Core benchmarks 位于 `./benchmarks/core`。

Focused zero-allocation JSON path：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Core pressure benchmarks：

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16 ./benchmarks/core
```

Internal data-structure benchmarks：

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./internal/disruptor ./internal/jsoncodec
```

Independent comparison benchmarks：

```bash
cd benchmarks/compare
GOWORK=off go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

PowerShell equivalent：

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

Stress tests 默认跳过。需要显式启用：

```bash
GOARK_LOG_STRESS=1 GOWORK=off go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

PowerShell：

```powershell
$env:GOARK_LOG_STRESS='1'
$env:GOWORK='off'
go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

## CI Layers

Short CI：`.github/workflows/ci.yml`

- 运行 root-module tests。
- 运行 comparison-module tests。
- 对 async、rolling、file 和 JSON paths 运行 focused race subset。
- 运行 short benchmark smoke，捕获 broken benchmark entry points。

Long pressure workflow：`.github/workflows/pressure.yml`

- Manual `workflow_dispatch`。
- Daily scheduled run。
- 设置 `GOARK_LOG_STRESS=1`。
- 在 race 下运行 `TestStress`。
- 上传 core、internal 和 comparison benchmark artifacts。

Manual trigger：

```bash
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

需要 HTTP proxy 时：

```powershell
$env:HTTP_PROXY='http://172.16.8.171:9444'
$env:HTTPS_PROXY='http://172.16.8.171:9444'
gh workflow run pressure.yml --ref dev -f benchtime=5s -f count=3
```

## Hot-Path Design

- `JSONLayout` 手写编码常见 `slog.Value` kinds。
- Direct JSON appender 写固定 JSON event shape，不走通用 layout dispatch。
- `NewNativeLogger` 避开通用 `slog.Record` facade。
- `LogAttrs3` 写固定三属性 event，避免 common path 上的 variadic slice allocation。
- `JSONTemplateLayout` 一次性编译 resolvers；built-in resolvers 直接 append JSON。
- internal JSON fallback 只在受支持的 Go/architecture 组合上使用 Sonic；Go 1.27+ 或不受支持的架构使用标准库路径，避免 unsupported Sonic fast path 输出 runtime warning。
- `LevelName` 对 built-in levels 使用 no-lock path；只有注册 custom levels 后才 fallback 到 registry。
- `AsyncLogger` 和 `AsyncAppender` 使用内部 bounded ring buffer。
- Rolling compression 和 delete actions 可由 background worker 串行化，避免 concurrent archive mutation。
- File appenders 使用 mutex 包住每个 writer，保证并发调用下 line integrity。

## Performance Budget

| 场景 | 预算 | 说明 |
| --- | --- | --- |
| JSONLayout with common fields | `0 B/op`, `0 allocs/op` | 不得退化为 reflection JSON encoding。 |
| JSONTemplate default template | `0 B/op`, `0 allocs/op` | Built-in resolvers 应直接 append。 |
| Direct native logger JSON with three attrs | `0 B/op`, `0 allocs/op` | 高吞吐路径的主 API 预算。 |
| Direct native JSON file parallel path | `0 B/op`, `0 allocs/op` | Buffered file write 必须保持完整 event lines。 |
| Internal ring buffer publish/pop | `0 B/op`, `0 allocs/op` | 不得替换为 allocation-heavy queue。 |
| `slog.Any` fallback | Sonic fast path 可用时快于 stdlib；其他平台不输出 unsupported runtime warning | 平台 fallback 可能不同；优先保证行为正确。 |
| Async logger block strategy | no dropped events | `AsyncDropped` 对 `block` 应保持 0。 |
| Rolling file parallel writes | complete lines | Active 和 archive files 必须包含完整 log events。 |
| Compare module | no core dependency pollution | zap/zerolog dependencies 留在 `benchmarks/compare`。 |

## 调优指南

| 目标 | 推荐设置 |
| --- | --- |
| 最低分配 structured output | Direct JSON appender 加 native `LogAttrs3`。 |
| 人类可读本地日志 | Console appender 加 PatternLayout。 |
| 容器日志 | JSON appender 到 stdout。 |
| Durable VM logs | Rolling file、`overflowStrategy: block`、显式 `Close`。 |
| 高 burst tolerance | 增大 async `queueSize` 和 `batchSize`。 |
| Queue pressure 下更好的 tail latency | 在明确 loss semantics 后使用 `drop-debug` 或 `sync-fallback`。 |
| Compliance/audit logs | 避免 lossy overflow；只对 audit sink 考虑 `flushOnWrite`。 |
| Caller fields | 只在需要的窄 logger 或 appender ref 上启用 `includeLocation`。 |
| Expensive dynamic payloads | 优先 typed `slog` values；最热路径避免大型 `slog.Any`。 |

## 示例数字

以下示例数字是历史本地 regression references，来自 Windows i9-11900KF、Go 1.25、`GOWORK=off`。这些不是 release claims；v0.0.2 发布前要重新运行当前 worktree 命令。

| 场景 | 示例结果 |
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

Stress tests 覆盖：

- multi-producer async logger block-strategy drain；
- async queues 满时 concurrent close wakeups；
- rolling file multi-producer 写真实文件；
- complete JSON/text line validation；
- `Close` 后 async gzip rolling action completion。

Pressure benchmarks 覆盖：

- async queue、batch、overflow 和 wait-strategy 组合；
- JSON file parallel write，含 buffering 和 non-buffering；
- rolling file plain、gzip sync 和 gzip async；
- caller-location cost；
- direct native logger JSON file paths。
