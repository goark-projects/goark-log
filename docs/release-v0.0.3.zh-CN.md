# v0.0.3 发布检查清单

[English](release-v0.0.3.md)

在准备打 tag 的精确提交上运行以下门禁：

```bash
git diff --check
GOWORK=off go mod verify
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go test -race ./... -count=1
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

在 `benchmarks/compare` 中单独运行 `go test ./...` 和 `go vet ./...`。
运行 `RELEASE.md` 列出的全部公开示例，检查中英文文档对应关系，并且只对干净的
`main` 提交打 tag。

GitHub Release 文案维护在
[github-release-v0.0.3.zh-CN.md](github-release-v0.0.3.zh-CN.md)。
