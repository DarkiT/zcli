// 完整示例：展示 zcli 全部能力的优雅调用方案。
//
// 涵盖：
//   - Builder 全量配置：Name / Logo / Version / GitInfo / Language / Debug / Mousetrap
//   - 服务配置：WorkDir / User / Executable / Arguments / ChRoot / EnvVars / Dependencies
//   - 服务接口：ServiceRunner + ServiceLifecycle → ManagedService
//   - 优雅关闭：ShutdownTimeouts / ServiceTimeouts / ShutdownCause
//   - 命令工厂：NewCommand + 全部 CommandOption
//   - 错误处理：ErrorBuilder / ErrorAggregator / LoggingErrorHandler / RecoveryErrorHandler
//   - 钩子系统：Validator / InitHook
//   - 标志导出：ExportFlagsForViper / GetSystemFlags
//   - 多语言：中文界面

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/darkit/zcli"
)

// ---------------------------------------------------------------------------
// Logo
// ---------------------------------------------------------------------------

const appLogo = `
 ██████╗ ██████╗ ███╗   ███╗██████╗ ██╗     ███████╗████████╗███████╗
██╔════╝██╔═══██╗████╗ ████║██╔══██╗██║     ██╔════╝╚══██╔══╝██╔════╝
██║     ██║   ██║██╔████╔██║██████╔╝██║     █████╗     ██║   █████╗
██║     ██║   ██║██║╚██╔╝██║██╔═══╝ ██║     ██╔══╝     ██║   ██╔══╝
╚██████╗╚██████╔╝██║ ╚═╝ ██║██║     ███████╗███████╗   ██║   ███████╗
 ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝     ╚══════╝╚══════╝   ╚═╝   ╚══════╝
`

// ---------------------------------------------------------------------------
// 领域模型
// ---------------------------------------------------------------------------

// Config 模拟外部配置源（文件 / 环境变量 / 远程中心）。
type Config struct {
	Profile   string
	DBHost    string
	DBPort    int
	RedisHost string
	RedisPort int
	WorkerNum int
	PollSecs  int
}

func loadConfig() *Config {
	return &Config{
		Profile:   "production",
		DBHost:    "localhost",
		DBPort:    5432,
		RedisHost: "localhost",
		RedisPort: 6379,
		WorkerNum: 4,
		PollSecs:  5,
	}
}

// Database 模拟数据库连接。
type Database struct{ dsn string }

func newDatabase(host string, port int) *Database {
	return &Database{dsn: fmt.Sprintf("postgres://%s:%d/app", host, port)}
}

func (db *Database) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (db *Database) Close() error { return nil }

// Cache 模拟缓存连接。
type Cache struct{ addr string }

func newCache(host string, port int) *Cache {
	return &Cache{addr: fmt.Sprintf("%s:%d", host, port)}
}

func (c *Cache) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (c *Cache) Close() error { return nil }

// ---------------------------------------------------------------------------
// 应用服务：集成 ServiceRunner + ServiceLifecycle
// ---------------------------------------------------------------------------

// AppService 是完整的生产级服务实现，同时满足 ServiceRunner 和 ServiceLifecycle。
type AppService struct {
	name  string
	cfg   *Config
	db    *Database
	cache *Cache

	stopCh   chan struct{}
	stopOnce sync.Once
	mu       sync.Mutex
}

func newAppService(cfg *Config) *AppService {
	return &AppService{
		name:   "complete-app",
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// --- ServiceRunner -----------------------------------------------------------

func (s *AppService) Name() string { return s.name }

func (s *AppService) Run(ctx context.Context) error {
	slog.Info("服务启动", "profile", s.cfg.Profile)

	// 检查关闭原因（可选，用于区分信号/服务停止/外部取消）
	if cause, ok := zcli.GetShutdownCause(ctx); ok {
		slog.Info("启动时检测到关闭原因", "reason", cause.Reason)
		return nil
	}

	ticker := time.NewTicker(time.Duration(s.cfg.PollSecs) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("收到停止信号，优雅退出中")

			// 区分停止原因做不同处理
			if cause, ok := zcli.GetShutdownCause(ctx); ok {
				switch cause.Reason {
				case zcli.ShutdownReasonSignal:
					slog.Info("收到系统信号", "signal", cause.Signal)
				case zcli.ShutdownReasonServiceStop:
					slog.Info("服务管理器请求停止")
				case zcli.ShutdownReasonExternalCancel:
					slog.Info("父 Context 被取消")
				}
			}
			return nil

		case <-s.stopCh:
			slog.Info("收到内部停止信号")
			return nil

		case <-ticker.C:
			s.processBatch()
		}
	}
}

func (s *AppService) Stop() error {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})

	// 错误聚合：尽可能关闭所有资源，收集所有错误
	aggr := zcli.NewErrorAggregator()

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			aggr.Add(zcli.NewError(zcli.ErrServiceStop).
				Service(s.name).
				Operation("db.close").
				Message("数据库关闭失败").
				Cause(err).
				Build())
		}
	}

	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			aggr.Add(zcli.NewError(zcli.ErrServiceStop).
				Service(s.name).
				Operation("cache.close").
				Message("缓存关闭失败").
				Cause(err).
				Build())
		}
	}

	slog.Info("服务清理完成")
	if aggr.HasErrors() {
		return zcli.CombineErrors(aggr.Errors()...)
	}
	return nil
}

func (s *AppService) processBatch() {
	slog.Info("处理业务批次", "workers", s.cfg.WorkerNum)
}

// --- ServiceLifecycle --------------------------------------------------------

func (s *AppService) BeforeStart() error {
	slog.Info("生命周期: BeforeStart — 初始化数据库和缓存连接")

	s.db = newDatabase(s.cfg.DBHost, s.cfg.DBPort)
	s.cache = newCache(s.cfg.RedisHost, s.cfg.RedisPort)

	// 结构化错误构建器演示
	if err := s.db.Ping(context.Background()); err != nil {
		return zcli.NewError(zcli.ErrConnection).
			Service(s.name).
			Operation("BeforeStart").
			Message("数据库连接测试失败").
			Context("host", s.cfg.DBHost).
			Context("port", s.cfg.DBPort).
			Cause(err).
			Build()
	}
	if err := s.cache.Ping(context.Background()); err != nil {
		return zcli.NewError(zcli.ErrConnection).
			Service(s.name).
			Operation("BeforeStart").
			Message("缓存连接测试失败").
			Context("host", s.cfg.RedisHost).
			Context("port", s.cfg.RedisPort).
			Cause(err).
			Build()
	}
	return nil
}

func (s *AppService) AfterStart() error {
	slog.Info("生命周期: AfterStart — 注册到服务发现")
	return nil
}

func (s *AppService) BeforeStop() error {
	slog.Info("生命周期: BeforeStop — 从服务发现注销")
	return nil
}

func (s *AppService) AfterStop() error {
	slog.Info("生命周期: AfterStop — 释放外部资源")
	return nil
}

// ---------------------------------------------------------------------------
// 自定义错误处理器
// ---------------------------------------------------------------------------

// slogAdapter 将 *slog.Logger 适配为 zcli.Logger 接口。
type slogAdapter struct{ logger *slog.Logger }

func (a slogAdapter) Error(msg string, fields ...any) { a.logger.Error(msg, fields...) }
func (a slogAdapter) Warn(msg string, fields ...any)  { a.logger.Warn(msg, fields...) }
func (a slogAdapter) Info(msg string, fields ...any)  { a.logger.Info(msg, fields...) }

// ---------------------------------------------------------------------------
// 子命令定义
// ---------------------------------------------------------------------------

func newInspectCommand(app *zcli.Cli) *zcli.Command {
	return zcli.NewCommand(
		"inspect",
		"查看应用配置和可导出标志",
		zcli.WithCommandLong("显示当前应用的基本信息、配置详情以及可导出给外部配置系统（如 Viper）的标志列表。"),
		zcli.WithCommandExample("  complete inspect\n  complete inspect --verbose"),
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
			cfg := app.Config()

			fmt.Fprintf(cmd.OutOrStdout(), "应用名称:   %s\n", cfg.Basic().Name)
			fmt.Fprintf(cmd.OutOrStdout(), "显示名称:   %s\n", cfg.Basic().DisplayName)
			fmt.Fprintf(cmd.OutOrStdout(), "版本号:     %s\n", cfg.Basic().Version)
			fmt.Fprintf(cmd.OutOrStdout(), "语言:       %s\n", cfg.Basic().Language)
			fmt.Fprintf(cmd.OutOrStdout(), "工作目录:   %s\n", cfg.Service().WorkDir)
			fmt.Fprintln(cmd.OutOrStdout())

			// 标志导出演示
			sysFlags := app.GetSystemFlags()
			bindable := app.GetFilteredFlagNames()
			fmt.Fprintf(cmd.OutOrStdout(), "系统标志 (%d): %v\n", len(sysFlags), sysFlags)
			fmt.Fprintf(cmd.OutOrStdout(), "可绑定标志:    %v\n", bindable)
			return nil
		}),
		zcli.WithCommandFlags(func(flags *zcli.FlagSet) {
			flags.Bool("verbose", false, "显示详细信息")
		}),
	)
}

func newHealthCommand() *zcli.Command {
	return zcli.NewCommand(
		"health",
		"健康检查",
		zcli.WithCommandLong("执行数据库和缓存的连通性检查。"),
		zcli.WithCommandAliases("hc", "ping"),
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
			cfg := loadConfig()
			db := newDatabase(cfg.DBHost, cfg.DBPort)
			cache := newCache(cfg.RedisHost, cfg.RedisPort)

			aggr := zcli.NewErrorAggregator()

			if err := db.Ping(context.Background()); err != nil {
				aggr.Add(fmt.Errorf("数据库不可达: %w", err))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "数据库   OK")
			}

			if err := cache.Ping(context.Background()); err != nil {
				aggr.Add(fmt.Errorf("缓存不可达: %w", err))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "缓存     OK")
			}

			if aggr.HasErrors() {
				return zcli.CombineErrors(aggr.Errors()...)
			}
			return nil
		}),
	)
}

func newConfigCommand() *zcli.Command {
	cmd := zcli.NewCommand(
		"config",
		"管理应用配置",
		zcli.WithCommandLong("查看、验证和导出应用配置。"),
		zcli.WithCommandArgs(zcli.ExactArgs(1)),
	)

	showCmd := zcli.NewCommand(
		"show",
		"显示当前配置",
		zcli.WithCommandRun(func(cmd *zcli.Command, args []string) {
			cfg := loadConfig()
			fmt.Fprintf(cmd.OutOrStdout(), "Profile:    %s\n", cfg.Profile)
			fmt.Fprintf(cmd.OutOrStdout(), "DB:         %s:%d\n", cfg.DBHost, cfg.DBPort)
			fmt.Fprintf(cmd.OutOrStdout(), "Redis:      %s:%d\n", cfg.RedisHost, cfg.RedisPort)
			fmt.Fprintf(cmd.OutOrStdout(), "Workers:    %d\n", cfg.WorkerNum)
			fmt.Fprintf(cmd.OutOrStdout(), "Poll:       %ds\n", cfg.PollSecs)
		}),
	)

	validateCmd := zcli.NewCommand(
		"validate",
		"验证配置有效性",
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
			cfg := loadConfig()
			if cfg.DBHost == "" {
				return fmt.Errorf("DBHost 不能为空")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "配置验证通过")
			return nil
		}),
	)

	cmd.AddCommand(showCmd, validateCmd)
	return cmd
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// --- 构建服务 ---

	cfg := loadConfig()
	svc := newAppService(cfg)
	managed := zcli.NewManagedService(svc, svc)

	workDir, _ := os.Getwd()
	execPath, _ := os.Executable()

	// --- Builder：全量配置 ---

	app, err := zcli.NewBuilder("zh"). // 中文界面
						WithName(svc.Name()).
						WithDisplayName("Complete App").
						WithDescription("zcli 全能力演示应用 — 展示 Builder、ServiceRunner、生命周期、标志导出、错误处理、多语言等全部特性。").
						WithVersion("3.0.0").
						WithLogo(appLogo). // ASCII Logo

		// Git 信息（通常由 CI 注入）
		WithGitInfo("a1b2c3d", "main", "v3.0.0").
		WithBuildTime("2026-06-15 10:30:00").
		WithDebug(false).

		// 服务基础配置
		WithServiceRunner(managed).
		WithWorkDir(workDir).
		WithEnvVar("APP_PROFILE", cfg.Profile).
		WithEnvVar("GOMAXPROCS", "4").

		// 高级服务配置（install 时生效）
		WithServiceUser("appuser").
		WithExecutable(execPath).
		WithArguments("run", "--profile", cfg.Profile).

		// 结构化依赖
		WithDependency("network-online.target", zcli.DependencyAfter).
		WithStructuredDependencies(
			zcli.Dependency{Name: "postgresql.service", Type: zcli.DependencyRequire},
			zcli.Dependency{Name: "redis.service", Type: zcli.DependencyWant},
		).

		// 平台选项
		WithServiceOption("Restart", "on-failure").
		WithServiceOption("RestartSec", 10).
		WithAllowSudoFallback(true).

		// 超时控制
		WithShutdownTimeouts(15*time.Second, 5*time.Second).
		WithServiceTimeouts(30*time.Second, 20*time.Second).

		// Windows 双击提示
		WithMousetrapDisabled(true).

		// 配置验证
		WithValidator(func(cfg *zcli.Config) error {
			if cfg.Basic().Name == "" {
				return fmt.Errorf("应用名称不能为空")
			}
			if cfg.Service().EnvVars["APP_PROFILE"] == "" {
				return fmt.Errorf("APP_PROFILE 不能为空")
			}
			return nil
		}).

		// 初始化钩子（命令执行前）
		WithInitHook(func(cmd *zcli.Command, args []string) error {
			logger.Info(
				"InitHook 触发",
				"command", cmd.CommandPath(),
				"args", args,
			)
			return nil
		}).

		// 错误处理器链
		WithErrorHandler(zcli.NewLoggingErrorHandler(slogAdapter{logger})).
		WithErrorHandler(zcli.NewRecoveryErrorHandler(3, 2*time.Second)).

		// 延迟收集命令
		WithCommand(newInspectCommand(nil)).
		WithCommand(newHealthCommand()).
		WithCommand(newConfigCommand()).

		// 构建
		BuildWithError()
	if err != nil {
		logger.Error("构建失败", "error", err)
		os.Exit(1)
	}

	// --- 设置全局标志 ---

	app.PersistentFlags().String("profile", cfg.Profile, "运行配置名称")
	app.PersistentFlags().Bool("debug", false, "启用调试日志")

	// --- 补注：inspect 命令需要 app 引用，用标记后的版本替换 ---
	// （Builder 阶段 app 尚未存在，build 后替换）
	if inspectCmd, _, _ := app.Command().Find([]string{"inspect"}); inspectCmd != nil {
		app.RemoveCommand(inspectCmd)
		app.AddCommand(newInspectCommand(app))
	}

	// --- 执行 ---

	if err := app.Execute(); err != nil {
		logger.Error("命令执行失败", "error", err)

		// 结构化错误检查
		if svcErr, ok := zcli.GetServiceError(err); ok {
			logger.Error(
				"服务错误详情",
				"code", svcErr.Code,
				"service", svcErr.Service,
				"operation", svcErr.Operation,
				"message", svcErr.Message,
			)
		}
		os.Exit(1)
	}
}
