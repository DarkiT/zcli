package zcli

import (
	"context"
	"time"
)

// RunFunc 服务运行函数签名
// 标准 Go 惯例：接收 context 用于优雅关闭，返回 error 用于错误处理
type RunFunc func(ctx context.Context) error

// StopFunc 服务停止函数签名
// 返回 error 用于报告停止过程中的错误
type StopFunc func() error

// Basic 基础配置
type Basic struct {
	Name              string // 服务名称
	DisplayName       string // 显示名称
	Description       string // 服务描述
	Version           string // 版本
	Logo              string // Logo 路径
	Language          string // 使用语言
	MousetrapDisabled bool   // 禁用 Windows 双击运行提示
	NoColor           bool   // 禁用彩色输出
	SilenceErrors     bool   // 禁止打印错误
	SilenceUsage      bool   // 禁止打印使用说明
}

// Runtime 运行时配置
type Runtime struct {
	Run       RunFunc      // 启动函数，标准签名：func(ctx context.Context) error
	Stop      StopFunc     // 停止函数，标准签名：func() error
	BuildInfo *VersionInfo // 构建信息

	// ShutdownInitial 在取消 Run(ctx) 后，等待主服务优雅退出的时长，默认 15s
	ShutdownInitial time.Duration
	// ShutdownGrace 在主服务收到停止信号后，保留给 stop hook / 最终清理的额外时长，默认 5s
	ShutdownGrace time.Duration
	// StartTimeout 启动超时，写入 daemon Config.Timeout.Start
	StartTimeout time.Duration
	// StopTimeout 停止超时，写入 daemon Config.Timeout.Stop
	StopTimeout time.Duration
	// ErrorHandlers 错误处理器链
	ErrorHandlers []ErrorHandler
}

// Config 统一配置结构
// 字段已私有化，通过 getter 方法访问以防止外部直接修改
type Config struct {
	basic   *Basic          // 基础配置（私有）
	service *ServiceConfig  // 服务配置（私有）
	runtime *Runtime        // 运行时配置（私有）
	ctx     context.Context // 上下文
}

// Basic 返回基础配置的副本，防止外部修改
func (c *Config) Basic() Basic {
	if c.basic == nil {
		return Basic{}
	}
	return *c.basic
}

// Service 返回服务配置的深拷贝，防止外部通过切片/map 修改内部状态
func (c *Config) Service() ServiceConfig {
	return cloneService(c.service)
}

// Runtime 返回运行时配置的副本
func (c *Config) Runtime() Runtime {
	return cloneRuntime(c.runtime)
}

// Context 返回配置的上下文
func (c *Config) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// =====================
// 只读视图（性能优化）
// =====================

// ConfigView 只读配置视图，避免频繁深拷贝
// 注意：调用方不应修改返回的指针内容
type ConfigView struct {
	config *Config
}

// View 创建只读视图
func (c *Config) View() *ConfigView {
	return &ConfigView{config: c}
}

// Basic 返回基础配置的只读引用
func (cv *ConfigView) Basic() *Basic {
	return cv.config.basic
}

// Service 返回服务配置的只读引用
func (cv *ConfigView) Service() *ServiceConfig {
	return cv.config.service
}

// Runtime 返回运行时配置的只读引用
func (cv *ConfigView) Runtime() *Runtime {
	return cv.config.runtime
}

// Context 返回配置的上下文
func (cv *ConfigView) Context() context.Context {
	return cv.config.Context()
}

// WithSilenceErrors 设置是否静默错误输出
func WithSilenceErrors(enabled bool) Option {
	return func(c *Config) {
		if c.basic == nil {
			c.basic = &Basic{}
		}
		c.basic.SilenceErrors = enabled
	}
}

// WithSilenceUsage 设置是否静默用法输出
func WithSilenceUsage(enabled bool) Option {
	return func(c *Config) {
		if c.basic == nil {
			c.basic = &Basic{}
		}
		c.basic.SilenceUsage = enabled
	}
}

// Option CLI 选项函数
type Option func(*Config)

// WithConfig 创建配置选项（深拷贝传入的配置）
func WithConfig(cfg *Config) Option {
	return func(c *Config) {
		if cfg == nil {
			return
		}

		c.basic = cloneBasicPtr(cfg.basic)
		c.service = cloneServicePtr(cfg.service)
		c.runtime = cloneRuntimePtr(cfg.runtime)
		c.ctx = cfg.ctx
	}
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		basic: &Basic{
			Language:      "en",
			NoColor:       false,
			SilenceErrors: false,
			SilenceUsage:  false,
		},
		service: &ServiceConfig{
			EnvVars: make(map[string]string),
			Options: make(ServiceOptions),
		},
		runtime: &Runtime{
			ShutdownInitial: 15 * time.Second,
			ShutdownGrace:   5 * time.Second,
			StopTimeout:     20 * time.Second,
		},
		ctx: context.Background(),
	}
}

// =====================
// 内部拷贝工具
// =====================

// cloneBasicPtr 复制基础配置指针，防止外部修改原对象
func cloneBasicPtr(src *Basic) *Basic {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}

// cloneService 深拷贝服务配置，包含切片与 map
func cloneService(src *ServiceConfig) ServiceConfig {
	if src == nil {
		return ServiceConfig{}
	}

	dst := ServiceConfig{
		Name:              src.Name,
		DisplayName:       src.DisplayName,
		Description:       src.Description,
		Version:           src.Version,
		WorkDir:           src.WorkDir,
		Username:          src.Username,
		Executable:        src.Executable,
		ChRoot:            src.ChRoot,
		AllowSudoFallback: src.AllowSudoFallback,
	}

	if len(src.Arguments) > 0 {
		dst.Arguments = append([]string(nil), src.Arguments...)
	}
	if len(src.Dependencies) > 0 {
		dst.Dependencies = append([]string(nil), src.Dependencies...)
	}
	if len(src.StructuredDeps) > 0 {
		dst.StructuredDeps = append([]Dependency(nil), src.StructuredDeps...)
	}
	if src.EnvVars != nil {
		dst.EnvVars = make(map[string]string, len(src.EnvVars))
		for k, v := range src.EnvVars {
			dst.EnvVars[k] = v
		}
	}
	if src.Options != nil {
		dst.Options = make(ServiceOptions, len(src.Options))
		for k, v := range src.Options {
			dst.Options[k] = v
		}
	}
	return dst
}

// cloneServicePtr 返回服务配置指针副本
func cloneServicePtr(src *ServiceConfig) *ServiceConfig {
	if src == nil {
		return nil
	}
	clone := cloneService(src)
	return &clone
}

// cloneRuntime 深拷贝运行时配置，复制 BuildInfo，保留函数指针引用
func cloneRuntime(src *Runtime) Runtime {
	if src == nil {
		return Runtime{}
	}

	dst := Runtime{
		Run:             src.Run,
		Stop:            src.Stop,
		ShutdownInitial: src.ShutdownInitial,
		ShutdownGrace:   src.ShutdownGrace,
		StartTimeout:    src.StartTimeout,
		StopTimeout:     src.StopTimeout,
	}

	if len(src.ErrorHandlers) > 0 {
		dst.ErrorHandlers = append([]ErrorHandler(nil), src.ErrorHandlers...)
	}

	if src.BuildInfo != nil {
		dst.BuildInfo = cloneVersionInfo(src.BuildInfo)
	}

	return dst
}

// cloneRuntimePtr 返回运行时配置指针副本
func cloneRuntimePtr(src *Runtime) *Runtime {
	if src == nil {
		return nil
	}
	clone := cloneRuntime(src)
	return &clone
}

// cloneVersionInfo 深拷贝版本信息，确保 Debug 原子标志保持原值
func cloneVersionInfo(src *VersionInfo) *VersionInfo {
	if src == nil {
		return nil
	}

	clone := &VersionInfo{
		Version:      src.Version,
		GoVersion:    src.GoVersion,
		GitCommit:    src.GitCommit,
		GitBranch:    src.GitBranch,
		GitTag:       src.GitTag,
		Platform:     src.Platform,
		Architecture: src.Architecture,
		Compiler:     src.Compiler,
		BuildTime:    src.BuildTime,
	}

	clone.Debug.Store(src.Debug.Load())
	return clone
}
