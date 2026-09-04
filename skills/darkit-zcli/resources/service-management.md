# 服务管理和并发控制指南

> 最近更新（2026-06-16）：`ManagedService` 正常取消归一为成功退出；停止钩子在并发关闭路径只执行一次；官方 examples E2E 已覆盖 Ctrl+C 不打印 `Usage:` / `context canceled`。

## 概述

ZCli 提供完整的服务管理和并发控制系统，支持前台/后台双模式运行、优雅关闭、状态管理和生命周期钩子等企业级特性。

## 服务接口系统

### ServiceRunner 核心接口（唯一入口）

```go
type ServiceRunner interface {
    // Run 运行服务主逻辑，接收上下文用于优雅关闭
    Run(ctx context.Context) error

    // Stop 停止服务，执行清理工作
    Stop() error

    // Name 返回服务名称
    Name() string
}
```

**设计原则**：
- **上下文驱动**：通过 context.Context 实现优雅关闭
- **错误返回**：所有操作返回错误而非 panic
- **可识别性**：每个服务都有唯一名称
- **并发安全**：`session` / `exitChan` / `service` / 强制退出定时器等共享状态统一由 `sm.mu` 保护，
  `stopMu` 只串行化停止流程本身（锁序 `stopMu` → `mu`）；停止意图另有专属状态，见"停止意图"一节

### 基本服务实现

```go
type MyService struct {
    config ServiceConfig
    db     *sql.DB
    stopCh chan struct{}
}

func (s *MyService) Run(ctx context.Context) error {
    slog.Info("服务启动")

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            slog.Info("收到关闭信号")
            return nil
        case <-s.stopCh:
            return nil
        case <-ticker.C:
            if err := s.doWork(); err != nil {
                return err
            }
        }
    }
}

func (s *MyService) Stop() error {
    slog.Info("停止服务")
    close(s.stopCh)
    return s.db.Close()
}

func (s *MyService) Name() string {
    return "my-service"
}
```

## 运行模式

### 前台运行模式

适用于开发和调试：

```bash
./myapp run
```

特点：
- 实时显示日志输出
- 支持 Ctrl+C 优雅退出
- Interactive 模式检测

### 后台服务模式

适用于生产环境：

```bash
# 安装为系统服务
sudo ./myapp install

# 服务管理
sudo ./myapp start
sudo ./myapp stop
sudo ./myapp restart
sudo ./myapp status

# 卸载服务
sudo ./myapp uninstall
```

特点：
- 后台运行，由系统服务管理器控制
- 支持开机自启、自动重启
- 完整的生命周期管理

## 服务生命周期

### ServiceLifecycle 接口

```go
type ServiceLifecycle interface {
    BeforeStart() error  // 服务启动前调用
    AfterStart() error   // 服务启动后调用
    BeforeStop() error   // 服务停止前调用
    AfterStop() error    // 服务停止后调用
}
```

### 生命周期执行顺序

```
启动流程:
1. BeforeStart()  -> 初始化资源（数据库、缓存、配置等）
2. Run(ctx)       -> 启动服务主逻辑
3. AfterStart()   -> 注册服务（服务发现、健康检查等）

停止流程:
1. BeforeStop()   -> 注销服务（从服务发现移除）
2. Stop()         -> 停止服务主逻辑
3. AfterStop()    -> 释放资源（关闭连接、保存状态等）
```

### 生命周期实现示例

```go
type AppLifecycle struct {
    service *AppService
    db      *sql.DB
}

func (l *AppLifecycle) BeforeStart() error {
    slog.Info("初始化资源...")

    // 连接数据库
    db, err := sql.Open("postgres", "postgres://...")
    if err != nil {
        return fmt.Errorf("连接数据库失败: %w", err)
    }
    l.db = db
    l.service.SetDatabase(db)

    return nil
}

func (l *AppLifecycle) AfterStart() error {
    slog.Info("注册到服务发现...")
    // 注册到 Consul 等
    return nil
}

func (l *AppLifecycle) BeforeStop() error {
    slog.Info("从服务发现注销...")
    return nil
}

func (l *AppLifecycle) AfterStop() error {
    slog.Info("释放资源...")
    if l.db != nil {
        return l.db.Close()
    }
    return nil
}
```

## 运行状态边界

当前主线不公开独立的 `ServiceState` / `AddStateListener` API。运行状态由以下来源判定：

- CLI 前台模式：`Run(ctx)` 返回值和进程退出码
- 系统服务模式：底层 `github.com/darkit/daemon` 的 `Status()`
- 业务内部：服务实现自行维护健康状态或暴露 `health` 命令

框架内部使用原子状态保护重复运行、重复停止和后台强制退出；此外还有一组受 `sm.mu` 保护的状态
承载**停止意图**，保证先于运行落地的 Stop 不会被后续 Run 吞掉（见"停止意图"一节）。这些状态均不作为公开 API 承诺。

## 信号处理（RunWait 机制）

### 默认流程

```
Run() → 内部创建 runCtx（可取消）
    ↓
RunWait() 监听信号 → 捕获后以 ShutdownReasonSignal 触发停止流程
    ↓
取消运行上下文 → 并发执行用户 Stop()（由 stopFuncOnce 保证仅一次）
    ↓
Run() 退出，RunWait 返回
```

### 默认处理的信号

| 信号 | 触发方式 | 行为 |
|------|----------|------|
| SIGINT | Ctrl+C | RunWait 触发停止流程 |
| SIGTERM | kill PID | RunWait 触发停止流程 |
| SIGQUIT | Ctrl+\ | RunWait 触发停止流程（**仅非 Windows**，Windows 只监听 SIGINT/SIGTERM） |

> 信号监听通过注入 daemon 的 `OptionRunWait` 钩子实现（内部用 `signal.Notify`，不是
> `signal.NotifyContext`）。若用户自带 `OptionRunWait`，框架不再注入默认监听。
>
> 关于超时：15s + 5s 分级预算**只在 CLI 前台 `run` 命令路径**生效（先等主服务退出，再等清理收尾），
> 后台托管路径不走这段。且 `ShutdownInitial` / `ShutdownGrace` 显式传 0 不会禁用等待，
> 而是回落为 3s + 2s。详见 `context-lifecycle.md`。

### 自定义信号处理（在业务中追加，而非替换框架监听）

```go
func runService(ctx context.Context) error {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGUSR1)

    for {
        select {
        case <-ctx.Done():
            return nil
        case sig := <-sigChan:
            switch sig {
            case syscall.SIGHUP:
                reloadConfig()
            case syscall.SIGUSR1:
                printStatus()
            }
        }
    }
}
```

## 优雅关闭

### 优雅关闭流程

```
RunWait 捕获 SIGINT/SIGTERM/SIGQUIT
    ↓
sm.Stop() 取消传给用户 Run(ctx) 的 context
    ↓
Run(ctx) 收到 ctx.Done() 并返回 nil
    ↓
必要时执行 Stop() / 生命周期停止钩子
    ↓
Execute 返回 nil，进程以 0 退出
```

正常关闭不应向 Cobra 返回 `context.Canceled`，否则 Cobra 会按命令错误打印 `Error:` 和 `Usage:`。
`ManagedService` 会把 `context.Canceled`、`context.DeadlineExceeded` 和 `ShutdownCause` 归一为成功关闭，并保证 `BeforeStop` / `Stop` / `AfterStop` 在并发关闭路径只执行一次。

### HTTP 服务优雅关闭

```go
func (s *HTTPService) Run(ctx context.Context) error {
    server := &http.Server{Addr: ":8080"}

    errChan := make(chan error, 1)
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            errChan <- err
        }
    }()

    select {
    case <-ctx.Done():
        // 优雅关闭
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        return server.Shutdown(shutdownCtx)
    case err := <-errChan:
        return err
    }
}
```

## 停止意图

Stop 取消的是调用那一刻的运行会话。当 `stop` 与 `run` 命令几乎同时下发时，那次取消可能
**没有兑现对象**，而随后启动的 Run 会重建会话继续运行——停止语义就此丢失，且 Stop 早已
返回 `nil`，调用方无从察觉。

为此框架保留了一组**停止意图**状态（`stopRequested` / `stopCause`，受 `sm.mu` 保护）：

| Stop 发生时的状态 | 处置 |
|-------------------|------|
| 有服务在运行 | 取消运行上下文即为兑现，意图随即清除 |
| 无服务在运行 | 这次取消打在空处，把意图留存下来，等随后启动的 Run 消费 |

Run 在建立运行上下文的临界区内消费意图。命中时：

- 立即用该意图的原因取消运行上下文
- **不执行 Run 方法**——编写清理逻辑时不要假设 Run 一定被调用过
- 返回 `nil`

意图被消费后即清除，因此一次 Stop 只终止一次运行，不会影响之后的 restart。

> 为什么"有服务在跑"时反而要清除意图：否则 `restart`（stop 之后紧接 start）会被上一次的
> 停止意图吞掉，服务再也起不来。

## 超时控制

### 分级超时保护

分级预算**只在 CLI 前台 `run` 命令路径生效**（后台托管路径不走这段）：

```
外部取消 commandCtx（信号 / 父级 ctx 取消）
    ↓
等待 ShutdownInitial（默认 15s），给主服务优雅退出的时间
    ↓
[仍在运行] 才触发停止流程：取消运行上下文 + 执行用户 Stop()
    ↓
等待 ShutdownGrace（默认 5s）完成清理
    ↓
[仍未退出] 记录警告并返回 nil（命令层面放弃等待）
```

与之**并行**的是强制退出保护，它从首次 Stop 就开始计时，而非排在分级之后：

```
首次 Stop 发生 → scheduleForceExit(StopTimeout) 启动 20s 倒计时
    ↓
Run 正常返回 → defer cancelForceExit() 撤销倒计时
    ↓
[20s 到点仍未撤销] 后台模式直接强制退出
```

注意：

- `ShutdownInitial` / `ShutdownGrace` 显式传 0 **不会禁用**等待，而是回落为 3s + 2s
- `StopTimeout <= 0` 时强制退出保护**关闭**（`scheduleForceExit` 直接返回）
- 交互模式（`service.Interactive()`）下不排强制退出定时器

### 自定义超时

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

## 错误处理

### 结构化错误（英文化示例）

```go
func (s *MyService) Run(ctx context.Context) error {
    if err := s.initialize(); err != nil {
        return zcli.NewError(zcli.ErrServiceStart).
            Service(s.Name()).
            Operation("initialize").
            Message("failed to initialize resources").
            Cause(err).
            Build()
    }
    return nil
}
```

### 错误聚合

```go
func (s *MyService) Stop() error {
    aggr := zcli.NewErrorAggregator()

    if err := s.closeDatabase(); err != nil {
        aggr.Add(err)
    }
    if err := s.closeCache(); err != nil {
        aggr.Add(err)
    }

    if aggr.HasErrors() {
        return aggr.Error()
    }
    return nil
}
```

## 最佳实践

### 1. 始终监听 ctx.Done()

```go
for {
    select {
    case <-ctx.Done():
        return nil
    case work := <-workChan:
        process(work)
    }
}
```

### 2. 使用 WaitGroup 管理 Goroutines

```go
func (s *WorkerService) Run(ctx context.Context) error {
    var wg sync.WaitGroup

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            s.worker(id, ctx)
        }(i)
    }

    <-ctx.Done()
    wg.Wait()  // 等待所有工作完成
    return nil
}
```

### 3. 实现健康检查

```go
func (s *Service) Run(ctx context.Context) error {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        if s.healthy.Load() {
            w.WriteHeader(http.StatusOK)
            fmt.Fprintf(w, "OK")
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
    })

    go http.ListenAndServe(":8081", nil)

    // 主逻辑...
}
```

### 4. 合理设置超时

```go
// 快速服务
fastService := zcli.NewBuilder("zh").
    WithShutdownTimeouts(5*time.Second, 3*time.Second).
    Build()

// 慢服务（需要保存大量状态）
slowService := zcli.NewBuilder("zh").
    WithShutdownTimeouts(60*time.Second, 30*time.Second).
    Build()
```
