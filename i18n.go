package zcli

import (
	"fmt"
	"strings"
)

// ServiceMessages 服务相关消息
type ServiceMessages struct {
	// 操作相关
	Install   string // 安装服务
	Uninstall string // 卸载服务
	Start     string // 启动服务
	Stop      string // 停止服务
	Restart   string // 重启服务
	Status    string // 查看状态
	Run       string // 在前台运行服务

	// 状态相关
	StatusFormat string // 状态格式化字符串
	Running      string // 正在运行
	Stopped      string // 已停止
	Unknown      string // 未知状态
	Success      string // 操作成功

	// 错误状态
	NotInstalled   string // 服务未安装
	AlreadyExists  string // 服务已存在
	AlreadyRunning string // 服务已在运行
	AlreadyStopped string // 服务已停止

	// 错误消息
	ErrGetStatus          string // 获取服务状态失败
	ErrStartService       string // 启动服务失败
	ErrStopService        string // 停止服务失败
	ErrRestartService     string // 重启服务失败
	ErrCreateConfig       string // 创建服务配置失败
	ErrCreateService      string // 创建服务实例失败
	ErrRunService         string // 运行服务失败
	ErrPathNotExist       string // 路径不存在
	ErrGetPathInfo        string // 获取路径信息失败
	ErrInsufficientPerm   string // 权限不足
	ErrGetExecPath        string // 获取可执行文件路径失败
	ErrExecFilePermission string // 可执行文件权限检查失败
	ErrWorkDirPermission  string // 工作目录权限检查失败
	ErrChrootPermission   string // chroot目录权限检查失败
	ErrInstallService     string // 安装服务失败
	ErrUninstallService   string // 卸载服务失败
	ErrServiceNotFound    string // 服务未找到

	// 调试和信息消息
	ServiceStopTimeout    string // 服务未能在指定时间内退出
	StopTimeoutReinvoke   string // 等待超时，再次调用停止函数
	ServiceStopTimedOut   string // 服务未能在规定时间内退出
	ServiceAlreadyRunning string // 服务已在运行中
}

// CommandMessages 命令相关消息
type CommandMessages struct {
	// 基础命令信息
	Usage   string // 用法说明
	Command string // 命令

	// 命令选项相关
	Flags string // 命令参数

	// UI 相关文本
	Options      string // 选项标题
	DefaultValue string // 默认值格式
	Examples     string // 示例标题
	HelpUsage    string // 帮助使用说明

	// 命令类型
	AvailableCommands string // 可用命令
	SystemCommands    string // 系统命令

	// 帮助相关
	HelpCommand string // 帮助命令
	HelpDesc    string // 帮助命令描述

	// 版本相关
	Version     string // 版本命令
	VersionDesc string // 版本描述
}

// ErrorMessages 错误相关消息
type ErrorMessages struct {
	Prefix           string // 错误前缀
	UnknownHelpTopic string // 未知帮助主题
}

// Language 语言包定义
type Language struct {
	Service ServiceMessages
	Command CommandMessages
	Error   ErrorMessages
}

// 预定义语言包
var (
	defaultLang = "zh" // 默认语言
	languages   = map[string]*Language{
		"zh": zhCN, // 注册中文语言包
		"en": enUS, // 注册英文语言包
	}

	// 中文语言包
	zhCN = &Language{
		Service: ServiceMessages{
			// 操作相关
			Run:       "运行服务",
			Install:   "安装服务",
			Uninstall: "卸载服务",
			Start:     "启动服务",
			Stop:      "停止服务",
			Restart:   "重启服务",
			Status:    "查看状态",

			// 状态相关
			StatusFormat: "服务 %s: %s",
			Running:      "正在运行",
			Stopped:      "已停止",
			Unknown:      "未知状态",
			Success:      "执行成功",

			// 错误状态
			NotInstalled:   "服务未安装",
			AlreadyExists:  "服务已存在",
			AlreadyRunning: "服务已在运行",
			AlreadyStopped: "服务已停止",

			// 错误消息
			ErrGetStatus:          "获取服务状态失败",
			ErrStartService:       "启动服务失败",
			ErrStopService:        "停止服务失败",
			ErrRestartService:     "重启服务失败",
			ErrCreateConfig:       "创建服务配置失败",
			ErrCreateService:      "创建服务实例失败",
			ErrRunService:         "运行服务失败",
			ErrPathNotExist:       "路径不存在: %s",
			ErrGetPathInfo:        "获取路径信息失败: %v",
			ErrInsufficientPerm:   "权限不足: 需要 %v, 当前 %v",
			ErrGetExecPath:        "获取可执行文件路径失败: %v",
			ErrExecFilePermission: "可执行文件 %s 权限检查失败: %v",
			ErrWorkDirPermission:  "工作目录 %s 权限检查失败: %v",
			ErrChrootPermission:   "chroot 目录 %s 权限检查失败: %v",
			ErrInstallService:     "安装服务失败: %v",
			ErrUninstallService:   "卸载服务失败: %v",
			ErrServiceNotFound:    "服务 %s 未安装",

			// 调试和信息消息
			ServiceStopTimeout:    "服务未能在%d秒内正常退出，强制结束进程",
			StopTimeoutReinvoke:   "等待超时，再次调用停止函数",
			ServiceStopTimedOut:   "服务未能在规定时间内退出，标记为已停止",
			ServiceAlreadyRunning: "服务已在运行中",
		},
		Command: CommandMessages{
			Usage:   "用法",
			Command: "命令",

			// 命令选项相关
			Flags: "参数",

			// UI 相关文本
			Options:      "选项",
			DefaultValue: "(默认值: %s)",
			Examples:     "示例",
			HelpUsage:    "使用 '%s [command] --help' 获取命令的更多信息。",

			// 命令类型
			AvailableCommands: "可用命令",
			SystemCommands:    "系统命令",

			// 帮助相关
			HelpCommand: "获取帮助",
			HelpDesc:    "获取帮助",

			// 版本相关
			Version:     "Ver",
			VersionDesc: "显示版本信息",
		},
		Error: ErrorMessages{
			Prefix:           "错误: ",
			UnknownHelpTopic: "未知的帮助主题: %v",
		},
	}

	// 英文语言包
	enUS = &Language{
		Service: ServiceMessages{
			// Operations
			Run:       "Run Service",
			Install:   "Install Service",
			Uninstall: "Uninstall Service",
			Start:     "Start Service",
			Stop:      "Stop Service",
			Restart:   "Restart Service",
			Status:    "Service Status",

			// Status
			StatusFormat: "Service %s: %s",
			Running:      "Running",
			Stopped:      "Stopped",
			Unknown:      "Unknown",
			Success:      "Success",

			// Error states
			NotInstalled:   "Service not installed",
			AlreadyExists:  "Service already exists",
			AlreadyRunning: "Service is already running",
			AlreadyStopped: "Service is already stopped",

			// Error messages
			ErrGetStatus:          "Failed to get service status",
			ErrStartService:       "Failed to start service",
			ErrStopService:        "Failed to stop service",
			ErrRestartService:     "Failed to restart service",
			ErrCreateConfig:       "Failed to create service configuration",
			ErrCreateService:      "Failed to create service instance",
			ErrRunService:         "Failed to run service",
			ErrPathNotExist:       "Path does not exist: %s",
			ErrGetPathInfo:        "Failed to get path information: %v",
			ErrInsufficientPerm:   "Insufficient permissions: required %v, current %v",
			ErrGetExecPath:        "Failed to get executable path: %v",
			ErrExecFilePermission: "Executable file %s permission check failed: %v",
			ErrWorkDirPermission:  "Working directory %s permission check failed: %v",
			ErrChrootPermission:   "Chroot directory %s permission check failed: %v",
			ErrInstallService:     "Failed to install service: %v",
			ErrUninstallService:   "Failed to uninstall service: %v",
			ErrServiceNotFound:    "Service %s is not installed",

			// Debug and info messages
			ServiceStopTimeout:    "Service failed to exit within %d seconds, force terminating process",
			StopTimeoutReinvoke:   "Timeout waiting, calling stop functions again",
			ServiceStopTimedOut:   "Service failed to exit within timeout period, marked as stopped",
			ServiceAlreadyRunning: "Service is already running",
		},
		Command: CommandMessages{
			Usage:   "Usage",
			Command: "Command",
			Flags:   "Flags",

			// UI 相关文本
			Options:      "Options",
			DefaultValue: "(default: %s)",
			Examples:     "Examples",
			HelpUsage:    "Use '%s [command] --help' for more information about a command.",

			// 命令类型
			AvailableCommands: "Available Commands",
			SystemCommands:    "System Commands",

			// 帮助相关
			HelpCommand: "Help",
			HelpDesc:    "Help about any command",

			// 版本相关
			Version:     "Ver",
			VersionDesc: "Show version information",
		},
		Error: ErrorMessages{
			Prefix:           "Error: ",
			UnknownHelpTopic: "Unknown help topic: %v",
		},
	}
)

// AddLanguage 注册新的语言包
func AddLanguage(lang string, l *Language) error {
	if err := validateLanguagePack(l); err != nil {
		return err
	}
	if lang != "" && l != nil {
		languages[lang] = l
	}
	return nil
}

// SetDefaultLanguage 设置默认语言
func SetDefaultLanguage(lang string) {
	if _, ok := languages[lang]; ok {
		defaultLang = lang
	}
}

// getServiceLanguage 获取语言包
func getServiceLanguage(lang string) *Language {
	if lang == "" {
		lang = defaultLang
	}
	if l, ok := languages[lang]; ok {
		return l
	}
	return languages[defaultLang]
}

// validateLanguagePack 验证语言包完整性
func validateLanguagePack(l *Language) error {
	if l == nil {
		return fmt.Errorf("language pack is nil")
	}

	// 验证服务消息
	if err := validateServiceMessages(l.Service); err != nil {
		return fmt.Errorf("service messages: %w", err)
	}

	// 验证命令消息
	if err := validateCommandMessages(l.Command); err != nil {
		return fmt.Errorf("command messages: %w", err)
	}

	// 验证错误消息
	if err := validateErrorMessages(l.Error); err != nil {
		return fmt.Errorf("error messages: %w", err)
	}

	return nil
}

// validateServiceMessages 验证服务消息完整性
func validateServiceMessages(m ServiceMessages) error {
	fields := []struct {
		value, name string
	}{
		{m.Run, "Run"},
		{m.Install, "Install"},
		{m.Uninstall, "Uninstall"},
		{m.Start, "Start"},
		{m.Stop, "Stop"},
		{m.Restart, "Restart"},
		{m.Status, "Status"},
		{m.StatusFormat, "StatusFormat"},
		{m.Running, "Running"},
		{m.Stopped, "Stopped"},
		{m.Unknown, "Unknown"},
		{m.Success, "Success"},
		{m.NotInstalled, "NotInstalled"},
		{m.AlreadyExists, "AlreadyExists"},
		{m.AlreadyRunning, "AlreadyRunning"},
		{m.AlreadyStopped, "AlreadyStopped"},
		{m.ErrGetStatus, "ErrGetStatus"},
		{m.ErrStartService, "ErrStartService"},
		{m.ErrStopService, "ErrStopService"},
		{m.ErrRestartService, "ErrRestartService"},
		{m.ErrCreateConfig, "ErrCreateConfig"},
		{m.ErrCreateService, "ErrCreateService"},
		{m.ErrRunService, "ErrRunService"},
		{m.ErrPathNotExist, "ErrPathNotExist"},
		{m.ErrGetPathInfo, "ErrGetPathInfo"},
		{m.ErrInsufficientPerm, "ErrInsufficientPerm"},
		{m.ErrGetExecPath, "ErrGetExecPath"},
		{m.ErrExecFilePermission, "ErrExecFilePermission"},
		{m.ErrWorkDirPermission, "ErrWorkDirPermission"},
		{m.ErrChrootPermission, "ErrChrootPermission"},
		{m.ErrInstallService, "ErrInstallService"},
		{m.ErrUninstallService, "ErrUninstallService"},
		{m.ErrServiceNotFound, "ErrServiceNotFound"},
		{m.ServiceStopTimeout, "ServiceStopTimeout"},
		{m.StopTimeoutReinvoke, "StopTimeoutReinvoke"},
		{m.ServiceStopTimedOut, "ServiceStopTimedOut"},
		{m.ServiceAlreadyRunning, "ServiceAlreadyRunning"},
	}

	return validateFields(fields)
}

// validateCommandMessages 验证命令消息完整性
func validateCommandMessages(m CommandMessages) error {
	fields := []struct {
		value, name string
	}{
		{m.Usage, "Usage"},
		{m.Command, "Command"},
		{m.Flags, "Flags"},
		{m.Options, "Options"},
		{m.DefaultValue, "DefaultValue"},
		{m.Examples, "Examples"},
		{m.HelpUsage, "HelpUsage"},
		{m.AvailableCommands, "AvailableCommands"},
		{m.SystemCommands, "SystemCommands"},
		{m.HelpCommand, "HelpCommand"},
		{m.HelpDesc, "HelpDesc"},
		{m.Version, "Version"},
		{m.VersionDesc, "VersionDesc"},
	}

	return validateFields(fields)
}

// validateErrorMessages 验证错误消息完整性
func validateErrorMessages(m ErrorMessages) error {
	fields := []struct {
		value, name string
	}{
		{m.Prefix, "Prefix"},
		{m.UnknownHelpTopic, "UnknownHelpTopic"},
	}

	return validateFields(fields)
}

// validateFields 通用字段验证
func validateFields(fields []struct{ value, name string }) error {
	var missing []string
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	return nil
}
