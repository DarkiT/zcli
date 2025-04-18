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
		sm.ExitWithTimeout(15*time.Second, fmt.Sprintf(c.lang.Service.ServiceStopTimeout, 15), 1)
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
			sm.runRun(cmd, args)
		}
	}
}

// sManager 服务管理器，封装service库的底层功能
type sManager struct {
	commands     *Cli               // CLI实例引用
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
	sm := &sManager{
		commands: cmd,
		ctx:      ctx,
		cancel:   cancel,
		exitChan: make(chan struct{}),
	}

	// 初始化为未执行状态
	sm.stopExecuted.Store(false)

	// 创建服务配置
	config, err := sm.createServiceConfig()
	if err != nil {
		cancel() // 取消上下文
		return nil, fmt.Errorf(cmd.lang.Service.ErrCreateConfig+": %v", err)
	}
	sm.config = config

	// 创建服务实例
	svc, err := service.New(sm, config)
	if err != nil {
		cancel() // 取消上下文
		return nil, fmt.Errorf(cmd.lang.Service.ErrCreateService+": %v", err)
	}
	sm.service = svc

	// 设置上下文监听器，当上下文取消时执行停止函数
	go func() {
		<-ctx.Done()
		// 如果服务正在运行且尚未执行过停止操作，执行停止函数
		if sm.running.Load() && !sm.stopExecuted.Load() {
			_ = sm.Stop(sm.service)

			// 确保退出应用程序，防止卡死
			sm.ExitWithTimeout(15*time.Second, fmt.Sprintf(cmd.lang.Service.ServiceStopTimeout, 15), 1)
		}
	}()

	return sm, nil
}

// Start 实现 service.Interface 接口，启动服务
func (sm *sManager) Start(s service.Service) error {
	// 重置停止标志
	sm.stopExecuted.Store(false)

	// 防止重复启动
	if sm.running.Load() {
		return fmt.Errorf(sm.commands.lang.Service.ServiceAlreadyRunning)
	}

	// 确保退出通道已关闭
	select {
	case <-sm.exitChan:
		// 通道已关闭，重新创建
		sm.exitChan = make(chan struct{})
	default:
		// 通道未关闭，不需要操作
	}

	// 标记为运行状态
	sm.running.Store(true)

	// 启动主服务
	go func() {
		defer sm.running.Store(false)

		// 执行用户定义的运行函数
		if sm.commands.config.Runtime.Run != nil {
			sm.commands.config.Runtime.Run()
		}

		// 监听退出信号
		select {
		case <-sm.exitChan:
			// 收到退出通道信号
		case <-sm.ctx.Done():
			// 收到上下文取消信号
			// 如果是由上下文取消触发，主动关闭退出通道
			select {
			case <-sm.exitChan:
				// 通道已关闭，不需要操作
			default:
				// 关闭退出通道
				close(sm.exitChan)
			}
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

	// 检查退出通道是否已关闭，如未关闭则关闭
	select {
	case <-sm.exitChan:
		// 通道已关闭，不需要操作
	default:
		// 发送退出信号
		close(sm.exitChan)
	}

	// 标记为非运行状态
	sm.running.Store(false)

	return nil
}

// createServiceConfig 创建服务配置
func (sm *sManager) createServiceConfig() (*service.Config, error) {
	// 从CLI配置创建服务配置
	config := &service.Config{
		Name:             sm.commands.config.Basic.Name,
		DisplayName:      sm.commands.config.Basic.DisplayName,
		Description:      sm.commands.config.Basic.Description,
		UserName:         sm.commands.config.Service.Username,
		Arguments:        sm.commands.config.Service.Arguments,
		Executable:       sm.commands.config.Service.Executable,
		Dependencies:     sm.commands.config.Service.Dependencies,
		WorkingDirectory: sm.commands.config.Service.WorkDir,
		ChRoot:           sm.commands.config.Service.ChRoot,
		Option:           sm.commands.config.Service.Options,
		EnvVars:          sm.commands.config.Service.EnvVars,
	}

	// 获取可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf(sm.commands.lang.Service.ErrGetExecPath, err)
	}
	if config.Executable == "" {
		config.Executable = execPath
	}

	// 获取工作目录
	if config.WorkingDirectory == "" {
		config.WorkingDirectory = filepath.Dir(execPath)
	}

	// 非Windows系统检查文件权限
	if runtime.GOOS != "windows" {
		// 检查可执行文件路径权限
		if err := checkPermissions(config.Executable, 0o500, sm.commands.lang); err != nil {
			return nil, fmt.Errorf(sm.commands.lang.Service.ErrExecFilePermission, config.Executable, err)
		}
	}

	// 检查工作目录权限
	if err := checkPermissions(config.WorkingDirectory, 0o700, sm.commands.lang); err != nil {
		return nil, fmt.Errorf(sm.commands.lang.Service.ErrWorkDirPermission, config.WorkingDirectory, err)
	}

	// 检查 ChRoot 目录权限（如果启用）
	if config.ChRoot != "" {
		if err := checkPermissions(config.ChRoot, 0o700, sm.commands.lang); err != nil {
			return nil, fmt.Errorf(sm.commands.lang.Service.ErrChrootPermission, config.ChRoot, err)
		}
	}

	return config, nil
}

// runRun 运行服务（在前台）
func (sm *sManager) runRun(_ *cobra.Command, args []string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 构建运行参数
	serviceArgs := args
	if len(serviceArgs) == 0 && len(sm.commands.config.Service.Arguments) > 0 {
		serviceArgs = sm.commands.config.Service.Arguments
	}

	// 如果有参数变化，重新创建服务实例
	if len(serviceArgs) > 0 {
		sm.config.Arguments = serviceArgs
		svc, err := service.New(sm, sm.config)
		if err != nil {
			_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.ErrCreateService+": %v\n", err)
			return
		}
		sm.service = svc
	}

	// 确保状态重置
	sm.stopExecuted.Store(false)

	// 创建一个用于监控服务运行状态的通道
	runDone := make(chan struct{})

	// 在另一个goroutine中运行服务
	go func() {
		defer close(runDone)
		// 启动服务
		if err := sm.service.Run(); err != nil {
			_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.ErrRunService+": %v\n", err)
		}
	}()

	// 使用单独的函数处理服务结束和超时逻辑，使代码更清晰
	sm.waitForServiceCompletion(runDone)
}

// waitForServiceCompletion 等待服务完成或响应信号
func (sm *sManager) waitForServiceCompletion(runDone chan struct{}) {
	// 等待服务退出或收到信号
	select {
	case <-runDone:
		// 服务自行退出，无需额外处理
		return

	case <-sm.ctx.Done():
		// 收到取消信号，尝试优雅停止

		// 使用超时机制等待服务退出
		select {
		case <-runDone:
			// 服务响应信号成功退出
			return

		case <-time.After(3 * time.Second):
			// 超时3秒，尝试调用停止函数
			if !sm.stopExecuted.Load() {
				_, _ = sm.commands.colors.Warning.Println(sm.commands.lang.Service.StopTimeoutReinvoke)
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
				_, _ = sm.commands.colors.Warning.Println(sm.commands.lang.Service.ServiceStopTimedOut)
				sm.running.Store(false)
				sm.stopExecuted.Store(true)
				return
			}
		}
	}
}

// callStopFunctions 调用用户注册的所有停止函数
func (sm *sManager) callStopFunctions() {
	if sm.commands.config.Runtime.Stop != nil {
		for _, stop := range sm.commands.config.Runtime.Stop {
			if stop != nil {
				stop()
			}
		}
	}
}

// buildBaseCommand 构建基础命令模板
func (sm *sManager) buildBaseCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:                   use,
		Short:                 short,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
	}
}

// newInstallCmd 创建安装服务命令
func (sm *sManager) newInstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("install", sm.commands.lang.Service.Install)
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// 构建运行参数，通常安装服务时需要包含 "run" 参数
		serviceArgs := append([]string{"run"}, args...)
		sm.config.Arguments = serviceArgs

		// 重新创建服务实例
		svc, err := service.New(sm, sm.config)
		if err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrCreateService+": %v", err)
		}
		sm.service = svc

		// 检查服务是否已安装
		status, err := sm.service.Status()
		if err == nil && status != service.StatusUnknown {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyExists)
			return nil
		}

		// 安装服务
		if err = sm.service.Install(); err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrInstallService, err)
		}

		_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Success)
		return nil
	}
	return cmd
}

// newUninstallCmd 创建卸载服务命令
func (sm *sManager) newUninstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("uninstall", sm.commands.lang.Service.Uninstall)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// 先停止服务
		_ = sm.service.Stop()

		// 卸载服务
		if err := sm.service.Uninstall(); err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrUninstallService, err)
		}

		_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Success)
		return nil
	}
	return cmd
}

// newStartCmd 创建启动服务命令
func (sm *sManager) newStartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("start", sm.commands.lang.Service.Start)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrGetStatus+": %v", err)
		}

		if status == service.StatusRunning {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyRunning)
			return nil
		}

		// 启动服务
		if err = sm.service.Start(); err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrStartService+": %v", err)
		}

		_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Success)
		return nil
	}
	return cmd
}

// newStopCmd 创建停止服务命令
func (sm *sManager) newStopCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("stop", sm.commands.lang.Service.Stop)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrGetStatus+": %v", err)
		}

		if status == service.StatusStopped {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyStopped)
			return nil
		}

		// 停止服务
		if err = sm.service.Stop(); err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrStopService+": %v", err)
		}

		_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Success)
		return nil
	}
	return cmd
}

// newRestartCmd 创建重启服务命令
func (sm *sManager) newRestartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("restart", sm.commands.lang.Service.Restart)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		// 检查服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrGetStatus+": %v", err)
		}

		if status == service.StatusUnknown {
			return fmt.Errorf(sm.commands.lang.Service.ErrServiceNotFound, sm.commands.config.Basic.Name)
		}

		// 先停止服务
		_ = sm.service.Stop()

		// 等待服务完全停止
		time.Sleep(time.Second * 2)

		// 启动服务
		if err = sm.service.Start(); err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrRestartService+": %v", err)
		}

		_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Success)
		return nil
	}
	return cmd
}

// newStatusCmd 创建查看服务状态命令
func (sm *sManager) newStatusCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("status", sm.commands.lang.Service.Status)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sm.mu.RLock()
		defer sm.mu.RUnlock()

		// 获取服务状态
		status, err := sm.service.Status()
		if err != nil {
			return fmt.Errorf(sm.commands.lang.Service.ErrGetStatus+": %v", err)
		}

		// 根据状态显示不同的消息
		switch status {
		case service.StatusRunning:
			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Running)
		case service.StatusStopped:
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Stopped)
		case service.StatusUnknown:
			_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.NotInstalled)
		default:
			_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.Unknown)
		}

		return nil
	}
	return cmd
}

// newRunCmd 创建在前台运行服务的命令
func (sm *sManager) newRunCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("run", sm.commands.lang.Service.Run)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		sm.runRun(cmd, args)
	}
	return cmd
}

// ExitWithTimeout 设置一个超时强制退出机制
// 参数说明：
//   - timeout: 超时时间（如 15*time.Second）
//   - debugMsg: 退出前的调试信息（可选）
//   - exitCode: 退出状态码（默认0）
func (sm *sManager) ExitWithTimeout(timeout time.Duration, debugMsg string, exitCode int) {
	time.AfterFunc(timeout, func() {
		if debugMsg != "" {
			_, _ = sm.commands.colors.Error.Println(debugMsg)
		}
		os.Exit(exitCode)
	})
}

// checkPermissions 检查文件或目录的权限
func checkPermissions(path string, requiredPerm os.FileMode, lang *Language) error {
	// 检查路径是否存在
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(lang.Service.ErrPathNotExist, path)
		}
		return fmt.Errorf(lang.Service.ErrGetPathInfo, err)
	}

	// 检查是否有足够的权限
	perm := fileInfo.Mode() & os.ModePerm
	if perm&requiredPerm != requiredPerm {
		return fmt.Errorf(lang.Service.ErrInsufficientPerm, requiredPerm, perm)
	}

	return nil
}
