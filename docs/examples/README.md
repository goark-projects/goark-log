# Configuration Examples

[简体中文](README.zh-CN.md)

This directory contains copyable configuration files used by the documentation.
The files are validated by `TestDocsExamples_whenLoaded_shouldBuildOptions`, so
they must remain loadable by the current `LoadOptions` implementation.

| File | Purpose |
| --- | --- |
| [console.yml](console.yml) | Development console output with PatternLayout and ANSI highlighting. |
| [json-stdout.yml](json-stdout.yml) | Container stdout JSON logging. |
| [production-rolling.yml](production-rolling.yml) | Production rolling JSON files with async logger, gzip, and retention. |
| [split-audit.yml](split-audit.yml) | Separate application and audit logs with different durability and retention. |
| [async-appender.yml](async-appender.yml) | Appender-level async around a selected JSON file sink. |
| [rewrite-routing.yml](rewrite-routing.yml) | Attribute rewrite plus tenant-based routing. |
| [goark-log.properties](goark-log.properties) | Flat properties configuration for deployment systems that render key-value files. |
| [log4j2-style.xml](log4j2-style.xml) | XML configuration using supported Log4j2-style element names. |

The examples only use built-in core appenders and layouts. HTTP, Socket,
Syslog network output, broker sinks, database sinks, observability exporters,
and script engines require external plugin modules.
