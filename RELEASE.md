# Release Process

[简体中文](RELEASE.zh-CN.md)

`goark-log` releases must be cut from a clean, validated commit. Do not tag from
an unverified local worktree.

## Branch Rule

- Develop and validate on `dev`.
- Merge or fast-forward to `main` according to the repository process.
- Tag only from `main`.
- Keep public tag notes and GitHub release notes in English by default.

## Required Gates

```bash
git status --short --branch
git diff --check
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -race -count=1 ./...
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

Run the comparison module separately:

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

Run the public demos:

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

## Version Checklist

1. Update `CHANGELOG.md` and `CHANGELOG.zh-CN.md`.
2. Update the version checklist and GitHub release notes, such as
   `docs/release-v0.0.2.md` and `docs/github-release-v0.0.2.md`.
3. Verify that every English public Markdown file has a `.zh-CN.md` counterpart.
4. Verify that every file under `docs/examples` with extension `.yml`, `.yaml`, `.json`, `.toml`, `.xml`, or `.properties` loads through `LoadOptions`.
5. Confirm the core dependency boundary: zap and zerolog remain confined to `benchmarks/compare`.
6. Confirm unsupported external integrations are documented as plugins or separate modules, not built-in core features.
7. Tag from `main` only after all gates pass on the exact release commit.
