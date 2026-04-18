# Official Examples

This directory contains the current, supported examples for `github.com/darkit/zcli`.

These examples are intentionally aligned with the current public API surface:

- Default example language is English.
- `BuildWithError()` is preferred for examples that may fail validation.
- `WithServiceRunner(...)` is the recommended path for non-trivial services.
- `WithService(...)` remains the concise option for small services.

## Examples

### `cli-tool`

Recommended starting point for a CLI-only application.

Highlights:

- `NewBuilder("en")`
- `BuildWithError()`
- `WithValidator(...)`
- `WithInitHook(...)`
- `zcli.Command` as a Cobra-compatible command type

Run:

```bash
go run ./examples/cli-tool --help
go run ./examples/cli-tool status --profile prod
go run ./examples/cli-tool echo hello world
```

### `function-service`

Recommended starting point for a small service that does not need a dedicated service object.

Highlights:

- `WithService(run, stop)`
- `WithServiceTimeouts(...)`
- `WithDependency(...)`
- default shutdown budget of `15s` for `Run(ctx)` and `5s` for final cleanup

Run:

```bash
go run ./examples/function-service --help
go run ./examples/function-service run
```

### `service-runner`

Recommended starting point for non-trivial services with explicit dependencies and a dedicated service type.

Highlights:

- `ServiceRunner` interface
- `WithServiceRunner(...)`
- explicit `Run / Stop / Name`
- dependency injection friendly structure

Run:

```bash
go run ./examples/service-runner --help
go run ./examples/service-runner run
```

### `service-config`

Advanced service registration example focused on install-time configuration.

Highlights:

- `WithServiceUser(...)`
- `WithExecutable(...)`
- `WithArguments(...)`
- `WithStructuredDependencies(...)`
- `WithServiceOption(...)`
- `WithServiceOptionsMap(...)`
- `WithAllowSudoFallback(...)`
- `WithServiceConfig(...)`

Run:

```bash
go run ./examples/service-config --help
go run ./examples/service-config inspect
```

### `flag-export`

Reference example for integrating zcli flags with configuration systems or other packages.

Highlights:

- `ExportFlagsForViper(...)`
- `GetBindableFlagSets(...)`
- `GetFilteredFlagNames(...)`
- system flag filtering

Run:

```bash
go run ./examples/flag-export --help
go run ./examples/flag-export inspect --region eu-west-1 --output yaml
```

## Notes

- Service examples automatically register `run`, `install`, `start`, `stop`, `restart`, `status`, and `uninstall`.
- Real `install/start/stop` flows still require platform-specific permissions.
- If you need the shortest possible setup for an internal tool, see `QuickCLI`, `QuickService`, and `QuickServiceWithStop` in the main package API.
