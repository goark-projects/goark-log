# Configuration Model

[简体中文](configuration.zh-CN.md)

This page explains how configuration files are discovered, parsed, resolved,
validated, and reloaded. See [configuration-reference.md](configuration-reference.md)
for every field and alias.

## Supported Formats

| Format | Parser behavior |
| --- | --- |
| YAML | Structured model with strict known fields. |
| JSON | Decoded through the same structured model as YAML. |
| TOML | Decoded to a generic map, marshaled into the YAML model, then validated with the same structured rules. |
| XML | Log4j2-style `<Configuration>` with explicit elements for appenders, filters, async logger, and loggers. |
| properties | Java properties style `key=value`, `key:value`, or whitespace-separated `key value`. |

## Wrappers

YAML, JSON, and TOML can use one of three shapes:

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

Use exactly one shape per file. A file that mixes top-level fields with a wrapper
or uses both `configuration` and `goark.log` is rejected.

## Discovery Order

`LoadOptions`, `NewConfiguredHandler`, `NewConfigured`, `ConfigureDefault`, and
`NewConfiguredLoggerContext` use this order:

1. `WithConfigPath(path)`.
2. `os.Getenv(EnvConfigPath)`, where `EnvConfigPath` is `GOARK_LOG_CONFIG`; override the key with `WithConfigEnvKey`.
3. Boot property resolver keys `goark.log.config`, `goark.logging.config`, and `logging.config`.
4. Default files under the working directory: `conf/goark-log.yml`, `.yaml`, `.json`, `.xml`, `.toml`, `.properties`.
5. `DefaultOptions()`.

Relative paths are resolved against the current working directory, or the
directory set by `WithConfigWorkingDir`.

## Lookups

Configuration text supports `${namespace:key}` and `${namespace:key:-fallback}`.
Property shorthand `${NAME}` and `${NAME:-fallback}` resolves through `prop` and
`property` after the file `properties` section is loaded.

Built-in namespaces:

| Namespace | Keys |
| --- | --- |
| `env` | Operating system environment variables. |
| `sys` | `pid`, `processId`, `process-id`, `hostname`, `host`, `cwd`, `workdir`, `workingDir`, `working-dir`, `os`, `arch`. |
| `go` | `version`, `os`, `arch`. |
| `date` | Any supported date pattern, `RFC3339`, `RFC3339NANO`, `UNIX`, `UNIX_MILLIS`, `UNIX_MICROS`, `UNIX_NANOS`. |
| `prop`, `property` | File-local `properties` entries. |

`$$` escapes a literal dollar sign. Missing values without fallback are errors.
Namespaces `jndi`, `ldap`, and `rmi` cannot be registered.

## Levels

Built-in levels are `ALL`, `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`,
and `OFF`. `WARNING` is accepted as `WARN`. Numeric levels are accepted.

`customLevels` and `custom-levels` register names in the process default level
registry. They should be used deliberately because the registry is process-wide.

## Logger Routing

The router uses longest-prefix matching. A rule named `goark.orm` matches
`goark.orm` and `goark.orm.mapper`; a more specific rule wins.

Named loggers are additive by default. With additivity enabled, the named
logger's appenders are merged with root appenders and its filters are followed
by root filters. With `additivity: false`, the named logger must declare at
least one appender.

Appender references can be plain strings or objects:

```yaml
appenderRefs:
  - console
  - ref: rolling
    level: warn
    includeLocation: true
    filterRefs: [auditMarker]
```

## Reload

`ConfigReloader.Reload` loads a complete new `Options` value and calls
`Handler.Reload`. Router replacement is atomic: the old appenders are closed
only after the new runtime is built.

Handler-level async runtime shape cannot change during reload. Changing from
sync to async, changing queue size, batch size, overflow strategy, wait strategy,
wait options, or include-location returns an error and leaves the old runtime
active.

## Validation

Configuration loading fails for:

- nil context or canceled context.
- unsupported file extension.
- unknown YAML/JSON/TOML structured fields.
- invalid wrapper mixing.
- unknown appender, layout, or filter type.
- missing required appender references.
- cyclic filter references.
- duplicate appender names after properties aliases.
- invalid byte sizes, intervals, cron expressions, file permissions, booleans, or integers.
- rolling size policy with `filePattern` that lacks `%i`.
- direct-write rolling with gzip.
