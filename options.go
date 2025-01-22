package zcli

// Option 定义配置选项函数类型
type Option func(*option)

// option 服务选项
type option struct {
	Name             string                 // 服务名称
	DisplayName      string                 // 显示名称
	Description      string                 // 服务描述
	UserName         string                 // 运行用户
	Arguments        []string               // 启动参数
	Executable       string                 // 可执行文件路径
	Dependencies     []string               // 服务依赖
	WorkingDirectory string                 // 工作目录
	ChRoot           string                 // 根目录
	Option           map[string]interface{} // 自定义选项
	EnvVars          map[string]string      // 环境变量
	Version          string                 // 版本
	Logo             string                 // Logo路径
	Language         string                 // 使用语言
	NoColor          bool                   // 禁用彩色输出
	VersionInfo      *VersionInfo           // 构建信息
	Run              func()                 // 启动函数，用于调用上层服务主函数
	Stop             func()                 // 停止函数，用于停止服务时调用上层停止函数
}

// WithName 设置服务名称
func WithName(name string) Option {
	return func(o *option) {
		o.Name = name
	}
}

// WithDisplayName 设置显示名称
func WithDisplayName(name string) Option {
	return func(o *option) {
		o.DisplayName = name
	}
}

// WithDescription 设置服务描述
func WithDescription(desc string) Option {
	return func(o *option) {
		o.Description = desc
	}
}

// WithUserName 设置运行用户
func WithUserName(user string) Option {
	return func(o *option) {
		o.UserName = user
	}
}

// WithArguments 设置启动参数
func WithArguments(args ...string) Option {
	return func(o *option) {
		o.Arguments = args
	}
}

// WithExecutable 设置可执行文件路径
func WithExecutable(path string) Option {
	return func(o *option) {
		o.Executable = path
	}
}

// WithDependencies 设置服务依赖
func WithDependencies(deps ...string) Option {
	return func(o *option) {
		o.Dependencies = deps
	}
}

// WithWorkingDirectory 设置工作目录
func WithWorkingDirectory(dir string) Option {
	return func(o *option) {
		o.WorkingDirectory = dir
	}
}

// WithChRoot 设置根目录
func WithChRoot(root string) Option {
	return func(o *option) {
		o.ChRoot = root
	}
}

// WithOption 设置单个自定义选项
func WithOption(key string, value interface{}) Option {
	return func(o *option) {
		if o.Option == nil {
			o.Option = make(map[string]interface{})
		}
		o.Option[key] = value
	}
}

// WithOptions 设置多个自定义选项
func WithOptions(opts map[string]interface{}) Option {
	return func(o *option) {
		if o.Option == nil {
			o.Option = make(map[string]interface{})
		}
		for k, v := range opts {
			o.Option[k] = v
		}
	}
}

// WithEnvVar 设置单个环境变量
func WithEnvVar(key, value string) Option {
	return func(o *option) {
		if o.EnvVars == nil {
			o.EnvVars = make(map[string]string)
		}
		o.EnvVars[key] = value
	}
}

// WithEnvVars 设置多个环境变量
func WithEnvVars(vars map[string]string) Option {
	return func(o *option) {
		if o.EnvVars == nil {
			o.EnvVars = make(map[string]string)
		}
		for k, v := range vars {
			o.EnvVars[k] = v
		}
	}
}

// WithVersion 设置版本
func WithVersion(version string) Option {
	return func(o *option) {
		o.Version = version
	}
}

// WithLogo 设置Logo路径
func WithLogo(path string) Option {
	return func(o *option) {
		o.Logo = path
	}
}

// WithLanguage 设置使用语言
func WithLanguage(lang string) Option {
	return func(o *option) {
		o.Language = lang
	}
}

// WithNoColor 设置是否禁用彩色输出
func WithNoColor(disable bool) Option {
	return func(o *option) {
		o.NoColor = disable
	}
}

// WithBuildInfo 设置构建信息
func WithBuildInfo(info *VersionInfo) Option {
	return func(o *option) {
		o.VersionInfo = info
	}
}

// WithRun 设置启动函数
func WithRun(fn func()) Option {
	return func(o *option) {
		o.Run = fn
	}
}

// WithStop 设置停止函数
func WithStop(fn func()) Option {
	return func(o *option) {
		o.Stop = fn
	}
}
