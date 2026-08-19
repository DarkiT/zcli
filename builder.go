package zcli

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"
)

// CommandFactory 在 CLI 实例就绪后构造命令。
// 用于需要读取 App/Config/Flag 导出能力的命令，避免 Build 后再 Find/Remove/Add。
type CommandFactory func(app *Cli) *Command

type flagSetup struct {
	persistent bool
	configure  func(flags *FlagSet)
}

// Builder CLI构建器 - 增强版
type Builder struct {
	config       *Config
	cli          *Cli
	pendingCmds  []*Command
	cmdFactories []CommandFactory
	flagSetups   []flagSetup
	validators   []func(*Config) error
	buildErrs    []error
	built        bool
	service      ServiceRunner
	initHooks    []InitHook
}

// InitHook 在命令执行前执行的初始化钩子
// 返回 error 则中断命令执行
type InitHook func(cmd *Command, args []string) error

// NewBuilder 创建CLI构建器
// lang参数可选，指定默认语言
func NewBuilder(lang ...string) *Builder {
	b := &Builder{
		config: NewConfig(),
	}

	// 如果提供了语言参数，设置默认语言
	if len(lang) > 0 && lang[0] != "" {
		b.WithLanguage(lang[0])
	}

	return b
}

// WithDefaultConfig 使用默认配置
func (b *Builder) WithDefaultConfig() *Builder {
	b.WithLanguage("en").
		WithVersion("1.0.0").
		WithDebug(false)
	return b
}

// WithName 设置名称
func (b *Builder) WithName(name string) *Builder {
	b.config.basic.Name = name
	return b
}

// WithDisplayName 设置显示名称
func (b *Builder) WithDisplayName(name string) *Builder {
	b.config.basic.DisplayName = name
	return b
}

// WithDescription 设置描述
func (b *Builder) WithDescription(desc string) *Builder {
	b.config.basic.Description = desc
	return b
}

// WithVersion 设置版本
func (b *Builder) WithVersion(version string) *Builder {
	b.ensureBuildInfo().Version = version
	b.config.basic.Version = version
	return b
}

// WithLogo 设置Logo
func (b *Builder) WithLogo(logo string) *Builder {
	b.config.basic.Logo = logo
	return b
}

// WithLanguage 设置语言
func (b *Builder) WithLanguage(lang string) *Builder {
	b.config.basic.Language = lang
	return b
}

// WithCustomService 配置服务（高级用法）
// 允许传入自定义配置函数来修改配置
func (b *Builder) WithCustomService(fn func(*Config)) *Builder {
	if fn == nil {
		return b
	}
	fn(b.config)
	return b
}

// WithServiceConfig 直接配置服务扩展字段。
func (b *Builder) WithServiceConfig(fn func(*ServiceConfig)) *Builder {
	if fn == nil {
		return b
	}
	fn(b.config.service)
	return b
}

// WithCommand 添加命令
func (b *Builder) WithCommand(cmd *Command) *Builder {
	if cmd == nil {
		return b
	}
	if b.built && b.cli != nil {
		b.cli.AddCommand(cmd)
		return b
	}
	b.pendingCmds = append(b.pendingCmds, cmd)
	return b
}

// WithCommands 批量添加命令，语义与多次 WithCommand 相同。
func (b *Builder) WithCommands(cmds ...*Command) *Builder {
	for _, cmd := range cmds {
		b.WithCommand(cmd)
	}
	return b
}

// WithCommandFactory 注册需要 *Cli 才能构造的命令。
func (b *Builder) WithCommandFactory(factory CommandFactory) *Builder {
	if factory == nil {
		return b
	}
	if b.built && b.cli != nil {
		if cmd := factory(b.cli); cmd != nil {
			b.cli.AddCommand(cmd)
		}
		return b
	}
	b.cmdFactories = append(b.cmdFactories, factory)
	return b
}

// WithInitHook 注册初始化钩子
func (b *Builder) WithInitHook(hook InitHook) *Builder {
	if hook != nil {
		b.initHooks = append(b.initHooks, hook)
	}
	return b
}

// WithGitInfo 设置Git信息
func (b *Builder) WithGitInfo(commitID, branch, tag string) *Builder {
	info := b.ensureBuildInfo()
	info.GitCommit = commitID
	info.GitBranch = branch
	info.GitTag = tag
	return b
}

// WithDebug 设置调试模式
func (b *Builder) WithDebug(debug bool) *Builder {
	b.ensureBuildInfo().Debug.Store(debug)
	return b
}

// WithWorkDir 设置工作目录
func (b *Builder) WithWorkDir(dir string) *Builder {
	b.config.service.WorkDir = dir
	return b
}

// WithServiceUser 设置系统服务运行用户。
func (b *Builder) WithServiceUser(username string) *Builder {
	b.config.service.Username = username
	return b
}

// WithExecutable 设置系统服务安装时使用的可执行文件路径。
func (b *Builder) WithExecutable(path string) *Builder {
	b.config.service.Executable = path
	return b
}

// WithArguments 设置系统服务运行参数。
// 传入空参数列表会显式清空默认的 "run" 参数。
func (b *Builder) WithArguments(args ...string) *Builder {
	b.config.service.Arguments = append([]string(nil), args...)
	return b
}

// WithChRoot 设置 chroot 目录。
func (b *Builder) WithChRoot(dir string) *Builder {
	b.config.service.ChRoot = dir
	return b
}

// WithBuildTime 设置构建时间。
// 接受 time.DateTime 或 RFC3339；无法解析时累积到 BuildWithError，不再静默丢弃。
func (b *Builder) WithBuildTime(buildTime string) *Builder {
	if buildTime == "" {
		return b
	}
	parsedTime, err := parseBuildTime(buildTime)
	if err != nil {
		b.addBuildError(fmt.Errorf("invalid build time %q: %w", buildTime, err))
		return b
	}
	b.ensureBuildInfo().BuildTime = parsedTime
	return b
}

// WithEnvVar 设置环境变量
func (b *Builder) WithEnvVar(key, value string) *Builder {
	if key == "" {
		return b
	}
	if b.config.service.EnvVars == nil {
		b.config.service.EnvVars = make(map[string]string)
	}
	b.config.service.EnvVars[key] = value
	return b
}

// WithEnvVars 批量合并环境变量，空 key 会被忽略。
func (b *Builder) WithEnvVars(vars map[string]string) *Builder {
	for key, value := range vars {
		b.WithEnvVar(key, value)
	}
	return b
}

// WithDependencies 设置依赖
func (b *Builder) WithDependencies(deps ...string) *Builder {
	structured := make([]Dependency, 0, len(deps))
	for _, dep := range deps {
		if dep == "" {
			continue
		}
		structured = append(structured, Dependency{Name: dep, Type: DependencyRequire})
	}
	b.config.service.StructuredDeps = structured
	b.config.service.Dependencies = nil
	return b
}

// WithLegacyDependencies 设置 daemon 原生字符串依赖。
func (b *Builder) WithLegacyDependencies(deps ...string) *Builder {
	b.config.service.Dependencies = append([]string(nil), deps...)
	b.config.service.StructuredDeps = nil
	return b
}

// WithStructuredDependencies 追加结构化依赖，语义与多次 WithDependency 相同。
// 若要覆盖已有列表，使用 SetStructuredDependencies。
func (b *Builder) WithStructuredDependencies(deps ...Dependency) *Builder {
	b.config.service.Dependencies = nil
	if len(deps) == 0 {
		return b
	}
	b.config.service.StructuredDeps = append(b.config.service.StructuredDeps, deps...)
	return b
}

// SetStructuredDependencies 覆盖结构化依赖列表。
func (b *Builder) SetStructuredDependencies(deps ...Dependency) *Builder {
	b.config.service.Dependencies = nil
	b.config.service.StructuredDeps = append([]Dependency(nil), deps...)
	return b
}

// WithDependency 追加单个结构化依赖。
func (b *Builder) WithDependency(name string, depType DependencyType) *Builder {
	if name == "" {
		return b
	}
	b.config.service.Dependencies = nil
	b.config.service.StructuredDeps = append(
		b.config.service.StructuredDeps,
		Dependency{Name: name, Type: depType},
	)
	return b
}

// WithServiceOption 设置单个 daemon 平台选项。
func (b *Builder) WithServiceOption(key string, value any) *Builder {
	if key == "" {
		return b
	}
	if b.config.service.Options == nil {
		b.config.service.Options = make(ServiceOptions)
	}
	b.config.service.Options[key] = value
	return b
}

// WithServiceOptionsMap 批量合并 daemon 平台选项。
func (b *Builder) WithServiceOptionsMap(options ServiceOptions) *Builder {
	if len(options) == 0 {
		return b
	}
	if b.config.service.Options == nil {
		b.config.service.Options = make(ServiceOptions)
	}
	maps.Copy(b.config.service.Options, options)
	return b
}

// WithAllowSudoFallback 设置 daemon 的 sudo/su 回退策略。
func (b *Builder) WithAllowSudoFallback(enabled bool) *Builder {
	b.config.service.AllowSudoFallback = enabled
	return b
}

// WithForce 设置 Install 覆盖已有服务配置，对齐 daemon Config.Force。
func (b *Builder) WithForce(enabled bool) *Builder {
	b.config.service.Force = enabled
	return b
}

// WithNotify 启用 systemd Type=notify / Windows 就绪通知。
func (b *Builder) WithNotify(enabled bool) *Builder {
	b.config.service.Notify = enabled
	return b
}

// WithRuntime 设置运行时配置
func (b *Builder) WithRuntime(rt *Runtime) *Builder {
	if rt == nil {
		b.addBuildError(errors.New("runtime cannot be nil"))
		return b
	}
	b.config.runtime = rt
	return b
}

// WithService 配置服务运行和停止函数（主入口）
// run: 服务运行函数，标准签名 func(ctx context.Context) error
// stop: 可选停止函数，标准签名 func() error
func (b *Builder) WithService(run RunFunc, stop ...StopFunc) *Builder {
	if run == nil {
		b.addBuildError(errors.New("service run function cannot be nil"))
		b.service = nil
		b.ensureRuntime().Run = nil
		b.ensureRuntime().Stop = nil
		return b
	}
	rt := b.ensureRuntime()
	rt.Run = run
	if len(stop) > 0 {
		rt.Stop = stop[0]
	} else {
		rt.Stop = nil
	}
	b.service = nil
	return b
}

// WithServiceRunner 配置优雅的服务接口（推荐方式）
func (b *Builder) WithServiceRunner(service ServiceRunner) *Builder {
	if service == nil {
		b.addBuildError(errors.New("service cannot be nil"))
		b.service = nil
		rt := b.ensureRuntime()
		rt.Run = nil
		rt.Stop = nil
		return b
	}
	b.service = service
	rt := b.ensureRuntime()
	rt.Run = service.Run
	rt.Stop = service.Stop
	return b
}

// WithShutdownTimeouts 配置优雅退出的分级超时时间
// initial: 首次等待时长；grace: 调用停止函数后的额外等待时长
func (b *Builder) WithShutdownTimeouts(initial, grace time.Duration) *Builder {
	rt := b.ensureRuntime()
	rt.ShutdownInitial = initial
	rt.ShutdownGrace = grace
	return b
}

// WithServiceTimeouts 配置服务启动/停止超时（写入 daemon Config.Timeout）
func (b *Builder) WithServiceTimeouts(start, stop time.Duration) *Builder {
	rt := b.ensureRuntime()
	rt.StartTimeout = start
	rt.StopTimeout = stop
	rt.daemonStopTimeoutSet = true
	return b
}

// WithRestartTimeout 设置 daemon 重启间隔 Timeout.Restart。
func (b *Builder) WithRestartTimeout(d time.Duration) *Builder {
	b.ensureRuntime().RestartTimeout = d
	return b
}

// WithMousetrapDisabled 禁用 Windows 双击运行提示
func (b *Builder) WithMousetrapDisabled(disabled bool) *Builder {
	b.config.basic.MousetrapDisabled = disabled
	return b
}

// WithErrorHandler 添加错误处理器
func (b *Builder) WithErrorHandler(handler ErrorHandler) *Builder {
	if handler != nil {
		rt := b.ensureRuntime()
		rt.ErrorHandlers = append(rt.ErrorHandlers, handler)
	}
	return b
}

// WithValidator 添加配置验证器
func (b *Builder) WithValidator(validator func(*Config) error) *Builder {
	if validator == nil {
		return b
	}
	b.validators = append(b.validators, validator)
	return b
}

// WithContext 设置构建与执行使用的根上下文。
func (b *Builder) WithContext(ctx context.Context) *Builder {
	if ctx == nil {
		ctx = context.Background()
	}
	b.config.ctx = ctx
	return b
}

// WithSilenceErrors 设置是否静默错误输出。
func (b *Builder) WithSilenceErrors(enabled bool) *Builder {
	b.config.basic.SilenceErrors = enabled
	return b
}

// WithSilenceUsage 设置是否静默用法输出。
func (b *Builder) WithSilenceUsage(enabled bool) *Builder {
	b.config.basic.SilenceUsage = enabled
	return b
}

// WithNoColor 设置是否禁用彩色输出。
func (b *Builder) WithNoColor(enabled bool) *Builder {
	b.config.basic.NoColor = enabled
	return b
}

// WithRootAction 设置无子命令时根命令的默认动作。
func (b *Builder) WithRootAction(action RootAction) *Builder {
	b.config.basic.RootAction = action
	return b
}

// WithPersistentFlags 在构建阶段配置根命令持久标志。
func (b *Builder) WithPersistentFlags(configure func(flags *FlagSet)) *Builder {
	return b.addFlagSetup(true, configure)
}

// WithFlags 在构建阶段配置根命令本地标志。
func (b *Builder) WithFlags(configure func(flags *FlagSet)) *Builder {
	return b.addFlagSetup(false, configure)
}

func (b *Builder) addFlagSetup(persistent bool, configure func(flags *FlagSet)) *Builder {
	if configure == nil {
		return b
	}
	if b.built && b.cli != nil {
		if persistent {
			configure(b.cli.PersistentFlags())
		} else {
			configure(b.cli.Flags())
		}
		return b
	}
	b.flagSetups = append(b.flagSetups, flagSetup{persistent: persistent, configure: configure})
	return b
}

// Build 构建CLI实例
func (b *Builder) Build() *Cli {
	cli, err := b.BuildWithError()
	if err != nil {
		panic(fmt.Sprintf("build failed: %v", err))
	}
	return cli
}

// BuildWithError 构建CLI实例，返回错误而不是panic
func (b *Builder) BuildWithError() (*Cli, error) {
	if b.built {
		return b.cli, nil
	}

	b.syncServiceRuntime()

	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	cli, err := b.finalizeCLI()
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}
	b.cli = cli
	b.built = true
	return b.cli, nil
}

func (b *Builder) syncServiceRuntime() {
	if b.service == nil {
		return
	}
	rt := b.ensureRuntime()
	rt.Run = b.service.Run
	rt.Stop = b.service.Stop
}

func (b *Builder) finalizeCLI() (*Cli, error) {
	cli := b.cli
	if cli == nil {
		cli = newAppBase(b.config)
	}

	pending := append([]*Command(nil), b.pendingCmds...)
	for _, factory := range b.cmdFactories {
		if factory == nil {
			continue
		}
		if cmd := factory(cli); cmd != nil {
			pending = append(pending, cmd)
		}
	}

	for _, setup := range b.flagSetups {
		if setup.configure == nil {
			continue
		}
		if setup.persistent {
			setup.configure(cli.PersistentFlags())
		} else {
			setup.configure(cli.Flags())
		}
	}

	if err := cli.applyBuilderAssembly(pending, b.initHooks); err != nil {
		return nil, err
	}

	b.pendingCmds = nil
	b.cmdFactories = nil
	b.flagSetups = nil
	b.initHooks = nil
	return cli, nil
}

func (b *Builder) ensureRuntime() *Runtime {
	if b.config.runtime == nil {
		b.config.runtime = &Runtime{
			ShutdownInitial: 15 * time.Second,
			ShutdownGrace:   5 * time.Second,
			StopTimeout:     20 * time.Second,
		}
	}
	return b.config.runtime
}

func (b *Builder) ensureBuildInfo() *VersionInfo {
	rt := b.ensureRuntime()
	if rt.BuildInfo == nil {
		rt.BuildInfo = NewVersion()
	}
	return rt.BuildInfo
}

func (b *Builder) addBuildError(err error) {
	if err != nil {
		b.buildErrs = append(b.buildErrs, err)
	}
}

func (b *Builder) syncServiceIdentity() {
	if b.config == nil || b.config.service == nil || b.config.basic == nil {
		return
	}
	if b.config.service.Name == "" {
		b.config.service.Name = b.config.basic.Name
	}
	if b.config.service.DisplayName == "" {
		b.config.service.DisplayName = b.config.basic.DisplayName
	}
	if b.config.service.Description == "" {
		b.config.service.Description = b.config.basic.Description
	}
	if b.config.service.Version == "" {
		b.config.service.Version = b.config.basic.Version
	}
}

func parseBuildTime(value string) (time.Time, error) {
	for _, layout := range []string{time.DateTime, time.RFC3339, time.RFC3339Nano} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("must be %q or RFC3339", time.DateTime)
}

// validate 验证配置
func (b *Builder) validate() error {
	var errs []error

	if b.config == nil || b.config.basic == nil {
		errs = append(errs, errors.New("application configuration is required"))
		return &BuildError{Errors: append(errs, b.buildErrs...)}
	}

	if b.config.basic.Name == "" {
		errs = append(errs, errors.New("application name is required"))
	} else if strings.TrimSpace(b.config.basic.Name) != b.config.basic.Name ||
		strings.ContainsAny(b.config.basic.Name, "/\\\x00\t\n\r ") {
		errs = append(errs, errors.New("application name must not contain spaces or path separators"))
	}

	errs = append(errs, b.buildErrs...)

	if rt := b.config.runtime; rt != nil {
		if rt.ShutdownInitial < 0 || rt.ShutdownGrace < 0 {
			errs = append(errs, errors.New("shutdown timeouts cannot be negative"))
		}
		if rt.StartTimeout < 0 || rt.StopTimeout < 0 || rt.RestartTimeout < 0 {
			errs = append(errs, errors.New("service timeouts cannot be negative"))
		}
		if rt.Run != nil {
			b.syncServiceIdentity()
			if b.config.service != nil {
				if err := b.config.service.Validate(); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	for i, validator := range b.validators {
		if validator == nil {
			continue
		}
		if err := validator(b.config); err != nil {
			errs = append(errs, fmt.Errorf("validator %d failed: %w", i+1, err))
		}
	}

	if len(errs) > 0 {
		return &BuildError{Errors: errs}
	}
	return nil
}

// BuildError 构建错误
type BuildError struct {
	Errors []error
}

func (be *BuildError) Error() string {
	if be == nil || len(be.Errors) == 0 {
		return "build failed"
	}
	if len(be.Errors) == 1 {
		return be.Errors[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "build failed, %d error(s):", len(be.Errors))
	for i, err := range be.Errors {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, err.Error())
	}
	return b.String()
}

func (be *BuildError) Unwrap() []error {
	if be == nil {
		return nil
	}
	return be.Errors
}

// ===================================================================
// 便利性API - 快速构建常见场景
// ===================================================================

// QuickService 快速创建服务应用（新签名）
func QuickService(name, displayName string, run RunFunc) *Cli {
	return NewBuilder("en").
		WithName(name).
		WithDisplayName(displayName).
		WithService(run).
		Build()
}

// QuickServiceWithStop 快速创建带停止函数的服务应用
func QuickServiceWithStop(name, displayName string, run RunFunc, stop StopFunc) *Cli {
	return NewBuilder("en").
		WithName(name).
		WithDisplayName(displayName).
		WithService(run, stop).
		Build()
}

// QuickCLI 快速创建基础CLI应用
func QuickCLI(name, displayName, description string) *Cli {
	return NewBuilder("en").
		WithName(name).
		WithDisplayName(displayName).
		WithDescription(description).
		Build()
}

// ===================================================================
// 链式配置增强
// ===================================================================

// WithDefaults 设置默认配置
func (b *Builder) WithDefaults() *Builder {
	if b.config.basic.Language == "" {
		b.config.basic.Language = "en"
	}
	if b.config.basic.Version == "" {
		b.config.basic.Version = "1.0.0"
	}
	return b
}

// WithQuickConfig 快速配置基本信息
func (b *Builder) WithQuickConfig(name, displayName, description, version string) *Builder {
	return b.WithName(name).
		WithDisplayName(displayName).
		WithDescription(description).
		WithVersion(version)
}

// WithServiceOptions 配置服务选项
func (b *Builder) WithServiceOptions(workDir string, envVars map[string]string, deps ...string) *Builder {
	if workDir != "" {
		b.WithWorkDir(workDir)
	}
	for k, v := range envVars {
		b.WithEnvVar(k, v)
	}
	if len(deps) > 0 {
		b.WithDependencies(deps...)
	}
	return b
}

// MustBuild 构建CLI实例，失败时panic（用于确定不会失败的场景）
func (b *Builder) MustBuild() *Cli {
	cli, err := b.BuildWithError()
	if err != nil {
		panic(fmt.Sprintf("MustBuild failed: %v", err))
	}
	return cli
}
