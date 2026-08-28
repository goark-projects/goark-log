# 更新日志

[English](CHANGELOG.md)

本文件只记录基于源码确认的用户可见变更。当前工作分支为 `dev`；release tag
应在验证完成后从 `main` 打出。

## v0.0.2 - 2026-08-28

### Added

- 新增 TOML 配置加载，和 YAML、JSON、Log4j2 风格 XML、Java properties 并列支持。
- 扩展 Log4j2 风格配置覆盖面，包括 rolling policy、rollover strategy、结构化
  appender 引用、组合 appender 和更多 filter 家族。
- 补齐 appender、appender ref、原生 logger、插件注册表、插件集合、status logger、
  logger context、message、marker、context attrs、context stack 和 throwable snapshot
  的公共 API 表面。
- 拆分内部实现包，覆盖 appender、async runtime、配置、layout、filter、routing、
  rolling file、lookup、log value、status 和插件构建。
- 重写双语文档体系，公共默认语言为英文，每个公开 Markdown 文件都有简体中文对应版本。
- 新增完整配置参考，覆盖包装结构、发现顺序、lookup namespace、async 选项、appender 字段、layout 字段、filter 字段、rolling policy、XML 元素和 properties 键。
- 新增基于当前实现的生产级 demo 和场景文档：`examples/production`、`examples/slf4j`、`examples/log4j2_config`。
- 新增可加载配置示例：console、容器 JSON、完整 JSON 流、生产滚动文件、审计路由、异步 failover、filter、JSON Template、TOML、properties 和 Log4j2 风格 XML。
- 文档示例测试改为扫描 `docs/examples` 下所有受支持配置文件，后续新增示例会自动进入校验。

### Changed

- 现有可运行示例尽量改为真实配置文件驱动，不再只是零散片段。
- 公共文档明确区分核心已实现能力和外部集成扩展边界。
- 根目录源码现在作为更小的公共 facade，具体实现迁入 `internal`。
- Benchmark 拆分为核心 benchmark 和独立对比模块，zap 与 zerolog 不进入核心模块图。

### Fixed

- 配置化 async、failover、routing、rewrite 等组合 appender 会一致应用 appender 级 filter。
- `createOnDemand` 开启且首条事件写入前关闭 appender 时，file 和 rolling-file appender
  不再创建或触碰惰性文件。
- rolling file 归档动作在压力下保持串行，避免 Windows 上 gzip/delete 并发竞争。
- 生产日志文件路径现在更防御性地校验和关闭缓冲文件生命周期状态。

## v0.0.1

`goark.dev/log` 下 Goark 日志核心的初始 tag 版本。

该版本包含 `slog.Handler` 运行时、命名 logger 路由、核心 appender 和 layout、
滚动文件、异步队列、filter、配置加载和显式插件注册。
