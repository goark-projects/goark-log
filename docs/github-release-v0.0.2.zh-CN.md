# goark-log v0.0.2

[English](github-release-v0.0.2.md)

`goark-log` v0.0.2 是 Goark 日志核心的生产就绪版本。它继续以 `log/slog` 作为
第一门面，并围绕 Log4j2 和 SLF4J 风格服务日志扩展运行时能力：更完整的配置、
滚动文件、异步队列、filter、layout、显式插件和双语文档。

## 重点变化

- 新增 TOML 配置支持，与 YAML、JSON、Log4j2 风格 XML 和 Java properties 并列。
- 扩展 Log4j2 风格运行时覆盖：rolling policy、rollover strategy、appender 引用、
  组合 appender、routing、rewrite、failover、async 和 filter chain。
- 新增生产形态文档和 demo，覆盖服务日志、SLF4J 风格参数化日志和 Log4j2 风格 XML 配置。
- 重建公共文档体系，默认英文，并为每个公开 Markdown 页面提供简体中文副本。
- 新增可加载配置示例，覆盖 console、容器 JSON、完整 JSON 流、生产滚动文件、审计路由、
  异步 failover、filter 覆盖、JSON Template、TOML、properties 和 XML。
- 可选网络 sink 和观测 sink 继续保留在核心之外，通过显式插件边界接入。

## 修复

- 修复组合 appender filter 应用逻辑，配置化 async、failover、routing、rewrite appender
  会一致应用 appender 级 filter chain。
- 修复惰性文件生命周期：`createOnDemand` 目标在首条事件前关闭时，file 和 rolling-file
  appender 不再创建或触碰目标文件。
- rolling 归档 gzip/delete 动作在压力下保持串行，避免 Windows 文件占用竞争。
- 加固生产文件日志路径，包括缓冲、权限、关闭阶段 layout 生命周期和目录校验。

## 性能与验证

Release candidate 已在 Windows/amd64 和 Go 1.27.0 上验证：

```bash
GOWORK=off go test -count=1 ./...
GOWORK=off go vet ./...
GOWORK=off go test -race -count=1 ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -run '^$' -bench . -benchmem -count=1 ./benchmarks/core
pushd benchmarks/compare
GOWORK=off go test -count=1 ./...
GOWORK=off go test -run '^$' -bench . -benchmem -count=1
popd
```

可运行示例已完成 smoke test：

```bash
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

核心热点 benchmark 在验证机器上确认原生 direct JSON 三属性路径为零分配。对比 benchmark
仍位于独立 `benchmarks/compare` 模块；本版本不宣称在吞吐上全面超越 zap 或 zerolog。

## 升级

```bash
go get goark.dev/log@v0.0.2
```

建议从这些文档开始：

- [README](../README.zh-CN.md)
- [生产指南](production-guide.zh-CN.md)
- [配置参考](configuration-reference.zh-CN.md)
- [配置示例](examples/README.zh-CN.md)
- [可运行示例](../examples/README.zh-CN.md)

## 边界

核心模块不内置 HTTP appender、socket appender、网络 syslog client、Kafka、Pulsar、
RabbitMQ、SMTP、数据库 sink、OpenTelemetry exporter、Prometheus exporter 或嵌入式
脚本运行时。这些能力应由独立模块通过显式插件注册接入。

**完整变更**：`v0.0.1...v0.0.2`
