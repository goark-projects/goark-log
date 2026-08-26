# 性能验证说明

本文档记录本地性能验证入口和当前优化边界。基准数字会随 CPU、Go 版本、Windows/Linux 调度器、杀毒软件和终端环境波动，提交前以当前工作树命令为准。

## 推荐命令

根模块：

```bash
go test -run '^$' -bench 'BenchmarkLayout/json$|BenchmarkNativeLoggerLogAttrs$|BenchmarkAsyncAppender$' -benchmem
go test -run '^$' -bench . -benchmem ./internal/disruptor
go test -run '^$' -bench . -benchmem ./internal/jsoncodec
go test -run '^$' -bench 'BenchmarkLayout/json-template$' -benchmem
```

独立对标模块：

```bash
cd benchmarks/compare
go test -run '^$' -bench . -benchmem
```

Windows 本地建议显式关闭父级 workspace：

```powershell
$env:GOWORK='off'
& 'D:\Program Files\go\bin\go.exe' test -run '^$' -bench . -benchmem
```

## 当前热路径结论

- `JSONLayout` 对常见 `slog` 基础类型走手写 `bytes.Buffer` 编码，目标是 `0 alloc/op`。
- 复杂 `slog.Any` fallback 使用 ByteDance Sonic；JSONTemplate resolver 选项解析也复用 Sonic 封装，模板字段顺序解析保留标准库流式 decoder。
- `LevelName` 对内置级别走无锁路径；注册自定义级别后才回到 registry map 查询。
- `NewNativeLogger` 绕过 `slog.Record` facade，`LogAttrs3` 可以在常见三字段场景走固定数组路径。
- `JSONAppender` 是完整 JSON 直写链路，适合极低分配文件或网络适配器前置缓冲。
- `AsyncLogger` 和 `AsyncAppender` 使用内部 ring buffer，支持批量 drain、队列满策略、等待策略参数和异步错误处理。

## 性能预算

| 场景 | 预算 | 说明 |
| --- | --- | --- |
| JSONLayout 基础字段 | 0 alloc/op | 不能把基础字段改成反射 JSON 编码。 |
| JSONTemplate 默认模板 | 0 alloc/op | resolver 热路径必须继续手写追加 JSON。 |
| Sonic fallback | 必须快于 stdlib fallback | 如果后续 Sonic 对某平台退化，应允许切回标准库或增加构建标签。 |
| internal/disruptor ring buffer | 0 alloc/op | ring buffer 不能退化为 channel 队列。 |
| native logger | 优先低于 slog facade | 对外对标时必须同时报告 native 和 slog facade 两条路径。 |
| compare 子模块 | 不能进入核心 go.mod | zap/zerolog 只允许在 `benchmarks/compare` 中出现。 |

## 当前本地样例

以下数字来自 Windows 本地 i9-11900KF、Go 1.25、`GOWORK=off` 的抽样，只作为回归参考，不作为跨机器承诺：

| 场景 | 样例结果 |
| --- | --- |
| `BenchmarkLayout/json` | 约 713 ns/op，0 B/op，0 allocs/op |
| `BenchmarkLayout/json-template` | 约 1251 ns/op，0 B/op，0 allocs/op |
| `BenchmarkNativeLoggerDirectJSON3` | 约 793 ns/op，0 B/op，0 allocs/op |
| `BenchmarkAsyncAppender/block` | 约 687 ns/op，0 B/op，0 allocs/op |
| `internal/disruptor` publish/pop | 约 17.25 ns/op，0 B/op，0 allocs/op |

下一阶段如果继续做极限优化，应集中在真实文件 I/O、强竞争生产者、异步队列容量和操作系统调度差异；不能为了微优化破坏 `Appender` 公共合同、自定义插件正确性或 async drain 语义。
