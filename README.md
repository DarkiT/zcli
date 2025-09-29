# ZCli - 企业级CLI框架和系统服务管理扩展包

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

ZCli 是一个基于 [Cobra](https://github.com/spf13/cobra) 的现代化企业级命令行应用框架，提供了优雅的API设计、完整的系统服务管理、强大的错误处理系统、现代化的多语言支持和生产级的稳定性保障。

## 🌟 最新改进 (v1.0+)

### ✨ 优雅的API设计
- **服务接口抽象**: 统一的 `ServiceRunner` 接口，支持依赖注入和模块化设计
- **链式Builder模式**: 流畅的API调用，支持 `WithServiceRunner`、`WithSimpleService`、`WithValidator` 等方法
- **错误优先设计**: 新增 `BuildWithError` 方法，提供更好的错误处理体验
- **便利性API**: `QuickService`、`QuickCLI` 等快速创建函数，简化常见使用场景

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
- **自定义Logo**: 支持ASCII艺术Logo展示
- **分组命令显示**: 自动区分普通命令和系统服务命令
- **优化的帮助系统**: 美观的帮助信息展示和命令补全

### 🔧 企业级系统服务集成
- **双运行模式**: 支持前台开发调试模式和后台服务模式
- **完整服务管理**: install/start/stop/restart/status/uninstall
- **高级服务配置**: 用户权限、工作目录、依赖关系、环境变量、chroot等
- **跨平台支持**: Windows、Linux、macOS全平台兼容
- **并发安全**: 经过严格的并发安全测试，生产环境可靠

### 🌍 现代化多语言系统
- **层次化语言包**: Service/UI/Error/Format四域架构
- **智能回退机制**: 缺失翻译自动回退到备用语言
- **便利性API**: ServiceLocalizer简化常用本地化操作
- **动态扩展**: 运行时注册自定义语言包

### 🛡️ 生产级稳定性
- **优雅信号处理**: 支持Ctrl+C等信号的优雅退出
- **分级超时机制**: 3+2秒分级超时确保服务可靠停止
- **权限检查**: 智能的文件和目录权限验证
- **错误恢复**: 全面的错误处理和日志记录

## 📦 安装

```bash
go get github.com/darkit/zcli
```

**最低要求**: Go 1.19+ (使用了 atomic.Bool 等新特性)

## 🚀 快速开始

### 现代API示例（推荐）

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/darkit/zcli"
)

func main() {
    // 方式1: 使用便利性API快速创建
    app := zcli.QuickService("myapp", "我的企业级应用", runService)
    
    // 方式2: 使用完整的Builder API（推荐用于复杂场景）
    app, err := zcli.NewBuilder("zh").
        WithName("myapp").
        WithDisplayName("我的企业级应用").
        WithDescription("这是一个演示企业级功能的应用").
        WithVersion("1.0.0").
        WithSimpleService("myapp", runService, stopService).  // 现代服务API
        WithValidator(func(cfg *zcli.Config) error {           // 配置验证
            if cfg.Basic.Name == "" {
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
            if cfg.Service.EnvVars["DB_HOST"] == "" {
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
        WithSimpleService("error-demo", runWithErrorHandling, nil).
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

### 服务配置的高级选项

通过 `WithService` 方法可以访问更多高级配置：

```go
app := zcli.NewBuilder("zh").
    WithName("secure-service").
    WithService(func(config *zcli.Config) {
        // 安全配置
        config.Service.Username = "serviceuser"           // 运行用户
        config.Service.ChRoot = "/var/chroot/myapp"       // chroot环境
        config.Service.Executable = "/usr/bin/myapp"     // 自定义可执行文件路径
        
        // 平台特定选项 (Linux systemd)
        config.Service.Options = map[string]interface{}{
            "Restart": "always",
            "RestartSec": 5,
            "LimitNOFILE": 65536,
        }
        
        // 更多环境变量
        config.Service.EnvVars["GOMAXPROCS"] = "4"
        config.Service.EnvVars["TZ"] = "Asia/Shanghai"
    }).
    WithSystemService(runService, stopService).
    Build()
```

## 🎯 自定义命令

### 添加普通命令

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

### 添加带标志的命令

```go
importCmd := &zcli.Command{
    Use:   "import",
    Short: "导入数据",
    Run: func(cmd *zcli.Command, args []string) {
        file, _ := cmd.Flags().GetString("file")
        format, _ := cmd.Flags().GetString("format")
        dryRun, _ := cmd.Flags().GetBool("dry-run")
        
        fmt.Printf("导入文件: %s, 格式: %s, 预演模式: %v\n", file, format, dryRun)
    },
}

// 添加标志
importCmd.Flags().StringP("file", "f", "", "导入文件路径")
importCmd.Flags().StringP("format", "t", "json", "文件格式 (json/csv/xml)")
importCmd.Flags().Bool("dry-run", false, "预演模式，不实际执行")

// 标记必需参数
importCmd.MarkFlagRequired("file")

app.AddCommand(importCmd)
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

### 自定义语言包

```go
// 创建自定义语言包
japaneseLanguage := &zcli.Language{
    Code: "ja",
    Name: "日本語",
    Service: zcli.ServiceDomain{
        Operations: zcli.ServiceOperations{
            Install:   "サービスをインストール",
            Start:     "サービスを開始",
            Stop:      "サービスを停止",
            Status:    "ステータス確認",
            // ... 其他必需字段
        },
        Status: zcli.ServiceStatus{
            Running:  "実行中",
            Stopped:  "停止済み",
            Success:  "成功",
            // ... 其他必需字段
        },
        Messages: zcli.ServiceMessages{
            Installing: "インストール中...",
            Starting:   "開始中...",
            // ... 其他必需字段
        },
    },
    // UI, Error, Format域也需要完整定义
    UI: zcli.UIDomain{ /* ... */ },
    Error: zcli.ErrorDomain{ /* ... */ },
    Format: zcli.FormatDomain{ /* ... */ },
}

// 注册自定义语言包
manager := zcli.NewLanguageManager("ja")
err := manager.RegisterLanguage(japaneseLanguage)
if err != nil {
    log.Fatal("语言包注册失败:", err)
}

// 使用自定义语言包
app := zcli.NewBuilder("ja").Build()
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
   import      导入数据

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

### 版本信息示例
```bash
$ myapp version

应用信息:
  名称: 我的企业级应用
  版本: v1.0.0
  描述: 这是一个演示企业级功能的应用

构建信息:
  Git提交: abc123
  Git分支: main
  Git标签: v1.0.0
  构建时间: 2024-07-02 10:30:00
  Go版本: go1.22.0
  运行模式: 生产模式
```

### 服务状态示例
```bash
$ myapp status
2024/07/02 10:15:32 INFO 服务 myapp: 正在运行

$ myapp install
2024/07/02 10:15:35 INFO 服务 myapp: 执行成功

$ myapp start  
2024/07/02 10:15:38 INFO 服务 myapp: 执行成功
```

## ⚙️ 高级特性

### 1. 前台vs服务运行模式

ZCli智能区分两种运行模式：

**前台运行模式** (`./myapp run`):
- 适用于开发和调试
- 实时显示日志输出
- 支持Ctrl+C优雅退出
- Interactive模式检测

**服务运行模式** (通过系统服务):
- 适用于生产环境部署
- 后台运行，由系统服务管理器控制
- 支持开机自启、自动重启等
- 完整的生命周期管理

### 2. 智能标志导出功能

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

框架自动排除 **23个系统标志**，确保只有业务标志传递给外部包：

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

#### 高级过滤选项

```go
// 自定义排除特定业务标志
flags := app.GetBindableFlagSets("debug", "internal-config")
WithBindPFlags(flags...)(config)

// 获取单个过滤后的标志集合
filtered := app.GetFilteredFlags("exclude1", "exclude2")

// 获取标志名称列表（便于调试）
allNames := app.GetFlagNames(true)  // 包含继承的标志
filteredNames := app.GetFilteredFlagNames("debug")
```

#### 典型集成场景

**与 Viper 配置管理集成：**
```go
// 1. 导出标志
flags := app.ExportFlagsForViper("sensitive-flag")

// 2. 绑定到 Viper
for _, flagSet := range flags {
    flagSet.VisitAll(func(flag *pflag.Flag) {
        viper.BindPFlag(flag.Name, flag)
    })
}

// 3. 使用配置
config := viper.GetString("config")
port := viper.GetInt("port")
```

**与第三方库集成：**
```go
// 获取所有标志集合
allFlagSets := app.GetAllFlagSets()

// 传递给第三方库
thirdPartyLib.WithFlags(allFlagSets...)

// 或者只传递业务标志
businessFlags := app.ExportFlagsForViper()
thirdPartyLib.WithBusinessFlags(businessFlags...)
```

#### API 参考

| 方法 | 用途 | 返回值 |
|------|------|--------|
| `ExportFlagsForViper(exclude...)` | 导出适用于Viper的标志 | `[]*FlagSet` |
| `GetAllFlagSets()` | 获取所有标志集合 | `[]*FlagSet` |
| `GetBindableFlagSets(exclude...)` | 获取可绑定的标志（排除系统标志） | `[]*FlagSet` |
| `GetFilteredFlags(exclude...)` | 获取单个过滤后的标志集合 | `*FlagSet` |
| `GetFlagNames(includeInherited)` | 获取标志名称列表 | `[]string` |
| `GetFilteredFlagNames(exclude...)` | 获取过滤后的标志名称 | `[]string` |
| `IsSystemFlag(name)` | 检查是否为系统标志 | `bool` |
| `GetSystemFlags()` | 获取所有系统标志列表 | `[]string` |

### 3. 智能语言包系统

```go
// 获取语言管理器
manager := zcli.NewLanguageManager("zh")

// 使用ServiceLocalizer简化操作
localizer := zcli.NewServiceLocalizer(manager, colors)

// 便利的API
localizer.LogSuccess("myapp", "install")    // 输出成功消息
localizer.LogError("createService", err)    // 输出错误消息
localizer.LogWarning("配置文件不存在")        // 输出警告消息

// 格式化错误消息
errorMsg := localizer.FormatError("pathNotExist", "/opt/app")
```

### 4. 并发安全保障

ZCli经过严格的并发安全测试：

```go
// 框架内部使用原子操作
var running atomic.Bool
var stopExecuted atomic.Bool

// 安全的channel操作
select {
case <-exitChan:
    // 已关闭
default:
    close(exitChan)  // 安全关闭
}

// 分级超时机制
select {
case <-done:
    return  // 正常退出
case <-time.After(3 * time.Second):
    // 第一级超时处理
    select {
    case <-done:
        return
    case <-time.After(2 * time.Second):
        // 第二级超时，强制退出
        forceTerminate()
    }
}
```

### 5. 错误处理和日志

```go
// 结构化错误处理
type ServiceError struct {
    Operation string
    Service   string
    Cause     error
}

// 自动日志记录
func (err ServiceError) Log(localizer *ServiceLocalizer) {
    localizer.LogError(err.Operation, err.Cause)
}

// 权限检查
if err := checkPermissions("/opt/app", 0o755); err != nil {
    return fmt.Errorf("权限检查失败: %w", err)
}
```

## 🔧 构建和部署

### 编译时注入构建信息

```bash
# 设置构建变量
VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse HEAD)
BRANCH=$(git rev-parse --abbrev-ref HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

# 编译
go build -ldflags "
  -X main.version=${VERSION}
  -X main.commit=${COMMIT}
  -X main.branch=${BRANCH}
  -X main.buildTime=${BUILD_TIME}
" -o myapp main.go
```

### 生产环境部署

```go
// 生产环境配置
app := zcli.NewBuilder("zh").
    WithName("production-service").
    WithVersion(version).           // 从构建时注入
    WithGitInfo(commit, branch, version).
    WithBuildTime(buildTime).
    WithDebug(false).               // 生产模式
    WithWorkDir("/opt/production-service").
    WithEnvVar("ENV", "production").
    WithDependencies("postgresql", "redis").
    WithSystemService(runService, stopService).
    Build()
```

## 📚 API 参考

### Builder API

#### 核心构建方法

| 方法 | 说明 | 示例 |
|------|------|------|
| `NewBuilder(lang)` | 创建新的Builder实例 | `zcli.NewBuilder("zh")` |
| `Build()` | 构建CLI实例（可能panic） | `builder.Build()` |
| `BuildWithError()` | 构建CLI实例，返回错误 | `app, err := builder.BuildWithError()` |

#### 基础配置方法

| 方法 | 说明 | 示例 |
|------|------|------|
| `WithName(name)` | 设置应用名称 | `.WithName("myapp")` |
| `WithDisplayName(name)` | 设置显示名称 | `.WithDisplayName("我的应用")` |
| `WithDescription(desc)` | 设置应用描述 | `.WithDescription("应用描述")` |
| `WithVersion(version)` | 设置版本号 | `.WithVersion("1.0.0")` |
| `WithLanguage(lang)` | 设置语言 | `.WithLanguage("zh")` |

#### 服务配置方法（现代API）

| 方法 | 说明 | 示例 |
|------|------|------|
| `WithServiceRunner(service)` | 配置服务接口实现 | `.WithServiceRunner(myService)` |
| `WithSimpleService(name, run, stop)` | 快速配置简单服务 | `.WithSimpleService("app", runFunc, stopFunc)` |
| `WithValidator(validator)` | 添加配置验证器 | `.WithValidator(validatorFunc)` |

#### 便利性API

| 方法 | 说明 | 示例 |
|------|------|------|
| `QuickService(name, desc, run)` | 快速创建服务应用 | `zcli.QuickService("app", "描述", runFunc)` |
| `QuickServiceWithStop(name, desc, run, stop)` | 快速创建带停止函数的服务 | `zcli.QuickServiceWithStop(...)` |
| `QuickCLI(name, display, desc)` | 快速创建CLI工具 | `zcli.QuickCLI("tool", "工具", "描述")` |

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

### 错误处理API

#### 错误创建和处理

| 函数 | 说明 | 示例 |
|------|------|------|
| `NewError(code)` | 创建错误构建器 | `zcli.NewError(zcli.ErrServiceStart)` |
| `NewErrorAggregator()` | 创建错误聚合器 | `aggregator := zcli.NewErrorAggregator()` |
| `GetServiceError(err)` | 获取服务错误 | `serviceErr, ok := zcli.GetServiceError(err)` |

#### 预定义错误函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `ErrServiceAlreadyRunning(name)` | 服务已运行错误 | `zcli.ErrServiceAlreadyRunning("myapp")` |
| `ErrServiceAlreadyStopped(name)` | 服务已停止错误 | `zcli.ErrServiceAlreadyStopped("myapp")` |
| `ErrServiceStartTimeout(name, timeout)` | 启动超时错误 | `zcli.ErrServiceStartTimeout("myapp", 30*time.Second)` |

#### 错误构建器API

```go
err := zcli.NewError(zcli.ErrServiceStart).
    Service("myapp").                    // 设置服务名
    Operation("start").                  // 设置操作
    Message("启动失败").                 // 设置消息
    Cause(originalError).                // 设置原因
    Context("key", "value").             // 添加上下文
    Build()                              // 构建错误
```

### 并发服务管理API

#### ConcurrentServiceManager

```go
manager := zcli.NewConcurrentServiceManager(serviceRunner, config)

// 配置方法
manager.SetStartTimeout(30 * time.Second)
manager.SetStopTimeout(10 * time.Second)
manager.SetErrorLogger(logFunc)
manager.AddStateListener(stateChangeFunc)

// 操作方法
err := manager.Start()
err := manager.Stop()
err := manager.Restart()

// 状态查询
state := manager.GetState()
isRunning := manager.IsRunning()
isStopped := manager.IsStopped()
stats := manager.GetStats()
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
// ✅ 推荐：添加配置验证
app, err := zcli.NewBuilder("zh").
    WithName("myapp").
    WithValidator(func(cfg *zcli.Config) error {
        if cfg.Basic.Name == "" {
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

## 📝 最佳实践

### 1. 优雅的服务生命周期管理

```go
func runService(ctx context.Context) {
    slog.Info("服务启动")
    
    // 设置定时器
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    // 设置健康检查
    healthTicker := time.NewTicker(30 * time.Second)
    defer healthTicker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            slog.Info("收到停止信号，开始优雅关闭")
            
            // 执行关闭前的清理工作
            cleanupResources()
            
            slog.Info("服务已安全停止")
            return
            
        case <-ticker.C:
            // 主要业务逻辑
            processMainTasks()
            
        case <-healthTicker.C:
            // 健康检查
            performHealthCheck()
        }
    }
}
```

### 2. 错误处理模式

```go
func stopService() {
    slog.Info("开始停止服务")
    
    // 设置停止超时
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 并发停止多个组件
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        stopDatabase(ctx)
    }()
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        stopCache(ctx)
    }()
    
    // 等待所有组件停止
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        slog.Info("所有组件已安全停止")
    case <-ctx.Done():
        slog.Warn("停止超时，部分组件可能未完全停止")
    }
}
```

### 3. 配置管理

```go
type AppConfig struct {
    Database DatabaseConfig `json:"database"`
    Cache    CacheConfig    `json:"cache"`
    Log      LogConfig      `json:"log"`
}

func loadConfig() *AppConfig {
    // 从环境变量或配置文件加载配置
    config := &AppConfig{}
    
    // 使用viper等配置库
    viper.SetConfigFile("config.json")
    if err := viper.ReadInConfig(); err == nil {
        viper.Unmarshal(config)
    }
    
    return config
}

func main() {
    config := loadConfig()
    
    app := zcli.NewBuilder("zh").
        WithName("myapp").
        WithWorkDir(config.WorkDir).
        WithSystemService(
            func() { runWithConfig(config) },
            func() { stopWithConfig(config) },
        ).
        Build()
    
    app.Execute()
}
```

## ⚠️ 注意事项

### 系统权限
- 服务管理功能需要适当的系统权限
- Linux/macOS: 通常需要sudo权限安装系统服务
- Windows: 需要管理员权限

### 平台兼容性
- 某些终端可能不支持彩色输出，框架会自动检测并降级
- Windows服务和Unix守护进程的配置选项有所不同
- 使用条件编译处理平台特定功能

### 语言包要求
- 自定义语言包必须实现所有必需字段
- 建议先复制内置语言包再修改
- 使用智能回退避免翻译缺失问题

### 并发安全
- 服务函数应该是线程安全的
- 避免在服务函数中使用全局变量
- 使用channel或context进行goroutine间通信

## 📚 依赖项

- **Go 1.23+**: 充分利用Go 1.23新特性（iter包、slices包、maps包等）
- **github.com/spf13/cobra**: 命令行框架基础
- **github.com/fatih/color**: 彩色输出支持  
- **github.com/darkit/syscore**: 跨平台系统服务管理

## 🆕 版本历史

### v2.0.0 (Latest)
- ✨ **新增服务接口抽象**: `ServiceRunner` 接口统一服务管理
- ✨ **增强Builder模式**: 新增 `WithServiceRunner`、`WithSimpleService`、`WithValidator` 方法
- ✨ **错误优先设计**: 新增 `BuildWithError` 方法，提供更好的错误处理
- ✨ **强大的错误处理系统**: 结构化错误、错误聚合器、预定义错误函数
- ✨ **并发安全保障**: 原子操作、线程安全设计、状态管理
- ✨ **便利性API**: `QuickService`、`QuickCLI` 等快速创建函数
- 🔧 **简化API设计**: 移除过度封装的模板代码，专注核心功能
- 🐛 **修复并发问题**: 解决服务管理中的race condition和死锁问题
- 📚 **完善文档**: 全面更新API文档和使用示例

### v1.x.x (Legacy)
- 基础CLI框架功能
- 系统服务管理
- 多语言支持
- 传统API设计

## 🤝 贡献

欢迎提交Issue和Pull Request！

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

**ZCli**: 让您的CLI应用更专业、更稳定、更易用。
