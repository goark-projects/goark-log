# 配置模型

[English](configuration.md)

本页说明配置文件如何被发现、解析、lookup、校验和重载。每个字段和别名见
[configuration-reference.zh-CN.md](configuration-reference.zh-CN.md)。

## 支持格式

| 格式 | 解析行为 |
| --- | --- |
| YAML | 使用严格 known fields 的结构化模型。 |
| JSON | 通过和 YAML 相同的结构化模型解码。 |
| TOML | 先解码为通用 map，再转成 YAML 模型，然后使用相同结构化规则校验。 |
| XML | Log4j2 风格 `<Configuration>`，显式描述 appender、filter、async logger 和 logger。 |
| properties | Java properties 风格，支持 `key=value`、`key:value` 或空白分隔 `key value`。 |

## 包装结构

YAML、JSON 和 TOML 可以使用三种结构之一：

```yaml
configuration:
  root:
    level: info
```

```yaml
goark:
  log:
    root:
      level: info
```

```yaml
root:
  level: info
```

单个文件只能使用一种结构。混用顶层字段和包装结构，或同时使用 `configuration`
和 `goark.log`，都会被拒绝。

## 发现顺序

`LoadOptions`、`NewConfiguredHandler`、`NewConfigured`、`ConfigureDefault`
和 `NewConfiguredLoggerContext` 使用以下顺序：

1. `WithConfigPath(path)`。
2. `os.Getenv(EnvConfigPath)`，其中 `EnvConfigPath` 为 `GOARK_LOG_CONFIG`；可用 `WithConfigEnvKey` 覆盖。
3. Boot property resolver 的 `goark.log.config`、`goark.logging.config`、`logging.config`。
4. 工作目录下默认文件：`conf/goark-log.yml`、`.yaml`、`.json`、`.xml`、`.toml`、`.properties`。
5. `DefaultOptions()`。

相对路径会相对当前工作目录解析，也可以通过 `WithConfigWorkingDir` 指定目录。

## Lookup

配置文本支持 `${namespace:key}` 和 `${namespace:key:-fallback}`。属性简写
`${NAME}` 和 `${NAME:-fallback}` 会在文件 `properties` 区段加载后通过 `prop`
和 `property` 解析。

内置 namespace：

| Namespace | Key |
| --- | --- |
| `env` | 操作系统环境变量。 |
| `sys` | `pid`、`processId`、`process-id`、`hostname`、`host`、`cwd`、`workdir`、`workingDir`、`working-dir`、`os`、`arch`。 |
| `go` | `version`、`os`、`arch`。 |
| `date` | 支持的日期 pattern、`RFC3339`、`RFC3339NANO`、`UNIX`、`UNIX_MILLIS`、`UNIX_MICROS`、`UNIX_NANOS`。 |
| `prop`、`property` | 文件本地 `properties` 条目。 |

`$$` 表示字面量美元符号。没有 fallback 的缺失值会报错。`jndi`、`ldap`、
`rmi` namespace 不能注册。

## 级别

内置级别为 `ALL`、`TRACE`、`DEBUG`、`INFO`、`WARN`、`ERROR`、`FATAL`、`OFF`。
`WARNING` 会按 `WARN` 解析。也可以使用数字级别。

`customLevels` 和 `custom-levels` 会向进程默认级别注册表注册名称。因为注册表是进程级的，应谨慎使用。

## Logger 路由

Router 使用最长前缀匹配。名为 `goark.orm` 的规则匹配 `goark.orm` 和
`goark.orm.mapper`；更具体的规则优先。

命名 logger 默认 additive。启用 additivity 时，命名 logger appender 会与 root
appender 合并，命名 logger filter 后会继续执行 root filter。设置
`additivity: false` 时，命名 logger 必须至少声明一个 appender。

Appender 引用可以是字符串或对象：

```yaml
appenderRefs:
  - console
  - ref: rolling
    level: warn
    includeLocation: true
    filterRefs: [auditMarker]
```

## Reload

`ConfigReloader.Reload` 会加载完整的新 `Options`，再调用 `Handler.Reload`。
Router 替换是原子的：只有新运行期构建成功后，旧 appender 才会关闭。

Handler 层 async 的运行期形态不能在 reload 时变化。从同步改异步、变更 queue size、
batch size、overflow strategy、wait strategy、wait options 或 include-location
都会返回错误，并保留旧运行期。

## 校验

以下情况会导致配置加载失败：

- nil context 或已取消 context。
- 不支持的文件扩展名。
- YAML/JSON/TOML 出现未知结构化字段。
- 包装结构混用。
- 未知 appender、layout 或 filter 类型。
- 缺少必需 appender 引用。
- filter 引用成环。
- properties alias 处理后 appender name 重复。
- byte size、interval、cron 表达式、文件权限、布尔值或整数无效。
- 启用 size policy 的 `filePattern` 缺少 `%i`。
- direct-write rolling 同时启用 gzip。
