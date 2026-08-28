# 配置示例

[English](README.md)

本目录包含文档使用的可复制配置文件。文件由 `TestDocsExamples_whenLoaded_shouldBuildOptions` 验证，必须保持能被当前 `LoadOptions` 实现加载。

| 文件 | 目的 |
| --- | --- |
| [console.yml](console.yml) | 使用 PatternLayout 和 ANSI highlight 的开发 console 输出。 |
| [json-stdout.yml](json-stdout.yml) | 容器 stdout JSON logging。 |
| [production-rolling.yml](production-rolling.yml) | 带 async logger、gzip 和 retention 的生产 rolling JSON files。 |
| [split-audit.yml](split-audit.yml) | 拆分应用日志和审计日志，并使用不同 durability 与 retention。 |
| [async-appender.yml](async-appender.yml) | 只围绕指定 JSON file sink 使用 appender-level async。 |
| [rewrite-routing.yml](rewrite-routing.yml) | 属性 rewrite 加 tenant-based routing。 |
| [goark-log.properties](goark-log.properties) | 适合部署系统渲染 key-value 文件的 flat properties 配置。 |
| [goark-log.toml](goark-log.toml) | 使用 dotted tables 和 structured config model 的 TOML 配置。 |
| [log4j2-style.xml](log4j2-style.xml) | 使用已支持 Log4j2-style element names 的 XML 配置。 |

这些示例只使用 core 内置 appender 和 layout。HTTP、Socket、Syslog network output、broker sinks、database sinks、observability exporters 和 script engines 需要外部插件模块。
