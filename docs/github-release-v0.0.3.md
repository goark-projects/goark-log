# goark-log 0.0.3

English | [中文](https://github.com/goark-projects/goark-log/blob/v0.0.3/docs/github-release-v0.0.3.zh-CN.md)

Release date: September 6, 2026

goark-log 0.0.3 aligns the standalone logging runtime with Goark Boot while
keeping `log/slog` first-class and the root package name simply `log`.

See the [changelog](https://github.com/goark-projects/goark-log/blob/v0.0.3/CHANGELOG.md#v003---2026-09-06)
for the complete version summary.

## :star: New Features

- Add Spring Boot-style console layout and logger-name abbreviation.
- Add ECS, GELF, and Logstash structured JSON formats and configurable output
  charsets.
- Add in-memory configuration, loaded-options customizers, runtime logger level
  control, and stronger rolling retention.

## :lady_beetle: Bug Fixes

- Write default console logs to stdout.
- Detect configuration content changes when timestamps and sizes are unchanged.
- Bridge Boot properties into lookups and align structured stack trace output.

## :hammer_and_wrench: Engineering

- Rename the root package to `log`.
- Standardize JSON work on Sonic, require Go 1.26, and split implementation and
  integration-test responsibilities into bounded packages and files.
- Add Linux, Windows, and macOS CI with race and benchmark smoke gates.

## :package: Installation

```bash
go get goark.dev/log@v0.0.3
```

## :white_check_mark: Verification

The candidate passed tests, vet, race tests, configuration integration, and
named benchmark smoke gates on Windows Go 1.26 and Debian Linux Go 1.27.
The native direct JSON three-attribute benchmark remained zero-allocation on
the Windows validation host; no universal performance claim is made.

## :heart: Contributors

Thank you to [@xigexb2](https://github.com/xigexb2) for making this release
possible.
