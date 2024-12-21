# Service Manager (服务管理器)

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

一个灵活的 Go 语言服务管理库，提供完整的服务生命周期管理功能。

## 主要特性

- 完整的服务生命周期管理（安装、卸载、启动、停止、重启）
- 多语言支持（内置中文和英文）
- 丰富的命令行参数系统
- 运行时配置管理
- 详细的构建信息支持
- 彩色控制台输出
- Windows 服务支持

## 安装

```bash
go get github.com/darkit/zcli
```

## 环境要求

- Go 1.16 或更高版本
- Windows、Linux 或 macOS 操作系统

## 依赖项

- github.com/kardianos/service
- github.com/fatih/color

## 快速开始

```go
package main

import (
    "log"
    "github.com/darkit/zcli"
)

func main() {
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "我的应用",
        Description: "这是我的应用服务",
        Version:     "1.0.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    if err := svc.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## 命令行使用

```bash
# 显示帮助信息
./myapp -h

# 安装服务
./myapp install

# 启动服务
./myapp start

# 停止服务
./myapp stop

# 显示状态
./myapp status

# 卸载服务
./myapp uninstall

# 使用中文
./myapp install --lang zh

# 使用自定义参数运行
./myapp --port 9090 --mode dev
```

## 参数系统

该库提供了灵活的参数管理系统：

```go
// 添加必需参数
svc.ParamManager().AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "配置文件路径",
    Required:    true,
})

// 添加枚举参数
svc.ParamManager().AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Long:        "mode",
    Description: "运行模式",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
})
```

## 多语言支持

内置中英文支持，可以轻松添加更多语言：

```go
// 添加新语言
svc.AddLanguage("fr", zcli.Messages{
    Install:   "Installer le service",
    Uninstall: "Désinstaller le service",
    Start:     "Démarrer le service",
    Stop:      "Arrêter le service",
    // ...
})

// 切换语言
svc.SetLanguage("fr")
```

## 构建信息

支持详细的构建信息：

```go
svc, err := zcli.New(&zcli.Options{
    Name:    "myapp",
    Version: "1.0.0",
    BuildInfo: zcli.NewBuildInfo().
        SetVersion("1.0.0").
        SetBuildTime(time.Now()).
        SetDebug(true),
})
```

## 运行时配置

所有参数和设置都在内存中管理：

```go
// 设置自定义配置值
svc.SetConfigValue("custom_key", "custom_value")

// 获取配置值
value, exists := svc.GetConfigValue("custom_key")

// 删除配置值
svc.DeleteConfigValue("custom_key")
```

## 服务事件

支持服务生命周期事件：

```go
svc, err := zcli.New(&zcli.Options{
    Name: "myapp",
    Run: func() error {
        // 服务运行逻辑
        return nil
    },
    Stop: func() error {
        // 服务停止逻辑
        return nil
    },
})
```

## 高级用法

查看 `examples` 目录获取更多高级用法示例。

## 许可证

本项目使用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。