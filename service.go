package zcli

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/kardianos/service"
)

// 颜色定义优化 - 使用对象池
var colorPool = sync.Pool{
	New: func() interface{} {
		return &ColorScheme{
			Title:         color.New(color.FgCyan, color.Bold),
			Success:       color.New(color.FgGreen, color.Bold),
			Error:         color.New(color.FgRed, color.Bold),
			Info:          color.New(color.FgWhite),
			Version:       color.New(color.FgWhite, color.Bold),
			BuildInfo:     color.New(color.FgYellow),
			Usage:         color.New(color.FgGreen),
			OptionsTitle:  color.New(color.FgMagenta, color.Bold),
			Option:        color.New(color.FgCyan),
			CommandsTitle: color.New(color.FgBlue, color.Bold),
			Command:       color.New(color.FgWhite),
			ExamplesTitle: color.New(color.FgGreen, color.Bold),
			Example:       color.New(color.FgYellow),
		}
	},
}

type ColorScheme struct {
	Title         *color.Color
	Success       *color.Color
	Error         *color.Color
	Info          *color.Color
	Version       *color.Color
	BuildInfo     *color.Color
	Usage         *color.Color
	OptionsTitle  *color.Color
	Option        *color.Color
	CommandsTitle *color.Color
	Command       *color.Color
	ExamplesTitle *color.Color
	Example       *color.Color
}

// BuildInfo 构建信息
type BuildInfo struct {
	Version      string      `json:"version"`
	BuildTime    time.Time   `json:"buildTime"`
	GoVersion    string      `json:"goVersion"`
	GitCommit    string      `json:"gitCommit"`
	GitBranch    string      `json:"gitBranch"`
	GitTag       string      `json:"gitTag"`
	Platform     string      `json:"platform"`
	Architecture string      `json:"architecture"`
	Compiler     string      `json:"compiler"`
	Debug        atomic.Bool `json:"debug"`
}

// SetDebug 是否开启调试模式
func (bi *BuildInfo) SetDebug(debug bool) *BuildInfo {
	bi.Debug.Store(debug)
	return bi
}

// SetVersion 设置构建版本号
func (bi *BuildInfo) SetVersion(version string) *BuildInfo {
	bi.Version = version
	return bi
}

// SetBuildTime 设置构建时间
func (bi *BuildInfo) SetBuildTime(t time.Time) *BuildInfo {
	bi.BuildTime = t
	return bi
}

// String 返回格式化的构建信息
func (bi *BuildInfo) String() string {
	var b strings.Builder
	b.Grow(256)
	fmt.Fprintf(&b, "Version:      %s\n", bi.Version)
	fmt.Fprintf(&b, "Go Version:   %s\n", bi.GoVersion)
	fmt.Fprintf(&b, "Compiler:     %s\n", bi.Compiler)
	fmt.Fprintf(&b, "Platform:     %s/%s\n", bi.Platform, bi.Architecture)
	if bi.GitBranch != "" {
		fmt.Fprintf(&b, "Git Branch:   %s\n", bi.GitBranch)
	}
	if bi.GitTag != "" {
		fmt.Fprintf(&b, "Git Tag:      %s\n", bi.GitTag)
	}
	if bi.GitCommit != "" {
		fmt.Fprintf(&b, "Git Commit:   %s\n", bi.GitCommit)
	}
	fmt.Fprintf(&b, "Build Mode:   %s\n", map[bool]string{true: "Debug", false: "Release"}[bi.Debug.Load()])
	fmt.Fprintf(&b, "Build Time:   %s\n", bi.BuildTime.Format(time.DateTime))
	return b.String()
}

// NewBuildInfo 创建构建信息
func NewBuildInfo() *BuildInfo {
	return &BuildInfo{
		Version:      "1.0.0",
		BuildTime:    time.Now(),
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Compiler:     runtime.Compiler,
	}
}

// Options 服务选项
type Options struct {
	Name             string       // 服务的名称，必填。建议不包含空格。
	DisplayName      string       // 显示名称，允许包含空格。
	Description      string       // 服务的详细描述，解释其功能和用法。
	UserName         string       // 以哪个用户身份运行服务。
	Arguments        []string     // 启动时的参数，允许自定义服务的执行配置。
	Executable       string       // 服务的可执行文件路径。
	Dependencies     []string     // 该服务依赖的其他服务或组件列表。
	WorkingDirectory string       // 初始工作目录，服务开始执行的目录。
	ChRoot           string       // 为服务更改根目录，增强安全性和隔离性。
	Option           sync.Map     // 其他配置选项，为服务提供灵活的设置。
	EnvVars          sync.Map     // 服务的环境变量，影响其运行行为。
	Version          string       // 服务的版本，便于跟踪和更新。
	Logo             string       // 服务logo的路径，用于品牌识别。
	Language         string       // 服务使用的编程语言，便于文档和支持。
	NoColor          bool         // 禁用颜色，标志位，用于禁用日志或用户界面的彩色输出。
	BuildInfo        *BuildInfo   // 构建信息，包含构建过程的详细信息，如构建时间和提交哈希。
	Run              func() error // 启动服务的函数，封装其主要逻辑。
	Stop             func() error // 停止服务的函数，确保资源的正确清理。
}

// Service 服务实例
type Service struct {
	opts     *Options
	svc      service.Service
	msgs     map[string]Messages
	mu       sync.RWMutex
	isWin    bool
	paramMgr *manager
	colors   *ColorScheme
	status   atomic.Value
}

// Messages 多语言消息
type Messages struct {
	Install      string // 安装服务提示信息
	Uninstall    string // 卸载服务提示信息
	Start        string // 启动服务提示信息
	Stop         string // 停止服务提示信息
	Restart      string // 重启服务提示信息
	Status       string // 服务状态提示信息
	Usage        string // 用法提示
	Commands     string // 命令列表
	StatusFormat string // 格式化状态字符串
	Running      string // 正在运行
	Stopped      string // 已停止
	Unknown      string // 未知状态
	Example      string // 示例
	Options      string // 选项说明
	Required     string // 必需项
	Default      string // 默认值
	Help         string // 帮助菜单
	Version      string // 版本信息
}

var defaultMessages = map[string]Messages{
	"en": {
		Install:      "Install the service",
		Uninstall:    "Uninstall the service",
		Start:        "Start the service",
		Stop:         "Stop the service",
		Restart:      "Restart the service",
		Status:       "Print service status",
		Usage:        "Usage: %s [options] [command]",
		Commands:     "Commands:",
		StatusFormat: "Service %s status: %s",
		Running:      "running",
		Stopped:      "stopped",
		Unknown:      "unknown",
		Example:      "Examples:",
		Options:      "Options:",
		Required:     "required",
		Default:      "default",
		Help:         "Show help menu",
		Version:      "Show version info",
	},
	"zh": {
		Install:      "安装服务",
		Uninstall:    "卸载服务",
		Start:        "启动服务",
		Stop:         "停止服务",
		Restart:      "重启服务",
		Status:       "服务状态",
		Usage:        "用法：%s [选项] [命令]",
		Commands:     "命令：",
		StatusFormat: "服务(%s)状态：%s",
		Running:      "运行中",
		Stopped:      "已停止",
		Unknown:      "未知",
		Example:      "示例：",
		Options:      "选项：",
		Required:     "必需",
		Default:      "默认值",
		Help:         "显示帮助菜单",
		Version:      "显示版本信息",
	},
}

// New 创建新服务实例
func New(opts *Options) (*Service, error) {
	// 1. 验证选项
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// 2. 创建服务实例
	s := &Service{
		opts:     opts,
		isWin:    runtime.GOOS == "windows",
		paramMgr: NewParamManager(),
		colors:   colorPool.Get().(*ColorScheme),
		msgs:     make(map[string]Messages),
	}

	// 4. 初始化消息映射
	s.mu.Lock()
	for lang, msgs := range defaultMessages {
		s.msgs[lang] = msgs
	}
	s.mu.Unlock()

	// 5. 设置颜色支持
	if opts.NoColor || (s.isWin && !isColorSupported()) {
		color.NoColor = true
	}

	// 6. 获取命令行参数 检查是否有参数，且第一个参数是否为控制命令
	args := os.Args[1:]
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		// 检查第一个参数是否在 ControlAction 列表中
		isControlCmd := false
		for _, cmd := range service.ControlAction {
			if firstArg == cmd {
				isControlCmd = true
				break
			}
		}
		// 如果是控制命令，则跳过第一个参数
		if isControlCmd || firstArg == "status" {
			args = os.Args[2:]
		}
	}

	// 7. 初始化服务管理器
	svcConfig := &service.Config{
		Name:             opts.Name,
		DisplayName:      opts.DisplayName,
		Description:      opts.Description,
		Arguments:        append(opts.Arguments, args...),
		Executable:       opts.Executable,
		Dependencies:     opts.Dependencies,
		WorkingDirectory: opts.WorkingDirectory,
		ChRoot:           opts.ChRoot,
		EnvVars:          make(map[string]string),
	}

	// 复制环境变量
	opts.EnvVars.Range(func(key, value interface{}) bool {
		svcConfig.EnvVars[key.(string)] = value.(string)
		return true
	})

	// 创建服务
	svc, err := service.New(s, svcConfig)
	if err != nil {
		return nil, err
	}
	s.svc = svc
	return s, nil
}

// Start 实现 service.Interface
func (s *Service) Start(_ service.Service) error {
	return s.opts.Run()
}

// Stop 实现 service.Interface
func (s *Service) Stop(_ service.Service) error {
	if s.opts.Stop != nil {
		return s.opts.Stop()
	}
	return nil
}

// Run 启动服务管理
func (s *Service) Run() error {
	msgs := s.getMessage()

	// 1. 创建自定义的 FlagSet
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// 2. 设置基本的 flag
	var help, version bool
	fs.BoolVar(&help, "h", false, msgs.Help)
	fs.BoolVar(&help, "help", false, msgs.Help)
	fs.BoolVar(&version, "v", false, msgs.Version)
	fs.BoolVar(&version, "version", false, msgs.Version)

	// 3. 创建参数值存储映射
	paramValues := make(map[string]*string)

	// 4. 注册自定义参数到 FlagSet
	s.paramMgr.mu.RLock()
	registeredFlags := make(map[string]bool)
	for _, param := range s.paramMgr.params {
		// 为每个参数创建一个值存储
		paramValue := new(string)
		*paramValue = param.Default
		paramValues[param.Name] = paramValue

		// 注册长参数
		if param.Long != "" && !registeredFlags[param.Long] {
			fs.StringVar(paramValue, param.Long, param.Default, param.Description)
			registeredFlags[param.Long] = true
		}

		// 注册短参数
		if param.Short != "" && !registeredFlags[param.Short] {
			fs.StringVar(paramValue, param.Short, param.Default, param.Description)
			registeredFlags[param.Short] = true
		}
	}
	s.paramMgr.mu.RUnlock()

	fs.Usage = s.showHelp

	// 5. 解析命令行参数
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	// 6. 从命令行参数更新值到 ParamManager
	s.paramMgr.mu.Lock()
	for name, paramValue := range paramValues {
		if param, ok := s.paramMgr.params[name]; ok {
			s.paramMgr.values[param.Name] = *paramValue
		}
	}
	s.paramMgr.mu.Unlock()

	switch {
	case help:
		s.showHelp()
		return nil
	case version:
		s.showVersion()
		return nil
	}

	if args := fs.Args(); len(args) > 0 {
		return s.handleCommand(args[0])
	}

	return s.svc.Run()
}

// showVersion 显示版本信息
func (s *Service) showVersion() {
	if s.opts.Logo != "" {
		s.colors.Title.Println(strings.TrimLeft(s.opts.Logo, "\n"))
		fmt.Println()
	}

	if s.opts.BuildInfo != nil {
		s.colors.Info.Print(s.opts.BuildInfo.String())
	} else if s.opts.Version != "" {
		s.colors.Info.Printf("Version: %s\n", s.opts.Version)
	}
}

// getMessage 获取消息
func (s *Service) getMessage() Messages {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lang := s.opts.Language
	if lang == "" {
		lang = "en"
	}

	if msgs, ok := s.msgs[lang]; ok {
		return msgs
	}

	return s.msgs["en"]
}

// showHelp 显示帮助信息
func (s *Service) showHelp() {
	msgs := s.getMessage()
	var buf strings.Builder
	buf.Grow(4096)

	if s.opts.Logo != "" {
		buf.WriteString(s.colors.Title.Sprint(strings.TrimLeft(s.opts.Logo, "\n")))
	}

	if s.opts.BuildInfo != nil {
		buf.WriteString(s.colors.BuildInfo.Sprint(s.opts.BuildInfo.String()))
		buf.WriteString("\n")
	} else if s.opts.Version != "" {
		buf.WriteString(s.colors.Version.Sprintf("Version: %s\n\n", s.opts.Version))
	}

	buf.WriteString(s.colors.Usage.Sprintf(msgs.Usage+"\n\n", os.Args[0]))

	// 选项部分
	buf.WriteString(s.colors.OptionsTitle.Sprint(msgs.Options + "\n"))
	buf.WriteString(s.colors.Option.Sprintf("  %-20s%s\n", "-h, --help", msgs.Help))
	buf.WriteString(s.colors.Option.Sprintf("  %-20s%s\n", "-v, --version", msgs.Version))

	// 显示所有参数
	s.paramMgr.mu.RLock()
	for _, name := range s.paramMgr.paramOrder {
		p := s.paramMgr.params[name]
		flags := make([]string, 0, 2)
		if p.Short != "" {
			flags = append(flags, "-"+strings.TrimPrefix(p.Short, "-"))
		}
		if p.Long != "" {
			flags = append(flags, "--"+strings.TrimPrefix(p.Long, "--"))
		}

		buf.WriteString(s.colors.Option.Sprintf("  %-20s%s%s%s\n",
			strings.Join(flags, ", "),
			p.Description,
			map[bool]string{true: fmt.Sprintf(" (%s)", msgs.Required), false: ""}[p.Required],
			map[bool]string{true: fmt.Sprintf(" (%s: %s)", msgs.Default, p.Default), false: ""}[p.Default != ""]))
	}
	s.paramMgr.mu.RUnlock()
	buf.WriteString("\n")

	// 命令部分
	buf.WriteString(s.colors.CommandsTitle.Sprint(msgs.Commands + "\n"))

	// 系统服务命令
	defaultCommands := []struct{ cmd, desc string }{
		{"install", msgs.Install},
		{"uninstall", msgs.Uninstall},
		{"start", msgs.Start},
		{"stop", msgs.Stop},
		{"restart", msgs.Restart},
		{"status", msgs.Status},
	}

	// 显示默认命令
	for _, cmd := range defaultCommands {
		buf.WriteString(s.colors.Command.Sprintf("  %-20s%s\n", cmd.cmd, cmd.desc))
	}

	// 显示自定义命令
	s.paramMgr.mu.RLock()
	for _, cmd := range s.paramMgr.commands {
		if !cmd.Hidden {
			buf.WriteString(s.colors.Command.Sprintf("  %-20s%s\n", cmd.Name, cmd.Description))
		}
	}
	s.paramMgr.mu.RUnlock()

	buf.WriteString("\n")

	// Examples section
	buf.WriteString(s.colors.ExamplesTitle.Sprint(msgs.Example + "\n"))
	buf.WriteString(s.colors.Example.Sprintf("  %s -h\n", os.Args[0]))
	buf.WriteString(s.colors.Example.Sprintf("  %s -v\n", os.Args[0]))

	fmt.Print(buf.String())
}

// handleCommand 处理命令
func (s *Service) handleCommand(cmd string) error {
	msgs := s.getMessage()

	// 初始化命令列表
	var cmdlist []string

	// 添加系统服务命令
	for _, v := range service.ControlAction {
		cmdlist = append(cmdlist, v)
	}
	cmdlist = append(cmdlist, "status")

	// 添加自定义命令
	s.paramMgr.mu.RLock()
	for cmdName := range s.paramMgr.commands {
		cmdlist = append(cmdlist, cmdName)
	}
	s.paramMgr.mu.RUnlock()

	// 首先验证命令是否存在
	validCmd := false
	for _, c := range cmdlist {
		if c == cmd {
			validCmd = true
			break
		}
	}

	if !validCmd {
		return fmt.Errorf("unknown command: %s\nAvailable commands: %s", cmd, strings.Join(cmdlist, ", "))
	}

	switch cmd {
	case "install":
		if err := s.svc.Install(); err != nil {
			return err
		}
		if err := s.svc.Start(); err != nil {
			return err
		}
		s.colors.Success.Printf("√ %s\n", msgs.Install)

	case "uninstall":
		_ = s.svc.Stop()
		if err := s.svc.Uninstall(); err != nil {
			return err
		}
		s.colors.Success.Printf("√ %s\n", msgs.Uninstall)

	case "start":
		if err := s.svc.Start(); err != nil {
			return err
		}
		s.colors.Success.Printf("√ %s\n", msgs.Start)

	case "stop":
		if err := s.svc.Stop(); err != nil {
			return err
		}
		s.colors.Success.Printf("√ %s\n", msgs.Stop)

	case "restart":
		if err := s.svc.Restart(); err != nil {
			return err
		}
		s.colors.Success.Printf("√ %s\n", msgs.Restart)

	case "status":
		status, err := s.svc.Status()
		if err != nil {
			return err
		}
		s.status.Store(status)
		s.printStatus(msgs)

	default:
		// 处理自定义命令
		s.paramMgr.mu.RLock()
		if cmd, exists := s.paramMgr.commands[cmd]; exists {
			s.paramMgr.mu.RUnlock()
			if cmd.Run != nil {
				cmd.Run()
				return nil
			}
			return fmt.Errorf("command %s has no Run function defined", cmd.Name)
		}
		s.paramMgr.mu.RUnlock()
	}

	return nil
}

// printStatus 打印状态信息
func (s *Service) printStatus(msgs Messages) {
	status := s.status.Load().(service.Status)
	statusText := map[service.Status]string{
		service.StatusRunning: msgs.Running,
		service.StatusStopped: msgs.Stopped,
	}[status]

	if statusText == "" {
		statusText = msgs.Unknown
	}

	switch statusText {
	case msgs.Running:
		s.colors.Success.Printf(msgs.StatusFormat+"\n", s.opts.Name, statusText)
	case msgs.Stopped:
		s.colors.Error.Printf(msgs.StatusFormat+"\n", s.opts.Name, statusText)
	default:
		s.colors.Info.Printf(msgs.StatusFormat+"\n", s.opts.Name, statusText)
	}
}

// SetParamManager 注册参数到参数管理器
func (s *Service) SetParamManager(paramMgr *manager) {
	s.paramMgr = paramMgr
}

// ParamManager 返回参数管理器
func (s *Service) ParamManager() *manager {
	return s.paramMgr
}

// AddLanguage 添加新的语言支持
func (s *Service) AddLanguage(lang string, msgs Messages) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs[lang] = msgs
}

// SetLanguage 设置当前语言
func (s *Service) SetLanguage(lang string) bool {
	if _, ok := s.msgs[lang]; !ok {
		return false
	}
	s.opts.Language = lang
	return true
}

// GetCurrentLanguage 获取当前语言
func (s *Service) GetCurrentLanguage() string {
	if s.opts.Language == "" {
		return "en"
	}
	return s.opts.Language
}

// GetBuildInfo 获取构建信息
func (s *Service) GetBuildInfo() *BuildInfo {
	return s.opts.BuildInfo
}

// SetBuildInfo 设置构建信息
func (s *Service) SetBuildInfo(bi *BuildInfo) {
	s.opts.BuildInfo = bi
}

// GetStatus 获取服务状态
func (s *Service) GetStatus() (service.Status, error) {
	status, err := s.svc.Status()
	if err != nil {
		return service.StatusUnknown, err
	}
	s.status.Store(status)
	return status, nil
}

// GetStatusText 获取服务状态文本
func (s *Service) GetStatusText() string {
	msgs := s.getMessage()
	status, err := s.GetStatus()
	if err != nil {
		return msgs.Unknown
	}

	switch status {
	case service.StatusRunning:
		return msgs.Running
	case service.StatusStopped:
		return msgs.Stopped
	default:
		return msgs.Unknown
	}
}

// EnableDebug 启用调试模式
func (s *Service) EnableDebug() {
	if s.opts.BuildInfo != nil {
		s.opts.BuildInfo.Debug.Store(true)
	}
}

// DisableDebug 禁用调试模式
func (s *Service) DisableDebug() {
	if s.opts.BuildInfo != nil {
		s.opts.BuildInfo.Debug.Store(false)
	}
}

// IsDebug 是否为调试模式
func (s *Service) IsDebug() bool {
	return s.opts.BuildInfo != nil && s.opts.BuildInfo.Debug.Load()
}

// SetLogo 设置Logo
func (s *Service) SetLogo(logo string) {
	s.opts.Logo = logo
}

// GetLogo 获取Logo
func (s *Service) GetLogo() string {
	return s.opts.Logo
}

// SetVersion 设置版本号
func (s *Service) SetVersion(version string) {
	s.opts.Version = version
	if s.opts.BuildInfo != nil {
		s.opts.BuildInfo.Version = version
	}
}

// GetVersion 获取版本号
func (s *Service) GetVersion() string {
	if s.opts.BuildInfo != nil {
		return s.opts.BuildInfo.Version
	}
	return s.opts.Version
}

// UpdateBuildInfo 更新构建信息
func (s *Service) UpdateBuildInfo(updates map[string]interface{}) error {
	if s.opts.BuildInfo == nil {
		return fmt.Errorf("BuildInfo is not initialized")
	}

	for k, v := range updates {
		switch k {
		case "Version":
			if str, ok := v.(string); ok {
				s.opts.BuildInfo.Version = str
			}
		case "GitCommit":
			if str, ok := v.(string); ok {
				s.opts.BuildInfo.GitCommit = str
			}
		case "GitBranch":
			if str, ok := v.(string); ok {
				s.opts.BuildInfo.GitBranch = str
			}
		case "GitTag":
			if str, ok := v.(string); ok {
				s.opts.BuildInfo.GitTag = str
			}
		case "BuildTime":
			if t, ok := v.(time.Time); ok {
				s.opts.BuildInfo.BuildTime = t
			}
		case "Debug":
			if b, ok := v.(bool); ok {
				s.opts.BuildInfo.Debug.Store(b)
			}
		default:
			return fmt.Errorf("unknown BuildInfo field: %s", k)
		}
	}
	return nil
}

// Reload 重新加载配置
func (s *Service) Reload() error {
	if err := s.paramMgr.Parse(); err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}
	return nil
}

// String 返回服务信息字符串
func (s *Service) String() string {
	var b strings.Builder
	b.Grow(512)

	fmt.Fprintf(&b, "Service: %s\n", s.opts.Name)
	fmt.Fprintf(&b, "Display Name: %s\n", s.opts.DisplayName)
	fmt.Fprintf(&b, "Description: %s\n", s.opts.Description)
	fmt.Fprintf(&b, "Language: %s\n", s.GetCurrentLanguage())

	if s.opts.BuildInfo != nil {
		b.WriteString("\nBuild Information:\n")
		b.WriteString(s.opts.BuildInfo.String())
	}

	fmt.Fprintf(&b, "\nStatus: %s\n", s.GetStatusText())

	return b.String()
}

// Close 关闭服务并清理资源
func (s *Service) Close() error {
	colorPool.Put(s.colors)
	return nil
}

// validateOptions 验证选项
func validateOptions(opts *Options) error {
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}
	if opts.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if opts.Run == nil {
		return fmt.Errorf("run function is required")
	}
	if opts.DisplayName == "" {
		opts.DisplayName = opts.Name // 使用服务名作为显示名
	}
	if opts.Version == "" {
		opts.Version = "1.0.0" // 设置默认版本号
	}
	if opts.Language == "" {
		opts.Language = "en" // 设置默认语言
	}
	if opts.BuildInfo == nil {
		opts.BuildInfo = NewBuildInfo()
	}
	return nil
}

// isColorSupported 检查系统是否支持彩色输出
func isColorSupported() bool {
	if runtime.GOOS != "windows" {
		return true
	}

	colorSupportEnvs := []string{
		"WT_SESSION",          // Windows Terminal
		"TERM_PROGRAM=vscode", // VS Code terminal
		"ConEmuANSI=ON",       // ConEmu
		"ANSICON",             // ANSICON
		"TERM=xterm",          // Git Bash
		"TERM=cygwin",         // Cygwin
		"CI",                  // CI environment
	}

	for _, env := range colorSupportEnvs {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			if os.Getenv(parts[0]) == parts[1] {
				return true
			}
		} else if os.Getenv(env) != "" {
			return true
		}
	}

	return false
}
