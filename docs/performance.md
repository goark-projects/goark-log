# 性能验证说明

本文档记录本地性能验证入口和当前优化边界。基准数字会随 CPU、Go 版本、Windows/Linux 调度器、杀毒软件和终端环境波动，提交前以当前工作树命令为准。

## 推荐命令

根模块：

```bash
go test -run '^$' -bench 'BenchmarkLayout/json$|BenchmarkNativeLoggerLogAttrs$|BenchmarkAsyncAppender$' -benchmem
go test -run '^$' -bench . -benchmem ./internal/disruptor
go test -run '^$' -bench . -benchmem ./internal/jsoncodec
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
- 复杂 `slog.Any` fallback 使用 ByteDance Sonic；本地抽样显示 Sonic 明显快于 `encoding/json` 且分配更少。
- `LevelName` 对内置级别走无锁路径；注册自定义级别后才回到 registry map 查询。
- `NewNativeLogger` 绕过 `slog.Record` facade，适合延迟敏感路径。
- `AsyncLogger` 和 `AsyncAppender` 使用内部 ring buffer，支持批量 drain 和队列满策略。

## 性能预算

| 场景 | 预算 | 说明 |
| --- | --- | --- |
| JSONLayout 基础字段 | 0 alloc/op | 不能把基础字段改成反射 JSON 编码。 |
| Sonic fallback | 必须快于 stdlib fallback | 如果后续 Sonic 对某平台退化，应允许切回标准库或增加构建标签。 |
| internal/disruptor ring buffer | 0 alloc/op | ring buffer 不能退化为 channel 队列。 |
| native logger | 优先低于 slog facade | 对外对标时必须同时报告 native 和 slog facade 两条路径。 |
| compare 子模块 | 不能进入核心 go.mod | zap/zerolog 只允许在 `benchmarks/compare` 中出现。 |

## 已知瓶颈

完整 logger 到 appender 的 JSON 链路仍有 `Event.Attrs` 跨接口逃逸，通常表现为 `1 alloc/op`。下一阶段优化应集中在：

- 专用 JSON 直写 appender 或 layout fast path；
- 可证明生命周期的属性快照策略；
- appender/filter 不需要存储事件时的零拷贝迭代接口；
- 避免为了微优化破坏 `Appender` 公共合同。

任何进一步优化都必须同时保留自定义 appender 的正确性和 async drain 语义。
