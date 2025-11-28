---
name: darkit-zcli
description: 帮助 Claude 使用 darkit-zcli 库构建企业级 CLI 的命令树、服务管理与国际化实践。
tags:
  - go
  - cli
  - automation
---

# DarkiT ZCLI 命令行技能

## 适用场景

- 构建企业级 CLI 应用，需要服务管理功能
- 实现前台/后台双模式运行的守护进程
- 需要优雅关闭和信号处理的长期运行服务
- 希望 CLI 与服务共享配置、依赖注入和日志

## 核心概念

### Builder 模式

使用 `NewBuilder` 创建 CLI 应用，支持链式配置：

```go
app, err := zcli.NewBuilder("zh").
    WithName("myapp").
    WithDisplayName("我的应用").
    WithVersion("1.0.0").
    WithService(runService, stopService).
    BuildWithError()

if err != nil {
    log.Fatal(err)
}

app.Execute()
```

### ServiceRunner 接口

实现 `ServiceRunner` 接口来定义服务行为：

```go
type ServiceRunner interface {
    Run(ctx context.Context) error  // 运行服务，监听 ctx.Done() 实现优雅关闭
    Stop() error                    // 停止服务，执行清理工作
    Name() string                   // 返回服务名称
}
```

### Context 生命周期

框架**自动管理** Context 生命周期：
- 自动创建带信号监听的 Context（SIGINT/SIGTERM/SIGQUIT）
- 用户按 Ctrl+C 时自动触发 `ctx.Done()`
- 3+2 秒分级超时保护机制

## 使用示例

### 最小服务应用

```go
package main

import (
    "context"
    "log/slog"
    "time"
    "github.com/darkit/zcli"
)

func main() {
    app := zcli.QuickService("myapp", "我的应用", runService)
    app.Execute()
}

func runService(ctx context.Context) error {
    slog.Info("服务启动")
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():  // 框架自动触发
            slog.Info("收到停止信号")
            return nil
        case <-ticker.C:
            slog.Info("服务运行中...")
        }
    }
}
```

### 完整配置示例

```go
app, err := zcli.NewBuilder("zh").
    WithName("enterprise-app").
    WithDisplayName("企业级应用").
    WithDescription("提供企业级功能的后台服务").
    WithVersion("2.1.0").
    WithWorkDir("/opt/enterprise-app").
    WithEnvVar("ENV", "production").
    WithDependencies("postgresql", "redis").
    WithServiceRunner(myService).
    WithValidator(func(cfg *zcli.Config) error {
        // 注意：必须使用 getter 方法访问 Config 字段
        if cfg.Basic().Name == "" {
            return fmt.Errorf("应用名称不能为空")
        }
        return nil
    }).
    BuildWithError()
```

## 指导原则

### Config 字段访问

Config 结构体字段是**私有的**，必须通过 getter 方法访问：

```go
// 正确
cfg.Basic().Name
cfg.Service().EnvVars
cfg.Runtime().Run

// 错误 - 编译会失败
cfg.Basic.Name      // 私有字段无法访问
cfg.Service.EnvVars // 私有字段无法访问
```

### 服务函数签名

使用标准 Go 惯例的函数签名：

```go
type RunFunc  func(ctx context.Context) error  // 运行函数
type StopFunc func() error                     // 停止函数
```

### 信号处理

在 Run 函数中**始终监听** `ctx.Done()`：

```go
func runService(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()  // 或 return nil
        case work := <-workChan:
            process(work)
        }
    }
}
```

## 常见陷阱

- **直接 `os.Exit`** 会跳过 Hook 与清理逻辑，使用 `return error` 代替
- **忽略 `ctx.Done()`** 会导致服务无法优雅关闭
- **使用 `cfg.Basic.Name`** 编译失败，必须使用 `cfg.Basic().Name`
- **存储 Context 到结构体** 违反 Go 官方建议，通过参数传递

## 资源导航

- [API 参考](resources/api-reference.md)
- [Context 生命周期](resources/context-lifecycle.md)
- [配置选项](resources/config-options.md)
- [服务管理](resources/service-management.md)
- [完整示例](resources/complete-example.md)
