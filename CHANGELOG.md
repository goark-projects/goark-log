# Changelog

[简体中文](CHANGELOG.zh-CN.md)

This project follows Go module semantic versioning rules.

## v0.0.2 - Unreleased

### Added

- Added default English README and separate Simplified Chinese README.
- Added detailed docs for programmatic API, configuration, appenders, layouts,
  filters, scenarios, extensibility, capability boundaries, performance, and
  v0.0.2 release validation.
- Added copyable configuration examples under `docs/examples` for console,
  JSON stdout, production rolling files, audit split, async appender,
  rewrite/routing, properties, and XML.
- Added an integration test that loads every copyable `docs/examples`
  configuration file.
- Added Simplified Chinese counterparts for every public documentation page
  while keeping English as the default documentation path.
- Added a localization coverage test for public Markdown documentation.
- Added TOML configuration loading through the same strict structured
  configuration contract used by YAML and JSON.

### Changed

- Moved root package tests into focused subpackages to keep the public root
  package smaller and clearer.
- Updated benchmark documentation to use `./benchmarks/core` for core
  benchmarks after the benchmark package split.
- Updated CI and pressure workflow benchmark commands to execute the
  `./benchmarks/core` package instead of the root package.
- Clarified that external-system appenders and observability exporters remain
  outside the core module and must be provided by explicit plugin modules.

### Validation

- Release validation checklist: `docs/release-v0.0.2.md`.
- Required local gates: `GOWORK=off go test ./...`, `GOWORK=off go vet ./...`,
  focused race tests, core benchmarks under `./benchmarks/core`, and comparison
  module tests under `benchmarks/compare`.

## v0.0.1 - 2026-08-27

### Added

- Added concurrency-safe `slog.Handler`, default logger assembly, and native
  low-allocation `Logger`.
- Added Console, File, RollingFile, JSONFile, Async, Failover, Routing, and
  Rewrite appenders.
- Added Pattern, Text, JSON, JSONTemplate, XML, CSV, GELF, RFC5424, YAML, and
  HTML layouts.
- Added root/logger hierarchical routing, additivity, appender-ref level,
  global filters, and local filters.
- Added context attributes, context stack, custom levels, marker, throwable, and
  opt-in caller location capture.
- Added YAML, JSON, XML, and properties configuration loading with file polling
  reload.
- Added size/time/cron/startup rolling policies, gzip compression, retained
  count, retained age, and delete actions.
- Added bounded async queues, batch drain, overflow strategies, wait
  strategies, and shutdown drain.
- Added explicit plugin registration, plugin set helpers, JSON Template
  resolver extension, and registrar generator.

### Performance

- Common JSON paths use handwritten `bytes.Buffer` encoding and avoid
  reflection for built-in `slog.Value` kinds.
- Complex `slog.Any` fallback uses ByteDance Sonic.
- Internal ring buffer, native three-attribute logging, direct JSON file output,
  and key layouts are covered by benchmarks.
- zap and zerolog comparison dependencies are isolated in `benchmarks/compare`.

### Security Boundary

- Default lookups are limited to local `env`, `sys`, `go`, `date`, and
  `property` namespaces.
- Remote lookup namespaces, script runtimes, external-system appenders, and
  observability exporters are not part of the core module.
- The default search path reserved `toml`; before v0.0.2 support, TOML input
  failed explicitly instead of being silently ignored.

### Validation

- Root module unit tests, race-focused tests, benchmark smoke tests, and compare
  module tests were included in the release validation path.
- Long stress coverage is available through the `pressure` workflow.
