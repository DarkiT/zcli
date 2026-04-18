package zcli

import (
	"context"
	"errors"

	service "github.com/darkit/daemon"
	"github.com/spf13/cobra"
)

// initService 初始化服务
func (c *Cli) initService() {
	// 使用外部上下文，信号处理由 service.RunWait 负责
	ctx, cancel := context.WithCancel(c.config.Context())

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

	// 添加服务命令并配置根命令运行函数
	c.addServiceCommands(sm)
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
	originalRunE := c.command.RunE
	c.command.Run = nil
	c.command.RunE = func(cmd *cobra.Command, args []string) error {
		switch {
		case originalRunE != nil:
			return sm.handleError(originalRunE(cmd, args))
		case originalRun != nil:
			originalRun(cmd, args)
			return nil
		default:
			return sm.executeRunCommand(cmd, args)
		}
	}
}

func (sm *sManager) buildBaseCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
	}
}

func (sm *sManager) wrapRunE(runE func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	if runE == nil {
		return nil
	}
	return func(cmd *cobra.Command, args []string) error {
		return sm.handleError(runE(cmd, args))
	}
}

func (sm *sManager) queryServiceStatus() (service.Status, error) {
	status, err := sm.service.Status()
	if err == nil {
		return status, nil
	}
	if errors.Is(err, service.ErrNotInstalled) {
		return status, ErrServiceNotInstalled(sm.Name()).WithCause(err)
	}
	return status, sm.wrapServiceError(err, ErrServiceStatus, "status")
}

func isNotInstalled(err error) bool {
	return IsErrorCode(err, ErrServiceNotFound)
}

// newInstallCmd 创建安装服务命令
func (sm *sManager) newInstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("install", sm.localizer.GetOperation("install"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		// 检查权限
		var err error

		// 创建服务实例
		if sm.service == nil {
			svc, createErr := service.New(sm.buildRunner(), sm.config)
			if createErr != nil {
				return WrapError(createErr, ErrServiceCreate, "install")
			}
			sm.service = svc
		}

		// 检查服务是否已安装
		status, statusErr := sm.queryServiceStatus()
		if statusErr == nil && status != service.StatusUnknown {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyExists")
			return nil
		}
		if statusErr != nil && !isNotInstalled(statusErr) {
			return statusErr
		}

		// 安装服务
		if err = sm.service.Install(); err != nil {
			return sm.wrapServiceError(err, ErrServiceInstall, "install")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "install")
		return nil
	})
	return cmd
}

// newUninstallCmd 创建卸载服务命令
func (sm *sManager) newUninstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("uninstall", sm.localizer.GetOperation("uninstall"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		status, statusErr := sm.queryServiceStatus()
		if statusErr == nil && status == service.StatusUnknown {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
			return nil
		}
		if statusErr != nil {
			if isNotInstalled(statusErr) {
				sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
				return nil
			}
			return statusErr
		}

		// 卸载服务
		if err := sm.service.Uninstall(); err != nil {
			return sm.wrapServiceError(err, ErrServiceUninstall, "uninstall")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "uninstall")
		return nil
	})
	return cmd
}

// newStartCmd 创建启动服务命令
func (sm *sManager) newStartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("start", sm.localizer.GetOperation("start"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.queryServiceStatus()
		if err != nil {
			return err
		}

		if status == service.StatusRunning {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyRunning")
			return nil
		}

		// 启动服务
		if err := sm.service.Start(); err != nil {
			return sm.wrapServiceError(err, ErrServiceStart, "start")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "start")
		return nil
	})
	return cmd
}

// newStopCmd 创建停止服务命令
func (sm *sManager) newStopCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("stop", sm.localizer.GetOperation("stop"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.queryServiceStatus()
		if err != nil {
			return err
		}

		if status == service.StatusStopped {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyStopped")
			return nil
		}

		// 停止服务
		if err := sm.service.Stop(); err != nil {
			return sm.wrapServiceError(err, ErrServiceStop, "stop")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "stop")
		return nil
	})
	return cmd
}

// newRestartCmd 创建重启服务命令
func (sm *sManager) newRestartCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("restart", sm.localizer.GetOperation("restart"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		// 检查服务状态
		status, err := sm.queryServiceStatus()
		if err != nil {
			return err
		}

		if status == service.StatusUnknown {
			return ErrServiceNotInstalled(sm.commands.config.basic.Name)
		}

		// 如果服务正在运行，先停止
		if status == service.StatusRunning {
			if err := sm.service.Stop(); err != nil {
				return sm.wrapServiceError(err, ErrServiceStop, "restart")
			}
		}

		// 启动服务
		if err := sm.service.Start(); err != nil {
			return sm.wrapServiceError(err, ErrServiceRestart, "restart")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "restart")
		return nil
	})
	return cmd
}

// newStatusCmd 创建查看状态命令
func (sm *sManager) newStatusCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("status", sm.localizer.GetOperation("status"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		// 获取服务状态
		status, err := sm.queryServiceStatus()
		if err != nil {
			if isNotInstalled(err) {
				sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
				return nil
			}
			return err
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
	})
	return cmd
}

// newRunCmd 创建运行服务命令
func (sm *sManager) newRunCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("run", sm.localizer.GetOperation("run"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		return sm.executeRunCommand(cmd, args)
	})
	return cmd
}
