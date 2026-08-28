# v0.0.2 Release Checklist

[简体中文](release-v0.0.2.zh-CN.md)

This checklist is for the next `goark-log` release candidate. Run it on the
exact commit that will be tagged.

GitHub release copy is maintained in
[github-release-v0.0.2.md](github-release-v0.0.2.md).

## Scope To Verify

| Area | Must be true |
| --- | --- |
| API | `slog` facade, native logger, context attrs, markers, messages, throwable snapshots, status logger, and plugin APIs compile and remain documented. |
| Configuration | YAML, JSON, TOML, XML, and properties examples load through `LoadOptions`. |
| Appenders | Console, file, JSON direct, rolling, async, failover, routing, and rewrite are covered by tests or runnable demos. |
| Layouts | Pattern, text, JSON, JSON Template, XML, CSV, GELF, RFC5424/syslog, YAML, and HTML are documented. |
| Filters | Every built-in filter family is documented and has integration coverage. |
| Rolling | Size/time/cron/startup policies, gzip, retention, delete actions, direct write, and async archive actions pass tests. |
| Reload | Explicit reload and `monitorInterval` behavior are verified. |
| Docs | Public Markdown has English default pages and Simplified Chinese counterparts. |
| Demos | All examples under `examples` run without external services. |

## Correctness Gates

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
```

Run the race suite when concurrency code changed:

```bash
GOWORK=off go test -race ./...
```

## Demo Smoke Gates

```bash
GOWORK=off go run ./examples/console
GOWORK=off go run ./examples/file
GOWORK=off go run ./examples/rolling
GOWORK=off go run ./examples/async
GOWORK=off go run ./examples/reload
GOWORK=off go run ./examples/extensibility
GOWORK=off go run ./examples/production
GOWORK=off go run ./examples/slf4j
GOWORK=off go run ./examples/log4j2_config
```

## Performance Gates

Core benchmark:

```bash
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

Comparison benchmark:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

Record command, Go version, OS, architecture, CPU, and commit before making any
performance statement.

## Release Steps

1. Ensure the worktree is clean except intended release changes.
2. Confirm `git diff --check` is clean.
3. Run correctness gates.
4. Run demo smoke gates.
5. Run benchmarks if release notes mention performance.
6. Update `CHANGELOG.md`, `CHANGELOG.zh-CN.md`, `RELEASE.md`, and `RELEASE.zh-CN.md`.
7. Update `docs/github-release-v0.0.2.md` and `docs/github-release-v0.0.2.zh-CN.md`.
8. Merge to `main`.
9. Tag from `main` only after the same commit passes the gates.

## Network Proxy

Use the proxy only when Go needs network access:

```bash
HTTP_PROXY=http://172.16.8.171:9444 HTTPS_PROXY=http://172.16.8.171:9444 ALL_PROXY=http://172.16.8.171:9444 go test ./...
```

## Non-Release Conditions

Do not tag if any of these are true:

| Condition | Reason |
| --- | --- |
| Full tests fail. | Correctness is not established. |
| Public Markdown localization fails. | Documentation contract is broken. |
| Config examples do not load. | Copyable docs are not trustworthy. |
| Demos fail. | User-facing examples are not production-grade. |
| Performance claims lack fresh benchmarks. | Claims would not be evidence-backed. |
| Core docs claim unsupported remote sinks or exporters. | Boundary is inaccurate. |
