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

	service "github.com/darkit/syscore"
	"github.com/spf13/cobra"
)

// initService 初始化服务
func (c *Cli) initService() {
	// 创建带信号处理的上下文
	ctx, cancel := signal.NotifyContext(
		c.config.ctx,    // 使用用户配置的上下文，而不是直接使用background
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // 终止信号
		syscall.SIGQUIT, // 退出信号
	)
	c.config.ctx = ctx

	// 检查是否设置了服务运行函数
	if c.config.Runtime.Run == nil {
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

	// 确保服务停止（增加对 stopExecuted 的检查，避免重复调用）
	if sm != nil && sm.running.Load() && !sm.stopExecuted.Load() {
		_ = sm.Stop(sm.service)
	} else if sm != nil && sm.stopExecuted.Load() {
		// 如果 Stop 已经被调用过，则跳过重复调用，仅执行用户定义的停止函数
		// 直接调用用户注册的停止函数，确保它们被执行
		c.executeStopFunctions()
	}

	// 如果服务没有及时退出，强制结束进程
	if sm != nil {
		timeoutMsg := sm.localizer.FormatError("timeout", 15)
		sm.ExitWithTimeout(15*time.Second, timeoutMsg, 1)
	}
}

// executeStopFunctions 执行所有已注册的停止函数
func (c *Cli) executeStopFunctions() {
	if c.config.Runtime.Stop != nil {
		for _, stop := range c.config.Runtime.Stop {
			if stop != nil {
				stop()
			}
		}
	}
}

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
			sm.executeRunCommand(cmd, args)
		}
	}
}

// sManager 服务管理器，封装service库的底层功能
type sManager struct {
	commands     *Cli               // CLI实例引用
	localizer    *ServiceLocalizer  // 服务本地化器
	ctx          context.Context    // 上下文，用于控制服务生命周期
	cancel       context.CancelFunc // 取消函数
	mu           sync.RWMutex       // 互斥锁，保证并发安全
	config       *service.Config    // 服务配置
	service      service.Service    // 服务实例
	exitChan     chan struct{}      // 退出通道
	running      atomic.Bool        // 运行状态标记
	stopExecuted atomic.Bool        // 停止方法执行标记
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
	sm.config = config

	// 创建服务实例
	svc, err := service.New(sm, config)
	if err != nil {
		cancel() // 取消上下文
		return nil, fmt.Errorf(localizer.FormatError("createService")+": %v", err)
	}
	sm.service = svc

	// 设置上下文监听器，当上下文取消时执行停止函数
	go func() {
		<-ctx.Done()
		// 如果服务正在运行且尚未执行过停止操作，执行停止函数
		if sm.running.Load() && !sm.stopExecuted.Load() {
			_ = sm.Stop(sm.service)

			// 确保退出应用程序，防止卡死
			timeoutMsg := localizer.FormatError("timeout", 15)
			sm.ExitWithTimeout(15*time.Second, timeoutMsg, 1)
		}
	}()

	return sm, nil
}

// Start 实现 service.Interface 接口，支持前台和服务模式
func (sm *sManager) Start(s service.Service) error {
	// 重置停止标志
	sm.stopExecuted.Store(false)

	// 防止重复启动
	if sm.running.Load() {
		return fmt.Errorf(sm.localizer.GetError("alreadyRunning"))
	}

	// 使用互斥锁保护 exitChan 的访问和修改
	sm.mu.Lock()
	// 安全地检查和重新创建退出通道
	select {
	case <-sm.exitChan:
		// 通道已关闭，重新创建
		sm.exitChan = make(chan struct{})
	default:
		// 通道未关闭，不需要操作
	}
	// 获取当前的 exitChan 引用，防止在使用过程中被其他 goroutine 修改
	exitChan := sm.exitChan
	sm.mu.Unlock()

	// 标记为运行状态
	sm.running.Store(true)

	// 启动主服务
	go func() {
		defer sm.running.Store(false)

		// 执行用户定义的运行函数
		if sm.commands.config.Runtime.Run != nil {
			// 优雅调用：传入context，用户可以选择使用或忽略
			// - 向下兼容：func() 忽略传入的context参数
			// - 推荐方式：func(ctx context.Context) 使用传入的context
			sm.commands.config.Runtime.Run(sm.ctx)
		}

		// 监听退出信号
		select {
		case <-exitChan:
			// 收到退出通道信号
		case <-sm.ctx.Done():
			// 上下文取消
		}

		// 如果是交互式模式，自动停止服务
		if service.Interactive() && !sm.stopExecuted.Load() {
			_ = sm.Stop(s)
		}
	}()

	return nil
}

// Stop 实现 service.Interface 接口，停止服务
func (sm *sManager) Stop(s service.Service) error {
	// 使用原子操作检查是否已经执行过停止操作
	// 如果已经执行过，直接返回，不重复执行
	if sm.stopExecuted.Swap(true) {
		return nil
	}

	// 如果没有在运行，直接返回
	if !sm.running.Load() {
		return nil
	}

	// 执行用户定义的停止函数 - 先执行这一步确保用户的停止逻辑被执行
	if sm.commands.config.Runtime.Stop != nil {
		for _, stop := range sm.commands.config.Runtime.Stop {
			if stop != nil {
				stop()
			}
		}
	}

	// 使用互斥锁和原子操作保护退出通道的关闭操作
	sm.mu.Lock()
	defer sm.mu.Unlock()
	select {
	case <-sm.exitChan:
		// 通道已关闭，不需要操作
	default:
		// 安全地关闭退出信号通道
		close(sm.exitChan)
	}

	// 标记为非运行状态
	sm.running.Store(false)

	return nil
}

// createServiceConfig 创建服务配置
func (sm *sManager) createServiceConfig() (*service.Config, error) {
	// 从CLI配置创建完整的服务配置
	config := &service.Config{
		Name:        sm.commands.config.Basic.Name,
		DisplayName: sm.commands.config.Basic.DisplayName,
		Description: sm.commands.config.Basic.Description,
	}

	// 根据操作系统设置不同的配置
	switch runtime.GOOS {
	case "windows":
		// Windows服务配置
		config.Arguments = []string{"run"}
		if sm.commands.config.Service.Arguments != nil {
			config.Arguments = sm.commands.config.Service.Arguments
		}
	default:
		// Unix-like系统配置
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf(sm.localizer.FormatError("getExecPath", err))
		}

		// 设置可执行文件路径
		config.Executable = execPath
		if sm.commands.config.Service.Executable != "" {
			config.Executable = sm.commands.config.Service.Executable
		}

		// 设置工作目录
		config.WorkingDirectory = filepath.Dir(execPath)
		if sm.commands.config.Service.WorkDir != "" {
			config.WorkingDirectory = sm.commands.config.Service.WorkDir
		}

		// 设置运行参数
		config.Arguments = []string{"run"}
		if sm.commands.config.Service.Arguments != nil {
			config.Arguments = sm.commands.config.Service.Arguments
		}

		// 设置其他配置选项
		config.UserName = sm.commands.config.Service.Username
		config.Dependencies = sm.commands.config.Service.Dependencies
		config.ChRoot = sm.commands.config.Service.ChRoot
		config.Option = sm.commands.config.Service.Options
		config.EnvVars = sm.commands.config.Service.EnvVars

		// 验证权限
		if err := checkPermissions(config.Executable, 0o755, sm.localizer); err != nil {
			return nil, fmt.Errorf(sm.localizer.FormatError("execPermission", config.Executable, err))
		}

		if config.WorkingDirectory != "" {
			if err := checkPermissions(config.WorkingDirectory, 0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf(sm.localizer.FormatError("workDirPermission", config.WorkingDirectory, err))
			}
		}

		if config.ChRoot != "" {
			if err := checkPermissions(config.ChRoot, 0o755, sm.localizer); err != nil {
				return nil, fmt.Errorf(sm.localizer.FormatError("chrootPermission", config.ChRoot, err))
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
	if len(serviceArgs) == 0 && len(sm.commands.config.Service.Arguments) > 0 {
		serviceArgs = sm.commands.config.Service.Arguments
	}

	// 如果有参数变化，重新创建服务实例
	if len(serviceArgs) > 0 {
		sm.mu.Lock()
		sm.config.Arguments = serviceArgs
		svc, err := service.New(sm, sm.config)
		if err != nil {
			sm.mu.Unlock()
			sm.localizer.LogError("createService", err)
			return fmt.Errorf(sm.localizer.FormatError("createService")+": %v", err)
		}
		sm.service = svc
		sm.mu.Unlock()
	}

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
		select {
		case <-runDone:
			// 服务响应信号成功退出
			return

		case <-time.After(3 * time.Second):
			// 超时3秒，尝试调用停止函数
			if !sm.stopExecuted.Load() {
				sm.localizer.LogWarning(sm.localizer.GetError("timeoutWarning"))
				// 如果尚未执行过，则调用 Stop 方法
				_ = sm.Stop(sm.service)
			} else {
				// 如果已经执行过 Stop，则直接调用停止函数
				sm.callStopFunctions()
			}

			// 再等待2秒
			select {
			case <-runDone:
				// 在额外调用stop后成功退出
				return

			case <-time.After(2 * time.Second):
				// 总计5秒后仍未退出，标记为已停止
				sm.localizer.LogWarning(sm.localizer.GetError("forceTerminate"))
				sm.running.Store(false)
				sm.stopExecuted.Store(true)
				return
			}
		}
	}
}

// callStopFunctions 调用停止函数
func (sm *sManager) callStopFunctions() {
	if sm.commands.config.Runtime.Stop != nil {
		for _, stop := range sm.commands.config.Runtime.Stop {
			if stop != nil {
				stop()
			}
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
			svc, createErr := service.New(sm, sm.config)
			if createErr != nil {
				return fmt.Errorf(sm.localizer.FormatError("createService")+": %v", createErr)
			}
			sm.service = svc
		}

		// 检查服务是否已安装
		status, _ := sm.service.Status()
		if status != service.StatusUnknown {
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "alreadyExists")
			return nil
		}

		// 安装服务
		if err = sm.service.Install(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("installFailed"), err)
		}

		sm.localizer.LogSuccess(sm.commands.config.Basic.Name, "install")
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
			return fmt.Errorf(sm.localizer.FormatError("uninstallFailed"), err)
		}

		sm.localizer.LogSuccess(sm.commands.config.Basic.Name, "uninstall")
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
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "alreadyRunning")
			return nil
		}

		// 启动服务
		if err := sm.service.Start(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("startFailed")+": %v", err)
		}

		sm.localizer.LogSuccess(sm.commands.config.Basic.Name, "start")
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
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "alreadyStopped")
			return nil
		}

		// 停止服务
		if err := sm.service.Stop(); err != nil {
			return fmt.Errorf(sm.localizer.FormatError("stopFailed")+": %v", err)
		}

		sm.localizer.LogSuccess(sm.commands.config.Basic.Name, "stop")
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
			return fmt.Errorf(sm.localizer.FormatError("notFound", sm.commands.config.Basic.Name))
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

		sm.localizer.LogSuccess(sm.commands.config.Basic.Name, "restart")
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
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "running")
		case service.StatusStopped:
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "stopped")
		case service.StatusUnknown:
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "notInstalled")
		default:
			sm.localizer.LogInfo(sm.commands.config.Basic.Name, "unknown")
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
			return fmt.Errorf(localizer.FormatError("pathNotExist", path))
		}
		return fmt.Errorf(localizer.FormatError("getPathInfo", err))
	}

	// 检查是否有足够的权限
	currentPerm := fileInfo.Mode() & os.ModePerm

	// 对于可执行文件，检查是否有执行权限
	if requiredPerm&0o111 != 0 && currentPerm&0o111 == 0 {
		return fmt.Errorf(localizer.FormatError("insufficientPerm",
			fmt.Sprintf("%o", requiredPerm),
			fmt.Sprintf("%o", currentPerm)))
	}

	// 对于目录，检查是否有读写权限
	if fileInfo.IsDir() && requiredPerm&0o600 != 0 && currentPerm&0o600 == 0 {
		return fmt.Errorf(localizer.FormatError("insufficientPerm",
			fmt.Sprintf("%o", requiredPerm),
			fmt.Sprintf("%o", currentPerm)))
	}

	return nil
}
