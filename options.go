package zcli

import "context"

// Basic 基础配置
type Basic struct {
	Name        string // 服务名称
	DisplayName string // 显示名称
	Description string // 服务描述
	Version     string // 版本
	Logo        string // Logo路径
	Language    string // 使用语言
	NoColor     bool   // 禁用彩色输出
}

// Service 服务配置
type Service struct {
	Username     string                 // 运行用户
	Arguments    []string               // 启动参数
	Executable   string                 // 可执行文件路径
	Dependencies []string               // 服务依赖
	WorkDir      string                 // 工作目录
	ChRoot       string                 // 根目录
	Options      map[string]interface{} // 自定义选项
	EnvVars      map[string]string      // 环境变量
}

// Runtime 运行时配置
type Runtime struct {
	Run       func()       // 启动函数，用于调用上层服务主函数
	Stop      []func()     // 停止函数，用于停止服务时调用上层停止函数
	BuildInfo *VersionInfo // 构建信息
}

// Config 统一配置结构
type Config struct {
	Basic   *Basic          // 基础配置
	Service *Service        // 服务配置
	Runtime *Runtime        // 运行时配置
	ctx     context.Context // 上下文
}

// Option CLI选项函数
type Option func(*Config)

// WithConfig 创建配置选项
func WithConfig(cfg *Config) Option {
	return func(c *Config) {
		*c = *cfg
	}
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Basic: &Basic{
			Language: "zh",
			NoColor:  false,
		},
		Service: &Service{
			EnvVars: make(map[string]string),
			Options: make(map[string]interface{}),
		},
		Runtime: &Runtime{},
		ctx:     context.Background(),
	}
}
