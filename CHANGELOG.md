# 更新日志

本文档记录 `goark.dev/log` 的公开版本变化。版本号遵循 Go Module 语义化版本规则。

## v0.0.1 - 2026-08-27

### 新增

- 提供基于 `log/slog` 的并发安全 `Handler`、默认 logger 装配和原生低分配 `Logger`。
- 支持 Console、File、RollingFile、JSONFile、Async、Failover、Routing、Rewrite 等本地和组合型输出端。
- 支持 Pattern、Text、JSON、JSONTemplate、XML、CSV、GELF、RFC5424、YAML、HTML 等布局。
- 支持 root/logger 层级路由、additivity、appenderRef 级别、全局 filter 和局部 filter。
- 支持上下文属性、上下文栈、自定义级别、marker、throwable 和 caller 位置采集。
- 支持 YAML、JSON、XML、properties 配置加载，以及文件轮询 reload。
- 支持 size/time/cron/startup 滚动策略、gzip 压缩、保留数量、保留时间和删除动作。
- 支持有界异步队列、批量 drain、队列满策略、等待策略和关闭 drain。
- 提供显式插件注册、插件集合 helper、JSON Template resolver 扩展点和 registrar 生成器。

### 性能

- 常见 JSON 热路径使用手写 `bytes.Buffer` 编码，避免基础字段退化到反射编码。
- 复杂 `slog.Any` fallback 通过 ByteDance Sonic 编码。
- 内部 ring buffer、原生三属性日志、直接 JSON 文件写入和关键 layout benchmark 进入回归覆盖。
- `benchmarks/compare` 独立维护 zap、zerolog 对比依赖，不污染核心模块依赖。

### 安全边界

- lookup 默认只开放 `env`、`sys`、`go`、`date` 和 `property` 等本地安全 namespace。
- 远程 lookup、脚本引擎运行时、外部系统 appender 和观测导出不进入核心仓库。
- TOML 配置显式报错，避免配置文件被静默忽略后误判为已生效。

### 验证

- 根模块单元测试、race 子集、benchmark smoke 和 compare 子模块测试纳入 GitHub Actions。
- 长压测通过 `pressure` workflow 手动触发或定时运行。
- 本地发布门禁见 `RELEASE.md`。
