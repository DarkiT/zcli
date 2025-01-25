package zcli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	service "github.com/darkit/syscore"
	"github.com/spf13/cobra"
)

// initService 初始化服务
func (c *Cli) initService() {
	// 创建服务管理器
	sm, err := newServiceManager(c)
	if err != nil {
		//_, _ = c.colors.Error.Println(err)
		return
	}

	// 直接初始化并添加所有服务命令
	c.command.AddCommand(
		sm.newInstallCmd(),
		sm.newUninstallCmd(),
		sm.newStartCmd(),
		sm.newStopCmd(),
		sm.newRestartCmd(),
		sm.newStatusCmd(),
	)
}

// sManager 服务管理器
type sManager struct {
	commands *Cli
	mu       sync.RWMutex
	config   *service.Config
	service  service.Service
	running  atomic.Bool
}

// newServiceManager 创建服务管理器实例
func newServiceManager(cmd *Cli) (*sManager, error) {
	sm := &sManager{
		commands: cmd,
	}

	// 创建服务实例
	svc, err := sm.initSystemService()
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	sm.service = svc

	return sm, nil
}

// Start 实现 service.Interface 接口
func (sm *sManager) Start(s service.Service) error {
	// 防止重复启动
	if sm.running.Load() {
		return fmt.Errorf("service is already running")
	}

	if sm.commands.config.Runtime.Run == nil {
		return fmt.Errorf("no run function provided")
	}

	// 标记为运行状态
	sm.running.Store(true)

	// 启动主服务
	go func() {
		defer sm.running.Store(false)

		// 执行用户定义的运行函数
		sm.commands.config.Runtime.Run()

		// 如果是交互式模式，则自动停止
		if service.Interactive() {
			_ = s.Stop()
		}
	}()

	return nil
}

// Stop 实现 service.Interface 接口
func (sm *sManager) Stop(s service.Service) error {
	// 如果没有在运行，直接返回
	if !sm.running.Load() {
		return nil
	}

	// 执行用户定义的停止函数
	if sm.commands.config.Runtime.Stop != nil {
		for _, stop := range sm.commands.config.Runtime.Stop {
			stop()
		}
	}

	// 标记为非运行状态
	sm.running.Store(false)

	return nil
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

		if len(args) > 0 {
			sm.commands.config.Service.Arguments = args

			svr, err := sm.initSystemService()
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}
			sm.service = svr
		}

		// 检查服务是否已安装
		status, err := sm.service.Status()
		if err == nil && status != service.StatusUnknown {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyExists)
			return nil
		}

		// 安装服务
		if err = sm.service.Install(); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}

		// 启动服务
		if err = sm.service.Start(); err != nil {
			return fmt.Errorf("service installed but failed to start: %w", err)
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
			return fmt.Errorf("failed to uninstall service: %w", err)
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
			return fmt.Errorf("failed to get service status: %w", err)
		}

		if status == service.StatusRunning {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyRunning)
			return nil
		}

		// 启动服务
		if err = sm.service.Start(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
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
			return fmt.Errorf("failed to get service status: %w", err)
		}

		if status == service.StatusStopped {
			_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.config.Basic.Name, sm.commands.lang.Service.AlreadyStopped)
			return nil
		}

		// 停止服务
		if err = sm.service.Stop(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
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
			return fmt.Errorf("failed to get service status: %w", err)
		}

		if status == service.StatusUnknown {
			return fmt.Errorf("service %s not installed", sm.commands.config.Basic.Name)
		}

		// 重启服务
		if err = sm.service.Restart(); err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
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
			return fmt.Errorf("failed to get service status: %w", err)
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

// initSystemService 初始化系统服务
func (sm *sManager) initSystemService() (service.Service, error) {
	// 创建服务配置
	sm.config = &service.Config{
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
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	if sm.config.Executable == "" {
		sm.config.Executable = execPath
	}

	// 获取工作目录
	if sm.config.WorkingDirectory == "" {
		sm.config.WorkingDirectory = filepath.Dir(execPath)
	}

	if runtime.GOOS != "windows" {
		// 检查可执行文件路径权限
		if err := checkPermissions(sm.config.Executable, 0o500); err != nil {
			return nil, fmt.Errorf("executable %s permission check failed: %v", sm.config.Executable, err)
		}
	}

	// 检查 WorkingDirectory 权限
	if err := checkPermissions(sm.config.WorkingDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("working directory %s permission check failed: %v", sm.config.WorkingDirectory, err)
	}

	// 检查 ChRoot 目录权限（如果启用）
	if sm.config.ChRoot != "" {
		if err := checkPermissions(sm.config.ChRoot, 0o700); err != nil {
			return nil, fmt.Errorf("chroot directory %s permission check failed: %v", sm.config.ChRoot, err)
		}
	}

	return service.New(sm, sm.config)
}

// controlSystemService 控制系统服务
func (sm *sManager) controlSystemService(action string) (err error) {
	if contains(service.ControlAction[:], action) {
		if err = service.Control(sm.service, action); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unknown action: %s", action)
}

// 判断字符串是否在切片中
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// checkPermissions 检查当前用户是否具备对指定路径的权限
func checkPermissions(path string, requiredPerm os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// 检查路径是否是一个目录（如果是目录）
	if info.IsDir() {
		if info.Mode().Perm()&requiredPerm < requiredPerm {
			return fmt.Errorf("insufficient permissions for directory: %s, current: %#o, required: %#o",
				path,
				info.Mode().Perm(),
				requiredPerm)
		}
	} else {
		if info.Mode().Perm()&requiredPerm < requiredPerm {
			return fmt.Errorf("insufficient permissions for file: %s, current: %#o, required: %#o",
				path,
				info.Mode().Perm(),
				requiredPerm)
		}
	}

	return nil
}
