# Service Manager (服务管理器)

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

一个灵活的 Go 语言服务管理库，提供完整的服务生命周期管理功能。

## 主要特性

- 完整的服务生命周期管理（安装、卸载、启动、停止、重启、状态查询）
- 多语言支持（内置中文和英文）
- 丰富的命令行参数系统，支持参数验证
- 运行时配置管理
- 详细的构建信息支持
- 可自定义主题的彩色控制台输出
- 跨平台服务支持（Windows/Linux/macOS）
- 自定义命令支持
- 参数验证和枚举值支持
- 并发参数处理
- 调试模式支持

## 安装

```bash
go get github.com/darkit/zcli
```

## 环境要求

- Go 1.21.5 或更高版本
- Windows、Linux 或 macOS 操作系统

## 依赖项

- github.com/kardianos/service v1.2.2
- github.com/fatih/color v1.18.0

## 快速开始

```go
package main

import (
    "fmt"
    "log/slog"
    "time"
    "github.com/darkit/zcli"
)

func main() {
    // 创建构建信息
    buildInfo := zcli.NewBuildInfo().
        SetDebug(true).
        SetVersion("1.0.0").
        SetBuildTime(time.Now())

    // 创建服务
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "我的应用",
        Description: "这是一个示例应用服务",
        Version:     "1.0.0",
        Language:    "zh",
        BuildInfo:   buildInfo,
        Run: func() error {
            slog.Info("服务正在运行...")
            return nil
        },
        Stop: func() error {
            slog.Info("服务正在停止...")
            return nil
        },
    })
    if err != nil {
        slog.Error("创建服务失败", "error", err)
        return
    }

    // 添加参数
    pm := svc.ParamManager()
    
    // 添加配置文件参数
    pm.AddParam(&zcli.Parameter{
        Name:        "config",
        Short:       "c",
        Long:        "config", 
        Description: "配置文件路径",
        Required:    true,
        Type:        "string",
    })

    // 添加端口参数（带验证）
    pm.AddParam(&zcli.Parameter{
        Name:        "port",
        Short:       "p",
        Long:        "port",
        Description: "服务端口",
        Default:     "8080",
        Type:        "string",
        Validate: func(val string) error {
            if val == "0" {
                return fmt.Errorf("端口不能为0")
            }
            return nil
        },
    })

    // 添加模式参数（带枚举值）
    pm.AddParam(&zcli.Parameter{
        Name:        "mode",
        Short:       "m",
        Long:        "mode",
        Description: "运行模式",
        Default:     "prod",
        EnumValues:  []string{"dev", "test", "prod"},
        Type:        "string",
    })

    // 运行服务
    if err := svc.Run(); err != nil {
        slog.Error("服务运行失败", "error", err)
    }
}
```

## 命令行使用

```bash
# 显示帮助信息
./myapp -h

# 显示版本信息
./myapp -v

# 安装并启动服务
./myapp install

# 启动服务
./myapp start

# 停止服务
./myapp stop

# 重启服务
./myapp restart

# 显示服务状态
./myapp status

# 卸载服务
./myapp uninstall

# 使用自定义参数运行
./myapp --port 9090 --mode dev --config config.toml

# 使用中文
./myapp --lang zh
```

## 参数系统

该库提供了完整的参数管理系统：

```go
// 添加必需参数
pm.AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "配置文件路径",
    Required:    true,
    Type:        "string",
})

// 添加带验证的参数
pm.AddParam(&zcli.Parameter{
    Name:        "workers",
    Short:       "w",
    Long:        "workers",
    Description: "工作线程数",
    Default:     "5",
    Type:        "string",
    Validate: func(val string) error {
        if val == "0" {
            return fmt.Errorf("工作线程数不能为0")
        }
        return nil
    },
})

// 添加枚举参数
pm.AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Long:        "mode",
    Description: "运行模式",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
    Type:        "string",
})

// 获取参数值
port := pm.GetString("port")
workers := pm.GetInt("workers")
isDebug := pm.GetBool("debug")
```

## 构建信息

支持详细的构建信息，配合构建脚本使用：

```bash
#!/bin/bash
VERSION="1.0.0"
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

go build \
    -ldflags "-X main.version=${VERSION} \
              -X buildInfo.buildTime=${BUILD_TIME} \
              -X buildInfo.gitCommit=${GIT_COMMIT} \
              -X buildInfo.gitBranch=${GIT_BRANCH} \
              -X buildInfo.gitTag=${GIT_TAG}" \
    -o myapp
```

## 自定义命令

支持添加自定义命令：

```go
pm.AddCommand("custom", "自定义命令描述", func() {
    // 自定义命令逻辑
    return nil
}, false)
```

## 配置管理

运行时配置管理：

```go
// 设置配置值
svc.SetConfigValue("lastStartTime", time.Now().Unix())

// 获取配置值
value, exists := svc.GetConfigValue("lastStartTime")

// 删除配置值
svc.DeleteConfigValue("lastStartTime")

// 检查配置是否存在
exists := svc.HasConfigValue("lastStartTime")

// 获取所有配置键
keys := svc.GetConfigKeys()
```

## 调试模式

支持调试模式：

```go
// 启用调试模式
svc.EnableDebug()

// 禁用调试模式
svc.DisableDebug()

// 检查调试状态
isDebug := svc.IsDebug()
```

## 示例

查看 `examples` 目录获取完整的工作示例。

## 许可证

本项目使用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。