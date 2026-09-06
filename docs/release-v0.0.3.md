# v0.0.3 Release Checklist

[中文](release-v0.0.3.zh-CN.md)

Run these gates on the exact commit to be tagged:

```bash
git diff --check
GOWORK=off go mod verify
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go test -race ./... -count=1
GOWORK=off go test -run '^$' -bench . -benchmem ./benchmarks/core
```

Run `go test ./...` and `go vet ./...` separately in `benchmarks/compare`.
Run every public example listed in `RELEASE.md`, verify English and Chinese
documentation parity, and tag only a clean `main` commit.

GitHub release copy is maintained in
[github-release-v0.0.3.md](github-release-v0.0.3.md).
