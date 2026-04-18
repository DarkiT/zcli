package zcli

import (
	"context"
	"fmt"
	"io"
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

type sManager struct {
	commands       *Cli
	localizer      *ServiceLocalizer
	mu             sync.RWMutex
	session        *serviceRunSession
	config         *service.Config
	service        service.Service
	exitChan       chan struct{}
	running        atomic.Bool
	stopExecuted   atomic.Bool
	stopFuncOnce   atomic.Bool
	forceExitOnce  atomic.Bool
	forceExitTimer *time.Timer
	errorHandlers  []ErrorHandler
	stopMu         sync.Mutex
	runnerDone     chan struct{}
	runnerErr      chan error
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
	if _, ok := config.Option["RunWait"]; !ok {
		signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
		if serviceManagerGOOS != "windows" {
			signals = append(signals, syscall.SIGQUIT)
		}
		config.Option["RunWait"] = func() {
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

	if sm.commands.config.runtime.StartTimeout > 0 {
		config.Timeout.Start = sm.commands.config.runtime.StartTimeout
	}
	if sm.commands.config.runtime.StopTimeout > 0 {
		config.Timeout.Stop = sm.commands.config.runtime.StopTimeout
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
	}
	if svcCfg.Arguments != nil {
		config.Arguments = append([]string(nil), svcCfg.Arguments...)
	}
	if svcCfg.Options != nil {
		config.Option = make(service.KeyValue, len(svcCfg.Options))
		for key, value := range svcCfg.Options {
			config.Option[key] = value
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

	if serviceManagerGOOS != "windows" {
		if err := checkPermissions(config.Executable, 0o755, sm.localizer); err != nil {
			return nil, fmt.Errorf("%s", sm.localizer.FormatError("execPermission", config.Executable, err))
		}
		if config.WorkingDirectory != "" {
			if err := checkPermissions(config.WorkingDirectory, os.ModeDir|0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf("%s", sm.localizer.FormatError("workDirPermission", config.WorkingDirectory, err))
			}
		}
		if config.ChRoot != "" {
			if err := checkPermissions(config.ChRoot, os.ModeDir|0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf("%s", sm.localizer.FormatError("chrootPermission", config.ChRoot, err))
			}
		}
	}

	return config, nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
