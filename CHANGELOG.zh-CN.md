# 更新日志

[English](CHANGELOG.md)

本文件只记录基于源码确认的用户可见变更。当前工作分支为 `dev`；release tag
应在验证完成后从 `main` 打出。

## Unreleased

### Added

- 重写双语文档体系，公共默认语言为英文，每个公开 Markdown 文件都有简体中文对应版本。
- 新增完整配置参考，覆盖包装结构、发现顺序、lookup namespace、async 选项、appender 字段、layout 字段、filter 字段、rolling policy、XML 元素和 properties 键。
- 新增基于当前实现的生产级 demo 和场景文档：`examples/production`、`examples/slf4j`、`examples/log4j2_config`。
- 新增可加载配置示例：console、容器 JSON、完整 JSON 流、生产滚动文件、审计路由、异步 failover、filter、JSON Template、TOML、properties 和 Log4j2 风格 XML。
- 文档示例测试改为扫描 `docs/examples` 下所有受支持配置文件，后续新增示例会自动进入校验。

### Changed

- 现有可运行示例尽量改为真实配置文件驱动，不再只是零散片段。
- 公共文档明确区分核心已实现能力和外部集成扩展边界。

## v0.0.1

`goark.dev/log` 下 Goark 日志核心的初始 tag 版本。

该版本包含 `slog.Handler` 运行时、命名 logger 路由、核心 appender 和 layout、
滚动文件、异步队列、filter、配置加载和显式插件注册。
