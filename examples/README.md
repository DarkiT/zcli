# 官方示例

本目录包含 `github.com/darkit/zcli` 当前支持的官方示例，与公开 API 保持一致。

约定：

- 示例默认语言为英文
- 可能失败的构建路径统一使用 `BuildWithError()`
- 非 trivial 服务推荐使用 `WithServiceRunner`
- 小型服务可使用 `WithService(run, stop)` 作为简洁选项

## cli-tool

纯 CLI 工具推荐起点。展示 Builder + Validator + InitHook + NewCommand 的组合用法。

```bash
go run ./examples/cli-tool --help
go run ./examples/cli-tool status --profile prod
go run ./examples/cli-tool echo hello world
```

## function-service

函数式服务的推荐起点，无需定义专用服务结构体。`WithService(run, stop)` 直接传入函数。

```bash
go run ./examples/function-service --help
go run ./examples/function-service run
```

## service-runner

实现 `ServiceRunner` 接口的推荐起点。适合有显式依赖和专用服务类型的场景，支持依赖注入。

```bash
go run ./examples/service-runner --help
go run ./examples/service-runner run
```

## service-config

高级服务注册示例，聚焦安装期配置：用户权限、可执行文件路径、运行参数、结构化依赖和平台选项。

```bash
go run ./examples/service-config --help
go run ./examples/service-config inspect
```

## flag-export

标志导出与外部配置系统（Viper 等）集成的参考示例，展示系统标志过滤和可绑定标志集导出。

```bash
go run ./examples/flag-export --help
go run ./examples/flag-export inspect --region eu-west-1 --output yaml
```

## complete

全能力演示，展示 zcli 全部特性的优雅调用方案：Builder 全量配置、Logo、版本信息（GitInfo/BuildTime）、ServiceRunner + ServiceLifecycle → ManagedService、结构化错误处理（ErrorBuilder/ErrorAggregator/LoggingErrorHandler/RecoveryErrorHandler）、Validator + InitHook 双钩子、标志导出、中文界面。

```bash
go run ./examples/complete --help
go run ./examples/complete inspect
go run ./examples/complete health
go run ./examples/complete config show
go run ./examples/complete --version
# 前台服务模式：Ctrl+C 应优雅退出，不打印 Usage / context canceled
# go run ./examples/complete
```

## 注意事项

- 所有服务示例自动注册 `run`、`install`、`start`、`stop`、`restart`、`status`、`uninstall` 共 7 个系统命令
- `install` / `start` / `stop` 流程需要平台特定权限
- 如需最短启动路径（内部工具），参考主包 API 中的 `QuickCLI`、`QuickService` 和 `QuickServiceWithStop`

## 自动化端到端回归

官方示例的主链路由 `tests/e2e/examples_test.go` 覆盖。重点回归：

- `cli-tool` 的 help/status/echo
- `flag-export` 的系统标志过滤与业务标志导出
- `service-config` 的配置 inspect 输出
- `function-service`、`service-runner`、`complete` 的 SIGINT 优雅退出；正常停止不得打印 `Usage:`、`context canceled` 或命令失败日志
