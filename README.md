# ZCli 更美观的命令行应用框架

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)
[![Coverage Status](https://coveralls.io/repos/github/darkit/zcli/badge.svg?branch=master)](https://coveralls.io/github/darkit/zcli?branch=master)

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
  - 支持自定义语言包

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
    "fmt"
    "log"
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

func main() {
    app := zcli.NewBuilder().
        WithName("myapp").
        WithDisplayName("我的应用").
        WithDescription("这是一个示例应用").
        WithLogo(logo).
        WithLanguage("zh").
        WithVersion("1.0.0").
        WithGitInfo("abc123", "master", "v1.0.0").
        WithDebug(true).
        WithWorkDir("/app").
        WithEnvVar("ENV", "prod").
        WithSystemService(run, stop).
        Build()

    // 添加命令行参数
    app.PersistentFlags().StringP("config", "c", "", "配置文件路径")

    // 执行应用
    if err := app.Execute(); err != nil {
        log.Fatal(err)
    }
}

func run() {
    for {
        time.Sleep(time.Second * 5)
        fmt.Println("服务运行中...")
    }
}

func stop() {
    fmt.Println("服务停止中...")
}
```

## 高级用法

### 自定义命令

```go
configCmd := &zcli.Command{
    Use:   "config",
    Short: "配置管理",
    Run: func(cmd *zcli.Command, args []string) {
        fmt.Println("配置管理...")
    },
}

app.AddCommand(configCmd)
```

### 服务配置

```go
app := zcli.NewBuilder().
    WithName("myapp").
    WithWorkDir("/opt/myapp").
    WithEnvVar("ENV", "production").
    WithDependencies("mysql", "redis").
    Build()
```

### 版本信息

```go
app := zcli.NewBuilder().
    WithVersion("1.0.0").
    WithGitInfo("abc123", "master", "v1.0.0").
    WithDebug(true).
    Build()
```

### 自定义语言包

```go
customLang := &zcli.Language{
    Service: zcli.ServiceMessages{
        Install: "安装服务",
        Start:   "启动服务",
        // ...
    },
}

zcli.AddLanguage("custom", customLang)
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

## 依赖

- Go 1.21 或更高版本
- github.com/spf13/cobra
- github.com/fatih/color

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

[MIT License](LICENSE)
