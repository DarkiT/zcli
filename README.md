# ZCli 更友好的命令行界面和系统服务管理扩展包

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

ZCli 是一个基于 [cobra](https://github.com/spf13/cobra) 的命令行应用框架，提供了更友好的命令行界面和系统服务管理功能。

## 特性

- 友好的命令行界面
  - 支持彩色输出，自动检测终端颜色支持
  - 自定义 Logo 显示
  - 分组显示命令（普通命令和系统命令）
  - 优化的帮助信息展示
  - 支持命令补全

- 系统服务集成
  - 一键集成系统服务管理功能
  - 支持服务的安装、卸载、启动、停止、重启和状态查询
  - 支持服务配置的自定义（工作目录、环境变量等）
  - 跨平台支持 (Windows, Linux, macOS)

- 国际化支持
  - 内置中文和英文支持
  - 可扩展的语言包系统
  - 支持自定义语言包和错误消息

- 版本信息管理
  - 详细的版本信息展示
  - 支持构建信息的自定义
  - 支持调试/发布模式切换

## 安装

```bash
go get github.com/darkit/zcli
```

## 快速开始

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "github.com/darkit/zcli"
)

const logo = `
███████╗ ██████╗██╗     ██╗
╚══███╔╝██╔════╝██║     ██║
  ███╔╝ ██║     ██║     ██║
 ███╔╝  ██║     ██║     ██║
███████╗╚██████╗███████╗██║
╚══════╝ ╚═════╝╚══════╝╚═╝
`

// 全局变量用于控制服务状态
var isRunning = true

func main() {
    workDir, _ := os.UserHomeDir()
    app := zcli.NewBuilder().
        WithName("myapp").
        WithDisplayName("我的应用").
        WithDescription("这是一个示例应用").
        WithLogo(logo).
        WithLanguage("zh").
        WithVersion("1.0.0").
        WithGitInfo("abc123", "master", "v1.0.0").
        WithDebug(true).
        WithWorkDir(workDir).
        WithEnvVar("ENV", "prod").
        WithSystemService(run, stop).
        Build()

    // 添加全局标志
    app.PersistentFlags().BoolP("debug", "d", false, "启用调试模式")
    app.PersistentFlags().StringP("config", "c", "", "配置文件路径")

    // 执行应用
    if err := app.Execute(); err != nil {
        slog.Error(err.Error())
    }
}

// 服务主函数
func run() {
    slog.Info("服务已启动")
    isRunning = true

    // 创建定时器
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    // 服务主循环 - 使用for range简化定时器处理
    for range ticker.C {
        if !isRunning {
            break
        }
        slog.Info("服务正在运行...")
    }
}

// 服务停止函数
func stop() {
    slog.Warn("服务停止中...")
    isRunning = false
    slog.Info("服务已停止")
}
```

## 高级用法

### 自定义命令和子命令

```go
// 创建配置管理主命令
configCmd := &zcli.Command{
    Use:   "config",
    Short: "配置管理",
    Run: func(cmd *zcli.Command, args []string) {
        slog.Info("配置管理", "name", cmd.Name(), "args", args)
    },
}

// 创建配置查看子命令
showCmd := &zcli.Command{
    Use:                   "show",
    Short:                 "查看当前配置",
    DisableFlagParsing:    true, // 禁用标志解析
    DisableFlagsInUseLine: true, // 在使用说明中禁用标志
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("当前配置:")
        fmt.Println("- 服务名称: myapp")
        fmt.Println("- 显示名称: 我的应用")
        fmt.Println("- 版本: 1.0.0")
    },
}

// 创建配置更新子命令
updateCmd := &zcli.Command{
    Use:                   "update",
    Short:                 "更新配置",
    DisableFlagParsing:    true,
    DisableFlagsInUseLine: true,
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("更新配置...")
    },
}

// 添加子命令到配置管理命令
configCmd.AddCommand(showCmd, updateCmd)
// 添加配置管理命令到主应用
app.AddCommand(configCmd)
```

### 服务配置

```go
app := zcli.NewBuilder().
    WithName("myapp").
    WithDisplayName("我的应用").
    WithDescription("这是一个示例应用").
    WithWorkDir("/opt/myapp").
    WithEnvVar("ENV", "production").
    WithEnvVar("LOG_LEVEL", "info").
    WithDependencies("mysql", "redis").
    WithSystemService(run, stop).
    Build()
```

### 版本信息

```go
app := zcli.NewBuilder().
    WithVersion("1.0.0").
    WithGitInfo("abc123", "master", "v1.0.0").
    WithBuildInfo("2023-06-01", "go1.22", "amd64").
    WithDebug(true).
    Build()
```

### 多语言支持

ZCli内置了中文和英文支持，可以通过以下方式轻松切换：

```go
// 方法一：通过NewBuilder参数直接指定语言（推荐）
app := zcli.NewBuilder("zh").Build()  // 中文
app := zcli.NewBuilder("en").Build()  // 英文

// 方法二：通过WithLanguage方法设置
app := zcli.NewBuilder().WithLanguage("zh").Build()
app := zcli.NewBuilder().WithLanguage("en").Build()
```

以上两种方法效果完全相同，方法一只是方法二的便捷实现，为了让调用更简洁。

自定义语言包：

```go
customLang := &zcli.Language{
    Service: zcli.ServiceMessages{
        Install:          "安装服务",
        Start:            "启动服务",
        ErrGetStatus:     "获取服务状态失败",
        ErrStartService:  "启动服务失败",
        // ... 其他必要字段
    },
    Command: zcli.CommandMessages{
        // ... 命令相关翻译
    },
    Error: zcli.ErrorMessages{
        // ... 错误相关翻译
    },
}

// 注册自定义语言包
zcli.AddLanguage("custom", customLang)

// 使用自定义语言包（两种方式）
app := zcli.NewBuilder("custom").Build()
// 或
app := zcli.NewBuilder().WithLanguage("custom").Build()
```

## 命令行界面示例

```bash
$ myapp --help

███████╗ ██████╗██╗     ██╗
╚══███╔╝██╔════╝██║     ██║
  ███╔╝ ██║     ██║     ██║
 ███╔╝  ██║     ██║     ██║
███████╗╚██████╗███████╗██║
╚══════╝ ╚═════╝╚══════╝╚═╝ v1.0.0

这是一个示例应用

用法:
   myapp [参数]
   myapp [命令] [参数]

选项:
   -h, --help      获取帮助
   -v, --version   显示版本信息
   -d, --debug     启用调试模式
   -c, --config    配置文件路径

可用命令:
   help        获取帮助
   config      配置管理
   show        查看当前配置
   update      更新配置

系统命令:
   run         运行服务
   start       启动服务
   stop        停止服务
   status      查看状态
   restart     重启服务
   install     安装服务
   uninstall   卸载服务

使用 'myapp [command] --help' 获取命令的更多信息。
```

## 注意事项

1. 服务管理功能需要适当的系统权限，特别是安装和卸载服务时
2. Windows 下某些终端可能不支持彩色输出，系统会自动检测并降级
3. 自定义语言包需要实现所有必需的字段
4. 服务循环设计应当保持简洁，避免复杂的嵌套逻辑

## 依赖

- Go 1.22 或更高版本
- github.com/spf13/cobra
- github.com/fatih/color

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

[MIT License](LICENSE)
