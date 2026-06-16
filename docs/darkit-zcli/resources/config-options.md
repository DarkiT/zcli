# DarkiT-ZCli 配置选项

## 重要说明

Config 结构体的字段是**私有的**（小写字母开头），必须通过 getter 方法访问。getter 方法返回的是**副本**，防止外部修改内部状态。

## Config 结构

```go
type Config struct {
    basic   *Basic          // 私有字段 - 通过 Basic() 访问
    service *ServiceConfig  // 私有字段 - 通过 Service() 访问
    runtime *Runtime        // 私有字段 - 通过 Runtime() 访问
    ctx     context.Context // 私有字段 - 通过 Context() 访问
}

// Getter 方法
func (c *Config) Basic() Basic           // 返回 Basic 副本
func (c *Config) Service() ServiceConfig // 返回 ServiceConfig 深拷贝
func (c *Config) Runtime() Runtime       // 返回 Runtime 副本
func (c *Config) Context() context.Context
```

## Basic 基础配置

```go
type Basic struct {
    Name              string // 服务名称
    DisplayName       string // 显示名称
    Description       string // 服务描述
    Version           string // 版本
    Logo              string // Logo 文本
    Language          string // 使用语言
    MousetrapDisabled bool   // 禁用 Windows 双击运行提示
    NoColor           bool   // 禁用彩色输出
    SilenceErrors     bool   // 禁止 Cobra 自动打印错误
    SilenceUsage      bool   // 禁止 Cobra 自动打印用法
}
```

**访问示例**：

```go
func validateConfig(cfg *zcli.Config) error {
    basic := cfg.Basic()  // 获取副本

    if basic.Name == "" {
        return fmt.Errorf("应用名称不能为空")
    }
    if basic.Version == "" {
        return fmt.Errorf("版本号不能为空")
    }
    return nil
}
```

## ServiceConfig 服务配置

```go
type ServiceConfig struct {
    Name              string            // 服务名称
    DisplayName       string            // 显示名称
    Description       string            // 服务描述
    Version           string            // 版本号
    WorkDir           string            // 工作目录
    Username          string            // 系统服务运行用户
    Dependencies      []string          // daemon 原生字符串依赖
    StructuredDeps    []Dependency      // daemon 结构化依赖
    EnvVars           map[string]string // 环境变量
    Arguments         []string          // 系统服务启动参数
    Executable        string            // 可执行文件路径
    ChRoot            string            // chroot 根目录
    Options           ServiceOptions    // daemon 平台特定配置
    AllowSudoFallback bool              // 是否允许 sudo/su 回退
}
```

**访问示例**：

```go
func validateServiceConfig(cfg *zcli.Config) error {
    svc := cfg.Service()  // 获取深拷贝

    if svc.EnvVars["DB_HOST"] == "" {
        return fmt.Errorf("数据库主机不能为空")
    }

    for _, dep := range svc.Dependencies {
        if !isServiceRunning(dep) {
            return fmt.Errorf("依赖服务 %s 未运行", dep)
        }
    }

    return nil
}
```

## Runtime 运行时配置

```go
type Runtime struct {
    Run             RunFunc        // 启动函数
    Stop            StopFunc       // 停止函数
    BuildInfo       *VersionInfo   // 构建信息
    ShutdownInitial time.Duration  // 取消 Run(ctx) 后的主服务退出时长（默认 15s）
    ShutdownGrace   time.Duration  // stop hook / 最终清理的额外时长（默认 5s）
    StartTimeout    time.Duration  // daemon 启动超时
    StopTimeout     time.Duration  // daemon 停止超时，默认 20s
    ErrorHandlers   []ErrorHandler // 错误处理器链
}
```

**函数签名**：

```go
type RunFunc  func(ctx context.Context) error  // 运行函数签名
type StopFunc func() error                     // 停止函数签名
```

## VersionInfo 构建信息

```go
type VersionInfo struct {
    Version      string           // 版本号
    GoVersion    string           // Go 版本
    GitCommit    string           // Git 提交哈希
    GitBranch    string           // Git 分支
    GitTag       string           // Git 标签
    Platform     string           // 平台
    Architecture string           // 架构
    Compiler     string           // 编译器
    BuildTime    time.Time        // 构建时间
    Debug        atomic.Bool      // 调试模式
}
```

**配置示例**：

```go
app, _ := zcli.NewBuilder("zh").
    WithName("myapp").
    WithVersion("1.0.0").
    WithGitInfo("abc123", "main", "v1.0.0").
    WithBuildTime("2024-01-01 12:00:00").
    WithDebug(false).
    BuildWithError()
```

## Builder 配置方法总结

### 基础配置

| 方法 | 说明 | 示例 |
|------|------|------|
| `WithName(name)` | 设置服务名称 | `.WithName("myapp")` |
| `WithDisplayName(name)` | 设置显示名称 | `.WithDisplayName("我的应用")` |
| `WithDescription(desc)` | 设置应用描述 | `.WithDescription("描述")` |
| `WithVersion(version)` | 设置版本号 | `.WithVersion("1.0.0")` |
| `WithLogo(logo)` | 设置 Logo | `.WithLogo(asciiArt)` |
| `WithLanguage(lang)` | 设置语言 | `.WithLanguage("zh")` |

### 服务配置

| 方法 | 说明 | 示例 |
|------|------|------|
| `WithService(run, stop...)` | 配置服务函数 | `.WithService(runFunc, stopFunc)` |
| `WithServiceRunner(runner)` | 配置服务接口 | `.WithServiceRunner(myService)` |
| `WithWorkDir(dir)` | 设置工作目录 | `.WithWorkDir("/opt/app")` |
| `WithEnvVar(key, value)` | 添加环境变量 | `.WithEnvVar("ENV", "prod")` |
| `WithServiceUser(username)` | 设置系统服务运行用户 | `.WithServiceUser("svc-app")` |
| `WithExecutable(path)` | 设置系统服务可执行文件 | `.WithExecutable("/opt/app/bin/app")` |
| `WithArguments(args...)` | 设置系统服务参数；空列表可清空默认 `run` | `.WithArguments("run", "--profile", "prod")` |
| `WithChRoot(dir)` | 设置 chroot 根目录 | `.WithChRoot("/srv/jail")` |
| `WithDependencies(deps...)` | 设置 require 型结构化依赖 | `.WithDependencies("postgresql")` |
| `WithLegacyDependencies(deps...)` | 设置 daemon 原生字符串依赖 | `.WithLegacyDependencies("network.target")` |
| `WithStructuredDependencies(deps...)` | 批量设置结构化依赖 | `.WithStructuredDependencies(dep)` |
| `WithDependency(name, type)` | 追加单个结构化依赖 | `.WithDependency("redis", zcli.DependencyAfter)` |
| `WithServiceOption(key, value)` | 设置单个 daemon 平台选项 | `.WithServiceOption(service.OptionRestart, "always")` |
| `WithServiceOptionsMap(options)` | 批量合并 daemon 平台选项 | `.WithServiceOptionsMap(opts)` |
| `WithAllowSudoFallback(enabled)` | 设置 sudo/su 回退策略 | `.WithAllowSudoFallback(true)` |
| `WithServiceTimeouts(start, stop)` | 设置服务启动/停止超时 | `.WithServiceTimeouts(15*time.Second, 20*time.Second)` |

### 构建配置

| 方法 | 说明 | 示例 |
|------|------|------|
| `WithValidator(fn)` | 添加验证器 | `.WithValidator(validateFunc)` |
| `WithContext(ctx)` | 设置上下文 | `.WithContext(ctx)` |
| `WithRuntime(rt)` | 替换运行时配置 | `.WithRuntime(rt)` |
| `WithInitHook(hook)` | 添加命令执行前初始化钩子 | `.WithInitHook(loadConfig)` |
| `WithErrorHandler(handler)` | 添加错误处理器 | `.WithErrorHandler(zcli.NewRecoveryErrorHandler(3, time.Second))` |
| `WithMousetrapDisabled(true)` | 禁用 Windows 双击提示 | `.WithMousetrapDisabled(true)` |
| `WithDefaultConfig()` | 使用默认配置 | `.WithDefaultConfig()` |

## 常见错误

### 错误：直接访问私有字段

```go
// 编译错误：cfg.basic is unexported
if cfg.basic.Name == "" {
    // ...
}
```

### 正确：使用 getter 方法

```go
// 正确用法
if cfg.Basic().Name == "" {
    // ...
}
```

### 错误：尝试修改返回的副本

```go
// 这不会修改原始配置，因为返回的是副本
basic := cfg.Basic()
basic.Name = "new-name"  // 只修改了副本
```

### 正确：通过 Builder 配置

```go
// 配置应在 Builder 阶段完成
app := zcli.NewBuilder("zh").
    WithName("new-name").  // 正确方式
    Build()
```

## 完整配置示例

```go
app, err := zcli.NewBuilder("zh").
    // 基础配置
    WithName("enterprise-app").
    WithDisplayName("企业级应用").
    WithDescription("提供企业级功能的后台服务").
    WithVersion("2.1.0").
    WithLogo(logo).

    // 服务配置
    WithWorkDir("/opt/enterprise-app").
    WithServiceUser("enterprise").
    WithEnvVar("ENV", "production").
    WithEnvVar("LOG_LEVEL", "info").
    WithDependency("postgresql.service", zcli.DependencyAfter).
    WithDependency("redis.service", zcli.DependencyAfter).
    WithAllowSudoFallback(true).

    // 运行时配置
    WithServiceRunner(myService).
    WithServiceTimeouts(15*time.Second, 20*time.Second).

    // 验证配置
    WithValidator(func(cfg *zcli.Config) error {
        if cfg.Basic().Name == "" {
            return fmt.Errorf("应用名称不能为空")
        }
        if cfg.Service().EnvVars["ENV"] == "" {
            return fmt.Errorf("环境变量 ENV 不能为空")
        }
        return nil
    }).

    BuildWithError()

if err != nil {
    log.Fatal(err)
}
```
