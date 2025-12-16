package zcli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	service "github.com/darkit/daemon"
	"github.com/spf13/cobra"
)

// initService 初始化服务
func (c *Cli) initService() {
	// 使用外部上下文，信号处理由 service.RunWait 负责
	ctx, cancel := context.WithCancel(c.config.ctx)

	// 检查是否设置了服务运行函数
	if c.config.runtime.Run == nil {
		cancel()
		return
	}

	// 创建服务管理器
	sm, err := newServiceManager(c, ctx, cancel)
	if err != nil {
		_, _ = c.colors.Error.Printf("%v\n", err)
		return
	}

	// 设置信号处理
	go c.setupSignalHandler(sm)

	// 添加服务命令并配置根命令运行函数
	c.addServiceCommands(sm)
}

// setupSignalHandler 设置信号处理器
func (c *Cli) setupSignalHandler(sm *sManager) {
	<-c.config.ctx.Done()

	// 15 秒强制退出兜底
	if sm == nil {
		return
	}
	timeoutMsg := sm.localizer.FormatError("timeout", 15)
	sm.ExitWithTimeout(15*time.Second, timeoutMsg, 1)
}

// executeStopFunctions 执行所有已注册的停止函数

// addServiceCommands 添加服务管理命令
func (c *Cli) addServiceCommands(sm *sManager) {
	// 向命令行应用添加服务管理命令
	c.command.AddCommand(
		sm.newRunCmd(),
		sm.newInstallCmd(),
		sm.newUninstallCmd(),
		sm.newStartCmd(),
		sm.newStopCmd(),
		sm.newRestartCmd(),
		sm.newStatusCmd(),
	)

	// 设置根命令的运行函数，处理直接运行的情况
	originalRun := c.command.Run
	c.command.Run = func(cmd *cobra.Command, args []string) {
		if originalRun != nil {
			originalRun(cmd, args)
		} else {
			// 如果没有设置Run函数，则默认执行run命令
			if err := sm.executeRunCommand(cmd, args); err != nil {
				sm.localizer.LogError("runFailed", err)
			}
		}
	}
}

// sManager 服务管理器，基于 ServiceRunner 接口实现
type sManager struct {
	commands     *Cli
	localizer    *ServiceLocalizer
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	config       *service.Config
	service      service.Service
	exitChan     chan struct{}
	running      atomic.Bool
	stopExecuted atomic.Bool
	stopFuncOnce atomic.Bool
}

// newServiceManager 创建服务管理器实例
func newServiceManager(cmd *Cli, ctx context.Context, cancel context.CancelFunc) (*sManager, error) {
	// 创建服务本地化器
	localizer := NewServiceLocalizer(GetLanguageManager(), cmd.colors)

	sm := &sManager{
		commands:  cmd,
		localizer: localizer,
		ctx:       ctx,
		cancel:    cancel,
		exitChan:  make(chan struct{}),
	}

	// 初始化为未执行状态
	sm.stopExecuted.Store(false)

	// 创建服务配置
	config, err := sm.createServiceConfig()
	if err != nil {
		cancel() // 取消上下文
		return nil, fmt.Errorf(localizer.FormatError("createConfig")+": %v", err)
	}

	// 设置 RunWait 选项，由我们接管信号处理
	if config.Option == nil {
		config.Option = make(service.KeyValue)
	}
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGQUIT)
	}
	config.Option["RunWait"] = func() {
		sigCtx, stop := signal.NotifyContext(context.Background(), signals...)
		defer stop()
		ctx := sm.getCtx()
		select {
		case <-sigCtx.Done():
			// 捕获外部信号后主动触发停止逻辑，确保 sm.ctx 被取消
			_ = sm.Stop()
		case <-ctx.Done():
		}
	}

	sm.config = config

	// 创建 ServiceRunner（daemon 包）并绑定 StopFunc，Run 与 CLI 上下文绑定
	svcRunner := sm.buildRunner()

	// 创建服务实例（采用 Runner 适配）
	svc, err := service.New(svcRunner, config)
	if err != nil {
		cancel() // 取消上下文
		return nil, fmt.Errorf(localizer.FormatError("createService")+": %v", err)
	}
	sm.service = svc

	return sm, nil
}

// buildRunner 构造符合 daemon 的 ServiceRunner
func (sm *sManager) buildRunner() *service.ServiceRunner {
	return &service.ServiceRunner{
		RunFunc: func(_ context.Context) error {
			return sm.Run(sm.getCtx())
		},
		StopFunc: func(_ context.Context) error {
			return sm.Stop()
		},
	}
}

// getCtx 安全获取当前运行上下文，避免读写竞态
func (sm *sManager) getCtx() context.Context {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.ctx
}

// Run 实现 ServiceRunner 接口
func (sm *sManager) Run(ctx context.Context) error {
	// 将 daemon 传入的 ctx 包装为可取消上下文，确保 Stop 能正确触发 runService 的 ctx.Done()
	runCtx, runCancel := context.WithCancel(ctx)

	sm.mu.Lock()
	sm.ctx = runCtx
	sm.cancel = runCancel
	// 退出通道每次运行前重置
	select {
	case <-sm.exitChan:
		sm.exitChan = make(chan struct{})
	default:
	}
	sm.stopExecuted.Store(false)
	sm.running.Store(true)
	sm.mu.Unlock()
	defer runCancel()

	// 兜底：若 runCtx 被取消但业务未退出，15s 后强制结束
	go func(runCtx context.Context) {
		<-runCtx.Done()
		timeoutMsg := sm.localizer.FormatError("timeout", 15)
		sm.ExitWithTimeout(15*time.Second, timeoutMsg, 1)
	}(runCtx)

	defer sm.running.Store(false)

	if sm.commands.config.runtime.Run != nil {
		if err := sm.commands.config.runtime.Run(runCtx); err != nil {
			_, _ = sm.localizer.colors.Error.Printf("服务运行错误: %v\n", err)
		}
	}

	// 等待退出信号或上下文取消
	select {
	case <-sm.exitChan:
	case <-ctx.Done():
	}

	// 交互模式下补偿停止
	if service.Interactive() && !sm.stopExecuted.Load() {
		_ = sm.Stop()
	}
	return nil
}

// Stop 实现 ServiceRunner 接口
func (sm *sManager) Stop() error {
	if sm.stopExecuted.Swap(true) {
		return nil
	}

	// 执行用户停止函数一次；若未提供停止函数则给出提示
	if sm.commands.config.runtime.Stop != nil && !sm.stopFuncOnce.Swap(true) {
		if err := sm.commands.config.runtime.Stop(); err != nil {
			_, _ = sm.localizer.colors.Error.Printf("停止函数执行错误: %v\n", err)
		}
	} else if sm.commands.config.runtime.Stop == nil && !sm.stopFuncOnce.Swap(true) {
		sm.localizer.LogWarning("%s", "未配置停止函数，跳过清理逻辑")
	}

	// 取消上下文
	sm.mu.Lock()
	if sm.cancel != nil {
		sm.cancel()
	}
	select {
	case <-sm.exitChan:
	default:
		close(sm.exitChan)
	}
	sm.mu.Unlock()

	sm.running.Store(false)
	return nil
}

// Name 返回服务名称
func (sm *sManager) Name() string {
	return sm.commands.config.basic.Name
}

// createServiceConfig 创建服务配置
func (sm *sManager) createServiceConfig() (*service.Config, error) {
	// 从CLI配置创建完整的服务配置
	config := &service.Config{
		Name:        sm.commands.config.basic.Name,
		DisplayName: sm.commands.config.basic.DisplayName,
		Description: sm.commands.config.basic.Description,
	}

	// 根据操作系统设置不同的配置
	switch runtime.GOOS {
	case "windows":
		// Windows服务配置
		config.Arguments = []string{"run"}
		if sm.commands.config.service.Arguments != nil {
			config.Arguments = sm.commands.config.service.Arguments
		}
	default:
		// Unix-like系统配置
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("%s", sm.localizer.FormatError("getExecPath", err))
		}

		// 设置可执行文件路径
		config.Executable = execPath
		if sm.commands.config.service.Executable != "" {
			config.Executable = sm.commands.config.service.Executable
		}

		// 设置工作目录
		config.WorkingDirectory = filepath.Dir(execPath)
		if sm.commands.config.service.WorkDir != "" {
			config.WorkingDirectory = sm.commands.config.service.WorkDir
		}

		// 设置运行参数
		config.Arguments = []string{"run"}
		if sm.commands.config.service.Arguments != nil {
			config.Arguments = sm.commands.config.service.Arguments
		}

		// 设置其他配置选项
		config.UserName = sm.commands.config.service.Username
		config.Dependencies = sm.commands.config.service.Dependencies
		config.ChRoot = sm.commands.config.service.ChRoot
		config.Option = sm.commands.config.service.Options
		config.EnvVars = sm.commands.config.service.EnvVars

		// 验证权限
		if err := checkPermissions(config.Executable, 0o755, sm.localizer); err != nil {
			return nil, fmt.Errorf("%s", sm.localizer.FormatError("execPermission", config.Executable, err))
		}

		if config.WorkingDirectory != "" {
			if err := checkPermissions(config.WorkingDirectory, 0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf("%s", sm.localizer.FormatError("workDirPermission", config.WorkingDirectory, err))
			}
		}

		if config.ChRoot != "" {
			if err := checkPermissions(config.ChRoot, 0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf("%s", sm.localizer.FormatError("chrootPermission", config.ChRoot, err))
			}
		}
	}

	return config, nil
}

// executeRunCommand 执行运行命令，支持前台和服务模式
func (sm *sManager) executeRunCommand(_ *cobra.Command, args []string) error {
	// 如果服务正在运行，显示警告并退出
	if sm.running.Load() {
		sm.localizer.LogError("alreadyRunning", nil)
		return nil
	}

	// 处理运行参数
	serviceArgs := args
	if len(serviceArgs) == 0 && len(sm.commands.config.service.Arguments) > 0 {
		serviceArgs = sm.commands.config.service.Arguments
	}

	sm.mu.Lock()
	// 每次执行 run 前重建子上下文，确保取消信号能传递到当前 service
	if sm.cancel != nil {
		sm.cancel()
	}
	sm.ctx, sm.cancel = context.WithCancel(sm.commands.config.ctx)

	// 如果有参数变化，重新创建服务实例
	if len(serviceArgs) > 0 {
		sm.config.Arguments = serviceArgs
		svc, err := service.New(sm.buildRunner(), sm.config)
		if err != nil {
			// 避免泄漏新创建的上下文
			sm.cancel()
			sm.mu.Unlock()
			sm.localizer.LogError("createService", err)
			return fmt.Errorf(sm.localizer.FormatError("createService")+": %v", err)
		}
		sm.service = svc
	}
	sm.mu.Unlock()

	// 重置状态
	sm.stopExecuted.Store(false)

	// 创建监控通道
	runDone := make(chan struct{})

	// 在goroutine中运行服务
	go func() {
		defer close(runDone)
		defer func() {
			if r := recover(); r != nil {
				sm.localizer.LogError("runFailed", fmt.Errorf("%v", r))
			}
		}()

		// 启动服务 - 这会调用用户的Run函数
		if err := sm.service.Run(); err != nil {
			sm.localizer.LogError("runFailed", err)
		}
	}()

	// 等待服务完成，支持前台/服务模式区分
	sm.waitForServiceCompletion(runDone)
	return nil
}

// waitForServiceCompletion 等待服务完成，支持交互式和服务模式
func (sm *sManager) waitForServiceCompletion(runDone chan struct{}) {
	select {
	case <-runDone:
		// 服务正常退出
		return

	case <-sm.ctx.Done():
		// 收到取消信号，尝试优雅停止

		// 如果是交互式模式，安全地关闭退出通道
		sm.mu.Lock()
		defer sm.mu.Unlock()
		select {
		case <-sm.exitChan:
			// 通道已关闭，不需要操作
		default:
			// 安全地关闭退出通道，通知服务停止
			close(sm.exitChan)
		}

		// 使用超时机制等待服务退出
		initialWait := sm.commands.config.runtime.ShutdownInitial
		if initialWait <= 0 {
			initialWait = 3 * time.Second
		}
		graceWait := sm.commands.config.runtime.ShutdownGrace
		if graceWait <= 0 {
			graceWait = 2 * time.Second
		}

		select {
		case <-runDone:
			// 服务响应信号成功退出
			return

		case <-time.After(initialWait):
			// 超时3秒，尝试调用停止函数
			if !sm.stopExecuted.Load() {
				sm.localizer.LogWarning("%s", sm.localizer.GetError("timeoutWarning"))
				// 如果尚未执行过，则调用 Stop 方法
				_ = sm.Stop()
			} else {
				// 如果已经执行过 Stop，则直接调用停止函数
				sm.callStopFunctions()
			}

			// 再等待用户停止逻辑和额外宽限时间
			select {
			case <-runDone:
				// 在额外调用stop后成功退出
				return

			case <-time.After(graceWait):
				// 若用户 Stop 耗时超出宽限，仅记录警告，状态由运行 goroutine 最终更新
				sm.localizer.LogWarning("%s", sm.localizer.GetError("forceTerminate"))
				return
			}
		}
	}
}

// callStopFunctions 调用停止函数
func (sm *sManager) callStopFunctions() {
	if sm.commands.config.runtime.Stop != nil {
		if err := sm.commands.config.runtime.Stop(); err != nil {
			// 记录停止错误
			_, _ = sm.localizer.colors.Error.Printf("停止函数执行错误: %v\n", err)
		}
	}
}

// buildBaseCommand 构建基础命令
func (sm *sManager) buildBaseCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
	}
}

// newInstallCmd 创建安装服务命令
func (sm *sManager) newInstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("install", sm.localizer.GetOperation("install"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 检查权限
		var err error

		// 创建服务实例
		if sm.service == nil {
			svc, createErr := service.New(sm.buildRunner(), sm.config)
			if createErr != nil {
				return fmt.Errorf(sm.localizer.FormatError("createService")+": %v", createErr)
			}
			sm.service = svc
		}

		// 检查服务是否已安装
		status, _ := sm.service.Status()
		if status != service.StatusUnknown {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyExists")
			return nil
		}

		// 安装服务
		if err = sm.service.Install(); err != nil {
			return fmt.Errorf("%s: %w", sm.localizer.FormatError("installFailed"), err)
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "install")
		return nil
	}
	return cmd
}

// newUninstallCmd 创建卸载服务命令
func (sm *sManager) newUninstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("uninstall", sm.localizer.GetOperation("uninstall"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 卸载服务
		if err := sm.service.Uninstall(); err != nil {
			return fmt.Errorf("%s: %w", sm.localizer.FormatError("uninstallFailed"), err)
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "uninstall")
		return nil
	}
	return cmd
}

// newStartCmd 创建启动服务命令
func (sm *sManager) newStartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("start", sm.localizer.GetOperation("start"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.localizer.FormatError("getStatus")+": %v", err)
		}

		if status == service.StatusRunning {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyRunning")
			return nil
		}

		// 启动服务
		if err := sm.service.Start(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("startFailed")+": %v", err)
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "start")
		return nil
	}
	return cmd
}

// newStopCmd 创建停止服务命令
func (sm *sManager) newStopCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("stop", sm.localizer.GetOperation("stop"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.localizer.FormatError("getStatus")+": %v", err)
		}

		if status == service.StatusStopped {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyStopped")
			return nil
		}

		// 停止服务
		if err := sm.service.Stop(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("stopFailed")+": %v", err)
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "stop")
		return nil
	}
	return cmd
}

// newRestartCmd 创建重启服务命令
func (sm *sManager) newRestartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("restart", sm.localizer.GetOperation("restart"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.localizer.FormatError("getStatus")+": %v", err)
		}

		if status == service.StatusUnknown {
			return fmt.Errorf("%s", sm.localizer.FormatError("notFound", sm.commands.config.basic.Name))
		}

		// 如果服务正在运行，先停止
		if status == service.StatusRunning {
			if err := sm.service.Stop(); err != nil {
				return fmt.Errorf(sm.localizer.FormatError("stopFailed")+": %v", err)
			}
		}

		// 启动服务
		if err := sm.service.Start(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("restartFailed")+": %v", err)
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "restart")
		return nil
	}
	return cmd
}

// newStatusCmd 创建查看状态命令
func (sm *sManager) newStatusCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("status", sm.localizer.GetOperation("status"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// 获取服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.localizer.FormatError("getStatus")+": %v", err)
		}

		// 显示状态
		switch status {
		case service.StatusRunning:
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "running")
		case service.StatusStopped:
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "stopped")
		case service.StatusUnknown:
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
		default:
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "unknown")
		}

		return nil
	}
	return cmd
}

// newRunCmd 创建运行服务命令
func (sm *sManager) newRunCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("run", sm.localizer.GetOperation("run"))
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return sm.executeRunCommand(cmd, args)
	}
	return cmd
}

// ExitWithTimeout 在指定时间后强制退出程序
func (sm *sManager) ExitWithTimeout(timeout time.Duration, debugMsg string, exitCode int) {
	go func() {
		time.Sleep(timeout)
		if debugMsg != "" {
			_, _ = fmt.Fprintln(os.Stderr, debugMsg)
		}
		os.Exit(exitCode)
	}()
}

// checkPermissions 检查文件或目录的权限
func checkPermissions(path string, requiredPerm os.FileMode, localizer *ServiceLocalizer) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s", localizer.FormatError("pathNotExist", path))
		}
		return fmt.Errorf("%s", localizer.FormatError("getPathInfo", err))
	}

	currentPerm := fileInfo.Mode() & os.ModePerm

	// 路径类型校验：仅在显式要求目录时校验
	if requiredPerm&os.ModeDir != 0 && !fileInfo.IsDir() {
		return fmt.Errorf("%s", localizer.FormatError("needDir", path))
	}

	// 分别检查读/写/执行权限，更清晰地提示缺失项
	type permCheck struct {
		bit  os.FileMode
		name string
	}
	checks := []permCheck{
		{0o400, "read"},
		{0o200, "write"},
		{0o100, "exec"},
	}
	for _, p := range checks {
		if requiredPerm&p.bit != 0 && currentPerm&p.bit == 0 {
			return fmt.Errorf("%s", localizer.FormatError("insufficientPerm",
				p.name,
				fmt.Sprintf("%o", currentPerm)))
		}
	}

	return nil
}
