package zcli

// newChineseLanguage 创建中文语言包
func newChineseLanguage() *Language {
	return &Language{
		Code: "zh",
		Name: "中文",
		Service: ServiceDomain{
			Operations: ServiceOperations{
				Install:   "安装服务",
				Uninstall: "卸载服务",
				Start:     "启动服务",
				Stop:      "停止服务",
				Restart:   "重启服务",
				Run:       "运行服务",
				Status:    "查看状态",
			},
			Status: ServiceStatus{
				Running:        "正在运行",
				Stopped:        "已停止",
				Unknown:        "未知状态",
				NotInstalled:   "服务未安装",
				AlreadyExists:  "服务已存在",
				AlreadyRunning: "服务已在运行",
				AlreadyStopped: "服务已停止",
				Success:        "执行成功",
			},
			Messages: ServiceMessages{
				Installing:     "正在安装服务...",
				Uninstalling:   "正在卸载服务...",
				Starting:       "正在启动服务...",
				Stopping:       "正在停止服务...",
				Restarting:     "正在重启服务...",
				CheckingStatus: "正在检查服务状态...",
				TimeoutWarning: "等待超时，再次调用停止函数",
				ForceTerminate: "服务未能在规定时间内退出，标记为已停止",
			},
		},
		UI: UIDomain{
			Commands: CommandUI{
				Usage:             "用法",
				Options:           "选项",
				Examples:          "示例",
				Flags:             "flags",
				AvailableCommands: "可用命令",
				SystemCommands:    "系统命令",
				DefaultValue:      "(默认值: %s)",
			},
			Help: HelpUI{
				Command:     "获取帮助",
				Description: "获取帮助",
				Usage:       "使用 '%s [command] --help' 获取命令的更多信息",
			},
			Version: VersionUI{
				Command:     "Ver",
				Description: "显示版本信息",
				Label:       "v",
			},
		},
		Error: ErrorDomain{
			Prefix: "错误: ",
			Service: ServiceErrors{
				CreateConfig:    "创建服务配置失败",
				CreateService:   "创建服务实例失败",
				GetStatus:       "获取服务状态失败",
				StartFailed:     "启动服务失败",
				StopFailed:      "停止服务失败",
				RestartFailed:   "重启服务失败",
				InstallFailed:   "安装服务失败",
				UninstallFailed: "卸载服务失败",
				RunFailed:       "运行服务失败",
				NotFound:        "服务 %s 未安装",
				AlreadyRunning:  "服务已在运行中",
				Timeout:         "服务未能在%d秒内正常退出，强制结束进程",
				TimeoutWarning:  "等待超时，再次调用停止函数",
				ForceTerminate:  "服务未能在规定时间内退出，标记为已停止",
			},
			System: SystemErrors{
				PathNotExist:      "路径不存在: %s",
				GetPathInfo:       "获取路径信息失败: %v",
				NeedDir:           "路径必须是目录: %s",
				NeedFile:          "路径必须是文件: %s",
				InsufficientPerm:  "权限不足，需要 %s，当前 %s",
				GetExecPath:       "获取可执行文件路径失败: %v",
				ExecPermission:    "可执行文件权限检查失败 %s: %v",
				WorkDirPermission: "工作目录权限检查失败 %s: %v",
				ChrootPermission:  "chroot目录权限检查失败 %s: %v",
			},
			Help: HelpErrors{
				UnknownTopic: "未知的帮助主题: %v",
			},
		},
		Format: FormatDomain{
			ServiceStatus:    "服务 %s: %s",
			ErrorWithDetail:  "错误: %s",
			TimeoutMessage:   "超时 %d 秒",
			PermissionDenied: "权限被拒绝: 需要 %v, 当前 %v",
			PathError:        "路径错误: %s",
		},
	}
}

// newEnglishLanguage 创建英文语言包
func newEnglishLanguage() *Language {
	return &Language{
		Code: "en",
		Name: "English",
		Service: ServiceDomain{
			Operations: ServiceOperations{
				Install:   "Install Service",
				Uninstall: "Uninstall Service",
				Start:     "Start Service",
				Stop:      "Stop Service",
				Restart:   "Restart Service",
				Run:       "Run Service",
				Status:    "Service Status",
			},
			Status: ServiceStatus{
				Running:        "Running",
				Stopped:        "Stopped",
				Unknown:        "Unknown",
				NotInstalled:   "Service not installed",
				AlreadyExists:  "Service already exists",
				AlreadyRunning: "Service is already running",
				AlreadyStopped: "Service is already stopped",
				Success:        "Success",
			},
			Messages: ServiceMessages{
				Installing:     "Installing service...",
				Uninstalling:   "Uninstalling service...",
				Starting:       "Starting service...",
				Stopping:       "Stopping service...",
				Restarting:     "Restarting service...",
				CheckingStatus: "Checking service status...",
				TimeoutWarning: "Timeout waiting, calling stop functions again",
				ForceTerminate: "Service failed to exit within timeout period, marked as stopped",
			},
		},
		UI: UIDomain{
			Commands: CommandUI{
				Usage:             "Usage",
				Options:           "Options",
				Examples:          "Examples",
				Flags:             "flags",
				AvailableCommands: "Available Commands",
				SystemCommands:    "System Commands",
				DefaultValue:      "(default: %s)",
			},
			Help: HelpUI{
				Command:     "Help",
				Description: "Help about any command",
				Usage:       "Use '%s [command] --help' for more information about a command",
			},
			Version: VersionUI{
				Command:     "Ver",
				Description: "Show version information",
				Label:       "v",
			},
		},
		Error: ErrorDomain{
			Prefix: "Error: ",
			Service: ServiceErrors{
				CreateConfig:    "Failed to create service configuration",
				CreateService:   "Failed to create service instance",
				GetStatus:       "Failed to get service status",
				StartFailed:     "Failed to start service",
				StopFailed:      "Failed to stop service",
				RestartFailed:   "Failed to restart service",
				InstallFailed:   "Failed to install service",
				UninstallFailed: "Failed to uninstall service",
				RunFailed:       "Failed to run service",
				NotFound:        "Service %s is not installed",
				AlreadyRunning:  "Service is already running",
				Timeout:         "Service failed to exit within %d seconds, force terminating process",
				TimeoutWarning:  "Timeout waiting, calling stop functions again",
				ForceTerminate:  "Service failed to exit within timeout period, marked as stopped",
			},
			System: SystemErrors{
				PathNotExist:      "Path does not exist: %s",
				GetPathInfo:       "Failed to get path info: %v",
				NeedDir:           "Path must be a directory: %s",
				NeedFile:          "Path must be a file: %s",
				InsufficientPerm:  "Insufficient permissions, required %s, current %s",
				GetExecPath:       "Failed to get executable path: %v",
				ExecPermission:    "Executable file permission check failed %s: %v",
				WorkDirPermission: "Working directory permission check failed %s: %v",
				ChrootPermission:  "Chroot directory permission check failed %s: %v",
			},
			Help: HelpErrors{
				UnknownTopic: "Unknown help topic: %v",
			},
		},
		Format: FormatDomain{
			ServiceStatus:    "Service %s: %s",
			ErrorWithDetail:  "Error: %s",
			TimeoutMessage:   "Timeout after %d seconds",
			PermissionDenied: "Permission denied: required %v, current %v",
			PathError:        "Path error: %s",
		},
	}
}

// GlobalLanguageManager 全局语言包管理器
var GlobalLanguageManager = NewLanguageManager("en")

// SetLanguage 设置全局语言
func SetLanguage(langCode string) error {
	return GlobalLanguageManager.SetPrimary(langCode)
}

// RegisterLanguage 注册语言包到全局管理器
func RegisterLanguage(lang *Language) error {
	return GlobalLanguageManager.RegisterLanguage(lang)
}

// GetLanguageManager 获取全局语言包管理器
func GetLanguageManager() *LanguageManager {
	return GlobalLanguageManager
}

// GetText 从全局管理器获取文本
func GetText(path string) string {
	return GlobalLanguageManager.GetText(path)
}
