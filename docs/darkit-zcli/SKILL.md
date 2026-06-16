---
name: darkit-zcli
description: zcli 工作区本地导航层。用于构建、审查、重构、编写文档或排查基于 zcli 的 Go CLI 应用及 zcli 框架本身的问题，特别是涉及 NewBuilder/BuildWithError、QuickCLI/QuickService、Cobra 命令树、标志导出（ExportFlagsForViper）、ServiceRunner/WithService、daemon 服务安装配置、i18n 语言包、优雅关闭（ShutdownCause）以及官方示例对齐的场景。
---

# darkit-zcli

本 skill 是 `github.com/darkit/zcli` 工作区的本地导航层，不替代完整 API 文档。

## 入口判断

收到 zcli 相关任务时，先确定属于哪个领域：

| 领域         | 典型信号                                              |
| ------------ | ----------------------------------------------------- |
| 示例         | "跑一下示例"、"examples"、"快速开始"                  |
| Builder/配置 | "怎么构建"、"配置选项"、"validator"、"BuildWithError" |
| 命令树/UI    | "添加命令"、"帮助界面"、"completion"、"NewCommand"    |
| 服务生命周期 | "run/install/start/stop"、"优雅关闭"、"ShutdownCause" |
| i18n/输出    | "多语言"、"中文输出"、"语言包"                        |
| 标志导出     | "Viper"、"ExportFlagsForViper"、"系统标志"            |
| 排障         | "报错"、"不工作"、"timeout"                           |

## 工作流

1. 按领域打开对应资源文件或官方示例
2. 仅在第一份资源不够用时再检视源码
3. 用测试文件确认行为边界
4. 当文档与源码或测试冲突时，以源码和测试为准

## 按领域路由

### 快速开始 / 公开使用

读 `README.md` → 读 `examples/README.md`

### Builder、配置、钩子、验证

读 `./resources/config-options.md` → 检视 `builder.go`、`options.go`

### 命令树、帮助、版本、补全、标志

读 `examples/cli-tool`、`examples/flag-export` → 检视 `command.go`、`command_factory.go`、`command_flags.go`、`zcli.go`

### 服务生命周期、前台/后台、优雅关闭

读 `./resources/context-lifecycle.md`、`./resources/service-management.md` → 检视 `service.go`、`service_runtime.go`、`service_support.go`、`shutdown_cause.go`

### 服务命令语义（run/install/start/stop/restart/status/uninstall）

检视 `service_commands.go`

### 服务接口抽象（ServiceRunner / FuncService / ManagedService）

检视 `service_interface.go`

### 高级服务注册（Executable / ChRoot / AllowSudoFallback / 平台 Options）

检视 `builder.go`、`service.go`、`service_interface.go` → 交叉参考 `examples/service-config`

### i18n / 帮助渲染 / 语言切换

检视 `i18n.go`、`i18n_manager.go`、`i18n_localizer.go`、`i18n_builtin.go` → 当前示例默认英文，中文需显式指定语言参数

### 标志导出 / Viper 集成

读 `examples/flag-export` → 检视 `command_flags.go`

### 结构化错误 / ErrorBuilder / ErrorAggregator

检视 `errors.go`

### 排障

先读 `./resources/troubleshooting.md` → 打开最小相关测试文件

### 设计理由 / 审计历史

读 `DESIGN.md`、`./resources/design-audit-2026-03.md`

## 高信号源码文件

| 文件                   | 承载内容                                                            |
| ---------------------- | ------------------------------------------------------------------- |
| `builder.go`           | Builder 链式构建器，所有 `With*` 方法，Quick 函数                   |
| `options.go`           | Config 三层架构，RunFunc/StopFunc 签名，深拷贝工具                  |
| `command.go`           | Cli 核心结构体，Execute 系列方法，标志/命令管理方法                 |
| `command_factory.go`   | NewCommand 工厂，CommandOption 函数族，Args 校验器                  |
| `command_flags.go`     | 标志导出、系统标志过滤、GetBindableFlagSets                         |
| `service.go`           | sManager 创建、buildRunner、createServiceConfig、权限检查           |
| `service_commands.go`  | 7 个系统命令装配（run/install/start/stop/restart/status/uninstall） |
| `service_runtime.go`   | 服务运行时：managed run/stop/wait、executeRunCommand                |
| `service_support.go`   | 权限检查、强制退出调度、错误处理器链                                |
| `service_interface.go` | ServiceRunner 接口、BaseService、FuncService、ManagedService        |
| `shutdown_cause.go`    | ShutdownCause 结构与 3 种关闭原因                                   |
| `errors.go`            | ErrorCode、ServiceError、ErrorBuilder、ErrorAggregator              |
| `i18n.go`              | 四域语言包结构定义                                                  |
| `i18n_manager.go`      | LanguageManager、回退策略                                           |
| `i18n_builtin.go`      | 内置 zh/en 语言包                                                   |
| `app_alias.go`         | App = Cli 别名、NewApp、newAppBase                                  |
| `version.go`           | VersionInfo、NewVersion、String()                                   |
| `zcli.go`              | applyBuilderAssembly、configureRootCommand、addHelpCommand          |

## 高信号测试文件

| 文件                                | 覆盖领域                                   |
| ----------------------------------- | ------------------------------------------ |
| `enhanced_test.go`                  | Builder 增强功能集成测试                   |
| `service_alignment_test.go`         | 服务管理与 CLI 对齐验证                    |
| `service_command_semantics_test.go` | 7 个命令语义正确性                         |
| `service_signal_test.go`            | 信号处理与优雅关闭                         |
| `tests/e2e/examples_test.go`        | 官方 examples 端到端与 SIGINT 静默退出回归 |
| `service_force_exit_test.go`        | 强制退出路径                               |
| `service_concurrent_test.go`        | 并发安全                                   |
| `service_edge_cases_test.go`        | 边界条件                                   |
| `i18n_test.go`                      | 多语言系统                                 |

## 当前能力速查

- 生产代码优先使用 `BuildWithError()`；`Build()` 在验证失败时 panic
- 非 trivial 服务推荐 `WithServiceRunner(...)`；小型服务可用 `WithService(run, stop)`
- `WithServiceRunner(nil)` 不会 panic——错误累积到 `BuildWithError()` 阶段
- `WithCommand()` 在 Builder 未构建时延迟收集，构建后直接追加
- 默认关闭预算：`ShutdownInitial=15s` + `ShutdownGrace=5s`
- 默认 `StopTimeout=20s`，后台模式超时触发强制退出
- `ExportFlagsForViper` 自动排除 23 个系统标志
- 示例默认英文，中文需 `NewBuilder("zh")`
- `ManagedService` 正常取消会返回 `nil`，并保证 `BeforeStop` / `Stop` / `AfterStop` 在并发关闭路径只执行一次
- 官方 E2E 位于 `tests/e2e/examples_test.go`，会断言 Ctrl+C 不打印 `Usage:` / `context canceled` / 命令失败日志

## 约束

- 保持加载路径窄：不要默认读取所有资源文件
- 本 skill 是仓库本地工作流指引，不是源码替代品
- 更新文档或示例时，确保与当前代码库和测试保持一致
- 示例路径只参考 `examples/` 目录下的 6 个官方示例（含 `examples/complete`），不依赖已移除的历史路径
