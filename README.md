# ZCli

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

ZCli 是基于 [Cobra](https://github.com/spf13/cobra) 生态的企业级 CLI 框架和系统服务管理扩展包。它在 Cobra/pflag 之上提供 Builder 装配层、服务生命周期管理、结构化错误处理、多语言支持和跨平台系统服务注册。

## 安装

```bash
go get github.com/darkit/zcli
```

当前源码基线：Go 1.25.0（以 `go.mod` 为准）。

## 快速开始

完整可运行示例见 `examples/` 目录：

| 示例                        | 说明                                                          |
| --------------------------- | ------------------------------------------------------------- |
| `examples/cli-tool`         | 纯 CLI 工具：Builder + Validator + InitHook + NewCommand      |
| `examples/function-service` | 函数式服务：WithService(run, stop) + 超时设置                 |
| `examples/service-runner`   | 接口式服务：ServiceRunner + 依赖注入                          |
| `examples/service-config`   | 高级服务配置：用户/执行路径/结构化依赖/平台选项               |
| `examples/flag-export`      | 标志导出：ExportFlagsForViper + 系统标志过滤                  |
| `examples/complete`         | 全能力演示：Logo、GitInfo、ManagedService、错误处理、中文界面 |

### 纯 CLI 应用

```go
app, err := zcli.NewBuilder("en").
    WithQuickConfig("notesctl", "Notes CLI", "Recommended CLI-only example built with zcli.", "1.0.0").
    WithSilenceUsage(true).
    WithValidator(func(cfg *zcli.Config) error {
        if strings.Contains(cfg.Basic().Name, " ") {
            return fmt.Errorf("application name must not contain spaces")
        }
        return nil
    }).
    WithInitHook(func(cmd *zcli.Command, args []string) error {
        profile, _ := cmd.Root().PersistentFlags().GetString("profile")
        slog.Info("init hook", "profile", profile, "command", cmd.CommandPath())
        return nil
    }).
    WithPersistentFlags(func(flags *zcli.FlagSet) {
        flags.String("profile", "dev", "Runtime profile")
    }).
    WithCommands(statusCmd, echoCmd).
    BuildWithError()
if err != nil {
    os.Exit(zcli.ExitUsage)
}
os.Exit(app.Run())
```

### 函数式服务（小型应用）

```go
app, err := zcli.NewBuilder("en").
    WithName("mailer-worker").
    WithDisplayName("Mailer Worker").
    WithDescription("Recommended function-based service example for small applications.").
    WithVersion("1.0.0").
    WithService(runMailerWorker, stopMailerWorker).
    WithWorkDir(workDir).
    WithEnvVar("APP_ENV", "production").
    WithDependency("network-online.target", zcli.DependencyAfter).
    WithServiceTimeouts(15*time.Second, 20*time.Second).
    BuildWithError()
```

### 接口式服务（推荐用于非 trivial 场景）

```go
type queueWorker struct{ /* ... */ }

func (w *queueWorker) Run(ctx context.Context) error { /* ... */ }
func (w *queueWorker) Stop() error { /* ... */ }
func (w *queueWorker) Name() string { return "queue-worker" }

app, err := zcli.NewBuilder("en").
    WithName(worker.Name()).
    WithDisplayName("Queue Worker").
    WithDescription("Recommended ServiceRunner example for non-trivial services.").
    WithVersion("1.0.0").
    WithServiceRunner(worker).
    WithServiceTimeouts(15*time.Second, 20*time.Second).
    BuildWithError()
```

## 架构概览

### 类型别名：Cobra/pflag 直通

ZCli 通过类型别名将 Cobra 和 pflag 作为底层真相源，同时用 `zcli` 命名空间提供统一的入口词汇：

```go
type Command = cobra.Command          // 所有 Cobra 字段/方法直接可用
type FlagSet = pflag.FlagSet          // 与 pflag 生态完全互操作
type Flag    = pflag.Flag
type App     = Cli                    // App 是 Cli 的语义别名
```

高级用户随时可通过 `app.Command()` 拿到原生 `*cobra.Command`。

### 三层配置架构

```
Config
├── Basic     -- 名称/版本/语言/Logo/颜色开关
├── Service   -- 服务配置：用户/路径/依赖/chroot/平台选项
└── Runtime   -- Run/Stop 函数、超时设置、错误处理器链
```

配置字段已私有化，对外通过 getter 返回副本：

```go
cfg.Basic()   // 返回 Basic（值拷贝）
cfg.Service() // 返回 ServiceConfig（深拷贝）
cfg.Runtime() // 返回 Runtime（深拷贝）
```

性能敏感场景可用 `cfg.View()` 获取只读视图，避免深拷贝开销。

### Builder：链式装配

Builder 是推荐的构造入口。方法链延迟收集配置，在 `Build()` / `BuildWithError()` / `MustBuild()` 阶段统一落盘：

```go
NewBuilder(lang...)    → 创建 Builder
  .WithName / WithVersion / WithLanguage / ...
  .WithService(run, stop) / WithServiceRunner(svc)
  .WithValidator(fn) / WithInitHook(fn) / WithErrorHandler(handler)
  .WithCommand(cmd)    → 延迟收集，不提前触发构建
BuildWithError()        → 返回 (*Cli, error)，推荐
Build()                 → 失败时 panic
MustBuild()             → BuildWithError 包装，失败 panic
```

### 服务生命周期

服务应用自动注册 8 个系统命令：

| 命令         | 说明                         |
| ------------ | ---------------------------- |
| `run`        | 前台运行（开发/调试）        |
| `install`    | 安装为系统服务               |
| `reinstall`  | 覆盖重装（对齐 daemon Force）|
| `start`      | 启动系统服务                 |
| `stop`       | 停止系统服务                 |
| `restart`    | 重启系统服务                 |
| `status`     | 查看服务状态                 |
| `uninstall`  | 卸载系统服务                 |

`install` / `reinstall` 支持透传服务自启参数，两者行为一致：命令之后的所有
token（含未注册的 flag）原样作为服务启动参数保存，daemon 拉起时执行
`Executable + 参数`：

```sh
./app install client --a=111 --b=22   # 自启时运行 ./app client --a=111 --b=22
./app reinstall client --a=111 --b=22 # 覆盖重装，透传方式与 install 相同
```

- 提供参数时**整体替换**服务自启参数（含默认 `run` 与构建期 `WithArguments`）；
  不带参数则保持现状。
- 第一个非 flag token（通常是服务子命令）之后的参数不再被解析、原样作为服务
  参数保存；`install --help` 仍由 cobra 正常响应。
- 注意：这些参数在服务启动时仍会被二进制自身的 cobra 解析一次，目标子命令/flag
  需在二进制中注册，或用 `--` 分隔（如 `./app install client -- --a=111`）。

优雅关闭流程：

```
收到停止信号（SIGINT/SIGTERM/SIGQUIT）
    ↓
取消 Run(ctx)，让服务主循环开始优雅退出
    ↓
正常取消视为成功退出，Execute 返回 nil，不打印 Usage / context canceled
    ↓
等待 ShutdownInitial（默认 15s）
    ↓
[超时] 执行 Stop() → 等待 ShutdownGrace（默认 5s）
    ↓
[超时] 后台模式触发 StopTimeout 强制退出
```

## API 参考

### Builder 方法

#### 基础配置

```go
NewBuilder(lang ...string) *Builder  // 创建 Builder，可选语言参数

WithName(name string) *Builder
WithDisplayName(name string) *Builder
WithDescription(desc string) *Builder
WithVersion(version string) *Builder
WithLanguage(lang string) *Builder
WithLogo(logo string) *Builder
WithDebug(debug bool) *Builder
WithGitInfo(commitID, branch, tag string) *Builder
WithBuildTime(buildTime string) *Builder
WithDefaults() *Builder               // 补全 Language/Version 默认值
WithQuickConfig(name, displayName, description, version string) *Builder
WithDefaultConfig() *Builder          // 一键设置 en + 1.0.0 + debug=false
```

#### 服务配置

```go
WithService(run RunFunc, stop ...StopFunc) *Builder
WithServiceRunner(service ServiceRunner) *Builder   // 传 nil 则构建期报错
WithWorkDir(dir string) *Builder
WithEnvVar(key, value string) *Builder
WithServiceUser(username string) *Builder
WithExecutable(path string) *Builder
WithArguments(args ...string) *Builder   // 传空列表可显式清空默认 "run"
WithChRoot(dir string) *Builder
WithDependencies(deps ...string) *Builder              // require 型字符串依赖
WithLegacyDependencies(deps ...string) *Builder         // daemon 原生字符串依赖
WithStructuredDependencies(deps ...Dependency) *Builder // 追加结构化依赖
SetStructuredDependencies(deps ...Dependency) *Builder  // 覆盖结构化依赖
WithDependency(name string, depType DependencyType) *Builder // 单个结构化依赖
WithServiceOption(key string, value any) *Builder       // daemon 平台选项
WithServiceOptionsMap(options ServiceOptions) *Builder   // 批量 daemon 平台选项
WithAllowSudoFallback(enabled bool) *Builder
WithForce(enabled bool) *Builder                 // Install 覆盖已有配置
WithNotify(enabled bool) *Builder                // systemd Type=notify / Windows 就绪通知
WithCustomService(fn func(*Config)) *Builder
WithServiceConfig(fn func(*ServiceConfig)) *Builder
```

#### 超时与钩子

```go
WithShutdownTimeouts(initial, grace time.Duration) *Builder   // 优雅退出分级超时
WithServiceTimeouts(start, stop time.Duration) *Builder        // daemon 启动/停止超时
WithRestartTimeout(d time.Duration) *Builder                   // daemon 重启间隔 Timeout.Restart
WithInitHook(hook InitHook) *Builder
WithValidator(validator func(*Config) error) *Builder
WithErrorHandler(handler ErrorHandler) *Builder
```

#### 其他

```go
WithCommand(cmd *Command) *Builder            // 延迟收集命令
WithCommands(cmds ...*Command) *Builder       // 批量延迟收集
WithCommandFactory(fn CommandFactory) *Builder // 构建期注入需要 *Cli 的命令
WithPersistentFlags(func(*FlagSet)) *Builder // 构建期配置根持久标志
WithFlags(func(*FlagSet)) *Builder           // 构建期配置根本地标志
WithEnvVars(vars map[string]string) *Builder
WithSilenceErrors(enabled bool) *Builder
WithSilenceUsage(enabled bool) *Builder
WithNoColor(enabled bool) *Builder
WithRootAction(action RootAction) *Builder // auto/run/help
WithContext(ctx context.Context) *Builder
WithMousetrapDisabled(disabled bool) *Builder // 禁用 Windows 双击提示
WithRuntime(rt *Runtime) *Builder
WithServiceOptions(workDir string, envVars map[string]string, deps ...string) *Builder
```

#### 终止方法

```go
BuildWithError() (*Cli, error)  // 返回错误，推荐
Build() *Cli                     // 失败时 panic
MustBuild() *Cli                 // BuildWithError 包装，失败 panic
```

### Cli / App 方法

```go
// 执行
Execute() error
ExecuteC() (*Command, error)
ExecuteContext(ctx context.Context) error
ExecuteContextC(ctx context.Context) (*Command, error)
Run() int                      // Execute + 退出码，适合 os.Exit(app.Run())
Main()                         // 执行并以退出码结束进程
ExitCode(err error) int        // 0 成功 / 2 用法或配置 / 1 其他失败

// 命令管理
AddCommand(cmds ...*Command)
Commands() []*Command
Command() *Command         // 返回底层 *cobra.Command

// 标志访问
Flags() *FlagSet
PersistentFlags() *FlagSet
LocalFlags() *FlagSet
InheritedFlags() *FlagSet
Flag(name string) *Flag

// 配置访问
Config() *Config
```

### 核心类型

```go
// RunFunc 服务运行函数签名
type RunFunc func(ctx context.Context) error

// StopFunc 服务停止函数签名
type StopFunc func() error

// ServiceRunner 服务接口（推荐用于非 trivial 场景）
type ServiceRunner interface {
    Run(ctx context.Context) error
    Stop() error
    Name() string
}

// InitHook 初始化钩子，返回 error 则中断命令执行
type InitHook func(cmd *Command, args []string) error
```

### 依赖类型

```go
DependencyAfter   // 当前服务在该依赖之后启动
DependencyBefore  // 当前服务在该依赖之前启动
DependencyRequire // 必须依赖，失败则当前服务不启动
DependencyWant    // 期望依赖，失败不阻止当前服务启动

// 依赖构造函数（对齐 daemon，用于 WithDependency / WithStructuredDependencies）
After(name string) Dependency
Before(name string) Dependency
Require(name string) Dependency
Want(name string) Dependency

// 平台选项常量（用于 WithServiceOption / WithServiceOptionsMap）
OptionRestart / OptionRestartSec / OptionRunWait / OptionForceInstall /
OptionNotify / OptionLimitNOFILE

// Ready 在 WithNotify(true) 后通知 systemd/Windows 服务已就绪
Ready() error
```

### 命令工厂

```go
NewCommand(use, short string, opts ...CommandOption) *Command

// CommandOption 函数列表
WithCommandAliases(aliases ...string) CommandOption
WithCommandLong(long string) CommandOption
WithCommandExample(example string) CommandOption
WithCommandRun(run func(cmd *Command, args []string)) CommandOption
WithCommandRunE(runE func(cmd *Command, args []string) error) CommandOption
WithCommandArgs(args PositionalArgs) CommandOption
WithCommandGroup(groupID string) CommandOption
WithCommandValidArgs(args ...Completion) CommandOption
WithCommandCompletion(completion CompletionFunc) CommandOption
WithCommandFlags(configure func(flags *FlagSet)) CommandOption
WithCommandPersistentFlags(configure func(flags *FlagSet)) CommandOption
WithCommandSubcommands(children ...*Command) CommandOption
WithCommandHidden(hidden bool) CommandOption
WithCommandSilenceUsage(enabled bool) CommandOption
WithCommandSilenceErrors(enabled bool) CommandOption
WithCommandPreRunE(fn) CommandOption
WithCommandPersistentPreRunE(fn) CommandOption
```

### 便利函数

```go
QuickCLI(name, displayName, description string) *Cli
QuickService(name, displayName string, run RunFunc) *Cli
QuickServiceWithStop(name, displayName string, run RunFunc, stop StopFunc) *Cli
NewApp(opts ...Option) *App
NewCli(opts ...Option) *Cli
```

### 标志导出

```go
ExportFlagsForViper(excludeFlags ...string) []*FlagSet   // 导出给 Viper 绑定（按名字去重）
GetBindableFlagSets(excludeFlags ...string) []*FlagSet    // 过滤系统标志后的绑定集合
GetAllFlagSets() []*FlagSet
GetFilteredFlags(excludeFlags ...string) *FlagSet
GetFlagNames(includeInherited bool) []string
GetFilteredFlagNames(excludeFlags ...string) []string
GetSystemFlags() []string                                 // 调试用：列出系统标志
IsSystemFlag(flagName string) bool
```

### 结构化错误

```go
// 错误码常量
ErrConfigValidation / ErrServiceCreate / ErrServiceStart / ErrServiceStop / ...

// 错误构建器（流式 API）
NewError(code ErrorCode) *ErrorBuilder
    .Operation(operation string) *ErrorBuilder
    .Service(service string) *ErrorBuilder
    .Message(message string) *ErrorBuilder
    .Messagef(format string, args ...any) *ErrorBuilder
    .Cause(cause error) *ErrorBuilder
    .Context(key string, value any) *ErrorBuilder
    .Stack(stack []string) *ErrorBuilder
    .Build() *ServiceError

// 预定义错误函数
ErrServiceAlreadyRunning(service string) *ServiceError
ErrServiceAlreadyStopped(service string) *ServiceError
ErrServiceNotInstalled(service string) *ServiceError
ErrServiceStartTimeout(service string, timeout time.Duration) *ServiceError
ErrServiceStopTimeout(service string, timeout time.Duration) *ServiceError
ErrPermissionDenied(path, required, current string) *ServiceError
ErrConfigValidationFailed(details []error) *ServiceError

// 错误工具
IsServiceError(err error) bool
GetServiceError(err error) (*ServiceError, bool)
IsErrorCode(err error, code ErrorCode) bool
WrapError(err error, code ErrorCode, operation string) *ServiceError
CombineErrors(errs ...error) error

// 错误聚合器
NewErrorAggregator() *ErrorAggregator
    .Add(err) / .HasErrors() / .Count() / .Errors() / .Error() / .Clear()
```

### 上下文与关闭原因

```go
// ShutdownCause 获取
GetShutdownCause(ctx context.Context) (*ShutdownCause, bool)

// 关闭原因枚举
ShutdownReasonSignal         // 系统信号（SIGINT/SIGTERM/SIGQUIT）
ShutdownReasonServiceStop    // 服务管理器停止请求
ShutdownReasonExternalCancel // 父 Context 被取消
```

### 多语言

```go
SetLanguage(lang string) error
GetLanguageManager() *LanguageManager
NewLanguageManager(lang string) *LanguageManager

// Language 结构包含四域
type Language struct {
    Code    string
    Service ServiceDomain   // 服务操作/状态/消息
    UI      UIDomain        // 命令/帮助/版本界面
    Error   ErrorDomain     // 服务/系统/帮助错误
    Format  FormatDomain    // 格式化模板
}
```

## 常见模式

### 模式一：纯 CLI 工具

```go
app, _ := zcli.NewBuilder("zh").
    WithName("mytool").
    WithVersion("1.0.0").
    BuildWithError()

app.AddCommand(zcli.NewCommand("greet [name]",
    "打印问候语",
    zcli.WithCommandSilenceUsage(true),
    zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
        name := "world"
        if len(args) > 0 {
            name = args[0]
        }
        fmt.Fprintf(cmd.OutOrStdout(), "Hello, %s!\n", name)
        return nil
    }),
))

os.Exit(app.Run())
```

### 模式二：函数式服务

```go
app, _ := zcli.NewBuilder("zh").
    WithName("myworker").
    WithService(func(ctx context.Context) error {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return nil
            case <-ticker.C:
                doWork()
            }
        }
    }, func() error {
        cleanup()
        return nil
    }).
    BuildWithError()

app.Execute()
```

### 模式三：接口式服务（依赖注入）

```go
type MyService struct {
    db    *sql.DB
    cache *redis.Client
}

func (s *MyService) Run(ctx context.Context) error { /* ... */ }
func (s *MyService) Stop() error                    { /* ... */ }
func (s *MyService) Name() string                   { return "my-service" }

svc := &MyService{db: db, cache: cache}

app, _ := zcli.NewBuilder("zh").
    WithName(svc.Name()).
    WithServiceRunner(svc).
    WithValidator(func(cfg *zcli.Config) error {
        if cfg.Basic().Name != svc.Name() {
            return fmt.Errorf("name mismatch")
        }
        return nil
    }).
    BuildWithError()

app.Execute()
```

### 模式四：标志绑定到 Viper

```go
app, _ := zcli.NewBuilder("zh").
    WithName("app").
    BuildWithError()

app.PersistentFlags().String("config", "", "配置文件路径")
app.PersistentFlags().Bool("debug", false, "启用调试日志")

// 导出业务标志（自动排除 help/version/completion 等系统标志）
flagSets := app.ExportFlagsForViper()

// 传给 Viper 的 WithBindPFlags
viperConfig := config.NewConfig(config.WithBindPFlags(flagSets...))
```

## 注意事项

- 生产代码优先使用 `BuildWithError()` 而不是 `Build()`，`Build()` 在验证失败时会 panic
- `WithServiceRunner(nil)` / `WithService(nil)` / `WithRuntime(nil)` / 非法 `WithBuildTime` 不会 panic——错误会累积到 `BuildWithError()` 阶段返回
- `WithCommand()` / `WithCommands()` / `WithCommandFactory()` 在 Builder 未构建时延迟收集，构建后则直接追加
- 需要读取 `*Cli` 的命令用 `WithCommandFactory`，不要再 Build 后 Find/Remove/Add
- 命令路径重复时最后注册的生效（与 Cobra 行为一致）
- 系统标志（help/version/completion 等共 23 个）在 `ExportFlagsForViper` 中自动排除，且同一标志只导出一次
- 帮助输出到 stdout；未知主题仍走 stderr
- 系统服务命令按 `ServiceCommandGroupID` 分组，不再按 `status`/`run` 等名字误伤用户命令
- 纯服务无子命令默认 `run`；存在用户子命令时默认显示帮助，可用 `WithRootAction` 覆盖
- `GetServiceError` / `IsServiceError` / `CombineErrors` 走 error chain，包装后仍可识别 `*ServiceError`

## 依赖

| 依赖                       | 用途                         |
| -------------------------- | ---------------------------- |
| `github.com/spf13/cobra`   | 命令树、补全与标志互操作底座 |
| `github.com/spf13/pflag`   | 命令行标志解析               |
| `github.com/fatih/color`   | ANSI 彩色输出                |
| `github.com/darkit/daemon` | 跨平台系统服务管理           |

## 版本历史

### v0.5.0（当前）

> 破坏性变更（相对 v0.4.x）：`(*Cli).Config()` 返回 `*Config`；
> `WithStructuredDependencies` 改为追加语义（覆盖用 `SetStructuredDependencies`）；
> `GetBindableFlagSets` 返回去重后的单个合并 `FlagSet`；`NewCli` 复用 `NewApp` 装配流水线；
> 恢复标准 `completion` 子命令；服务命令改按 `ServiceCommandGroupID` 分组；
> 纯服务无子命令默认 `run`、混合应用默认 `help`（可用 `WithRootAction` 覆盖）。

- 修复 `attachInitHooks` 重复调用问题
- `sync.Once` 确保 `stopChan` 只关闭一次
- Windows 颜色检测改用安全 DLL 加载
- `ManagedService.Run()` 正确返回生命周期钩子错误；正常取消返回 `nil`，避免 Ctrl+C 打印 Usage / context canceled
- 新增 `WithErrorHandler` 支持自定义错误处理器链
- 新增 `Config.View()` 只读视图，避免频繁深拷贝
- 命令管理方法按职责拆分为多个文件
- 测试稳定性：支持 `ZCLI_TEST_TIMEOUT_MULTIPLIER` 环境变量
- Builder 构建路径收束：`Build()` 复用 `BuildWithError()`；非法 `WithBuildTime` / `WithRuntime(nil)` / `WithService(nil)` 改为构建期累积错误
- 新增构建期装配糖：`WithCommands`、`WithCommandFactory`、`WithPersistentFlags`、`WithFlags`、`WithEnvVars`、`WithSilence*`
- 新增 `Run()` / `Main()` / `ExitCode()`；`CombineErrors` 与 `GetServiceError` 走 error chain
- 帮助改写 stdout；服务命令按 `ServiceCommandGroupID` 分组，不再按名字误伤用户命令
- `ExportFlagsForViper` 按标志名去重；服务装配失败回传到 `BuildWithError`
- 根命令策略：纯服务默认 `run`，混合应用默认 help，可用 `WithRootAction` 覆盖
- 恢复标准 `completion` 子命令；`WithStructuredDependencies` 改为追加，覆盖用 `SetStructuredDependencies`
- `install` / `reinstall` 支持透传服务自启参数：`./app install client --a=111 --b=22`，第一个非 flag token 之后原样保存，整体替换 daemon Arguments
- 对抗审查修复：`CombineErrors` 单错误原样返回（不再被 `errors.Join` 包装）、`AddCommand(nil)` 防御、`ErrorAggregator` 并发安全
- 新增依赖构造函数 `After`/`Before`/`Require`/`Want` 与平台选项常量（`OptionRestart` 等）、`Ready()`

### v0.2.0

- 新增 `ServiceRunner` 接口，统一服务抽象
- Builder 新增 `WithServiceRunner`、`WithValidator` 方法
- 新增 `BuildWithError`，错误优先设计
- 结构化错误系统：`ErrorCode`、`ErrorBuilder`、`ErrorAggregator`、预定义错误函数
- 并发安全保障：`atomic.Bool` 状态管理
- 便利性 API：`NewApp`、`NewCommand`、`QuickService`、`QuickCLI`、`QuickServiceWithStop`
- 修复服务管理中的 race condition 和死锁

### v0.1.x

- 基础 CLI 框架功能
- 系统服务管理
- 多语言支持

## 许可证

[MIT](LICENSE)
