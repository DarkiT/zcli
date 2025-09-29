package zcli

import (
	"fmt"
	"reflect"
	"strings"
)

// =============================================================================
// 层次化语言包结构设计
// =============================================================================

// Language 新的层次化语言包结构
type Language struct {
	Code    string        // 语言代码 (如: "zh", "en")
	Name    string        // 语言名称 (如: "中文", "English")
	Service ServiceDomain // 服务域
	UI      UIDomain      // 界面域
	Error   ErrorDomain   // 错误域
	Format  FormatDomain  // 格式化域
}

// ServiceDomain 服务域 - 专注于服务相关的所有文本
type ServiceDomain struct {
	Operations ServiceOperations // 服务操作
	Status     ServiceStatus     // 服务状态
	Messages   ServiceMessages   // 服务消息
}

// ServiceOperations 服务操作相关文本
type ServiceOperations struct {
	Install   string // 安装
	Uninstall string // 卸载
	Start     string // 启动
	Stop      string // 停止
	Restart   string // 重启
	Run       string // 运行
	Status    string // 查看状态
}

// ServiceStatus 服务状态相关文本
type ServiceStatus struct {
	Running        string // 正在运行
	Stopped        string // 已停止
	Unknown        string // 未知状态
	NotInstalled   string // 未安装
	AlreadyExists  string // 已存在
	AlreadyRunning string // 已在运行
	AlreadyStopped string // 已停止
	Success        string // 成功
}

// ServiceMessages 服务操作过程中的提示消息
type ServiceMessages struct {
	Installing     string // 正在安装...
	Uninstalling   string // 正在卸载...
	Starting       string // 正在启动...
	Stopping       string // 正在停止...
	Restarting     string // 正在重启...
	CheckingStatus string // 正在检查状态...
	TimeoutWarning string // 超时警告
	ForceTerminate string // 强制终止
}

// UIDomain 界面域 - 专注于用户界面相关文本
type UIDomain struct {
	Commands CommandUI // 命令界面
	Help     HelpUI    // 帮助界面
	Version  VersionUI // 版本界面
}

// CommandUI 命令界面相关文本
type CommandUI struct {
	Usage             string // 用法
	Options           string // 选项
	Examples          string // 示例
	Flags             string // 参数
	AvailableCommands string // 可用命令
	SystemCommands    string // 系统命令
	DefaultValue      string // 默认值格式: "(default: %s)"
}

// HelpUI 帮助界面相关文本
type HelpUI struct {
	Command     string // 帮助命令
	Description string // 帮助描述
	Usage       string // 帮助使用说明格式: "Use '%s [command] --help' for more information"
}

// VersionUI 版本界面相关文本
type VersionUI struct {
	Command     string // 版本命令
	Description string // 版本描述
	Label       string // 版本标签
}

// ErrorDomain 错误域 - 集中管理所有错误信息
type ErrorDomain struct {
	Prefix  string        // 错误前缀
	Service ServiceErrors // 服务错误
	System  SystemErrors  // 系统错误
	Help    HelpErrors    // 帮助错误
}

// ServiceErrors 服务相关错误
type ServiceErrors struct {
	CreateConfig    string // 创建配置失败
	CreateService   string // 创建服务失败
	GetStatus       string // 获取状态失败
	StartFailed     string // 启动失败
	StopFailed      string // 停止失败
	RestartFailed   string // 重启失败
	InstallFailed   string // 安装失败
	UninstallFailed string // 卸载失败
	RunFailed       string // 运行失败
	NotFound        string // 服务未找到
	AlreadyRunning  string // 服务已在运行
	Timeout         string // 操作超时
	TimeoutWarning  string // 超时警告
	ForceTerminate  string // 强制终止
}

// SystemErrors 系统相关错误
type SystemErrors struct {
	PathNotExist      string // 路径不存在
	GetPathInfo       string // 获取路径信息失败
	InsufficientPerm  string // 权限不足
	GetExecPath       string // 获取可执行文件路径失败
	ExecPermission    string // 可执行文件权限检查失败
	WorkDirPermission string // 工作目录权限检查失败
	ChrootPermission  string // chroot目录权限检查失败
}

// HelpErrors 帮助相关错误
type HelpErrors struct {
	UnknownTopic string // 未知帮助主题
}

// FormatDomain 格式化域 - 提供格式化模板
type FormatDomain struct {
	ServiceStatus    string // "Service %s: %s"
	ErrorWithDetail  string // "Error: %s"
	TimeoutMessage   string // "Timeout after %d seconds"
	PermissionDenied string // "Required: %v, Current: %v"
	PathError        string // "Path error: %s"
}

// =============================================================================
// 智能语言包管理器
// =============================================================================

// LanguageManager 智能语言包管理器
type LanguageManager struct {
	primary  *Language // 主要语言包
	fallback *Language // 回退语言包
	registry map[string]*Language
}

// NewLanguageManager 创建语言包管理器
func NewLanguageManager(primaryLang string) *LanguageManager {
	manager := &LanguageManager{
		registry: make(map[string]*Language),
	}

	// 注册内置语言包
	manager.registry["zh"] = newChineseLanguage()
	manager.registry["en"] = newEnglishLanguage()

	// 设置主要语言包和回退语言包
	if lang, exists := manager.registry[primaryLang]; exists {
		manager.primary = lang
	} else {
		manager.primary = manager.registry["zh"] // 默认中文
	}

	manager.fallback = manager.registry["en"] // 回退到英文

	return manager
}

// RegisterLanguage 注册新的语言包
func (lm *LanguageManager) RegisterLanguage(lang *Language) error {
	if lang == nil || lang.Code == "" {
		return fmt.Errorf("invalid language: language or code is nil/empty")
	}

	if err := lm.validateLanguage(lang); err != nil {
		return fmt.Errorf("language validation failed: %v", err)
	}

	lm.registry[lang.Code] = lang
	return nil
}

// SetPrimary 设置主要语言
func (lm *LanguageManager) SetPrimary(langCode string) error {
	if lang, exists := lm.registry[langCode]; exists {
		lm.primary = lang
		return nil
	}
	return fmt.Errorf("language '%s' not found", langCode)
}

// GetPrimary 获取主要语言包
func (lm *LanguageManager) GetPrimary() *Language {
	return lm.primary
}

// GetText 智能获取文本，支持回退机制
func (lm *LanguageManager) GetText(path string) string {
	// 尝试从主要语言包获取
	if text := lm.getTextFromLanguage(lm.primary, path); text != "" {
		return text
	}

	// 回退到默认语言包
	if text := lm.getTextFromLanguage(lm.fallback, path); text != "" {
		return text
	}

	// 都没有找到，返回路径标识
	return fmt.Sprintf("[Missing: %s]", path)
}

// getTextFromLanguage 从指定语言包获取文本
func (lm *LanguageManager) getTextFromLanguage(lang *Language, path string) string {
	if lang == nil {
		return ""
	}

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}

	// 使用反射获取嵌套字段的值
	value := reflect.ValueOf(lang).Elem()
	for _, part := range parts {
		// 将首字母大写以匹配结构体字段
		fieldName := strings.Title(part)
		value = value.FieldByName(fieldName)
		if !value.IsValid() {
			return ""
		}
	}

	if value.Kind() == reflect.String {
		return value.String()
	}

	return ""
}

// validateLanguage 验证语言包完整性
func (lm *LanguageManager) validateLanguage(lang *Language) error {
	if lang.Code == "" {
		return fmt.Errorf("language code is required")
	}
	if lang.Name == "" {
		return fmt.Errorf("language name is required")
	}

	// 检查关键字段是否为空
	criticalPaths := []string{
		"service.operations.install",
		"service.operations.start",
		"service.operations.stop",
		"service.status.running",
		"service.status.stopped",
		"ui.commands.usage",
		"error.prefix",
	}

	for _, path := range criticalPaths {
		if text := lm.getTextFromLanguage(lang, path); text == "" {
			return fmt.Errorf("critical field missing: %s", path)
		}
	}

	return nil
}

// =============================================================================
// 便利性API - ServiceLocalizer
// =============================================================================

// ServiceLocalizer 服务本地化器，提供便利的API
type ServiceLocalizer struct {
	manager *LanguageManager
	colors  *colors
}

// NewServiceLocalizer 创建服务本地化器
func NewServiceLocalizer(manager *LanguageManager, colors *colors) *ServiceLocalizer {
	return &ServiceLocalizer{
		manager: manager,
		colors:  colors,
	}
}

// GetOperation 获取操作文本
func (sl *ServiceLocalizer) GetOperation(operation string) string {
	path := fmt.Sprintf("service.operations.%s", operation)
	return sl.manager.GetText(path)
}

// GetStatus 获取状态文本
func (sl *ServiceLocalizer) GetStatus(status string) string {
	path := fmt.Sprintf("service.status.%s", status)
	return sl.manager.GetText(path)
}

// GetError 获取错误文本
func (sl *ServiceLocalizer) GetError(errorType string) string {
	path := fmt.Sprintf("error.service.%s", errorType)
	if text := sl.manager.GetText(path); text != "" {
		return text
	}
	// 尝试系统错误
	path = fmt.Sprintf("error.system.%s", errorType)
	return sl.manager.GetText(path)
}

// GetFormat 获取格式化模板
func (sl *ServiceLocalizer) GetFormat(formatType string) string {
	path := fmt.Sprintf("format.%s", formatType)
	return sl.manager.GetText(path)
}

// LogError 记录错误日志
func (sl *ServiceLocalizer) LogError(errorType string, err error) {
	message := sl.GetError(errorType)
	if sl.colors != nil {
		_, _ = sl.colors.Error.Printf("%s: %v\n", message, err)
	} else {
		fmt.Printf("Error: %s: %v\n", message, err)
	}
}

// LogWarning 记录警告日志
func (sl *ServiceLocalizer) LogWarning(message string, args ...interface{}) {
	text := fmt.Sprintf(message, args...)
	if sl.colors != nil {
		_, _ = sl.colors.Warning.Println(text)
	} else {
		fmt.Printf("Warning: %s\n", text)
	}
}

// LogSuccess 记录成功日志
func (sl *ServiceLocalizer) LogSuccess(serviceName, operation string) {
	format := sl.GetFormat("serviceStatus")
	successText := sl.GetStatus("success")
	if sl.colors != nil {
		_, _ = sl.colors.Success.Printf(format+"\n", serviceName, successText)
	} else {
		fmt.Printf(format+"\n", serviceName, successText)
	}
}

// LogInfo 记录信息日志
func (sl *ServiceLocalizer) LogInfo(serviceName, status string) {
	format := sl.GetFormat("serviceStatus")
	statusText := sl.GetStatus(status)
	if sl.colors != nil {
		_, _ = sl.colors.Info.Printf(format+"\n", serviceName, statusText)
	} else {
		fmt.Printf(format+"\n", serviceName, statusText)
	}
}

// FormatError 格式化错误消息
func (sl *ServiceLocalizer) FormatError(errorType string, args ...interface{}) string {
	template := sl.GetError(errorType)
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

// FormatServiceStatus 格式化服务状态
func (sl *ServiceLocalizer) FormatServiceStatus(serviceName, status string) string {
	format := sl.GetFormat("serviceStatus")
	statusText := sl.GetStatus(status)
	return fmt.Sprintf(format, serviceName, statusText)
}

// =============================================================================
// 内置语言包定义
// =============================================================================

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
				Flags:             "参数",
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
				Label:       "版本",
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
				Flags:             "Flags",
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
				Label:       "Version",
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

// =============================================================================
// 全局语言包管理器实例
// =============================================================================

// GlobalLanguageManager 全局语言包管理器
var GlobalLanguageManager = NewLanguageManager("zh")

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
