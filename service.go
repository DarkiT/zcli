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
	// 检查是否设置了服务运行函数
	if c.config.Runtime.Run == nil {
		// 未设置服务运行函数，跳过服务命令初始化
		return
	}

	// 创建带信号处理的上下文
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // 终止信号
		syscall.SIGQUIT, // 退出信号
	)

	// 创建服务管理器
	sm, err := newServiceManager(c, ctx, cancel)
	if err != nil {
		_, _ = c.colors.Error.Printf("初始化服务管理器失败: %v\n", err)
		return
	}

	// 设置信号处理
	c.setupSignalHandler(ctx, sm)

	// 添加服务命令并配置根命令运行函数
	c.addServiceCommands(sm)
}

// setupSignalHandler 设置信号处理器
func (c *Cli) setupSignalHandler(ctx context.Context, sm *sManager) {
	go func() {
		<-ctx.Done()
		//_, _ = c.colors.Debug.Println("接收到系统信号，准备退出程序")

		// 确保服务停止
		if sm != nil && sm.running.Load() {
			_ = sm.Stop(sm.service)
		}

		// 直接调用用户注册的停止函数（双重保障）
		c.executeStopFunctions()

		// 如果服务没有及时退出，强制结束进程
		time.AfterFunc(15*time.Second, func() {
			_, _ = c.colors.Error.Println("服务未能在15秒内正常退出，强制结束进程")
			os.Exit(0) // 使用0作为退出码，因为这是预期中的退出
		})
	}()
}

// executeStopFunctions 执行所有已注册的停止函数
func (c *Cli) executeStopFunctions() {
	if c.config.Runtime.Stop != nil {
		for _, stop := range c.config.Runtime.Stop {
			if stop != nil {
				//_, _ = c.colors.Debug.Println("直接调用用户停止函数")
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
			//_, _ = cmd.colors.Debug.Println("上下文取消，触发服务停止")
			_ = sm.Stop(sm.service)

			// 确保退出应用程序，防止卡死
			time.AfterFunc(15*time.Second, func() {
				_, _ = cmd.colors.Debug.Println("强制结束进程")
				os.Exit(0)
			})
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
		return fmt.Errorf("服务已在运行中")
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
			//_, _ = sm.commands.colors.Debug.Println("服务收到退出通道信号")
		case <-sm.ctx.Done():
			//_, _ = sm.commands.colors.Debug.Println("服务收到上下文取消信号")
			// 如果是由上下文取消触发，主动关闭退出通道
			// 先检查通道是否已关闭
			select {
			case <-sm.exitChan:
				// 通道已关闭，不需要操作
			default:
				// 发送退出信号
				close(sm.exitChan)
			}
		}

		// 如果是交互式模式，自动停止
		// 但要检查是否已经执行过停止操作
		if service.Interactive() && !sm.stopExecuted.Load() {
			_, _ = sm.commands.colors.Debug.Println("交互模式下自动停止服务")
			_ = sm.Stop(s)
		}
	}()

	return nil
}

// Stop 实现 service.Interface 接口，停止服务
func (sm *sManager) Stop(s service.Service) error {
	// 使用原子操作检查是否已经执行过停止操作
	if sm.stopExecuted.Swap(true) {
		// 如果已经执行过，只输出调试信息并返回
		//_, _ = sm.commands.colors.Debug.Println("Stop方法已被调用过，跳过重复执行")
		return nil
	}

	// 如果没有在运行，直接返回
	if !sm.running.Load() {
		return nil
	}

	//_, _ = sm.commands.colors.Info.Println("服务正在停止...")

	// 执行用户定义的停止函数 - 先执行这一步确保用户的停止逻辑被执行
	if sm.commands.config.Runtime.Stop != nil {
		for _, stop := range sm.commands.config.Runtime.Stop {
			if stop != nil {
				//_, _ = sm.commands.colors.Debug.Println("执行用户定义的停止函数")
				stop()
			}
		}
	}

	// 检查退出通道是否已关闭
	select {
	case <-sm.exitChan:
		//_, _ = sm.commands.colors.Debug.Println("退出通道已经关闭")
	default:
		// 发送退出信号
		close(sm.exitChan)
		//_, _ = sm.commands.colors.Debug.Println("已关闭退出通道")
	}

	// 标记为非运行状态
	sm.running.Store(false)

	// 确保应用程序能够退出，使用短时延迟确认
	time.AfterFunc(200*time.Millisecond, func() {
		_, _ = sm.commands.colors.Debug.Println("检查服务是否已停止")
	})

	//_, _ = sm.commands.colors.Success.Println("服务已停止")
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
		return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
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
		if err := checkPermissions(config.Executable, 0o500); err != nil {
			return nil, fmt.Errorf("可执行文件 %s 权限检查失败: %v", config.Executable, err)
		}
	}

	// 检查工作目录权限
	if err := checkPermissions(config.WorkingDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("工作目录 %s 权限检查失败: %v", config.WorkingDirectory, err)
	}

	// 检查 ChRoot 目录权限（如果启用）
	if config.ChRoot != "" {
		if err := checkPermissions(config.ChRoot, 0o700); err != nil {
			return nil, fmt.Errorf("chroot 目录 %s 权限检查失败: %v", config.ChRoot, err)
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
			_, _ = sm.commands.colors.Error.Printf("创建服务实例失败: %v\n", err)
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
			_, _ = sm.commands.colors.Error.Printf("运行服务失败: %v\n", err)
		}
	}()

	// 等待服务退出或收到信号
	select {
	case <-runDone:
		_, _ = sm.commands.colors.Success.Println("服务已正常退出")
	case <-sm.ctx.Done():
		_, _ = sm.commands.colors.Debug.Println("接收到终止信号，等待服务退出...")

		// 等待服务退出，但最多等待3秒
		select {
		case <-runDone:
			_, _ = sm.commands.colors.Success.Println("服务已应信号退出")
		case <-time.After(3 * time.Second):
			if sm.commands.config.Runtime.Stop != nil {
				for _, stop := range sm.commands.config.Runtime.Stop {
					if stop != nil {
						_, _ = sm.commands.colors.Debug.Println("等待超时，再次调用停止函数")
						stop()
					}
				}
			}

			// 继续等待一段时间
			select {
			case <-runDone:
				_, _ = sm.commands.colors.Success.Println("服务在额外调用stop后退出")
			case <-time.After(2 * time.Second):
				_, _ = sm.commands.colors.Warning.Println("服务未能在总计5秒内退出，标记为已停止")
				// 确保服务已标记为停止状态
				sm.running.Store(false)
				sm.stopExecuted.Store(true)
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
			return fmt.Errorf("创建服务实例失败: %v", err)
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
			return fmt.Errorf("安装服务失败: %v", err)
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
			return fmt.Errorf("卸载服务失败: %v", err)
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
			return fmt.Errorf("服务 %s 未安装", sm.commands.config.Basic.Name)
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

// checkPermissions 检查文件或目录的权限
func checkPermissions(path string, requiredPerm os.FileMode) error {
	// 检查路径是否存在
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("路径不存在: %s", path)
		}
		return fmt.Errorf("获取路径信息失败: %v", err)
	}

	// 检查是否有足够的权限
	perm := fileInfo.Mode() & os.ModePerm
	if perm&requiredPerm != requiredPerm {
		return fmt.Errorf("权限不足: 需要 %v, 当前 %v", requiredPerm, perm)
	}

	return nil
}
