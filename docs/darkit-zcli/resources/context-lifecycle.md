# Context 生命周期管理

## 概述

ZCli 框架**自动管理** Context 生命周期，提供优雅的信号处理和资源清理机制。用户无需手动处理 SIGINT/SIGTERM 等信号。

## Context 的自动创建

框架内部自动创建带信号监听的 Context：

```go
// 框架内部自动执行（无需用户编写）
ctx, cancel := signal.NotifyContext(
    context.Background(),
    syscall.SIGINT,  // Ctrl+C
    syscall.SIGTERM, // 终止信号
    syscall.SIGQUIT, // 退出信号
)
```

## 生命周期流程

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

## 为什么使用 `func(ctx context.Context) error`？

### 优势

**1. 自动信号处理**

无需手动处理 SIGINT/SIGTERM：

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

**2. 优雅关闭链**

Context 可传递给所有依赖组件：

```go
func runService(ctx context.Context) error {
    // Context 传递给依赖组件
    db, _ := database.Connect(ctx)
    cache, _ := redis.Connect(ctx)

    // 当收到停止信号时，所有组件都能感知到
    // 数据库、缓存等会自动关闭连接
}
```

**3. 超时控制**

基于父 Context 创建子 Context：

```go
func runService(ctx context.Context) error {
    // 为某个操作设置超时
    opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    return doSomethingWithTimeout(opCtx)
}
```

### 对比：没有 Context（不推荐）

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

## 分级超时保护机制

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

### 自定义超时时间

```go
app := zcli.NewBuilder("zh").
    WithName("myapp").
    WithService(runService, stopService).
    WithShutdownTimeouts(
        10*time.Second,  // 优雅退出期
        5*time.Second,   // 清理期
    ).
    Build()
```

## 完整生命周期示例

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

## 最佳实践

### 1. 始终在 select 中监听 `ctx.Done()`

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

### 2. 传递 Context 给所有阻塞操作

```go
// 数据库查询
rows, err := db.QueryContext(ctx, sql)

// HTTP 请求
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

// gRPC 调用
conn, err := grpc.DialContext(ctx, addr)
```

### 3. 避免存储 Context（反模式）

```go
// 错误：不要将 context 存储在结构体中
type Service struct {
    ctx context.Context  // 违反 Go 官方建议
}

// 正确：通过参数传递
func (s *Service) Run(ctx context.Context) error {
    // 使用参数传递的 ctx
}
```

### 4. 使用 Context 传递请求范围的值

```go
// 传递请求 ID
ctx = context.WithValue(ctx, "requestID", uuid.New())

// 在下游获取
requestID := ctx.Value("requestID").(uuid.UUID)
```

### 5. 为子操作设置独立超时

```go
func processWithTimeout(ctx context.Context) error {
    // 即使父 Context 未取消，也限制此操作最多 5 秒
    opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return longRunningOperation(opCtx)
}
```

## 信号处理详解

### 默认处理的信号

| 信号 | 触发方式 | 行为 |
|------|----------|------|
| SIGINT | Ctrl+C | 触发 ctx.Done() |
| SIGTERM | kill PID | 触发 ctx.Done() |
| SIGQUIT | Ctrl+\ | 触发 ctx.Done() |

### 自定义信号处理

如需处理额外信号（如 SIGHUP 重载配置）：

```go
func runService(ctx context.Context) error {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP)

    for {
        select {
        case <-ctx.Done():
            return nil
        case sig := <-sigChan:
            if sig == syscall.SIGHUP {
                reloadConfig()
            }
        }
    }
}
```

## 调试技巧

### 模拟 Context 取消

```go
func TestGracefulShutdown(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        time.Sleep(100 * time.Millisecond)
        cancel()  // 模拟 Ctrl+C
    }()

    err := runService(ctx)
    if err != nil && err != context.Canceled {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### 检查 Context 取消原因

```go
case <-ctx.Done():
    switch ctx.Err() {
    case context.Canceled:
        slog.Info("收到取消信号")
    case context.DeadlineExceeded:
        slog.Info("操作超时")
    }
    return ctx.Err()
```
