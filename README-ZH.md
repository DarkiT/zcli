# Service Manager (服务管理器)

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

一个灵活的 Go 语言服务管理库，提供完整的服务生命周期管理功能。

## 核心功能

### 服务管理
- 完整的服务生命周期管理（安装、卸载、启动、停止、重启）
- 跨平台支持（Windows/Linux/macOS）
- 服务状态监控和报告
- 优雅关闭处理

### 参数系统
- 丰富的命令行参数管理
- 参数验证和枚举值支持
- 并发参数处理
- 短格式和长格式参数支持
- 必需参数强制检查
- 默认值支持
- 自定义验证规则

### 国际化
- 内置多语言支持
- 便捷的语言切换
- 可自定义消息模板

### 开发工具
- 调试模式与增强日志
- 自定义命令支持
- 彩色控制台输出与主题
- 详细的构建信息
- 完整的错误处理

## 安装

```bash
go get github.com/darkit/zcli
```

## 快速开始

这是一个最小的示例：

```go
package main

import (
    "github.com/darkit/zcli"
    "log/slog"
)

func main() {
    // 创建服务
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "我的应用",
        Description: "示例服务",
        Version:     "1.0.0",
        Run: func() error {
            slog.Info("服务正在运行...")
            return nil
        },
    })
    if err != nil {
        slog.Error("创建服务失败", "error", err)
        return
    }

    // 运行服务
    if err := svc.Run(); err != nil {
        slog.Error("服务运行失败", "error", err)
    }
}
```

## 高级用法

### 参数配置

```go
pm := svc.ParamManager()

// 添加必需参数
pm.AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "配置文件路径",
    Required:    true,
    Type:        "string",
})

// 添加带自定义验证的参数
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

// 添加带预定义值的枚举参数
pm.AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Long:        "mode",
    Description: "运行模式",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
    Type:        "string",
})
```

### 自定义命令

```go
// 添加版本命令
pm.AddCommand("version", "显示版本信息", func() {
    fmt.Printf("版本: %s\n", svc.GetVersion())
}, true)

// 添加检查命令
pm.AddCommand("check", "检查服务状态", func() {
    // 添加检查逻辑
}, false)
```

### 调试模式

```go
// 启用调试模式
svc.EnableDebug()

// 使用调试日志
if svc.IsDebug() {
    slog.Debug("调试信息...")
}

// 禁用调试模式
svc.DisableDebug()
```

### 错误处理

服务提供清晰的参数验证错误信息：

```bash
# 无效的参数值
$ ./myapp -p 0
Error: validation failed for parameter port: 端口不能为0

使用 './myapp --help' 获取更多信息。

# 缺少必需参数
$ ./myapp
Error: parameter 'config' is required

使用 './myapp --help' 获取更多信息。

# 无效的枚举值
$ ./myapp --mode invalid
Error: invalid value for parameter mode: must be one of [dev test prod]

使用 './myapp --help' 获取更多信息。
```

### 命令行界面

```bash
# 基本命令
./myapp install    # 安装服务
./myapp start     # 启动服务
./myapp stop      # 停止服务
./myapp restart   # 重启服务
./myapp status    # 显示状态
./myapp uninstall # 卸载服务

# 带参数运行或者带参数安装服务
./myapp --port 9090 --mode dev
./myapp install --port 9090 --mode dev

# 显示帮助
./myapp -h
./myapp --help

# 显示版本
./myapp -v
./myapp --version

# 自定义命令
./myapp version
./myapp check
```

## 完整示例

查看 [examples/main.go](examples/main.go) 获取完整的工作示例。

## 许可证

本项目使用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。