# goark-log 0.0.3

[English](https://github.com/goark-projects/goark-log/blob/v0.0.3/docs/github-release-v0.0.3.md) | 中文

发布日期：2026 年 9 月 6 日

goark-log 0.0.3 在保持 `log/slog` 第一等支持和根包名为 `log` 的同时，完成独立
日志运行时与 Goark Boot 的语义对齐。

完整版本摘要见[变更日志](https://github.com/goark-projects/goark-log/blob/v0.0.3/CHANGELOG.zh-CN.md#v003---2026-09-06)。

## :star: 新功能

- 新增 Spring Boot 风格控制台布局和 Logger 名称缩写。
- 新增 ECS、GELF、Logstash 结构化 JSON 格式和可配置输出字符集。
- 新增内存配置、已加载选项定制器、运行时日志级别控制和更完整的滚动保留策略。

## :lady_beetle: 问题修复

- 默认控制台日志写入 stdout。
- 当时间戳和文件大小未变化时，仍能检测配置内容变化。
- 将 Boot 属性接入 lookup，并对齐结构化堆栈跟踪输出。

## :hammer_and_wrench: 工程质量

- 根包改名为 `log`。
- JSON 工作统一使用 Sonic，要求 Go 1.26，并将实现和集成测试职责拆分到边界明确的
  包和文件。
- 新增 Linux、Windows 和 macOS CI，并包含 race 与 benchmark smoke 门禁。

## :package: 安装

```bash
go get goark.dev/log@v0.0.3
```

## :white_check_mark: 验证

候选版本已在 Windows Go 1.26 和 Debian Linux Go 1.27 上通过测试、vet、race、
配置集成和指定 benchmark smoke。Windows 验证机上的原生 direct JSON 三属性
benchmark 保持零分配；本版本不作普遍性能领先声明。

## :heart: 贡献者

感谢 [@xigexb2](https://github.com/xigexb2) 为本版本作出的贡献。
