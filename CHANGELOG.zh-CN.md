# 变更日志

[English](CHANGELOG.md)

本项目遵循 Go module 语义化版本规则。

## v0.0.2 - 未发布

### 新增

- 新增默认英文 README 和独立简体中文 README。
- 新增编程式 API、配置、Appender、Layout、Filter、场景、扩展、能力边界、性能、v0.0.2 发布验证等详细文档。
- 在 `docs/examples` 下新增可复制配置示例，覆盖 console、JSON stdout、生产滚动文件、审计拆分、异步 appender、rewrite/routing、properties 和 XML。
- 新增集成测试，加载每一个可复制的 `docs/examples` 配置文件。
- 为所有公开文档页面新增简体中文版本，同时保持英文作为默认文档路径。
- 新增公开 Markdown 文档 localization 覆盖测试。

### 变更

- 将根包测试移动到更聚焦的子包，降低 public root package 的文件密度并提升边界清晰度。
- benchmark 文档改为使用 `./benchmarks/core` 执行 core benchmark，以匹配 benchmark 包拆分后的结构。
- CI 和 pressure workflow 的 benchmark 命令改为执行 `./benchmarks/core`，不再从根包运行 core benchmark。
- 明确外部系统 appender 和 observability exporter 不属于 core module，必须由显式插件模块提供。

### 验证

- 发布验证清单：`docs/release-v0.0.2.zh-CN.md`。
- 必需本地关卡：`GOWORK=off go test ./...`、`GOWORK=off go vet ./...`、focused race tests、`./benchmarks/core` 下的 core benchmark，以及 `benchmarks/compare` 下的对比模块测试。

## v0.0.1 - 2026-08-27

### 新增

- 新增并发安全的 `slog.Handler`、默认 logger 组装逻辑和低分配 native `Logger`。
- 新增 Console、File、RollingFile、JSONFile、Async、Failover、Routing 和 Rewrite appenders。
- 新增 Pattern、Text、JSON、JSONTemplate、XML、CSV、GELF、RFC5424、YAML 和 HTML layouts。
- 新增 root/logger 层级路由、additivity、appender-ref level、global filters 和 local filters。
- 新增 context attributes、context stack、custom levels、marker、throwable 和可选 caller location capture。
- 新增 YAML、JSON、XML 和 properties 配置加载，以及文件轮询 reload。
- 新增 size/time/cron/startup rolling policies、gzip compression、retained count、retained age 和 delete actions。
- 新增 bounded async queues、batch drain、overflow strategies、wait strategies 和 shutdown drain。
- 新增显式插件注册、plugin set helpers、JSON Template resolver extension 和 registrar generator。

### 性能

- 常见 JSON 路径使用手写 `bytes.Buffer` 编码，避免 built-in `slog.Value` 类型走反射。
- 复杂 `slog.Any` fallback 使用 ByteDance Sonic。
- internal ring buffer、native three-attribute logging、direct JSON file output 和关键 layouts 均有 benchmark 覆盖。
- zap 和 zerolog 对比依赖隔离在 `benchmarks/compare`。

### 安全边界

- 默认 lookup 限制为本地 `env`、`sys`、`go`、`date` 和 `property` namespace。
- remote lookup namespace、script runtime、external-system appender 和 observability exporter 均不属于 core module。
- TOML 配置会显式失败，不会被静默忽略。

### 验证

- root module unit tests、focused race tests、benchmark smoke tests 和 compare module tests 已纳入发布验证路径。
- long stress coverage 可通过 `pressure` workflow 执行。
