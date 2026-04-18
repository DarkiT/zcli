package zcli

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
	NeedDir           string // 路径必须为目录
	NeedFile          string // 路径必须为文件
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
