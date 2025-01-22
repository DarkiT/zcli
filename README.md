# ZCli 更美观的命令行应用框架
[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

ZCli 是一个基于 [cobra](https://github.com/spf13/cobra) 的命令行应用框架，提供了更友好的命令行界面和系统服务管理功能。它专注于简化命令行应用的开发，并提供了丰富的国际化支持。

## 特性

### 1. 友好的命令行界面
- 支持彩色输出，自动检测终端颜色支持
- 自定义 Logo 显示
- 分组显示命令（普通命令和系统命令）
- 优化的帮助信息展示
- 支持命令补全

### 2. 系统服务集成
- 一键集成系统服务管理功能
- 支持服务的安装、卸载、启动、停止、重启和状态查询
- 支持服务配置的自定义（工作目录、环境变量等）
- 跨平台支持 (Windows, Linux, macOS)

### 3. 国际化支持
- 内置中文和英文支持
- 可扩展的语言包系统
- 支持自定义语言包

### 4. 版本信息管理
- 详细的版本信息展示
- 支持构建信息的自定义
- 支持调试/发布模式切换

## 快速开始

```go
package main

import (
    "fmt"
    "time"

    "github.com/darkit/zcli"
)

func main() {
    // 创建应用实例
    app := zcli.NewCli(
        zcli.WithName("myapp"),           // 设置应用名称
        zcli.WithDisplayName("My App"),   // 设置显示名称
        zcli.WithDescription("示例应用"), // 设置应用描述
        zcli.WithLanguage("zh"),          // 设置语言
        zcli.WithRun(run),                // 设置主函数
    )

    // 添加命令行参数
    app.PersistentFlags().StringP("config", "c", "", "配置文件路径")

    // 执行应用
    if err := app.Execute(); err != nil {
        fmt.Println(err)
    }
}

func run() {
    // 应用主逻辑
    for {
        time.Sleep(time.Second)
        fmt.Println("应用运行中...")
    }
}
```

## 高级特性

### 自定义服务配置

```go
zcli.NewCli(
    zcli.WithName("myapp"),
    zcli.WithWorkingDirectory("/opt/myapp"),
    zcli.WithEnvVar("ENV", "production"),
    zcli.WithDependencies("mysql", "redis"),
)
```

### 版本信息管理

```go
version := zcli.NewVersionInfo().
    SetVersion("1.0.0").
    SetDebug(true).
    SetBuildTime(time.Now())

zcli.NewCli(
    zcli.WithName("myapp"),
    zcli.WithBuildInfo(version),
)
```

### 自定义语言包

```go
customLang := &zcli.Language{
    Service: zcli.ServiceMessages{
        Install: "安装服务",
        Start:   "启动服务",
        // ...
    },
    // ...
}

zcli.AddLanguage("custom", customLang)
zcli.SetDefaultLanguage("custom")
```

## 命令行界面示例

```bash
$ myapp --help

███████╗████████╗ ██████╗  ██████╗ ██╗     
╚══███╔╝╚══██╔══╝██╔═══██╗██╔═══██╗██║     
  ███╔╝    ██║   ██║   ██║██║   ██║██║     
 ███╔╝     ██║   ██║   ██║██║   ██║██║     
███████╗   ██║   ╚██████╔╝╚██████╔╝███████╗
╚══════╝   ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝ Ver 1.0.0

这是一个示例应用

用法:
   myapp [参数]
   myapp [命令] [参数]

选项:
   -h, --help      获取帮助
   -v, --version   显示版本信息
   -c, --config    配置文件路径

可用命令:
   help        获取帮助
   config      配置管理

系统命令:
   start       启动服务
   stop        停止服务
   status      查看状态
   restart     重启服务
   install     安装服务
   uninstall   卸载服务

使用 'myapp [command] --help' 获取命令的更多信息。
```

## 注意事项

1. 服务管理功能需要适当的系统权限
2. Windows 下某些终端可能不支持彩色输出
3. 自定义语言包需要实现所有必需的字段

## 许可证

[MIT License](LICENSE)
