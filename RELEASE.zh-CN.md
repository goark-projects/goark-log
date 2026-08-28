# 发布流程

[English](RELEASE.md)

`goark-log` 必须从干净且已验证的提交发布。不要从未验证的本地 worktree 打 tag。

## 分支规则

- 在 `dev` 开发和验证。
- 按仓库流程合并或快进到 `main`。
- 只从 `main` 打 tag。
- 公开 tag note 和 GitHub release note 默认使用英文。

## 必跑 Gate

```bash
git status --short --branch
git diff --check
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go test ./internal/integration -run 'TestDocs(Examples|Localization)' -count=1
GOWORK=off go test -race -count=1 ./...
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

对比模块单独运行：

```bash
cd benchmarks/compare
GOWORK=off go test ./...
GOWORK=off go test -run '^$' -bench . -benchmem
```

运行公开 demo：

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

## 版本检查清单

1. 更新 `CHANGELOG.md` 和 `CHANGELOG.zh-CN.md`。
2. 更新版本检查清单和 GitHub release notes，例如 `docs/release-v0.0.2.md` 和
   `docs/github-release-v0.0.2.md`。
3. 确认每个英文公开 Markdown 文件都有 `.zh-CN.md` 对应版本。
4. 确认 `docs/examples` 下 `.yml`、`.yaml`、`.json`、`.toml`、`.xml`、`.properties` 文件都能通过 `LoadOptions` 加载。
5. 确认核心依赖边界：zap 和 zerolog 仍只在 `benchmarks/compare`。
6. 确认未内置的外部集成被写成插件或独立模块，而不是核心已支持能力。
7. 所有 gate 在目标提交上通过后，才从 `main` 打 tag。
