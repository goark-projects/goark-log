# Changelog

[简体中文](CHANGELOG.zh-CN.md)

This changelog records source-backed user-facing changes. The current working
branch is `dev`; release tags are cut from `main` after validation.

## Unreleased

### Added

- Rewritten bilingual documentation system with English as the default public
  language and Simplified Chinese counterparts for every public Markdown file.
- Exhaustive configuration reference covering wrappers, discovery order,
  lookup namespaces, async options, appender fields, layout fields, filter
  fields, rolling policies, XML elements, and properties keys.
- Production demo and scenario docs based on the current implementation:
  `examples/production`, `examples/slf4j`, and `examples/log4j2_config`.
- Loadable configuration examples for console, container JSON, complete JSON
  streams, production rolling files, audit routing, async failover, filters,
  JSON Template layout, TOML, properties, and Log4j2-style XML.
- Documentation test now scans every supported config file under
  `docs/examples`, so future examples are verified automatically.

### Changed

- Existing runnable examples now use real configuration files instead of
  one-off snippets where practical.
- Public documentation now separates implemented core features from external
  integration boundaries.

## v0.0.1

Initial tagged release of the Goark logging core under `goark.dev/log`.

The release included the `slog.Handler` runtime, named logger routing, core
appenders and layouts, rolling files, async queues, filters, configuration
loading, and explicit plugin registration.
