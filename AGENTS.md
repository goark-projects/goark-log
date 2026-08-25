# Goark Boot Agent Notes

## Project Skills

Use the project-local skills in `.claude/skills/` when the task matches their scope.
This directory is the canonical cross-tool skill copy for Claude Code, OpenCode, Codex, and other agents that can read project instructions.
A mirror exists in `.codex/skills/` for Codex-specific surfaces.

- Go bootstrapping, config loading, Viper integration, generics, reflection: `.claude/skills/golang-pro/SKILL.md`
- Startup, cancellation, lifecycle, and concurrent loading behavior: `.claude/skills/go-concurrency-patterns/SKILL.md`
- Bootstrap architecture and extension boundaries: `.claude/skills/architecture-patterns/SKILL.md` and `.claude/skills/architecture-designer/SKILL.md`
- Public APIs, package contracts, and module surfaces: `.claude/skills/api-and-interface-design/SKILL.md`
- Significant decisions and durable project context: `.claude/skills/documentation-and-adrs/SKILL.md`
- Multi-file features and staged refactors: `.claude/skills/incremental-implementation/SKILL.md`
- New features or unclear requirements: `.claude/skills/spec-driven-development/SKILL.md`
- Framework error codes, wrapping, and failure contracts: `.claude/skills/error-handling-patterns/SKILL.md`
- Config data contracts, profile precedence, and validation behavior: `.claude/skills/config-validate/SKILL.md`
- Bugs, test failures, and unexpected runtime behavior: `.claude/skills/systematic-debugging/SKILL.md`
- Refactoring for clarity without behavior changes: `.claude/skills/code-simplification/SKILL.md`
- Behavior changes and bug fixes: `.claude/skills/test-driven-development/SKILL.md`
- Reviews and quality checks: `.claude/skills/code-review-and-quality/SKILL.md`
- Security reviews and threat modeling when explicitly requested: `.claude/skills/security-best-practices/SKILL.md` and `.claude/skills/security-threat-model/SKILL.md`
- Future web/API modules: `.claude/skills/openapi-spec-generation/SKILL.md` and `.claude/skills/rest-api-conventions/SKILL.md`
- Observability, tracing, and performance work: `.claude/skills/observability-and-instrumentation/SKILL.md`, `.claude/skills/performance-engineer/SKILL.md`, and `.claude/skills/distributed-tracing/SKILL.md`
- GitHub Actions setup or failed PR checks: `.claude/skills/github-actions-templates/SKILL.md` and `.claude/skills/gh-fix-ci/SKILL.md`
- Spring Boot config semantics are reference material only: `.claude/skills/spring-boot-engineer/SKILL.md` and `.claude/skills/java-architect/SKILL.md`

Keep boot separate from the core runtime: boot owns startup conventions, config discovery, profile rules, and module assembly.
