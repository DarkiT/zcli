# 服务管理和并发控制指南

> 最近更新（2025-12-11）：✅ 统一迁移至 `ServiceRunner`，移除 `ConcurrentServiceManager`；✅ 信号处理改用 `RunWait` + `sm.Stop()`；✅ 错误消息示例全部英文化；⚠️ 请确保示例运行在 Go 1.23+。

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
- **并发安全**：内部通过 `getCtx()` 只读访问共享 Context，避免竞态

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

## 状态管理

### ServiceState 状态定义

```go
type ServiceState int32

const (
    StateStopped  ServiceState = iota  // 0: 已停止
    StateStarting                      // 1: 启动中
    StateRunning                       // 2: 运行中
    StateStopping                      // 3: 停止中
    StateError                         // 4: 错误状态
)
```

### 状态转换规则

```
StateStopped(已停止)
    ↓ Start()
StateStarting(启动中)
    ↓ 启动成功
StateRunning(运行中)
    ↓ Stop()
StateStopping(停止中)
    ↓ 停止完成
StateStopped(已停止)

任何状态 --错误--> StateError(错误)
```

### 状态监听器

```go
manager.AddStateListener(func(oldState, newState zcli.ServiceState) {
    slog.Info("状态变更",
        "old", oldState,
        "new", newState)

    switch newState {
    case zcli.StateRunning:
        sendNotification("服务已启动")
    case zcli.StateError:
        triggerAlert("服务进入错误状态")
    case zcli.StateStopped:
        slog.Info("服务已完全停止")
    }
})
```

## 信号处理（RunWait 机制）

### 默认流程

```
Run() → 内部创建 runCtx（可取消）
    ↓
RunWait() 监听 SIGINT/SIGTERM/SIGQUIT → 捕获后调用 sm.Stop()
    ↓
Stop() 优雅清理 → 返回后 RunWait 退出
```

### 默认处理的信号

| 信号 | 触发方式 | 行为 |
|------|----------|------|
| SIGINT | Ctrl+C | RunWait 触发停止流程 |
| SIGTERM | kill PID | RunWait 触发停止流程 |
| SIGQUIT | Ctrl+\ | RunWait 触发停止流程 |

> 兜底超时：15s 现在封装在 `Run()` 内部，无需手动处理；如需自定义请调用 `WithShutdownTimeouts`。

### 自定义信号处理（在业务中追加，而非替换框架监听）

```go
func runService(ctx context.Context) error {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGUSR1)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
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
1. 收到关闭信号
2. 停止接受新请求
3. 等待现有请求完成
4. 保存应用状态
5. 关闭数据库连接
6. 关闭其他资源
7. 退出程序
```

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

## 超时控制

### 分级超时保护

默认提供 15+5 秒分级关闭预算（可配置）：

```
RunWait 捕获信号 → sm.Stop()
    ↓
先取消 Run(ctx)，让主服务开始优雅退出
    ↓
[超时] 等待 15 秒后进入最终清理
    ↓
[超时] 再等待 5 秒（清理期）
    ↓
[超时] 强制终止进程（总计不超过 15s）
```

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
        return ctx.Err()
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
