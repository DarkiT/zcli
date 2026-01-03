# ZCli - 企业级 CLI 框架和系统服务管理扩展包

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

ZCli 是一个基于 [Cobra](https://github.com/spf13/cobra)的现代化企业级命令行应用框架，提供了优雅的 API 设计、完整的系统服务管理、强大的错误处理系统、现代化的多语言支持和生产级的稳定性保障。

## 🌟 最新改进 (v0.2.0)

### ✨ 优雅的 API 设计

- **服务接口抽象**: 统一的 `ServiceRunner` 接口，支持依赖注入和模块化设计
- **链式 Builder 模式**: 流畅的 API 调用，支持 `WithServiceRunner`、`WithService`、`WithValidator` 等方法
- **错误优先设计**: 新增 `BuildWithError` 方法，提供更好的错误处理体验
- **便利性 API**: `QuickService`、`QuickCLI` 等快速创建函数，简化常见使用场景

### 🛡️ 强大的错误处理系统

- **结构化错误**: 完整的错误代码体系，支持错误分类和追踪
- **错误聚合器**: 批量处理多个错误，提供统一的错误报告
- **预定义错误**: 常用错误的快速创建函数，如 `ErrServiceAlreadyRunning`
- **错误链追踪**: 支持错误原因链追踪和上下文信息附加

### 🔒 并发安全保障

- **原子操作**: 使用 `atomic.Bool` 等原子类型确保状态一致性
- **线程安全设计**: 经过严格的并发安全测试，支持多线程环境
- **状态管理**: 完善的服务生命周期状态管理，支持状态监听器
- **优雅关闭**: 分级超时机制，确保服务可靠停止

## ✨ 核心特性

### 🎨 优雅的命令行界面

- **智能彩色输出**: 自动检测终端能力，支持彩色和无彩色模式
- **自定义 Logo**: 支持 ASCII 艺术 Logo 展示
- **分组命令显示**: 自动区分普通命令和系统服务命令
- **优化的帮助系统**: 美观的帮助信息展示和命令补全

### 🔧 企业级系统服务集成

- **双运行模式**: 支持前台开发调试模式和后台服务模式
- **完整服务管理**: install/start/stop/restart/status/uninstall
- **高级服务配置**: 用户权限、工作目录、依赖关系、环境变量、chroot 等
- **跨平台支持**: Windows、Linux、macOS 全平台兼容
- **并发安全**: 经过严格的并发安全测试，生产环境可靠

### 🌍 现代化多语言系统

- **层次化语言包**: Service/UI/Error/Format 四域架构
- **智能回退机制**: 缺失翻译自动回退到备用语言
- **便利性 API**: ServiceLocalizer 简化常用本地化操作
- **动态扩展**: 运行时注册自定义语言包

### 🛡️ 生产级稳定性

- **优雅信号处理**: `RunWait` 默认监听 SIGINT/SIGTERM/SIGQUIT，未设置时自动注入；捕获后调用 `sm.Stop()`
- **分级超时机制**: 3+2 秒分级超时确保服务可靠停止
- **超时对齐**: 通过 `WithServiceTimeouts` 写入系统服务启动/停止超时
- **权限检查**: 智能的文件和目录权限验证
- **错误恢复**: 全面的错误处理和日志记录

## 📦 安装

```bash
go get github.com/darkit/zcli
```

**最低要求**: Go 1.23+ (使用了新特性)

## 🚀 快速开始

### 现代 API 示例（推荐）

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "time"

    "github.com/darkit/zcli"
)

func main() {
    // 方式1: 使用便利性API快速创建（最简单）
    app := zcli.QuickService("myapp", "我的企业级应用", runService)

    // 方式2: 使用完整的Builder API（推荐用于复杂场景）
    app, err := zcli.NewBuilder("zh").
        WithName("myapp").
        WithDisplayName("我的企业级应用").
        WithDescription("这是一个演示企业级功能的应用").
        WithVersion("1.0.0").
        WithService(runService, stopService).  // 配置服务入口
        WithValidator(func(cfg *zcli.Config) error {  // 配置验证
            if cfg.Basic().Name == "" {
                return fmt.Errorf("应用名称不能为空")
            }
            return nil
        }).
        BuildWithError()  // 错误优先的构建方式

    if err != nil {
        slog.Error("应用构建失败", "error", err)
        os.Exit(1)
    }

    // 添加全局选项
    app.PersistentFlags().BoolP("debug", "d", false, "启用调试模式")
    app.PersistentFlags().StringP("config", "c", "", "配置文件路径")

    // 执行应用
    if err := app.Execute(); err != nil {
        slog.Error("应用执行失败", "error", err)
        os.Exit(1)
    }
}

// 服务主函数 - 使用context优雅处理生命周期
func runService(ctx context.Context) error {
    slog.Info("服务已启动，等待停止信号...")

    // 创建定时器
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    // 服务主循环 - 使用context.Done()优雅处理停止
    for {
        select {
        case <-ctx.Done():
            slog.Info("收到停止信号，准备退出服务循环")
            return nil
        case <-ticker.C:
            slog.Info("服务正在运行...")

            // 模拟业务逻辑可能出现的错误
            if err := processBusinessLogic(); err != nil {
                return fmt.Errorf("业务逻辑处理失败: %w", err)
            }
        }
    }
}

// 服务停止函数 - 执行清理工作
func stopService() error {
    slog.Info("执行服务清理工作...")

    // 模拟清理工作
    time.Sleep(100 * time.Millisecond)

    slog.Info("服务清理完成，已安全停止")
    return nil
}

func processBusinessLogic() error {
    // 模拟业务逻辑
    return nil
}
```

### 运行方式

```bash
# 前台运行模式 - 开发调试使用
./myapp run              # 在前台运行，可以看到日志，Ctrl+C停止

# 服务管理模式 - 生产环境使用
./myapp install          # 安装为系统服务
./myapp start            # 启动服务
./myapp status           # 查看服务状态
./myapp stop             # 停止服务
./myapp restart          # 重启服务
./myapp uninstall        # 卸载服务

# 其他功能
./myapp help             # 查看帮助
./myapp version          # 查看版本信息
```

## 🔧 高级配置

### 服务接口抽象和依赖注入

```go
// 定义你的服务
type MyService struct {
    db     *Database
    cache  *Cache
    config *Config
}

func (s *MyService) Run(ctx context.Context) error {
    slog.Info("企业级服务已启动")

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            slog.Info("收到停止信号，优雅关闭")
            return nil
        case <-ticker.C:
            if err := s.processBusinessLogic(); err != nil {
                return fmt.Errorf("业务处理失败: %w", err)
            }
        }
    }
}

func (s *MyService) Stop() error {
    slog.Info("正在关闭服务...")

    // 关闭数据库连接
    if err := s.db.Close(); err != nil {
        return fmt.Errorf("数据库关闭失败: %w", err)
    }

    // 关闭缓存连接
    if err := s.cache.Close(); err != nil {
        return fmt.Errorf("缓存关闭失败: %w", err)
    }

    return nil
}

func (s *MyService) Name() string {
    return "enterprise-service"
}

func (s *MyService) processBusinessLogic() error {
    // 实现业务逻辑
    return nil
}

func main() {
    // 初始化依赖
    db := &Database{} // 初始化数据库
    cache := &Cache{} // 初始化缓存
    config := &Config{} // 加载配置

    // 创建服务实例
    service := &MyService{
        db:     db,
        cache:  cache,
        config: config,
    }

    // 使用服务接口抽象
    app, err := zcli.NewBuilder("zh").
        WithName("enterprise-app").
        WithDisplayName("企业级服务应用").
        WithDescription("提供企业级功能的后台服务").
        WithVersion("2.1.0").

        // 高级服务配置
        WithWorkDir("/opt/enterprise-app").
        WithEnvVar("ENV", "production").
        WithEnvVar("LOG_LEVEL", "info").
        WithEnvVar("DB_HOST", "localhost:5432").
        WithDependencies("postgresql", "redis").

        // 使用服务接口抽象
        WithServiceRunner(service).

        // 添加配置验证
        WithValidator(func(cfg *zcli.Config) error {
            if cfg.Service().EnvVars["DB_HOST"] == "" {
                return fmt.Errorf("数据库主机不能为空")
            }
            return nil
        }).

        // 错误优先构建
        BuildWithError()

    if err != nil {
        slog.Error("应用构建失败", "error", err)
        os.Exit(1)
    }

    if err := app.Execute(); err != nil {
        slog.Error("应用执行失败", "error", err)
        os.Exit(1)
    }
}
```

### 错误处理示例

```go
func main() {
    app, err := zcli.NewBuilder("zh").
        WithName("error-demo").
        WithService(runWithErrorHandling, stopService).
        BuildWithError()

    if err != nil {
        // 使用结构化错误处理
        if serviceErr, ok := zcli.GetServiceError(err); ok {
            slog.Error("服务错误",
                "code", serviceErr.Code,
                "service", serviceErr.Service,
                "operation", serviceErr.Operation,
                "message", serviceErr.Message)
        } else {
            slog.Error("构建失败", "error", err)
        }
        os.Exit(1)
    }

    app.Execute()
}

func runWithErrorHandling(ctx context.Context) error {
    // 使用错误聚合器收集多个错误
    aggregator := zcli.NewErrorAggregator()

    // 初始化各个组件
    if err := initDatabase(); err != nil {
        aggregator.Add(zcli.NewError(zcli.ErrServiceCreate).
            Service("database").
            Operation("init").
            Message("数据库初始化失败").
            Cause(err).
            Build())
    }

    if err := initCache(); err != nil {
        aggregator.Add(zcli.NewError(zcli.ErrServiceCreate).
            Service("cache").
            Operation("init").
            Message("缓存初始化失败").
            Cause(err).
            Build())
    }

    // 如果有错误，返回聚合的错误
    if aggregator.HasErrors() {
        return aggregator.Error()
    }

    // 服务主循环
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            // 处理业务逻辑
            time.Sleep(100 * time.Millisecond)
        }
    }
}

func initDatabase() error {
    // 模拟数据库初始化
    return nil
}

func initCache() error {
    // 模拟缓存初始化
    return nil
}
```

### 自定义命令

```go
// 创建配置管理主命令
configCmd := &zcli.Command{
    Use:   "config",
    Short: "配置管理",
    Long:  "管理应用程序的配置文件和设置",
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("配置管理功能")
    },
}

// 添加子命令
showCmd := &zcli.Command{
    Use:   "show",
    Short: "显示当前配置",
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("当前配置:")
        fmt.Println("- 服务名称: myapp")
        fmt.Println("- 工作目录: /opt/myapp")
        fmt.Println("- 运行模式: 生产模式")
    },
}

editCmd := &zcli.Command{
    Use:   "edit",
    Short: "编辑配置",
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("打开配置编辑器...")
    },
}

// 组织命令层次
configCmd.AddCommand(showCmd, editCmd)
app.AddCommand(configCmd)
```

## 🌍 多语言支持

### 内置语言切换

```go
// 中文界面
app := zcli.NewBuilder("zh").Build()

// 英文界面
app := zcli.NewBuilder("en").Build()

// 也可以用WithLanguage方法
app := zcli.NewBuilder().WithLanguage("zh").Build()
```

## 📊 命令行界面效果

### 帮助界面示例

```bash
$ myapp --help

███████╗ ██████╗██╗     ██╗
╚══███╔╝██╔════╝██║     ██║
  ███╔╝ ██║     ██║     ██║
 ███╔╝  ██║     ██║     ██║
███████╗╚██████╗███████╗██║
╚══════╝ ╚═════╝╚══════╝╚═╝ v1.0.0

我的企业级应用

用法:
   myapp [参数]
   myapp [command] [参数]

选项:
   -h, --help      获取帮助
   -v, --version   显示版本信息
   -d, --debug     启用调试模式
   -c, --config    配置文件路径

可用命令:
   help        获取帮助
   config      配置管理

系统命令:
   run         运行服务          # 前台运行模式
   start       启动服务          # 后台服务模式
   stop        停止服务
   status      查看状态
   restart     重启服务
   install     安装服务
   uninstall   卸载服务

使用 'myapp [command] --help' 获取命令的更多信息
```

## ⚙️ 高级特性

### 1. 前台 vs 服务运行模式

ZCli 智能区分两种运行模式：

**前台运行模式** (`./myapp run`):

- 适用于开发和调试
- 实时显示日志输出
- 支持 Ctrl+C 优雅退出
- Interactive 模式检测

**服务运行模式** (通过系统服务):

- 适用于生产环境部署
- 后台运行，由系统服务管理器控制
- 支持开机自启、自动重启等
- 完整的生命周期管理

**并发管理**

- sManager（默认）：面向 install/start/stop/status 等系统服务命令集，依赖 `github.com/darkit/daemon`，适合需要系统级安装/守护的场景。

### 2. Context 生命周期管理（自动信号处理）

ZCli 框架**自动管理** Context 生命周期，提供优雅的信号处理和资源清理机制。

#### Context 的自动创建

框架内部自动创建带信号监听的 Context，无需手动处理：

```go
// 框架内部自动执行（无需用户编写）
ctx, cancel := signal.NotifyContext(
    context.Background(),
    syscall.SIGINT,  // Ctrl+C
    syscall.SIGTERM, // 终止信号
    syscall.SIGQUIT, // 退出信号
)
```

#### 生命周期流程

```
用户启动应用
    ↓
框架创建带信号监听的 Context
    ↓
自动传递给 runService(ctx)
    ↓
用户按 Ctrl+C
    ↓
ctx.Done()被触发
    ↓
服务优雅退出
    ↓
执行 stopService() 清理资源
```

#### 为什么使用 `func(ctx context.Context) error`？

**✅ 优势**：

1. **自动信号处理** - 无需手动处理 SIGINT/SIGTERM

```go
func runService(ctx context.Context) error {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():  // Ctrl+C 时自动触发
            slog.Info("收到停止信号，优雅退出")
            return nil
        case <-ticker.C:
            // 业务逻辑
        }
    }
}
```

2. **优雅关闭链** - Context 可传递给所有依赖组件

```go
func runService(ctx context.Context) error {
    // Context 传递给依赖组件
    db, _ := database.Connect(ctx)
    cache, _ := redis.Connect(ctx)

    // 当收到停止信号时，所有组件都能感知到
    // 数据库、缓存等会自动关闭连接
}
```

3. **超时控制** - 基于父 Context 创建子 Context

```go
func runService(ctx context.Context) error {
    // 为某个操作设置超时
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    return doSomethingWithTimeout(ctx)
}
```

**❌ 如果没有 Context（不推荐）**：

```go
// 需要手动处理信号
func runService() error {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    for {
        select {
        case <-sigChan:  // 手动处理
            return nil
        default:
            // 业务逻辑
        }
    }
}
```

#### 完整生命周期示例

```go
func runService(ctx context.Context) error {
    slog.Info("服务启动")

    // 1. 初始化资源（传递 ctx）
    db, err := database.Connect(ctx)
    if err != nil {
        return err
    }
    defer db.Close()

    // 2. 创建子 Context（可选）
    workCtx, cancel := context.WithCancel(ctx)
    defer cancel()

    // 3. 启动后台任务
    go backgroundTask(workCtx)

    // 4. 主循环
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            // 框架收到 Ctrl+C 时自动触发
            slog.Info("收到停止信号")

            // 取消所有子任务
            cancel()

            // 等待子任务完成
            time.Sleep(100 * time.Millisecond)

            return nil

        case <-ticker.C:
            // 正常业务逻辑
            if err := doWork(ctx); err != nil {
                return err
            }
        }
    }
}

func stopService() error {
    slog.Info("执行最终清理")
    // 这里执行无法通过 context 传递的清理工作
    // 例如：关闭文件句柄、保存状态、刷新缓冲区等
    return nil
}
```

#### 分级超时保护机制

框架提供 **3+2 秒分级超时**保护：

```
收到停止信号（Ctrl+C）
    ↓
等待 3 秒（优雅退出期）
    ↓
[超时] 强制调用 stopService()
    ↓
再等待 2 秒（清理期）
    ↓
[超时] 强制终止进程
```

这确保即使服务卡死，也能在 5 秒内强制退出。

#### 最佳实践

1. **始终在 select 中监听 `ctx.Done()`**

```go
for {
    select {
    case <-ctx.Done():
        return ctx.Err()  // 返回取消原因
    case data := <-dataChan:
        processData(data)
    }
}
```

2. **传递 Context 给所有阻塞操作**

```go
// 数据库查询
rows, err := db.QueryContext(ctx, sql)

// HTTP 请求
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

// gRPC 调用
conn, err := grpc.DialContext(ctx, addr)
```

3. **避免存储 Context（反模式）**

```go
// ❌ 错误：不要将 context 存储在结构体中
type Service struct {
    ctx context.Context  // 违反 Go 官方建议
}

// ✅ 正确：通过参数传递
func (s *Service) Run(ctx context.Context) error {
    // 使用参数传递的 ctx
}
```

4. **使用 Context 传递请求范围的值**

```go
// 传递请求 ID
ctx = context.WithValue(ctx, "requestID", uuid.New())

// 在下游获取
requestID := ctx.Value("requestID").(uuid.UUID)
```

### 3. 智能标志导出功能

ZCli 提供了强大的命令行标志导出功能，专门用于与外部包（如 Viper 配置管理）的集成：

#### 基本用法

```go
// 创建应用并添加各种标志
app := zcli.NewBuilder("zh").
    WithName("myapp").
    Build()

// 添加业务标志
app.PersistentFlags().StringP("config", "c", "", "配置文件路径")
app.Flags().IntP("port", "p", 8080, "服务端口")
app.Flags().BoolP("debug", "d", false, "调试模式")

// 导出给外部包使用（自动排除系统标志）
flags := app.ExportFlagsForViper()
WithBindPFlags(flags...)(viperConfig)
```

#### 智能过滤机制

框架自动排除 **23 个系统标志**，确保只有业务标志传递给外部包：

```go
// 自动排除的系统标志包括：
// - 帮助系统：help, h
// - 版本系统：version, v
// - 补全系统：completion, completion-*, gen-completion
// - 内部调试：__complete, __completeNoDesc, no-descriptions
// - 配置系统：config-help, print-config, validate-config
// ... 总计23个系统标志

// 检查标志类型
isSystem := app.IsSystemFlag("help")     // true - 系统标志
isSystem := app.IsSystemFlag("config")   // false - 业务标志

// 获取所有系统标志列表（调试用）
systemFlags := app.GetSystemFlags()  // 返回23个系统标志的完整列表
```

#### API 参考

| 方法                               | 用途                             | 返回值       |
| ---------------------------------- | -------------------------------- | ------------ |
| `ExportFlagsForViper(exclude...)`  | 导出适用于 Viper 的标志          | `[]*FlagSet` |
| `GetAllFlagSets()`                 | 获取所有标志集合                 | `[]*FlagSet` |
| `GetBindableFlagSets(exclude...)`  | 获取可绑定的标志（排除系统标志） | `[]*FlagSet` |
| `GetFilteredFlags(exclude...)`     | 获取单个过滤后的标志集合         | `*FlagSet`   |
| `GetFlagNames(includeInherited)`   | 获取标志名称列表                 | `[]string`   |
| `GetFilteredFlagNames(exclude...)` | 获取过滤后的标志名称             | `[]string`   |
| `IsSystemFlag(name)`               | 检查是否为系统标志               | `bool`       |
| `GetSystemFlags()`                 | 获取所有系统标志列表             | `[]string`   |

## 📚 API 参考

### Builder API

#### 核心构建方法

| 方法               | 说明                        | 示例                                   |
| ------------------ | --------------------------- | -------------------------------------- |
| `NewBuilder(lang)` | 创建新的 Builder 实例       | `zcli.NewBuilder("zh")`                |
| `Build()`          | 构建 CLI 实例（可能 panic） | `builder.Build()`                      |
| `BuildWithError()` | 构建 CLI 实例，返回错误     | `app, err := builder.BuildWithError()` |

#### 基础配置方法

| 方法                    | 说明         | 示例                           |
| ----------------------- | ------------ | ------------------------------ |
| `WithName(name)`        | 设置应用名称 | `.WithName("myapp")`           |
| `WithDisplayName(name)` | 设置显示名称 | `.WithDisplayName("我的应用")` |
| `WithDescription(desc)` | 设置应用描述 | `.WithDescription("应用描述")` |
| `WithVersion(version)`  | 设置版本号   | `.WithVersion("1.0.0")`        |
| `WithLanguage(lang)`    | 设置语言     | `.WithLanguage("zh")`          |

#### 服务配置方法

| 方法                               | 说明                  | 示例                                                   |
| ---------------------------------- | --------------------- | ------------------------------------------------------ |
| `WithService(run, stop...)`        | 配置服务函数          | `.WithService(runFunc, stopFunc)`                      |
| `WithServiceRunner(service)`       | 配置服务接口实现      | `.WithServiceRunner(myService)`                        |
| `WithServiceTimeouts(start, stop)` | 配置服务启动/停止超时 | `.WithServiceTimeouts(30*time.Second, 20*time.Second)` |
| `WithValidator(validator)`         | 添加配置验证器        | `.WithValidator(validatorFunc)`                        |
| `WithWorkDir(dir)`                 | 设置工作目录          | `.WithWorkDir("/opt/app")`                             |
| `WithEnvVar(key, value)`           | 添加环境变量          | `.WithEnvVar("ENV", "prod")`                           |
| `WithDependencies(deps...)`        | 设置服务依赖          | `.WithDependencies("postgresql")`                      |

示例：将服务启动/停止超时写入系统服务配置（daemon Timeout）

```go
app, err := zcli.NewBuilder("zh").
    WithName("myapp").
    WithService(runService, stopService).
    WithServiceTimeouts(30*time.Second, 20*time.Second).
    BuildWithError()
```

#### 便利性 API

| 方法                                                 | 说明                     | 示例                                        |
| ---------------------------------------------------- | ------------------------ | ------------------------------------------- |
| `QuickService(name, displayName, run)`               | 快速创建服务应用         | `zcli.QuickService("app", "描述", runFunc)` |
| `QuickServiceWithStop(name, displayName, run, stop)` | 快速创建带停止函数的服务 | `zcli.QuickServiceWithStop(...)`            |
| `QuickCLI(name, display, desc)`                      | 快速创建 CLI 工具        | `zcli.QuickCLI("tool", "工具", "描述")`     |

### 服务接口

#### ServiceRunner 接口

```go
type ServiceRunner interface {
    Run(ctx context.Context) error  // 运行服务
    Stop() error                    // 停止服务
    Name() string                   // 服务名称
}
```

#### 实现示例

```go
type MyService struct{}

func (s *MyService) Run(ctx context.Context) error {
    // 服务运行逻辑
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            // 处理业务逻辑
        }
    }
}

func (s *MyService) Stop() error {
    // 停止逻辑
    return nil
}

func (s *MyService) Name() string {
    return "my-service"
}
```

### 错误处理 API

#### 错误创建和处理

| 函数                   | 说明           | 示例                                          |
| ---------------------- | -------------- | --------------------------------------------- |
| `NewError(code)`       | 创建错误构建器 | `zcli.NewError(zcli.ErrServiceStart)`         |
| `NewErrorAggregator()` | 创建错误聚合器 | `aggregator := zcli.NewErrorAggregator()`     |
| `GetServiceError(err)` | 获取服务错误   | `serviceErr, ok := zcli.GetServiceError(err)` |

#### 预定义错误函数

| 函数                                    | 说明           | 示例                                                   |
| --------------------------------------- | -------------- | ------------------------------------------------------ |
| `ErrServiceAlreadyRunning(name)`        | 服务已运行错误 | `zcli.ErrServiceAlreadyRunning("myapp")`               |
| `ErrServiceAlreadyStopped(name)`        | 服务已停止错误 | `zcli.ErrServiceAlreadyStopped("myapp")`               |
| `ErrServiceStartTimeout(name, timeout)` | 启动超时错误   | `zcli.ErrServiceStartTimeout("myapp", 30*time.Second)` |

#### 错误构建器 API

```go
err := zcli.NewError(zcli.ErrServiceStart).
    Service("myapp").                    // 设置服务名
    Operation("start").                  // 设置操作
    Message("启动失败").                 // 设置消息
    Cause(originalError).                // 设置原因
    Context("key", "value").             // 添加上下文
    Build()                              // 构建错误
```

## 🎯 设计原则和最佳实践

### 1. 错误优先设计

```go
// ✅ 推荐：使用BuildWithError
app, err := zcli.NewBuilder("zh").
    WithName("myapp").
    BuildWithError()
if err != nil {
    // 处理错误
}

// ❌ 不推荐：可能panic的Build
app := zcli.NewBuilder("zh").
    WithName("myapp").
    Build() // 可能panic
```

### 2. 依赖注入和接口抽象

```go
// ✅ 推荐：使用接口抽象
type DatabaseService interface {
    Connect() error
    Close() error
}

type MyService struct {
    db DatabaseService  // 依赖注入
}

app, _ := zcli.NewBuilder("zh").
    WithServiceRunner(myService).  // 接口抽象
    BuildWithError()
```

### 3. 配置验证

```go
// ✅ 推荐:添加配置验证
app, err := zcli.NewBuilder("zh").
    WithName("myapp").
    WithValidator(func(cfg *zcli.Config) error {
        if cfg.Basic().Name == "" {
            return fmt.Errorf("应用名称不能为空")
        }
        return nil
    }).
    BuildWithError()
```

### 4. 结构化错误处理

```go
// ✅ 推荐：使用结构化错误
aggregator := zcli.NewErrorAggregator()

if err := initDB(); err != nil {
    aggregator.Add(zcli.NewError(zcli.ErrServiceCreate).
        Service("database").
        Operation("init").
        Cause(err).
        Build())
}

if aggregator.HasErrors() {
    return aggregator.Error()
}
```

## ⚠️ 注意事项

### 系统权限

- 服务管理功能需要适当的系统权限
- Linux/macOS: 通常需要 sudo 权限安装系统服务
- Windows: 需要管理员权限

### 平台兼容性

- 某些终端可能不支持彩色输出，框架会自动检测并降级
- Windows 服务和 Unix 守护进程的配置选项有所不同
- 使用条件编译处理平台特定功能

### 并发安全

- 服务函数应该是线程安全的
- 避免在服务函数中使用全局变量
- 使用 channel 或 context 进行 goroutine 间通信

## 📚 依赖项

- **Go 1.23+**: 充分利用 Go 1.23 新特性
- **github.com/spf13/cobra**: 命令行框架基础
- **github.com/fatih/color**: 彩色输出支持
- **github.com/darkit/daemon**: 跨平台系统服务管理

## 🆕 版本历史

### v0.2.0 (Latest)

- ✨ **新增服务接口抽象**: `ServiceRunner` 接口统一服务管理
- ✨ **增强 Builder 模式**: 新增 `WithServiceRunner`、`WithValidator` 方法
- ✨ **错误优先设计**: 新增 `BuildWithError` 方法，提供更好的错误处理
- ✨ **强大的错误处理系统**: 结构化错误、错误聚合器、预定义错误函数
- ✨ **并发安全保障**: 原子操作、线程安全设计、状态管理
- ✨ **便利性 API**: `QuickService`、`QuickCLI` 等快速创建函数
- 🔧 **简化 API 设计**: 移除过度封装的模板代码，专注核心功能
- 🐛 **修复并发问题**: 解决服务管理中的 race condition 和死锁问题
- 📚 **完善文档**: 全面更新 API 文档和使用示例

### v0.1.x (Legacy)

- 基础 CLI 框架功能
- 系统服务管理
- 多语言支持
- 传统 API 设计

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发环境设置

```bash
git clone https://github.com/darkit/zcli.git
cd zcli
go mod tidy
go test -v  # 运行测试
```

### 测试

```bash
# 运行所有测试
go test -v

# 运行并发安全测试
go test -v -run TestServiceConcurrent

# 运行集成测试
go test -v -run TestFinalIntegration
```

## 📄 许可证

[MIT License](LICENSE)

---

**ZCli**: 让您的 CLI 应用更专业、更稳定、更易用。
