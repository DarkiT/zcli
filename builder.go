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
	b.WithLanguage("zh").
		WithVersion("1.0.0").
		WithDebug(false)
	return b
}

// WithName 设置名称
func (b *Builder) WithName(name string) *Builder {
	b.config.Basic.Name = name
	return b
}

// WithDisplayName 设置显示名称
func (b *Builder) WithDisplayName(name string) *Builder {
	b.config.Basic.DisplayName = name
	return b
}

// WithDescription 设置描述
func (b *Builder) WithDescription(desc string) *Builder {
	b.config.Basic.Description = desc
	return b
}

// WithVersion 设置版本
func (b *Builder) WithVersion(version string) *Builder {
	if b.config.Runtime.BuildInfo == nil {
		b.config.Runtime.BuildInfo = NewVersion()
	}
	b.config.Runtime.BuildInfo.Version = version
	b.config.Basic.Version = version // 同步更新基础配置
	return b
}

// WithLogo 设置Logo
func (b *Builder) WithLogo(logo string) *Builder {
	b.config.Basic.Logo = logo
	return b
}

// WithLanguage 设置语言
func (b *Builder) WithLanguage(lang string) *Builder {
	b.config.Basic.Language = lang
	return b
}

// WithService 配置服务
func (b *Builder) WithService(fn func(*Config)) *Builder {
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
func (b *Builder) WithGitInfo(commit, branch, tag string) *Builder {
	if b.config.Runtime.BuildInfo == nil {
		b.config.Runtime.BuildInfo = NewVersion()
	}
	b.config.Runtime.BuildInfo.GitCommit = commit
	b.config.Runtime.BuildInfo.GitBranch = branch
	b.config.Runtime.BuildInfo.GitTag = tag
	return b
}

// WithDebug 设置调试模式
func (b *Builder) WithDebug(debug bool) *Builder {
	if b.config.Runtime.BuildInfo == nil {
		b.config.Runtime.BuildInfo = NewVersion()
	}
	b.config.Runtime.BuildInfo.Debug.Store(debug)
	return b
}

// WithWorkDir 设置工作目录
func (b *Builder) WithWorkDir(dir string) *Builder {
	b.config.Service.WorkDir = dir
	return b
}

// WithBuildTime 设置构建时间
func (b *Builder) WithBuildTime(buildTime string) *Builder {
	parsedTime, err := time.Parse(time.DateTime, buildTime)
	if err == nil {
		b.config.Runtime.BuildInfo.BuildTime = parsedTime
	}
	return b
}

// WithEnvVar 设置环境变量
func (b *Builder) WithEnvVar(key, value string) *Builder {
	if b.config.Service.EnvVars == nil {
		b.config.Service.EnvVars = make(map[string]string)
	}
	b.config.Service.EnvVars[key] = value
	return b
}

// WithDependencies 设置依赖
func (b *Builder) WithDependencies(deps ...string) *Builder {
	b.config.Service.Dependencies = deps
	return b
}

// WithRuntime 设置运行时配置
func (b *Builder) WithRuntime(rt *Runtime) *Builder {
	b.config.Runtime = rt
	return b
}

// WithSystemService 配置系统服务（向下兼容）
// 支持两种调用方式：
//   - 向下兼容：func() { /* 用户自行处理停止逻辑 */ }
//   - 推荐方式：func(ctx context.Context) { /* 使用 ctx.Done() 优雅停止 */ }
func (b *Builder) WithSystemService(run func(...context.Context), stop ...func()) *Builder {
	b.config.Runtime.Run = run
	b.config.Runtime.Stop = stop
	return b
}

// WithServiceRunner 配置优雅的服务接口（推荐方式）
func (b *Builder) WithServiceRunner(service ServiceRunner) *Builder {
	if service == nil {
		panic("service cannot be nil")
	}
	b.service = service

	// 将ServiceRunner转换为现有的函数式API以保持兼容性
	b.config.Runtime.Run = func(ctxs ...context.Context) {
		ctx := context.Background()
		if len(ctxs) > 0 && ctxs[0] != nil {
			ctx = ctxs[0]
		}
		if err := service.Run(ctx); err != nil {
			fmt.Printf("服务运行错误: %v\n", err)
		}
	}

	b.config.Runtime.Stop = []func(){
		func() {
			if err := service.Stop(); err != nil {
				fmt.Printf("服务停止错误: %v\n", err)
			}
		},
	}

	return b
}

// WithSimpleService 快速创建简单服务
func (b *Builder) WithSimpleService(name string, runFunc func(context.Context) error, stopFunc func() error) *Builder {
	service := NewSimpleService(name, runFunc, stopFunc)
	return b.WithServiceRunner(service)
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

	// 执行验证
	if err := b.validate(); err != nil {
		panic(fmt.Sprintf("构建失败: %v", err))
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

	// 执行验证
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("构建失败: %w", err)
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
	if b.config.Basic.Name == "" {
		errs = append(errs, errors.New("应用名称不能为空"))
	}

	// 服务相关验证
	if b.config.Runtime.Run != nil && b.config.Basic.Name == "" {
		errs = append(errs, errors.New("配置了服务但未设置服务名称"))
	}

	// 执行自定义验证器
	for i, validator := range b.validators {
		if err := validator(b.config); err != nil {
			errs = append(errs, fmt.Errorf("验证器%d失败: %w", i+1, err))
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
	return fmt.Sprintf("构建失败，共%d个错误: %v", len(be.Errors), be.Errors[0])
}

func (be *BuildError) Unwrap() []error {
	return be.Errors
}

// ===================================================================
// 便利性API - 快速构建常见场景
// ===================================================================

// QuickService 快速创建服务应用
func QuickService(name, displayName string, runFunc func(context.Context) error) *Cli {
	return NewBuilder("zh").
		WithName(name).
		WithDisplayName(displayName).
		WithSimpleService(name, runFunc, nil).
		Build()
}

// QuickServiceWithStop 快速创建带停止函数的服务应用
func QuickServiceWithStop(name, displayName string, runFunc func(context.Context) error, stopFunc func() error) *Cli {
	return NewBuilder("zh").
		WithName(name).
		WithDisplayName(displayName).
		WithSimpleService(name, runFunc, stopFunc).
		Build()
}

// QuickCLI 快速创建基础CLI应用
func QuickCLI(name, displayName, description string) *Cli {
	return NewBuilder("zh").
		WithName(name).
		WithDisplayName(displayName).
		WithDescription(description).
		Build()
}

// FromTemplate 从模板创建Builder
func FromTemplate(template string) *Builder {
	b := NewBuilder("zh")

	switch template {
	case "web-service":
		return b.WithValidator(func(cfg *Config) error {
			if cfg.Runtime.Run == nil {
				return errors.New("web服务必须设置运行函数")
			}
			return nil
		})
	case "daemon":
		return b.WithValidator(func(cfg *Config) error {
			if cfg.Basic.Name == "" {
				return errors.New("daemon服务必须设置名称")
			}
			if cfg.Runtime.Run == nil {
				return errors.New("daemon服务必须设置运行函数")
			}
			return nil
		})
	case "cli-tool":
		return b.WithValidator(func(cfg *Config) error {
			if cfg.Basic.Name == "" {
				return errors.New("CLI工具必须设置名称")
			}
			return nil
		})
	default:
		return b
	}
}

// ===================================================================
// 链式配置增强
// ===================================================================

// WithDefaults 设置默认配置
func (b *Builder) WithDefaults() *Builder {
	if b.config.Basic.Language == "" {
		b.config.Basic.Language = "zh"
	}
	if b.config.Basic.Version == "" {
		b.config.Basic.Version = "1.0.0"
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
