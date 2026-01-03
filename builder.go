package zcli

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Builder CLI构建器 - 增强版
type Builder struct {
	config     *Config
	cli        *Cli
	validators []func(*Config) error
	built      bool
	service    ServiceRunner // 新增：支持ServiceRunner接口
}

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
	if b.config.runtime.BuildInfo == nil {
		b.config.runtime.BuildInfo = NewVersion()
	}
	b.config.runtime.BuildInfo.Version = version
	b.config.basic.Version = version // 同步更新基础配置
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
	fn(b.config)
	return b
}

// WithCommand 添加命令
func (b *Builder) WithCommand(cmd *Command) *Builder {
	if b.cli == nil {
		b.Build()
	}
	b.cli.AddCommand(cmd)
	return b
}

// WithGitInfo 设置Git信息
func (b *Builder) WithGitInfo(commitID, branch, tag string) *Builder {
	if b.config.runtime.BuildInfo == nil {
		b.config.runtime.BuildInfo = NewVersion()
	}
	b.config.runtime.BuildInfo.GitCommit = commitID
	b.config.runtime.BuildInfo.GitBranch = branch
	b.config.runtime.BuildInfo.GitTag = tag
	return b
}

// WithDebug 设置调试模式
func (b *Builder) WithDebug(debug bool) *Builder {
	if b.config.runtime.BuildInfo == nil {
		b.config.runtime.BuildInfo = NewVersion()
	}
	b.config.runtime.BuildInfo.Debug.Store(debug)
	return b
}

// WithWorkDir 设置工作目录
func (b *Builder) WithWorkDir(dir string) *Builder {
	b.config.service.WorkDir = dir
	return b
}

// WithBuildTime 设置构建时间
func (b *Builder) WithBuildTime(buildTime string) *Builder {
	parsedTime, err := time.Parse(time.DateTime, buildTime)
	if err == nil {
		b.config.runtime.BuildInfo.BuildTime = parsedTime
	}
	return b
}

// WithEnvVar 设置环境变量
func (b *Builder) WithEnvVar(key, value string) *Builder {
	if b.config.service.EnvVars == nil {
		b.config.service.EnvVars = make(map[string]string)
	}
	b.config.service.EnvVars[key] = value
	return b
}

// WithDependencies 设置依赖
func (b *Builder) WithDependencies(deps ...string) *Builder {
	b.config.service.Dependencies = deps
	return b
}

// WithRuntime 设置运行时配置
func (b *Builder) WithRuntime(rt *Runtime) *Builder {
	b.config.runtime = rt
	return b
}

// WithService 配置服务运行和停止函数（主入口）
// run: 服务运行函数，标准签名 func(ctx context.Context) error
// stop: 可选停止函数，标准签名 func() error
func (b *Builder) WithService(run RunFunc, stop ...StopFunc) *Builder {
	b.config.runtime.Run = run
	if len(stop) > 0 {
		b.config.runtime.Stop = stop[0]
	} else {
		b.config.runtime.Stop = nil
	}
	// 重置 ServiceRunner，保持与新的 run/stop 同步
	b.service = nil
	return b
}

// WithServiceRunner 配置优雅的服务接口（推荐方式）
func (b *Builder) WithServiceRunner(service ServiceRunner) *Builder {
	if service == nil {
		panic("service cannot be nil")
	}
	b.service = service

	// 将 ServiceRunner 直接赋值为新的标准签名
	b.config.runtime.Run = service.Run
	b.config.runtime.Stop = service.Stop

	return b
}

// WithShutdownTimeouts 配置优雅退出的分级超时时间
// initial: 首次等待时长；grace: 调用停止函数后的额外等待时长
func (b *Builder) WithShutdownTimeouts(initial, grace time.Duration) *Builder {
	b.config.runtime.ShutdownInitial = initial
	b.config.runtime.ShutdownGrace = grace
	return b
}

// WithServiceTimeouts 配置服务启动/停止超时（写入 daemon Config.Timeout）
func (b *Builder) WithServiceTimeouts(start, stop time.Duration) *Builder {
	b.config.runtime.StartTimeout = start
	b.config.runtime.StopTimeout = stop
	return b
}


// WithValidator 添加配置验证器
func (b *Builder) WithValidator(validator func(*Config) error) *Builder {
	b.validators = append(b.validators, validator)
	return b
}

// WithContext 设置名称
func (b *Builder) WithContext(ctx context.Context) *Builder {
	b.config.ctx = ctx
	return b
}

// Build 构建CLI实例
func (b *Builder) Build() *Cli {
	if b.built {
		return b.cli
	}

	// 若使用 ServiceRunner，确保 runtime 与之同步
	if b.service != nil {
		b.config.runtime.Run = b.service.Run
		b.config.runtime.Stop = b.service.Stop
	}

	// 执行验证
	if err := b.validate(); err != nil {
		panic(fmt.Sprintf("build failed: %v", err))
	}

	if b.cli == nil {
		b.cli = NewCli(WithConfig(b.config))
	}

	b.built = true
	return b.cli
}

// BuildWithError 构建CLI实例，返回错误而不是panic
func (b *Builder) BuildWithError() (*Cli, error) {
	if b.built {
		return b.cli, nil
	}

	if b.service != nil {
		b.config.runtime.Run = b.service.Run
		b.config.runtime.Stop = b.service.Stop
	}

	// 执行验证
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	if b.cli == nil {
		b.cli = NewCli(WithConfig(b.config))
	}

	b.built = true
	return b.cli, nil
}

// validate 验证配置
func (b *Builder) validate() error {
	var errs []error

	// 基本验证
	if b.config.basic.Name == "" {
		errs = append(errs, errors.New("application name is required"))
	}

	// 服务相关验证
	if b.config.runtime.Run != nil && b.config.basic.Name == "" {
		errs = append(errs, errors.New("service name must be set when service is configured"))
	}

	// 执行自定义验证器
	for i, validator := range b.validators {
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
	if len(be.Errors) == 1 {
		return be.Errors[0].Error()
	}
	return fmt.Sprintf("build failed, %d error(s): %v", len(be.Errors), be.Errors[0])
}

func (be *BuildError) Unwrap() []error {
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
