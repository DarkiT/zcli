package zcli

import (
	"errors"
	"strings"

	service "github.com/darkit/daemon"
	"github.com/spf13/cobra"
)

// ServiceCommandGroupID 标记由 zcli 注入的系统服务命令，避免按名字误伤用户命令。
const ServiceCommandGroupID = "zcli-service"

func isFrameworkServiceCommand(cmd *Command) bool {
	return cmd != nil && cmd.GroupID == ServiceCommandGroupID
}

// initService 初始化服务
func (c *Cli) initService() error {
	if c.config.runtime.Run == nil {
		return nil
	}

	sm, err := c.newServiceAssemblyManager()
	if err != nil {
		return err
	}

	c.attachServiceAssembly(sm)
	return nil
}

// attachServiceAssembly 把 service command adapter 与 root run 策略一起挂到当前 CLI。
func (c *Cli) attachServiceAssembly(sm *sManager) {
	if sm == nil {
		return
	}

	c.addServiceCommands(sm)
	c.attachServiceRootRun(sm)
}

// addServiceCommands 添加服务管理命令。
// 该阶段只负责命令树注入，不修改根命令默认运行策略。
func (c *Cli) addServiceCommands(sm *sManager) {
	if !c.command.ContainsGroup(ServiceCommandGroupID) {
		title := "System Commands"
		if c.lang != nil && c.lang.UI.Commands.SystemCommands != "" {
			title = c.lang.UI.Commands.SystemCommands
		}
		c.command.AddGroup(&Group{
			ID:    ServiceCommandGroupID,
			Title: title,
		})
	}
	c.command.AddCommand(
		sm.newRunCmd(),
		sm.newInstallCmd(),
		sm.newReinstallCmd(),
		sm.newUninstallCmd(),
		sm.newStartCmd(),
		sm.newStopCmd(),
		sm.newRestartCmd(),
		sm.newStatusCmd(),
	)
}

// attachServiceRootRun 设置根命令的运行策略，处理直接运行的情况。
func (c *Cli) attachServiceRootRun(sm *sManager) {
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
		case c.shouldRunServiceOnRoot():
			return sm.executeRunCommand(cmd, args)
		default:
			return cmd.Help()
		}
	}
}

func (c *Cli) shouldRunServiceOnRoot() bool {
	if c == nil || c.config == nil || c.config.basic == nil {
		return true
	}
	switch c.config.basic.RootAction {
	case RootActionRun:
		return true
	case RootActionHelp:
		return false
	default:
		return !c.hasUserCommands()
	}
}

func (c *Cli) hasUserCommands() bool {
	if c == nil || c.command == nil {
		return false
	}
	for _, cmd := range c.command.Commands() {
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			continue
		}
		if isFrameworkServiceCommand(cmd) {
			continue
		}
		if cmd.IsAvailableCommand() || cmd.Name() == "help" {
			return true
		}
	}
	return false
}

func (sm *sManager) buildBaseCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		GroupID: ServiceCommandGroupID,
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

func optionEnabled(options ServiceOptions, key string) bool {
	if options == nil {
		return false
	}
	enabled, ok := options[key].(bool)
	return ok && enabled
}

func (sm *sManager) shouldForceInstall() bool {
	if sm == nil {
		return false
	}
	if sm.commands != nil && sm.commands.config != nil && sm.commands.config.service != nil {
		if sm.commands.config.service.Force || optionEnabled(sm.commands.config.service.Options, service.OptionForceInstall) {
			return true
		}
	}
	if sm.config != nil && (sm.config.Force || optionEnabled(sm.config.Option, service.OptionForceInstall)) {
		return true
	}
	return false
}

func (sm *sManager) applyInstallForce() {
	if sm == nil || sm.config == nil || !sm.shouldForceInstall() {
		return
	}
	sm.config.Force = true
	if sm.config.Option == nil {
		sm.config.Option = make(service.KeyValue)
	}
	sm.config.Option[service.OptionForceInstall] = true
}

func isRecoverableStatusError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		msg := strings.ToLower(current.Error())
		if strings.Contains(msg, "failed state") || strings.Contains(msg, "unknown status") {
			return true
		}
	}
	return false
}

func (sm *sManager) classifyServicePresence(status service.Status, err error) (notInstalled bool, hard error) {
	if err == nil {
		return status == service.StatusUnknown, nil
	}
	if isNotInstalled(err) || errors.Is(err, service.ErrNotInstalled) {
		return true, nil
	}
	if isRecoverableStatusError(err) {
		return false, nil
	}
	return false, err
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
		if err := sm.verifyInstallPaths(); err != nil {
			return err
		}

		if !sm.shouldForceInstall() {
			status, statusErr := sm.queryServiceStatus()
			notInstalled, hardErr := sm.classifyServicePresence(status, statusErr)
			if hardErr != nil {
				return hardErr
			}
			if !notInstalled {
				sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyExists")
				return nil
			}
		}

		sm.applyInstallForce()
		if err = sm.service.Install(); err != nil {
			return sm.wrapServiceError(err, ErrServiceInstall, "install")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "install")
		return nil
	})
	return cmd
}

func (sm *sManager) newReinstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("reinstall", sm.localizer.GetOperation("reinstall"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		if sm.service == nil {
			svc, createErr := service.New(sm.buildRunner(), sm.config)
			if createErr != nil {
				return WrapError(createErr, ErrServiceCreate, "reinstall")
			}
			sm.service = svc
		}
		if err := sm.verifyInstallPaths(); err != nil {
			return err
		}
		if err := service.Reinstall(sm.service); err != nil {
			return sm.wrapServiceError(err, ErrServiceInstall, "reinstall")
		}
		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "reinstall")
		return nil
	})
	return cmd
}

// newUninstallCmd 创建卸载服务命令
func (sm *sManager) newUninstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("uninstall", sm.localizer.GetOperation("uninstall"))
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		status, statusErr := sm.queryServiceStatus()
		notInstalled, hardErr := sm.classifyServicePresence(status, statusErr)
		if hardErr != nil {
			return hardErr
		}
		if notInstalled {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
			return nil
		}

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
		notInstalled, hardErr := sm.classifyServicePresence(status, err)
		if hardErr != nil {
			return hardErr
		}
		if notInstalled {
			return ErrServiceNotInstalled(sm.commands.config.basic.Name)
		}

		if err == nil && status == service.StatusRunning {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyRunning")
			return nil
		}

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
		notInstalled, hardErr := sm.classifyServicePresence(status, err)
		if hardErr != nil {
			return hardErr
		}
		if notInstalled {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
			return nil
		}

		if err == nil && status == service.StatusStopped {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "alreadyStopped")
			return nil
		}

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
		notInstalled, hardErr := sm.classifyServicePresence(status, err)
		if hardErr != nil {
			return hardErr
		}
		if notInstalled {
			return ErrServiceNotInstalled(sm.commands.config.basic.Name)
		}

		if err := sm.service.Restart(); err != nil {
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
		notInstalled, hardErr := sm.classifyServicePresence(status, err)
		if hardErr != nil {
			return hardErr
		}
		if notInstalled {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, "notInstalled")
			return nil
		}

		if status == service.StatusUnknown {
			sm.localizer.LogInfo(sm.commands.config.basic.Name, status.String())
			return nil
		}
		sm.localizer.LogInfo(sm.commands.config.basic.Name, status.String())

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
