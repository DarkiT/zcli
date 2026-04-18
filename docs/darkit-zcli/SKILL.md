---
name: darkit-zcli
description: Repo-local guide for working on the current github.com/darkit/zcli workspace. Use when building, reviewing, refactoring, documenting, or troubleshooting zcli-based Go CLI apps or the zcli framework itself, especially around NewBuilder/BuildWithError, QuickCLI/QuickService, Cobra command trees, flag export, ServiceRunner/WithService, daemon-backed service install config, i18n/help/version behavior, official examples, and docs/tests alignment.
---

# darkit-zcli

Use this skill as a repo-local navigation layer for the current `zcli` workspace, not as a full API mirror.

## Work pattern

1. Identify the task area first: examples, builder/config, command tree/UI, service lifecycle, i18n/output, troubleshooting, or docs alignment.
2. Read one relevant resource or official example before opening broad repo context.
3. Inspect live code only after the first resource stops being enough.
4. Use tests to confirm behavior boundaries.
5. When docs conflict with live code or tests, trust live code and tests.

## Current official examples

Start with `examples/README.md`.

Current supported examples:

- `examples/cli-tool`
- `examples/function-service`
- `examples/service-runner`
- `examples/service-config`
- `examples/flag-export`

Do not rely on removed historical example paths such as `examples/main.go`, `examples/legacy`, `examples/demos`, or older variadic-service narratives.

## Route by task

- Quick start or public-facing usage:
  Read `README.md`, then `examples/README.md`.
- Builder, config, hooks, validation, or defaults:
  Read `./resources/config-options.md`, then inspect `builder.go` and `options.go`.
- Command tree, help, version, completion, flags, or Cobra behavior:
  Read `examples/cli-tool`, `examples/flag-export`, then inspect `command.go`, `command_*.go`, and `zcli.go`.
- Service lifecycle, foreground/background behavior, shutdown, or service command semantics:
  Read `./resources/context-lifecycle.md` and `./resources/service-management.md`, then inspect `service.go`, `service_commands.go`, `service_runtime.go`, `service_support.go`, and `service_interface.go`.
- Advanced service registration or daemon alignment:
  Inspect `builder.go`, `service.go`, and `service_interface.go`; cross-check `examples/service-config`.
- I18n, help rendering, output language, or Windows mousetrap behavior:
  Inspect `i18n*.go`, `command*.go`, `README.md`, and the official examples. English is the default in current official examples; Chinese is opt-in.
- Troubleshooting:
  Read `./resources/troubleshooting.md` first, then open the smallest relevant test file.
- Design rationale or compatibility boundaries:
  Read `DESIGN.md`.

## Current capability reminders

- Prefer `BuildWithError()` for production-style builder flows.
- Use `WithServiceRunner(...)` for non-trivial services with explicit dependencies or lifecycle ownership.
- Use `WithService(...)` for smaller services that only need `run` and optional `stop` functions.
- Advanced service registration currently includes:
  `WithServiceUser`, `WithExecutable`, `WithArguments`, `WithDependency`, `WithStructuredDependencies`, `WithLegacyDependencies`, `WithServiceOption`, `WithServiceOptionsMap`, `WithAllowSudoFallback`, and `WithServiceConfig`.
- Flag export is a first-class capability. Start with `ExportFlagsForViper(...)`, `GetBindableFlagSets(...)`, and `GetFilteredFlagNames(...)`.

## High-signal live code

- `builder.go`
- `options.go`
- `command.go`
- `command_*.go`
- `service.go`
- `service_commands.go`
- `service_runtime.go`
- `service_support.go`
- `service_interface.go`
- `i18n*.go`
- `README.md`
- `examples/README.md`
- `*_test.go`

## High-signal tests

- `enhanced_test.go`
- `service_alignment_test.go`
- `service_command_semantics_test.go`
- `service_signal_test.go`
- `service_force_exit_test.go`
- `service_concurrent_test.go`
- `service_edge_cases_test.go`

## Resource files

- `./resources/api-reference.md`
- `./resources/complete-example.md`
- `./resources/config-options.md`
- `./resources/context-lifecycle.md`
- `./resources/service-management.md`
- `./resources/troubleshooting.md`

## Constraints

- Keep the loading path narrow. Do not read every resource file by default.
- Treat this skill as repo-local workflow guidance, not as a substitute for live code.
- If you update docs or examples, keep them aligned with the current codebase and tests.
