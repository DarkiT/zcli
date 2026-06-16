# DarkiT-ZCli API 参考

> 契约说明：当前主线遵循 **兼容内核 + 增强装配 + 优雅糖层**。  
> 常规使用路径优先使用 `zcli` 暴露的类型和方法；高级互操作场景仍允许直达底层 Cobra / pflag 能力。  
> 详细边界见 `cobra-fusion-contract.md`。

## Builder API

### 创建 Builder

```go
func NewBuilder(lang ...string) *Builder
```

创建新的 Builder 实例。`lang` 可选；未传时使用默认配置，传入 `"zh"` / `"en"` 可指定初始语言。

**返回**：
- `*Builder`: Builder 实例

**示例**：

```go
builder := zcli.NewBuilder("zh")
```

### 基础配置方法

```go
func (b *Builder) WithName(name string) *Builder
```
设置服务名称（必需）。

---

```go
func (b *Builder) WithDisplayName(name string) *Builder
```
设置显示名称。

---

```go
func (b *Builder) WithDescription(desc string) *Builder
```
设置应用描述。

---

```go
func (b *Builder) WithVersion(version string) *Builder
```
设置版本号。

---

```go
func (b *Builder) WithLogo(logo string) *Builder
```
设置 Logo 文本（ASCII 艺术字）。

---

```go
func (b *Builder) WithLanguage(lang string) *Builder
```
设置语言。

---

```go
func (b *Builder) WithGitInfo(commitID, branch, tag string) *Builder
func (b *Builder) WithBuildTime(buildTime string) *Builder
func (b *Builder) WithDebug(debug bool) *Builder
func (b *Builder) WithDefaults() *Builder
func (b *Builder) WithDefaultConfig() *Builder
func (b *Builder) WithQuickConfig(name, displayName, description, version string) *Builder
```

设置构建信息、调试标记和常用默认值。

### 服务配置方法

```go
func (b *Builder) WithService(run RunFunc, stop ...StopFunc) *Builder
```

配置服务运行和停止函数。

**参数**：
- `run`: 运行函数，签名 `func(ctx context.Context) error`
- `stop`: 可选停止函数，签名 `func() error`

**示例**：

```go
builder.WithService(
    func(ctx context.Context) error {
        // 运行逻辑，监听 ctx.Done()
        <-ctx.Done()
        return nil
    },
    func() error {
        // 清理逻辑
        return nil
    },
)
```

---

```go
func (b *Builder) WithServiceRunner(runner ServiceRunner) *Builder
```

注入 ServiceRunner 接口实现（推荐方式）。

**参数**：
- `runner`: ServiceRunner 接口实例

**边界说明**：
- 当 `runner == nil` 时，不再立即 panic
- 该错误会在 `BuildWithError()` / `Build()` 阶段暴露；推荐使用 `BuildWithError()` 处理

---

```go
func (b *Builder) WithValidator(validator func(*Config) error) *Builder
```

注入配置验证器。

**参数**：
- `validator`: 验证函数

**示例**：

```go
// 注意：必须使用 getter 方法访问 Config 字段
builder.WithValidator(func(cfg *zcli.Config) error {
    if cfg.Basic().Name == "" {  // 使用 cfg.Basic() 而非 cfg.Basic
        return fmt.Errorf("名称不能为空")
    }
    return nil
})
```

---

```go
func (b *Builder) WithWorkDir(dir string) *Builder
```
设置工作目录。

---

```go
func (b *Builder) WithEnvVar(key, value string) *Builder
```
添加环境变量。

---

```go
func (b *Builder) WithDependencies(deps ...string) *Builder
```
设置 require 型结构化依赖，并清空 daemon 原生字符串依赖。

---

```go
func (b *Builder) WithLegacyDependencies(deps ...string) *Builder
func (b *Builder) WithStructuredDependencies(deps ...Dependency) *Builder
func (b *Builder) WithDependency(name string, depType DependencyType) *Builder
```

设置 daemon 原生字符串依赖或结构化依赖。

---

```go
func (b *Builder) WithServiceUser(username string) *Builder
func (b *Builder) WithExecutable(path string) *Builder
func (b *Builder) WithArguments(args ...string) *Builder
func (b *Builder) WithChRoot(dir string) *Builder
func (b *Builder) WithServiceOption(key string, value any) *Builder
func (b *Builder) WithServiceOptionsMap(options ServiceOptions) *Builder
func (b *Builder) WithAllowSudoFallback(enabled bool) *Builder
func (b *Builder) WithServiceConfig(fn func(*ServiceConfig)) *Builder
```

配置系统服务安装与平台特定选项。`WithArguments()` 传入空列表会显式清空默认的 `"run"` 参数。

---

```go
func (b *Builder) WithShutdownTimeouts(initial, grace time.Duration) *Builder
func (b *Builder) WithServiceTimeouts(start, stop time.Duration) *Builder
func (b *Builder) WithMousetrapDisabled(disabled bool) *Builder
func (b *Builder) WithRuntime(rt *Runtime) *Builder
func (b *Builder) WithContext(ctx context.Context) *Builder
```

设置运行时上下文、分级优雅关闭超时、daemon 启动/停止超时和 Windows mousetrap 行为。

---

```go
func (b *Builder) WithErrorHandler(handler ErrorHandler) *Builder
```
添加错误处理器到处理链。

**参数**：
- `handler`: 实现 `ErrorHandler` 接口的处理器

**示例**：

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
builder.WithErrorHandler(zcli.NewLoggingErrorHandler(slogAdapter{logger}))
builder.WithErrorHandler(zcli.NewRecoveryErrorHandler(3, time.Second))
```

### 构建方法

```go
func (b *Builder) BuildWithError() (*Cli, error)
```

构建应用并返回错误（推荐）。

**说明**：
- 会统一附加 pending commands 与 init hooks
- 适合处理 `WithServiceRunner(nil)`、validator 失败等配置错误

**返回**：
- `*Cli`: CLI 实例
- `error`: 构建错误

---

```go
func (b *Builder) Build() *Cli
```

构建应用，失败时 panic。

**返回**：
- `*Cli`: CLI 实例

## Cli API

### 执行方法

```go
func (c *Cli) Execute() error
```
执行应用主逻辑。

---

```go
func (c *Cli) ExecuteContext(ctx context.Context) error
```
使用指定上下文执行应用。

### 命令管理

```go
func (c *Cli) AddCommand(cmds ...*Command)
```

添加自定义命令。

**示例**：

```go
cmd := &zcli.Command{
    Use:   "custom",
    Short: "自定义命令",
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("执行自定义命令")
    },
}
app.AddCommand(cmd)
```

---

```go
func (c *Cli) Command() *Command
```

返回底层根命令，作为高级互操作的 raw escape hatch。`zcli.Command` 是 `cobra.Command` 类型别名，因此返回值仍是 Cobra 原生命令对象。

**使用建议**：
- 常规路径优先使用 `zcli.Command`
- 只有在必须调用底层 Cobra 特性、且 `Cli` 尚未提供对应快捷方法时，再使用该接口

### 标志管理

```go
func (c *Cli) Flags() *FlagSet
```
获取标志集合。

---

```go
func (c *Cli) PersistentFlags() *FlagSet
```
获取持久化标志。

---

```go
func (c *Cli) ExportFlagsForViper(exclude ...string) []*FlagSet
```

导出标志给 Viper，自动过滤 23 个系统标志。

**默认排除的系统标志**：
- 帮助系统：`help`, `h`
- 版本系统：`version`, `v`
- 补全系统：`completion`, `*-completion`, `gen-completion`
- Cobra 内部补全：`__complete`, `__completeNoDesc`, `no-descriptions`
- 兼容补全：`bash-completion`, `zsh-completion`, `fish-completion`, `powershell-completion`
- 配置系统常见辅助标志：`config-help`, `print-config`, `validate-config`

**示例**：

```go
flags := app.ExportFlagsForViper()
// 绑定到 Viper
for _, fs := range flags {
    viper.BindPFlags(fs)
}
```

## 兼容性边界

### 公开词汇

- 推荐对外使用 `*zcli.Command`
- 推荐对外使用 `*zcli.FlagSet`
- 推荐通过 `BuildWithError()` 处理构建阶段错误

### 原生互操作

以下场景仍允许直接使用底层 Cobra / pflag：

- `app.Command()` 取得根命令并调用尚未包进 `Cli` 的底层能力
- 与第三方库做 flag / completion / hook 级互操作
- 调试或渐进迁移期间，需要核对底层对象行为

### Go 版本说明

- 当前源码基线以仓库 `go.mod` 为准
- 若外部文档与 `go.mod` 存在不一致，以源码与验证结果为准，不以旧文案为准

## ServiceRunner 接口

```go
type ServiceRunner interface {
    Run(ctx context.Context) error
    Stop() error
    Name() string
}
```

### 方法说明

**Run 方法**：
- 运行服务主逻辑
- 必须监听 `ctx.Done()` 实现优雅关闭
- 正常关闭推荐返回 `nil`；底层会把 `context.Canceled` / `context.DeadlineExceeded` / `ShutdownCause` 归一为成功退出
- 返回业务 `error` 表示异常退出

**Stop 方法**：
- 停止服务，执行清理工作
- 在 Run 返回后调用

**Name 方法**：
- 返回服务名称

### 实现示例

```go
type MyService struct {
    db     *sql.DB
    stopCh chan struct{}
}

func (s *MyService) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-s.stopCh:
            return nil
        default:
            // 业务逻辑
        }
    }
}

func (s *MyService) Stop() error {
    close(s.stopCh)
    return s.db.Close()
}

func (s *MyService) Name() string {
    return "my-service"
}
```

## Config API

### 重要说明

Config 结构体字段是**私有的**，必须通过 getter 方法访问：

```go
type Config struct {
    basic   *Basic          // 私有字段
    service *ServiceConfig  // 私有字段
    runtime *Runtime        // 私有字段
    ctx     context.Context
}

// 必须通过这些方法访问
func (c *Config) Basic() Basic           // 返回基础配置副本
func (c *Config) Service() ServiceConfig // 返回服务配置副本
func (c *Config) Runtime() Runtime       // 返回运行时配置副本
func (c *Config) Context() context.Context
func (c *Config) View() *ConfigView      // 返回只读视图（性能优化）
```

### ConfigView 只读视图

`ConfigView` 提供只读访问，避免频繁深拷贝的性能开销：

```go
view := cfg.View()
name := view.Basic().Name       // 直接访问，无拷贝
envVars := view.Service().EnvVars
```

**注意**：调用方不应修改返回的指针内容。

### 正确使用示例

```go
func validateConfig(cfg *zcli.Config) error {
    // 正确：使用 getter 方法
    if cfg.Basic().Name == "" {
        return fmt.Errorf("应用名称不能为空")
    }
    if cfg.Basic().Version == "" {
        return fmt.Errorf("版本号不能为空")
    }
    if cfg.Service().EnvVars["DB_HOST"] == "" {
        return fmt.Errorf("数据库主机不能为空")
    }
    return nil
}
```

### 错误示例（编译失败）

```go
// 错误：直接访问私有字段
if cfg.Basic.Name == "" {     // 编译错误
if cfg.Service.EnvVars == nil // 编译错误
```

## 便利性 API

```go
func QuickService(name, displayName string, run RunFunc) *Cli
```
快速创建服务应用。

---

```go
func QuickServiceWithStop(name, displayName string, run RunFunc, stop StopFunc) *Cli
```
快速创建带停止函数的服务应用。

---

```go
func QuickCLI(name, displayName, description string) *Cli
```
快速创建基础 CLI 应用。

## Error API

### 创建错误

```go
func NewError(code ErrorCode) *ErrorBuilder
```

创建新的错误构建器。

### ErrorBuilder 方法

```go
func (eb *ErrorBuilder) Service(service string) *ErrorBuilder
func (eb *ErrorBuilder) Operation(operation string) *ErrorBuilder
func (eb *ErrorBuilder) Message(message string) *ErrorBuilder
func (eb *ErrorBuilder) Cause(cause error) *ErrorBuilder
func (eb *ErrorBuilder) Context(key string, value any) *ErrorBuilder
func (eb *ErrorBuilder) Build() *ServiceError
```

**链式调用示例**：

```go
err := zcli.NewError(zcli.ErrServiceStart).
    Service("database").
    Operation("connect").
    Message("连接失败").
    Cause(sqlErr).
    Context("host", "localhost").
    Build()
```

### 预定义错误函数

```go
func ErrServiceAlreadyRunning(service string) *ServiceError
func ErrServiceAlreadyStopped(service string) *ServiceError
func ErrServiceStartTimeout(service string, timeout time.Duration) *ServiceError
func ErrServiceStopTimeout(service string, timeout time.Duration) *ServiceError
```

### ErrorAggregator

```go
aggregator := zcli.NewErrorAggregator()

if err := initDB(); err != nil {
    aggregator.Add(err)
}
if err := initCache(); err != nil {
    aggregator.Add(err)
}

if aggregator.HasErrors() {
    return aggregator.Error()
}
```
