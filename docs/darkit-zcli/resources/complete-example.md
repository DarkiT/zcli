# 完整 CLI 应用示例

## 概述

本指南提供一个生产级的完整 CLI 应用示例，展示 ZCli 框架的所有核心功能。

## 项目结构

```
myapp/
├── main.go              # 应用入口
├── service.go           # 服务实现
├── lifecycle.go         # 生命周期管理
├── commands.go          # 自定义命令
├── config.yaml          # 配置文件
└── go.mod
```

## 完整源代码

### main.go

```go
package main

import (
    "fmt"
    "log"
    "log/slog"
    "os"
    "runtime"
    "time"

    "github.com/darkit/zcli"
    "github.com/spf13/viper"
)

var (
    // 编译时注入
    Version   = "dev"
    GitCommit = "unknown"
    BuildTime = "unknown"
)

const logo = `
 __  __          _
|  \/  |_   _   / \   _ __  _ __
| |\/| | | | | / _ \ | '_ \| '_ \
| |  | | |_| |/ ___ \| |_) | |_) |
|_|  |_|\__, /_/   \_\ .__/| .__/
        |___/        |_|   |_|
`

func main() {
    // 加载配置
    cfg, err := loadConfig()
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    // 创建服务
    svc := NewService(cfg)

    // 创建生命周期管理器
    lifecycle := NewLifecycle(cfg, svc)

    // 创建 CLI 应用
    app, err := zcli.NewBuilder("zh").
        WithName("myapp").
        WithDisplayName("我的企业级应用").
        WithDescription("提供企业级功能的后台服务").
        WithVersion(Version).
        WithLogo(logo).
        WithGitInfo(GitCommit, "main", "").
        WithServiceRunner(svc).
        // 注意：使用 getter 方法访问 Config 字段
        WithValidator(validateConfig).
        BuildWithError()

    if err != nil {
        slog.Error("创建应用失败", "error", err)
        os.Exit(1)
    }

    // 添加配置标志
    setupFlags(app)

    // 绑定标志到 Viper
    flags := app.ExportFlagsForViper()
    for _, fs := range flags {
        if err := viper.BindPFlags(fs); err != nil {
            log.Fatal(err)
        }
    }

    // 添加自定义命令
    addCustomCommands(app, cfg)

    // 执行应用
    if err := app.Execute(); err != nil {
        slog.Error("应用执行失败", "error", err)
        os.Exit(1)
    }
}

func loadConfig() (*AppConfig, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./configs")

    viper.SetEnvPrefix("MYAPP")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
        slog.Info("未找到配置文件，使用默认配置")
    }

    var cfg AppConfig
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("解析配置失败: %w", err)
    }

    return &cfg, nil
}

func setupFlags(app *zcli.Cli) {
    app.PersistentFlags().String("server.host", "0.0.0.0", "服务器地址")
    app.PersistentFlags().Int("server.port", 8080, "服务器端口")
    app.PersistentFlags().String("database.host", "localhost", "数据库主机")
    app.PersistentFlags().String("logging.level", "info", "日志级别")
}

// validateConfig 配置验证函数
// 重要：必须使用 getter 方法访问 Config 字段
func validateConfig(cfg *zcli.Config) error {
    // 正确：使用 cfg.Basic() 而非 cfg.Basic
    if cfg.Basic().Name == "" {
        return fmt.Errorf("应用名称不能为空")
    }
    if cfg.Basic().Version == "" {
        return fmt.Errorf("版本号不能为空")
    }
    return nil
}
```

### service.go

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"
)

type Service struct {
    config *AppConfig
    server *http.Server
    db     *Database
    stopCh chan struct{}
}

func NewService(cfg *AppConfig) *Service {
    return &Service{
        config: cfg,
        stopCh: make(chan struct{}),
    }
}

func (s *Service) Run(ctx context.Context) error {
    slog.Info("服务启动",
        "host", s.config.Server.Host,
        "port", s.config.Server.Port)

    // 配置 HTTP 服务器
    mux := http.NewServeMux()
    mux.HandleFunc("/health", s.healthHandler)
    mux.HandleFunc("/api/status", s.statusHandler)

    s.server = &http.Server{
        Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    // 启动服务器
    errChan := make(chan error, 1)
    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errChan <- err
        }
    }()

    slog.Info("HTTP 服务器已启动", "addr", s.server.Addr)

    // 等待关闭信号
    select {
    case <-ctx.Done():
        slog.Info("收到关闭信号")
        return s.shutdown()
    case <-s.stopCh:
        return nil
    case err := <-errChan:
        return err
    }
}

func (s *Service) Stop() error {
    slog.Info("停止服务")
    close(s.stopCh)
    return nil
}

func (s *Service) Name() string {
    return "myapp-service"
}

func (s *Service) shutdown() error {
    slog.Info("开始优雅关闭")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := s.server.Shutdown(ctx); err != nil {
        slog.Error("服务器关闭失败", "error", err)
        return err
    }

    slog.Info("HTTP 服务器已关闭")
    return nil
}

func (s *Service) SetDatabase(db *Database) {
    s.db = db
}

func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK")
}

func (s *Service) statusHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"status":"running","version":"%s"}`, s.config.Version)
}

// AppConfig 应用配置
type AppConfig struct {
    Version  string       `mapstructure:"version"`
    Server   ServerConfig `mapstructure:"server"`
    Database DBConfig     `mapstructure:"database"`
    Logging  LogConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}

type DBConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Name     string `mapstructure:"name"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}

type Database struct {
    // 数据库连接
}

func (d *Database) Close() error {
    return nil
}
```

### lifecycle.go

```go
package main

import (
    "fmt"
    "log/slog"
)

type Lifecycle struct {
    config  *AppConfig
    service *Service
    db      *Database
}

func NewLifecycle(cfg *AppConfig, svc *Service) *Lifecycle {
    return &Lifecycle{
        config:  cfg,
        service: svc,
    }
}

func (l *Lifecycle) BeforeStart() error {
    slog.Info("初始化资源...")

    // 连接数据库
    db, err := connectDatabase(l.config)
    if err != nil {
        return fmt.Errorf("连接数据库失败: %w", err)
    }
    l.db = db
    l.service.SetDatabase(db)

    slog.Info("数据库连接成功",
        "host", l.config.Database.Host,
        "port", l.config.Database.Port)

    return nil
}

func (l *Lifecycle) AfterStart() error {
    slog.Info("服务启动后处理...")
    // 注册到服务发现等
    return nil
}

func (l *Lifecycle) BeforeStop() error {
    slog.Info("准备停止服务...")
    return nil
}

func (l *Lifecycle) AfterStop() error {
    slog.Info("释放资源...")

    if l.db != nil {
        if err := l.db.Close(); err != nil {
            slog.Error("关闭数据库失败", "error", err)
        } else {
            slog.Info("数据库已关闭")
        }
    }

    return nil
}

func connectDatabase(cfg *AppConfig) (*Database, error) {
    // 实际的数据库连接逻辑
    return &Database{}, nil
}
```

### commands.go

```go
package main

import (
    "fmt"

    "github.com/darkit/zcli"
    "github.com/spf13/cobra"
)

func addCustomCommands(app *zcli.Cli, cfg *AppConfig) {
    // 配置命令
    configCmd := &cobra.Command{
        Use:   "config",
        Short: "配置管理",
    }

    configCmd.AddCommand(
        &cobra.Command{
            Use:   "show",
            Short: "显示当前配置",
            Run: func(cmd *cobra.Command, args []string) {
                fmt.Printf("服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
                fmt.Printf("数据库: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
                fmt.Printf("日志级别: %s\n", cfg.Logging.Level)
            },
        },
    )

    // 数据库命令
    dbCmd := &cobra.Command{
        Use:   "db",
        Short: "数据库管理",
    }

    dbCmd.AddCommand(
        &cobra.Command{
            Use:   "migrate",
            Short: "运行数据库迁移",
            RunE: func(cmd *cobra.Command, args []string) error {
                fmt.Println("运行数据库迁移...")
                return nil
            },
        },
        &cobra.Command{
            Use:   "reset",
            Short: "重置数据库",
            RunE: func(cmd *cobra.Command, args []string) error {
                fmt.Println("重置数据库...")
                return nil
            },
        },
    )

    app.AddCommand(configCmd, dbCmd)
}
```

### config.yaml

```yaml
version: "1.0.0"

server:
  host: 0.0.0.0
  port: 8080

database:
  host: localhost
  port: 5432
  name: myapp
  user: postgres
  password: password

logging:
  level: info
  format: json
```

## 使用示例

### 基本使用

```bash
# 查看帮助
$ ./myapp --help

# 查看版本
$ ./myapp --version

# 前台运行服务
$ ./myapp run

# 使用自定义配置
$ ./myapp run --config=./myconfig.yaml
```

### 配置管理

```bash
# 显示配置
$ ./myapp config show
```

### 数据库管理

```bash
# 运行迁移
$ ./myapp db migrate

# 重置数据库
$ ./myapp db reset
```

### 系统服务管理

```bash
# 安装为系统服务
$ sudo ./myapp install

# 启动服务
$ sudo ./myapp start

# 停止服务
$ sudo ./myapp stop

# 重启服务
$ sudo ./myapp restart

# 查看状态
$ ./myapp status

# 卸载服务
$ sudo ./myapp uninstall
```

### 环境变量配置

```bash
# 使用环境变量覆盖配置
$ export MYAPP_SERVER_PORT=9090
$ export MYAPP_DATABASE_HOST=prod-db.example.com
$ export MYAPP_LOGGING_LEVEL=debug

$ ./myapp run
```

## 构建脚本

### Makefile

```makefile
.PHONY: build run install clean

APP_NAME=myapp
VERSION=$(shell git describe --tags --always --dirty)
GIT_COMMIT=$(shell git rev-parse HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS=-ldflags "\
    -X main.Version=$(VERSION) \
    -X main.GitCommit=$(GIT_COMMIT) \
    -X main.BuildTime=$(BUILD_TIME)"

build:
	@echo "构建 $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "构建完成: bin/$(APP_NAME)"

run:
	go run .

install: build
	sudo cp bin/$(APP_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(APP_NAME)

clean:
	rm -rf bin/

test:
	go test -v ./...
```

## 关键点总结

### Config 字段访问

在 `WithValidator` 中访问 Config 字段时，**必须使用 getter 方法**：

```go
// 正确
func validateConfig(cfg *zcli.Config) error {
    if cfg.Basic().Name == "" {
        return fmt.Errorf("应用名称不能为空")
    }
    return nil
}

// 错误 - 编译失败
func validateConfig(cfg *zcli.Config) error {
    if cfg.Basic.Name == "" {  // cfg.Basic 是私有字段
        return fmt.Errorf("应用名称不能为空")
    }
    return nil
}
```

### Context 使用

在 `Run` 方法中监听 `ctx.Done()` 实现优雅关闭：

```go
func (s *Service) Run(ctx context.Context) error {
    select {
    case <-ctx.Done():  // 框架在收到 SIGINT/SIGTERM 时触发
        return s.shutdown()
    case <-s.stopCh:
        return nil
    }
}
```

### ServiceRunner 接口

实现三个方法：

```go
type ServiceRunner interface {
    Run(ctx context.Context) error  // 运行服务
    Stop() error                    // 停止服务
    Name() string                   // 返回服务名称
}
```
