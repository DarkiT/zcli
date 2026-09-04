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

// applyInstallArguments 将 install/reinstall 透传的 CLI 参数整体替换为服务自启参数。
// 参数写入 daemon config.Arguments（默认 ["run"] 或构建期 WithArguments 均被覆盖），
// 并强制重建 service 实例，确保 Install/Reinstall 使用新参数。
func (sm *sManager) applyInstallArguments(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if sm.config == nil {
		cfg, err := sm.createServiceConfig()
		if err != nil {
			return err
		}
		sm.config = cfg
	}
	sm.config.Arguments = append([]string(nil), args...)
	sm.setService(nil) // 新参数需在 service.New 时生效
	return nil
}

func (sm *sManager) wrapRunE(runE func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	if runE == nil {
		return nil
	}
	return func(cmd *cobra.Command, args []string) error {
		return sm.handleError(runE(cmd, args))
	}
}

// serviceOrError 取当前 service 实例快照；实例缺失时返回统一错误。
// install/reinstall 会先把实例置 nil 再重建，命令路径必须走快照，
// 否则并发下会裸读到 nil 而 panic。
func (sm *sManager) serviceOrError(operation string) (service.Service, error) {
	svc := sm.currentService()
	if svc == nil {
		return nil, NewError(ErrServiceCreate).Operation(operation).Service(sm.Name()).Build()
	}
	return svc, nil
}

func (sm *sManager) queryServiceStatus() (service.Status, error) {
	svc, err := sm.serviceOrError("status")
	if err != nil {
		return service.StatusUnknown, err
	}
	status, statusErr := svc.Status()
	if statusErr == nil {
		return status, nil
	}
	if errors.Is(statusErr, service.ErrNotInstalled) {
		return status, ErrServiceNotInstalled(sm.Name()).WithCause(statusErr)
	}
	return status, sm.wrapServiceError(statusErr, ErrServiceStatus, "status")
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
	// install 之后的 token 原样作为服务自启参数透传：关闭 flag 与 positional 的
	// 交错解析，第一个非 flag token（通常是服务子命令）之后的全部参数（含未注册
	// flag 如 --a=111）不再被 pflag 解析。注意不能用 DisableFlagParsing——
	// 那会让 install 之前的 root persistent flag 既不生效又混入服务参数。
	cmd.Flags().SetInterspersed(false)
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		if err := sm.applyInstallArguments(args); err != nil {
			return err
		}
		// 检查权限

		// 创建服务实例
		if sm.currentService() == nil {
			svc, createErr := service.New(sm.buildRunner(), sm.config)
			if createErr != nil {
				return WrapError(createErr, ErrServiceCreate, "install")
			}
			sm.setService(svc)
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

		svc, createErr := sm.serviceOrError("install")
		if createErr != nil {
			return createErr
		}
		sm.applyInstallForce()
		if err := svc.Install(); err != nil {
			return sm.wrapServiceError(err, ErrServiceInstall, "install")
		}

		sm.localizer.LogSuccess(sm.commands.config.basic.Name, "install")
		return nil
	})
	return cmd
}

// newReinstallCmd 创建覆盖重装服务命令
func (sm *sManager) newReinstallCmd() *cobra.Command {
	cmd := sm.buildBaseCommand("reinstall", sm.localizer.GetOperation("reinstall"))
	// 与 install 相同：reinstall 之后的 token 作为服务自启参数透传
	cmd.Flags().SetInterspersed(false)
	cmd.RunE = sm.wrapRunE(func(cmd *cobra.Command, args []string) error {
		if err := sm.applyInstallArguments(args); err != nil {
			return err
		}
		if sm.currentService() == nil {
			svc, createErr := service.New(sm.buildRunner(), sm.config)
			if createErr != nil {
				return WrapError(createErr, ErrServiceCreate, "reinstall")
			}
			sm.setService(svc)
		}
		if err := sm.verifyInstallPaths(); err != nil {
			return err
		}
		svc, createErr := sm.serviceOrError("reinstall")
		if createErr != nil {
			return createErr
		}
		if err := service.Reinstall(svc); err != nil {
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

		svc, svcErr := sm.serviceOrError("uninstall")
		if svcErr != nil {
			return svcErr
		}
		if err := svc.Uninstall(); err != nil {
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

		svc, svcErr := sm.serviceOrError("start")
		if svcErr != nil {
			return svcErr
		}
		if err := svc.Start(); err != nil {
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

		svc, svcErr := sm.serviceOrError("stop")
		if svcErr != nil {
			return svcErr
		}
		if err := svc.Stop(); err != nil {
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

		svc, svcErr := sm.serviceOrError("restart")
		if svcErr != nil {
			return svcErr
		}
		if err := svc.Restart(); err != nil {
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
