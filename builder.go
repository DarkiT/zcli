package zcli

// Builder CLI构建器
type Builder struct {
	config *Config
	cli    *Cli
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

// WithDefaultConfig 使用默认配置
func (b *Builder) WithDefaultConfig() *Builder {
	b.WithLanguage("zh").
		WithVersion("1.0.0").
		WithDebug(false)
	return b
}

// WithRuntime 设置运行时配置
func (b *Builder) WithRuntime(rt *Runtime) *Builder {
	b.config.Runtime = rt
	return b
}

// WithSystemService 快速配置为系统服务
func (b *Builder) WithSystemService(run func(), stop ...func()) *Builder {
	b.config.Runtime.Run = run
	b.config.Runtime.Stop = stop
	return b
}

// Build 构建CLI实例
func (b *Builder) Build() *Cli {
	if b.cli == nil {
		b.cli = NewCli(WithConfig(b.config))
	}
	return b.cli
}
