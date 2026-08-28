# Performance

[简体中文](performance.zh-CN.md)

This page documents how `goark-log` is designed and measured. It does not make
release performance claims without benchmark output from the exact commit being
published.

## Hot-Path Design

| Area | Current behavior |
| --- | --- |
| Facade | The standard `slog.Handler` path is supported for ordinary application code. |
| Native logger | `NewNativeLogger` provides a lower-allocation path and level-aware builder. |
| Fixed attr path | `LogAttrs3` avoids variadic slice construction for the common three-attribute event. |
| JSON direct | `NewJSONAppender` and configured `type: json` bypass generic layouts. |
| Caller data | `slog.Record.PC` and source formatting are used only when location is requested. |
| Async | Handler-level and appender-level queues use bounded ring buffers and explicit overflow strategies. |
| Rolling actions | Compression and deletion can run on a serial background worker. |
| Plugin boundary | Heavy optional dependencies stay outside the core module. |

## Benchmark Suites

Core benchmarks:

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

Focused hot-path benchmarks:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkNativeLoggerDirectJSON3|BenchmarkNativeLoggerDirectJSONParallel3' -benchmem ./benchmarks/core
```

Pressure benchmarks:

```bash
GOWORK=off go test -run '^$' -bench 'BenchmarkPressure' -benchmem ./benchmarks/core
```

Comparison benchmarks live in a separate module so zap and zerolog do not
become core dependencies:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

Use the proxy only when dependencies must be downloaded:

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```

## Interpreting Results

Benchmark numbers are valid only for the same machine, OS, Go version, commit,
and command. Report at least:

| Field | Example |
| --- | --- |
| Commit | `git rev-parse HEAD` |
| Go version | `go version` |
| OS and architecture | `go env GOOS GOARCH` |
| Command | Exact benchmark command. |
| Result | `ns/op`, `B/op`, and `allocs/op`. |

Do not claim superiority over zap, zerolog, slog, or another logger unless the
comparison benchmark was run on the exact release candidate and the workload is
named.

## Tuning Guide

| Goal | Setting |
| --- | --- |
| Lowest stdout overhead | Use JSON direct appender with `target: stdout`. |
| Lowest file overhead | Use JSON direct file or JSON layout with buffering enabled. |
| Stable latency for required logs | Use async overflow `block` and enough queue capacity. |
| Keep service moving for non-critical debug logs | Use async overflow `drop` or `drop-debug` with counters. |
| Preserve audit logs | Use `block` or synchronous file writes, `flushOnWrite: true`, and restrictive permissions. |
| Reduce caller overhead | Keep `includeLocation` disabled unless the route or layout needs caller data. |
| Reduce layout cost | Prefer JSON direct or simple pattern/text layouts for hot paths. |
| Reduce archive contention | Use rolling `compression.async: true` or `asyncActions: true`. |

## Async Metrics

`Handler` exposes async counters for handler-level async:

| Counter | Meaning |
| --- | --- |
| `AsyncDropped()` | Events dropped by overflow policy. |
| `AsyncFailed()` | Events that failed during asynchronous delivery. |

Appender-level async supports an error handler through programmatic options.
Configured appender-level async writes through the selected overflow behavior
and drains on close.

## Release Gate

Before publishing a release that mentions performance:

1. Run correctness tests first: `GOWORK=off go test ./...`.
2. Run `GOWORK=off go vet ./...`.
3. Run core benchmarks on the release candidate.
4. Run comparison benchmarks only from `benchmarks/compare`.
5. Record the exact command and environment in the release notes.
