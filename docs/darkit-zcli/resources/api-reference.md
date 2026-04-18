# DarkiT-ZCli API 参考

## Builder API

### 创建 Builder

```go
func NewBuilder(language string) *Builder
```

创建新的 Builder 实例。

**参数**：
- `language`: 语言代码（"zh" 或 "en"）

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
设置服务依赖。

---

```go
func (b *Builder) WithErrorHandler(handler ErrorHandler) *Builder
```
添加错误处理器到处理链。

**参数**：
- `handler`: 实现 `ErrorHandler` 接口的处理器

**示例**：

```go
builder.WithErrorHandler(zcli.LoggingErrorHandler{})
builder.WithErrorHandler(zcli.RecoveryErrorHandler{})
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
func (c *Cli) AddCommand(cmds ...*cobra.Command)
```

添加自定义命令。

**示例**：

```go
cmd := &cobra.Command{
    Use:   "custom",
    Short: "自定义命令",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("执行自定义命令")
    },
}
app.AddCommand(cmd)
```

### 标志管理

```go
func (c *Cli) Flags() *pflag.FlagSet
```
获取标志集合。

---

```go
func (c *Cli) PersistentFlags() *pflag.FlagSet
```
获取持久化标志。

---

```go
func (c *Cli) ExportFlagsForViper(exclude ...string) []*pflag.FlagSet
```

导出标志给 Viper，自动过滤 23 个系统标志。

**默认排除的系统标志**：
- 帮助系统：`help`, `h`
- 版本系统：`version`, `v`
- 补全系统：`completion`, `*-completion`, `gen-completion`
- 调试系统：`debug`, `verbose`, `quiet`
- 更多...

**示例**：

```go
flags := app.ExportFlagsForViper()
// 绑定到 Viper
for _, fs := range flags {
    viper.BindPFlags(fs)
}
```

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
- 返回 `nil` 表示正常退出
- 返回 `error` 表示异常退出

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
func (eb *ErrorBuilder) WithContext(key string, value any) *ErrorBuilder
func (eb *ErrorBuilder) Build() *ServiceError
```

**链式调用示例**：

```go
err := zcli.NewError(zcli.ErrServiceStart).
    Service("database").
    Operation("connect").
    Message("连接失败").
    Cause(sqlErr).
    WithContext("host", "localhost").
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
