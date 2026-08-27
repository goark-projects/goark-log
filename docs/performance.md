# 性能和压测

本文档记录 `goark-log` 的性能预算、压测入口和当前样例数字。所有数字都会受 CPU、Go 版本、操作系统调度、磁盘缓存、杀毒软件和 CI runner 抖动影响，发布前必须以当前工作树命令重新验证。

## 本地验证命令

根模块短测：

```bash
go test ./...
go vet ./...
```

长压测默认跳过，需要显式打开：

```bash
GOARK_LOG_STRESS=1 go test -race -run 'TestStress' -count=1 -timeout=20m ./...
```

核心压力 benchmark：

```bash
go test -run '^$' -bench 'BenchmarkPressure|BenchmarkAsyncLoggerParallel3|BenchmarkFileAppenderParallel|BenchmarkNativeLoggerDirectJSONFileParallel3' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
go test -run '^$' -bench . -benchmem ./internal/disruptor
go test -run '^$' -bench . -benchmem ./internal/jsoncodec
```

独立性能比较模块：

```bash
cd benchmarks/compare
go test ./...
go test -run '^$' -bench 'BenchmarkCompareParallelDiscard|BenchmarkPressureParallelFile' -benchmem -benchtime=10s -count=5 -cpu=1,4,16
```

Windows 本地建议显式关闭父级 workspace：

```powershell
$env:GOWORK='off'
$env:GOTOOLCHAIN='local'
$env:GOCACHE='G:\opensource\goark\.cache\go-build'
& 'D:\Program Files\go\bin\go.exe' test ./...
```

## CI 分层

短 CI：`.github/workflows/ci.yml`

- push 和 pull request 触发。
- 运行根模块测试、compare 子模块测试。
- 对 async、rolling、file、JSON 关键路径跑 race 子集。
- 对核心热路径 benchmark 做 1s smoke，防止 benchmark 入口损坏。

长压测：`.github/workflows/pressure.yml`

- 支持 `workflow_dispatch` 手动触发。
- 每日定时运行。
- 设置 `GOARK_LOG_STRESS=1`，执行 `TestStress` race。
- 输出 root、internal、compare benchmark artifact。

## 热路径设计

- `JSONLayout` 对常见 `slog` 基础类型走手写 `bytes.Buffer` 编码。
- 复杂 `slog.Any` fallback 使用 ByteDance Sonic；JSONTemplate resolver 选项解析也复用 Sonic 封装。
- `LevelName` 对内置级别走无锁路径；注册自定义级别后才回到 registry map 查询。
- `NewNativeLogger` 绕过 `slog.Record` facade，`LogAttrs3` 可以在常见三字段场景走固定数组路径。
- `JSONAppender` 是完整 JSON 直写链路，`NewJSONFileAppender` 直接写文件缓冲。
- `AsyncLogger` 和 `AsyncAppender` 使用内部 ring buffer，支持批量 drain、队列满策略、等待策略参数和异步错误处理。
- 文件类 appender 使用互斥锁保护单 writer，不能用无保护并发写换吞吐。

## 性能预算

| 场景 | 预算 | 说明 |
| --- | --- | --- |
| JSONLayout 基础字段 | 0 alloc/op | 基础字段不能退化为反射 JSON 编码。 |
| JSONTemplate 默认模板 | 0 alloc/op | resolver 热路径必须继续手写追加 JSON。 |
| Sonic fallback | 明显快于 stdlib fallback | 如果某平台退化，应允许构建标签或配置切回标准库。 |
| internal/disruptor ring buffer | 0 alloc/op | ring buffer 不能退化为 channel 队列。 |
| native logger direct JSON | 0 alloc/op | 固定三属性主路径不能产生堆分配。 |
| JSON file 并发写 | 0 alloc/op | 缓冲文件写入必须保持完整行边界。 |
| AsyncLogger block | 不丢事件 | block 策略下 `Close` 必须 drain，`AsyncDropped` 为 0。 |
| RollingFile 并发写 | 行完整 | active 和 archive 都必须是完整 JSON/text 行。 |
| compare 子模块 | 不污染核心 go.mod | zap/zerolog 只能位于 `benchmarks/compare`。 |

## 当前本地样例

以下数字来自 Windows 本地 i9-11900KF、Go 1.25、`GOWORK=off` 的抽样，只作为回归参考。

| 场景 | 样例结果 |
| --- | --- |
| `BenchmarkLayout/json` | 约 708 ns/op，0 B/op，0 allocs/op |
| `BenchmarkNativeLoggerDirectJSONParallel3` | 约 326.8 ns/op，0 B/op，0 allocs/op |
| `BenchmarkFileAppenderParallel` | 约 286.9 ns/op，0 B/op，0 allocs/op |
| `BenchmarkPressureAsyncLoggerQueueMatrix/q8192-b256-block-yield` | 约 1303 ns/op，257 B/op，2 allocs/op，0 dropped，0 failed |
| `BenchmarkPressureJSONFileParallel/buffered-256k` | 约 270.5 ns/op，0 B/op，0 allocs/op |
| `BenchmarkPressureRollingFileParallel/plain` | 约 482.5 ns/op，140 B/op，1 alloc/op |
| `internal/disruptor` publish/pop | 约 18 ns/op，0 B/op，0 allocs/op |
| `internal/jsoncodec` Sonic fallback | 约 630 ns/op，341 B/op，4 allocs/op |
| `internal/jsoncodec` stdlib fallback | 约 1335 ns/op，640 B/op，17 allocs/op |
| `benchmarks/compare` goark direct file parallel | 约 262.7 ns/op，0 B/op，0 allocs/op |
| `benchmarks/compare` zap file parallel | 约 317.8 ns/op，193 B/op，1 alloc/op |
| `benchmarks/compare` zerolog file parallel | 约 197.1 ns/op，0 B/op，0 allocs/op |

## 压测覆盖

`pressure_test.go` 覆盖：

- `AsyncLogger` 多生产者 block 策略 drain。
- `AsyncLogger` 队列满时并发关闭唤醒。
- `RollingFileAppender` 多生产者真实文件写入和 JSON 行完整性。
- `RollingFileAppender` 异步 gzip 动作在 `Close` 后完成。

`pressure_bench_test.go` 覆盖：

- async queue、batch、overflow、wait strategy 的关键组合。
- JSON file 并发写和 flushOnWrite。
- rolling file plain、gzip sync、gzip async。
- caller 采集开销。

`benchmarks/compare/pressure_bench_test.go` 覆盖：

- goark direct JSON file 并发写。
- goark rolling file 并发写。
- zap 和 zerolog 并发文件写。对非线程安全 buffered writer 使用互斥封装，保证比较代码自身不破坏并发写语义。
