package zcli

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	service "github.com/darkit/daemon"
)

var (
	exitFunc           = os.Exit
	serviceManagerGOOS = runtime.GOOS
	executablePath     = os.Executable
)

// serviceRunSession 描述单次服务命令执行的上下文边界：
// commandCtx 负责整次 run/start/stop 命令生命周期，
// serviceCtx 只负责传给用户 Run(ctx) 的服务运行生命周期。
type serviceRunSession struct {
	commandCtx    context.Context
	commandCancel context.CancelCauseFunc
	serviceCtx    context.Context
	serviceCancel context.CancelCauseFunc
}

// sManager 的共享状态由 sm.mu 统一保护：session、exitChan、service、
// runnerDone/runnerErr、forceExitTimer。stopMu 只串行化停止流程本身，
// 它不拥有任何字段——需要读 session 时必须在 stopMu 内再取 mu 快照。
//
// errorHandlers 由 handlersMu 单独保护：handleError 会在错误路径遍历它，
// 与 mu 解耦可避免锁嵌套。
type sManager struct {
	commands      *Cli
	localizer     *ServiceLocalizer
	mu            sync.RWMutex
	session       *serviceRunSession
	config        *service.Config
	service       service.Service
	exitChan      chan struct{}
	running       atomic.Bool
	stopExecuted  atomic.Bool
	stopFuncOnce  atomic.Bool
	forceExitOnce atomic.Bool
	// stopRequested / stopCause 承载「尚未兑现的停止意图」，归 mu 保护。
	//
	// 存在理由：Stop 取消的是调用那一刻的 session。若此时还没有运行中的服务
	// （例如 stop 与 run 命令几乎同时下发），这次取消就没有兑现对象，而随后
	// 启动的 Run 会重建 session 继续跑——停止语义就此丢失，且 Stop 仍返回 nil。
	// 保留意图后，Run 在启动临界区内消费它并立即停止。
	//
	// 反过来，若 Stop 时确有服务在运行，意图立即通过 cancel 兑现，标志随之
	// 清除——否则 restart（stop 后紧接 start）会被上一次的停止意图吞掉。
	stopRequested  bool
	stopCause      error
	forceExitTimer *time.Timer
	errorHandlers  []ErrorHandler
	handlersMu     sync.RWMutex
	stopMu         sync.Mutex
	runnerDone     chan struct{}
	runnerErr      chan error
}

// currentService 返回当前 service 实例快照，供命令路径在锁外使用。
func (sm *sManager) currentService() service.Service {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.service
}

// setService 替换当前 service 实例。
func (sm *sManager) setService(svc service.Service) {
	sm.mu.Lock()
	sm.service = svc
	sm.mu.Unlock()
}

// newServiceAssemblyManager 为 Cli 装配 service 能力。
// 它只负责把当前 CLI 的配置转换成 sManager，不负责命令注入或 root 运行策略。
func (c *Cli) newServiceAssemblyManager() (*sManager, error) {
	// 使用外部上下文，信号处理由 service.RunWait 负责
	ctx, cancel := context.WithCancel(c.config.Context())
	return newServiceManager(c, ctx, cancel)
}

// newServiceManager 创建服务管理器实例
func newServiceManager(cmd *Cli, ctx context.Context, cancel context.CancelFunc) (*sManager, error) {
	langManager := NewLanguageManager("en")
	if cmd != nil && cmd.config != nil && cmd.config.basic != nil && cmd.config.basic.Language != "" {
		if cmd.lang != nil && cmd.lang.Code == cmd.config.basic.Language {
			langManager = NewScopedLanguageManager(cmd.lang)
		} else {
			langManager = NewLanguageManager(cmd.config.basic.Language)
		}
	} else if cmd != nil && cmd.lang != nil {
		langManager = NewScopedLanguageManager(cmd.lang)
	}

	localizer := NewServiceLocalizer(langManager, cmd.colors)
	out := io.Writer(os.Stdout)
	errOut := io.Writer(os.Stderr)
	if cmd.command != nil {
		out = cmd.command.OutOrStdout()
		errOut = cmd.command.ErrOrStderr()
	}
	localizer.ConfigureOutput(out, errOut, cmd.config.basic.SilenceErrors, cmd.config.basic.SilenceUsage)

	sessionCtx := ctx
	if sessionCtx == nil {
		sessionCtx = context.Background()
	}
	commandCtx, commandCancel := context.WithCancelCause(sessionCtx)

	sm := &sManager{
		commands:      cmd,
		localizer:     localizer,
		session:       &serviceRunSession{commandCtx: commandCtx, commandCancel: commandCancel},
		exitChan:      make(chan struct{}),
		errorHandlers: cmd.config.runtime.ErrorHandlers,
	}

	sm.stopExecuted.Store(false)
	sm.forceExitOnce.Store(false)

	config, err := sm.createServiceConfig()
	if err != nil {
		cancel()
		return nil, fmt.Errorf(localizer.FormatError("createConfig")+": %v", err)
	}

	if config.Option == nil {
		config.Option = make(service.KeyValue)
	}
	if _, ok := config.Option[service.OptionRunWait]; !ok {
		signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
		if serviceManagerGOOS != "windows" {
			signals = append(signals, syscall.SIGQUIT)
		}
		config.Option[service.OptionRunWait] = func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, signals...)
			defer signal.Stop(sigCh)
			ctx := sm.getCtx()
			if ctx == nil {
				ctx = context.Background()
			}
			select {
			case sig := <-sigCh:
				_ = sm.stopWithCause(newShutdownCause(ShutdownReasonSignal, sig, nil), true)
			case <-ctx.Done():
			}
		}
	}

	sm.config = config
	svcRunner := sm.buildRunner()
	svc, err := service.New(svcRunner, config)
	if err != nil {
		cancel()
		return nil, WrapError(err, ErrServiceCreate, "create")
	}
	sm.service = svc

	return sm, nil
}

// buildRunner 构造符合 daemon 的 ServiceRunner
func (sm *sManager) buildRunner() *service.ServiceRunner {
	runner := &service.ServiceRunner{
		StartFunc: func(context.Context) error {
			return sm.startManagedRun()
		},
		StopFunc: func(ctx context.Context) error {
			return sm.stopManagedRun(ctx)
		},
	}

	if sm.commands.config.runtime.StartTimeout > 0 {
		runner.StartTimeout = sm.commands.config.runtime.StartTimeout
	}
	if sm.commands.config.runtime.StopTimeout > 0 {
		runner.StopTimeout = sm.commands.config.runtime.StopTimeout
	}

	return runner
}

// getCtx 返回当前命令级运行上下文。
func (sm *sManager) getCtx() context.Context {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.session == nil {
		return nil
	}
	return sm.session.commandCtx
}

func (sm *sManager) newCommandSessionLocked() *serviceRunSession {
	if sm.session != nil {
		if sm.session.serviceCancel != nil {
			sm.session.serviceCancel(newShutdownCause(ShutdownReasonServiceStop, nil, nil))
		}
		if sm.session.commandCancel != nil {
			sm.session.commandCancel(newShutdownCause(ShutdownReasonServiceStop, nil, nil))
		}
	}

	baseCtx := sm.commands.config.Context()
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	sm.session = &serviceRunSession{}
	sm.session.commandCtx, sm.session.commandCancel = context.WithCancelCause(baseCtx)
	return sm.session
}

func (sm *sManager) ensureCommandSessionLocked() *serviceRunSession {
	if sm.session == nil || sm.session.commandCtx == nil || sm.session.commandCtx.Err() != nil {
		return sm.newCommandSessionLocked()
	}
	return sm.session
}

func (sm *sManager) clearServiceContext(session *serviceRunSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.session != session {
		return
	}
	session.serviceCtx = nil
	session.serviceCancel = nil
}

// consumeStopIntentLocked 取出并清除尚未兑现的停止意图。
// 调用方必须持有 sm.mu。
// 返回 true 表示本次运行应当立即停止，不再执行用户 RunFunc。
func (sm *sManager) consumeStopIntentLocked() (error, bool) {
	if !sm.stopRequested {
		return nil, false
	}
	cause := sm.stopCause
	sm.stopRequested = false
	sm.stopCause = nil
	if cause == nil {
		cause = newShutdownCause(ShutdownReasonServiceStop, nil, nil)
	}
	return cause, true
}

// Name 返回服务名称
func (sm *sManager) Name() string {
	return sm.commands.config.basic.Name
}

// createServiceConfig 创建服务配置
func (sm *sManager) createServiceConfig() (*service.Config, error) {
	svcCfg := cloneService(sm.commands.config.service)
	displayName := sm.commands.config.basic.DisplayName
	if displayName == "" {
		displayName = sm.commands.config.basic.Name
	}

	config := &service.Config{
		Name:              sm.commands.config.basic.Name,
		DisplayName:       displayName,
		Description:       sm.commands.config.basic.Description,
		UserName:          svcCfg.Username,
		Arguments:         []string{"run"},
		Dependencies:      append([]string(nil), svcCfg.Dependencies...),
		StructuredDeps:    append([]service.Dependency(nil), svcCfg.StructuredDeps...),
		ChRoot:            svcCfg.ChRoot,
		EnvVars:           cloneStringMap(svcCfg.EnvVars),
		AllowSudoFallback: svcCfg.AllowSudoFallback,
		Force:             svcCfg.Force,
	}
	if svcCfg.Arguments != nil {
		config.Arguments = append([]string(nil), svcCfg.Arguments...)
	}
	if svcCfg.Options != nil {
		config.Option = make(service.KeyValue, len(svcCfg.Options))
		maps.Copy(config.Option, svcCfg.Options)
	}
	if svcCfg.Notify {
		if config.Option == nil {
			config.Option = make(service.KeyValue)
		}
		config.Option[service.OptionNotify] = true
	}
	if rt := sm.commands.config.runtime; rt != nil {
		config.Timeout = service.TimeoutConfig{
			Start:   rt.StartTimeout,
			Restart: rt.RestartTimeout,
		}
		if rt.daemonStopTimeoutSet {
			config.Timeout.Stop = rt.StopTimeout
		}
	}

	execPath := svcCfg.Executable
	if execPath == "" {
		var err error
		execPath, err = executablePath()
		if err != nil {
			return nil, fmt.Errorf("%s", sm.localizer.FormatError("getExecPath", err))
		}
	}
	config.Executable = execPath

	workDir := svcCfg.WorkDir
	if workDir == "" && execPath != "" {
		workDir = filepath.Dir(execPath)
	}
	config.WorkingDirectory = workDir

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (sm *sManager) verifyInstallPaths() error {
	if sm == nil || sm.config == nil || serviceManagerGOOS == "windows" {
		return nil
	}
	if err := checkPermissions(sm.config.Executable, 0o555, sm.localizer); err != nil {
		return fmt.Errorf("%s", sm.localizer.FormatError("execPermission", sm.config.Executable, err))
	}
	if sm.config.WorkingDirectory != "" {
		if err := checkPermissions(sm.config.WorkingDirectory, os.ModeDir|0o500, sm.localizer); err != nil {
			return fmt.Errorf("%s", sm.localizer.FormatError("workDirPermission", sm.config.WorkingDirectory, err))
		}
	}
	if sm.config.ChRoot != "" {
		if err := checkPermissions(sm.config.ChRoot, os.ModeDir|0o500, sm.localizer); err != nil {
			return fmt.Errorf("%s", sm.localizer.FormatError("chrootPermission", sm.config.ChRoot, err))
		}
	}
	return nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}
