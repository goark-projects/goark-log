# Changelog

Language: English | [简体中文](CHANGELOG.zh-CN.md)

This changelog records source-backed user-facing changes. The current working
branch is `dev`; release tags are cut from `main` after validation.

Release notes: [v0.0.3](docs/releases/v0.0.3.md)

## v0.0.3 - 2026-09-07

### Added

- Spring Boot-style console layout, logger-name abbreviations, structured JSON
  formats, configurable output charsets, and runtime level control.
- In-memory configuration loading and loaded-options customizers for Boot
  integration without temporary files.
- Stronger startup, size-based rolling, and retention behavior.

### Changed

- Renamed the root package to `log`, matching the module's public identity.
- Standardized JSON paths on Sonic and raised the Go baseline to 1.26.
- Split runtime, configuration, layout, appender, filter, rolling, and test
  responsibilities into focused packages and files.
- Added cross-platform CI and benchmark smoke gates.
- Aligned the core and independent comparison modules with the latest stable
  `golang.org/x` releases.

### Fixed

- Default console logging now writes to stdout.
- Configuration reload detects content changes even when file metadata is
  unchanged.
- Boot properties are available to logging lookups, and structured stack trace
  output follows the configured semantics.

## v0.0.2 - 2026-08-28

### Added

- TOML configuration loading alongside YAML, JSON, Log4j2-style XML, and Java
  properties.
- Log4j2-style configuration coverage for rolling policies, rollover strategy,
  structured appender references, composite appenders, and additional filter
  families.
- Public API surface for appenders, appender refs, the native logger, plugin
  registry, plugin sets, status logger, logger context, messages, markers,
  context attrs, context stack, and throwable snapshots.
- Focused internal packages for appenders, async runtime, configuration,
  layouts, filters, routing, rolling files, lookups, log values, status, and
  plugin construction.
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
- Root-level source files now act as a smaller public facade while
  implementation details live under `internal`.
- Benchmarks are split into core benchmarks and an isolated comparison module,
  keeping zap and zerolog outside the core module graph.

### Fixed

- Composite appender filters are applied consistently to configured async,
  failover, routing, and rewrite appenders.
- File and rolling-file appenders no longer create or touch a lazy file when
  `createOnDemand` is enabled and the appender is closed before the first
  event.
- Rolling file archive actions remain serialized under pressure, avoiding
  concurrent gzip/delete races on Windows.
- Production logging paths now validate and close buffered file lifecycle state
  more defensively.

## v0.0.1

Initial tagged release of the Goark logging core under `goark.dev/log`.

The release included the `slog.Handler` runtime, named logger routing, core
appenders and layouts, rolling files, async queues, filters, configuration
loading, and explicit plugin registration.
