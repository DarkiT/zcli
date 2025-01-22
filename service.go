package zcli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	service "github.com/darkit/syscore"
	"github.com/spf13/cobra"
)

// initService 初始化服务
func (c *Cli) initService() {
	// 创建服务管理器
	sm, err := newServiceManager(c)
	if err != nil {
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
	exit     chan struct{}
}

// newServiceManager 创建服务管理器实例
func newServiceManager(cmd *Cli) (*sManager, error) {
	sm := &sManager{
		commands: cmd,
		exit:     make(chan struct{}),
	}
	// 创建服务实例
	svc, err := sm.initSystemService()
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	sm.service = svc

	return sm, nil
}

// Start 添加service.Service接口Start方法
func (sm *sManager) Start(_ service.Service) error {
	if sm.commands.opts.Run != nil {
		go sm.iRun()
	}
	return nil
}

// Stop 添加service.Service接口Stop方法
func (sm *sManager) Stop(_ service.Service) error {
	if sm.commands.opts.Stop != nil {
		sm.commands.opts.Stop()
	}
	close(sm.exit)
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}

func (sm *sManager) iRun() {
	defer func() {
		// 判断运行模式
		if service.Interactive() {
			// 交互式模式：直接运行
			_ = sm.Stop(sm.service)
		} else {
			// 服务模式：通过服务管理器运行
			_ = sm.service.Stop()
		}
	}()
	sm.commands.opts.Run()
}

// newInstallCmd 创建安装服务命令
func (sm *sManager) newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "install",
		Short:                 sm.commands.lang.Service.Install,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true, // 允许未知标志，不报错
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sm.mu.Lock()
			defer sm.mu.Unlock()

			if len(args) > 0 {
				sm.commands.opts.Arguments = args

				svr, err := sm.initSystemService()
				// 有带参数的服务注册
				if err != nil {
					return fmt.Errorf("failed to create service: %w", err)
				}
				sm.service = svr
			}

			// 检查服务是否已安装
			status, err := sm.service.Status()
			if err == nil && status != service.StatusUnknown {
				_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.AlreadyExists)
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

			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Success)
			return nil
		},
	}
	return cmd
}

// newUninstallCmd 创建卸载服务命令
func (sm *sManager) newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "uninstall",
		Short:                 sm.commands.lang.Service.Uninstall,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		RunE: func(cmd *cobra.Command, args []string) error {
			sm.mu.Lock()
			defer sm.mu.Unlock()

			// 先停止服务
			_ = sm.service.Stop()

			// 卸载服务
			if err := sm.service.Uninstall(); err != nil {
				return fmt.Errorf("failed to uninstall service: %w", err)
			}

			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Success)
			return nil
		},
	}
}

// newStartCmd 创建启动服务命令
func (sm *sManager) newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "start",
		Short:                 sm.commands.lang.Service.Start,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		RunE: func(cmd *cobra.Command, args []string) error {
			sm.mu.Lock()
			defer sm.mu.Unlock()

			// 检查服务状态
			status, err := sm.service.Status()
			if err != nil {
				return fmt.Errorf("failed to get service status: %w", err)
			}

			if status == service.StatusRunning {
				_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.AlreadyRunning)
				return nil
			}

			// 启动服务
			if err = sm.service.Start(); err != nil {
				return fmt.Errorf("failed to start service: %w", err)
			}

			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Success)
			return nil
		},
	}
}

// newStopCmd 创建停止服务命令
func (sm *sManager) newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "stop",
		Short:                 sm.commands.lang.Service.Stop,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		RunE: func(cmd *cobra.Command, args []string) error {
			sm.mu.Lock()
			defer sm.mu.Unlock()

			// 检查服务状态
			status, err := sm.service.Status()
			if err != nil {
				return fmt.Errorf("failed to get service status: %w", err)
			}

			if status == service.StatusStopped {
				_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.AlreadyStopped)
				return nil
			}

			// 停止服务
			if err = sm.service.Stop(); err != nil {
				return fmt.Errorf("failed to stop service: %w", err)
			}

			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Success)
			return nil
		},
	}
}

// newRestartCmd 创建重启服务命令
func (sm *sManager) newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "restart",
		Short:                 sm.commands.lang.Service.Restart,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		RunE: func(cmd *cobra.Command, args []string) error {
			sm.mu.Lock()
			defer sm.mu.Unlock()

			// 检查服务状态
			status, err := sm.service.Status()
			if err != nil {
				return fmt.Errorf("failed to get service status: %w", err)
			}

			if status == service.StatusUnknown {
				return fmt.Errorf("service %s not installed", sm.commands.opts.Name)
			}

			// 重启服务
			if err = sm.service.Restart(); err != nil {
				return fmt.Errorf("failed to restart service: %w", err)
			}

			_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Success)
			return nil
		},
	}
}

// newStatusCmd 创建查看服务状态命令
func (sm *sManager) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "status",
		Short:                 sm.commands.lang.Service.Status,
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		SilenceErrors:         true, // 禁用错误输出
		SilenceUsage:          true, // 禁用使用说明输出
		RunE: func(cmd *cobra.Command, args []string) error {
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
				_, _ = sm.commands.colors.Success.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Running)
			case service.StatusStopped:
				_, _ = sm.commands.colors.Warning.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Stopped)
			case service.StatusUnknown:
				_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.NotInstalled)
			default:
				_, _ = sm.commands.colors.Error.Printf(sm.commands.lang.Service.StatusFormat+separator, sm.commands.opts.Name, sm.commands.lang.Service.Unknown)
			}

			return nil
		},
	}
}

// controlSystemService 控制系统服务
func (sm *sManager) controlSystemService(action string) (err error) {
	for _, v := range service.ControlAction {
		if action == v {
			switch action {
			case "install":
				err = sm.service.Install()
			case "uninstall":
				err = sm.service.Uninstall()
			case "start":
				err = sm.service.Start()
			case "stop":
				err = sm.service.Stop()
			case "restart":
				err = sm.service.Restart()
			}
			if err != nil {
				return fmt.Errorf("%s service failed: %w", action, err)
			}
			return nil
		}
	}

	return fmt.Errorf("unknown action: %s", action)
}

// initSystemService 初始化系统服务
func (sm *sManager) initSystemService() (service.Service, error) {
	// 创建服务配置
	sm.config = &service.Config{
		Name:             sm.commands.opts.Name,
		DisplayName:      sm.commands.opts.DisplayName,
		Description:      sm.commands.opts.Description,
		UserName:         sm.commands.opts.UserName,
		Arguments:        sm.commands.opts.Arguments,
		Executable:       sm.commands.opts.Executable,
		Dependencies:     sm.commands.opts.Dependencies,
		WorkingDirectory: sm.commands.opts.WorkingDirectory,
		ChRoot:           sm.commands.opts.ChRoot,
		Option:           sm.commands.opts.Option,
		EnvVars:          sm.commands.opts.EnvVars,
	}

	if sm.config.WorkingDirectory == "" {
		// 获取可执行文件路径
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}
		sm.config.WorkingDirectory = filepath.Dir(execPath)
	}

	if sm.config.Option == nil {
		sm.config.Option = map[string]interface{}{
			"StartLimitInterval": 5,                                              // 5秒内启动失败10次则停止
			"StartLimitBurst":    10,                                             // 最多允许重启10次
			"LimitNOFILE":        65535,                                          // 最大打开文件数
			"SuccessExitStatus":  "0 2",                                          // 退出码0和2都视为成功
			"ReloadSignal":       "SIGUSR1",                                      // 重载信号
			"Restart":            "always",                                       // 总是重启
			"Type":               "simple",                                       // 服务类型
			"PIDFile":            fmt.Sprintf("/var/run/%s.pid", sm.config.Name), // PID文件
		}
	}

	return service.New(sm, sm.config)
}
