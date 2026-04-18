package zcli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type (
	Flag               = pflag.Flag
	FlagSet            = pflag.FlagSet
	NormalizedName     = pflag.NormalizedName
	Command            = cobra.Command
	ShellCompDirective = cobra.ShellCompDirective
	CompletionOptions  = cobra.CompletionOptions
	FParseErrWhitelist = cobra.FParseErrWhitelist
	Group              = cobra.Group
	PositionalArgs     = cobra.PositionalArgs
)

var (
	EnablePrefixMatching     = cobra.EnablePrefixMatching
	EnableCommandSorting     = cobra.EnableCommandSorting
	EnableCaseInsensitive    = cobra.EnableCaseInsensitive
	EnableTraverseRunHooks   = cobra.EnableTraverseRunHooks
	MousetrapHelpText        = cobra.MousetrapHelpText
	MousetrapDisplayDuration = cobra.MousetrapDisplayDuration
	cobraGlobalsMu           sync.Mutex
)

func syncCobraGlobals() {
	cobra.EnablePrefixMatching = EnablePrefixMatching
	cobra.EnableCommandSorting = EnableCommandSorting
	cobra.EnableCaseInsensitive = EnableCaseInsensitive
	cobra.EnableTraverseRunHooks = EnableTraverseRunHooks
	cobra.MousetrapHelpText = MousetrapHelpText
	cobra.MousetrapDisplayDuration = MousetrapDisplayDuration
}

func (c *Cli) applyExecutionGlobals() func() {
	cobraGlobalsMu.Lock()
	prevPrefixMatching := cobra.EnablePrefixMatching
	prevCommandSorting := cobra.EnableCommandSorting
	prevCaseInsensitive := cobra.EnableCaseInsensitive
	prevTraverseRunHooks := cobra.EnableTraverseRunHooks
	prevMousetrapHelpText := cobra.MousetrapHelpText
	prevMousetrapDisplayDuration := cobra.MousetrapDisplayDuration

	syncCobraGlobals()
	if c != nil && c.config != nil && c.config.basic != nil && c.config.basic.MousetrapDisabled {
		cobra.MousetrapHelpText = ""
	}

	return func() {
		cobra.EnablePrefixMatching = prevPrefixMatching
		cobra.EnableCommandSorting = prevCommandSorting
		cobra.EnableCaseInsensitive = prevCaseInsensitive
		cobra.EnableTraverseRunHooks = prevTraverseRunHooks
		cobra.MousetrapHelpText = prevMousetrapHelpText
		cobra.MousetrapDisplayDuration = prevMousetrapDisplayDuration
		cobraGlobalsMu.Unlock()
	}
}

// Cli 是对 cobra.Command 的封装，提供更友好的命令行界面
type Cli struct {
	config  *Config
	colors  *colors
	lang    *Language
	command *cobra.Command
}

// NewCli 创建一个新的命令对象
func NewCli(opts ...Option) *Cli {
	cfg := NewConfig()
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(cfg)
		}
	}

	syncCobraGlobals()

	// 设置语言
	if cfg.basic.Language != "" {
		if err := SetLanguage(cfg.basic.Language); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "set language failed: %v\n", err)
		}
	}

	cmd := &Cli{
		config: cfg,
		colors: newColors(),
		lang:   GetLanguageManager().GetPrimary(),
		command: &cobra.Command{
			Use:           cfg.basic.Name, // 设置命令名称
			SilenceErrors: cfg.basic.SilenceErrors,
			SilenceUsage:  cfg.basic.SilenceUsage,
		},
	}

	// 如果写了服务描述则把服务描述作为命令描述
	if cmd.config.basic.Description != "" {
		cmd.command.Short = cmd.config.basic.Description
	}

	// 设置版本信息
	cmd.setupVersion()

	// 配置服务（如果需要）
	cmd.setupService()

	// 添加根命令
	cmd.addRootCommand(cmd.command)
	return cmd
}

// setupVersion 设置版本信息
func (c *Cli) setupVersion() {
	// 添加版本标志
	if c.config.runtime.BuildInfo != nil || c.config.basic.Version != "" {
		// 优先使用 BuildInfo.Version，如果需要可以被 Basic.Version 覆盖
		version := ""
		if c.config.runtime.BuildInfo != nil {
			version = c.config.runtime.BuildInfo.Version
		}
		if c.config.basic.Version != "" {
			version = c.config.basic.Version       // 允许覆盖
			if c.config.runtime.BuildInfo != nil { // 如果有 BuildInfo 则同步更新
				c.config.runtime.BuildInfo.Version = c.config.basic.Version // 同步更新 BuildInfo
			}
		}

		c.command.Version = version
		c.command.Flags().BoolP("version", "v", false, c.lang.UI.Version.Description)
	}

	// 如果有构建信息，重写版本命令
	if c.config.runtime.BuildInfo != nil {
		var buf strings.Builder
		defer buf.Reset()
		c.showVersion(&buf)
		c.command.SetVersionTemplate(buf.String())
	}
}

// setupService 设置服务相关功能
func (c *Cli) setupService() {
	// 只有同时设置了 Name 和 Run 函数才初始化服务
	if c.config.basic.Name != "" && c.config.runtime.Run != nil {
		// 如果配置了服务名称和启动函数则初始化服务
		c.initService()
	}
}

// ============================================================================
// 基础命令管理方法
// 用于管理命令的基本操作：添加子命令、获取命令信息、命令组管理等
// ============================================================================

// AddCommand 添加一个或多个子命令到当前命令
func (c *Cli) AddCommand(cmds ...*Command) {
	c.command.AddCommand(cmds...)
}

// AddGroup 添加一个或多个命令组到当前命令
func (c *Cli) AddGroup(groups ...*Group) {
	c.command.AddGroup(groups...)
}

// AllChildCommandsHaveGroup 检查是否所有子命令都已分配到命令组中
func (c *Cli) AllChildCommandsHaveGroup() bool {
	return c.command.AllChildCommandsHaveGroup()
}

// ArgsLenAtDash 获取命令行中 "--" 之前的参数数量
// 用于区分命令参数和传递给外部命令的参数
func (c *Cli) ArgsLenAtDash() int {
	return c.command.ArgsLenAtDash()
}

// CalledAs 返回命令在命令行中被调用时使用的名称
// 可能是命令的名称或别名
func (c *Cli) CalledAs() string {
	return c.command.CalledAs()
}

// CommandPath 返回从根命令到当前命令的完整路径
func (c *Cli) CommandPath() string {
	return c.command.CommandPath()
}

// CommandPathPadding 返回命令路径的填充长度
// 用于格式化输出时对齐
func (c *Cli) CommandPathPadding() int {
	return c.command.CommandPathPadding()
}

// Commands 返回当前命令的所有直接子命令
func (c *Cli) Commands() []*Command {
	return c.command.Commands()
}

// DebugFlags 打印调试信息，用于查看标志的归属与继承情况
func (c *Cli) DebugFlags() {
	c.command.DebugFlags()
}

// ContainsGroup 检查是否包含指定ID的命令组
func (c *Cli) ContainsGroup(groupID string) bool {
	return c.command.ContainsGroup(groupID)
}

// Context 返回命令的上下文
// 若未设置，则返回 nil（与原生 Cobra 行为一致）
func (c *Cli) Context() context.Context {
	return c.command.Context()
}

// ============================================================================
// 命令执行控制方法
// 用于执行命令和控制命令执行流程
// ============================================================================

// Execute 执行命令，这是启动应用程序的主要入口点
func (c *Cli) Execute() error {
	return c.ExecuteContext(c.config.ctx)
}

// ExecuteC 执行命令并返回选中的命令
// 主要用于需要访问选中命令的场景
func (c *Cli) ExecuteC() (*Command, error) {
	return c.ExecuteContextC(c.config.ctx)
}

// ExecuteContext 在指定的上下文中执行命令
// 可用于传递取消信号或超时控制
func (c *Cli) ExecuteContext(ctx context.Context) error {
	restore := c.applyExecutionGlobals()
	defer restore()
	return c.command.ExecuteContext(ctx)
}

// ExecuteContextC 在指定上下文中执行命令并返回选中的命令
func (c *Cli) ExecuteContextC(ctx context.Context) (*Command, error) {
	restore := c.applyExecutionGlobals()
	defer restore()
	return c.command.ExecuteContextC(ctx)
}

// ============================================================================
// 标志参数管理方法
// 用于管理命令行标志和参数
// ============================================================================

// Flag 获取指定名称的命令行标志
func (c *Cli) Flag(name string) *Flag {
	return c.command.Flag(name)
}

// GetFlagCompletionFunc 返回指定标志的补全函数（若已注册）
func (c *Cli) GetFlagCompletionFunc(flagName string) (cobra.CompletionFunc, bool) {
	return c.command.GetFlagCompletionFunc(flagName)
}

// FlagErrorFunc 返回处理标志错误的函数
func (c *Cli) FlagErrorFunc() func(*Command, error) error {
	return c.command.FlagErrorFunc()
}

// Flags 返回命令的本地标志集
func (c *Cli) Flags() *FlagSet {
	return c.command.Flags()
}

// HasAvailableFlags 检查命令是否有可用的标志
func (c *Cli) HasAvailableFlags() bool {
	return c.command.HasAvailableFlags()
}

// HasAvailableInheritedFlags 检查命令是否有可用的继承标志
func (c *Cli) HasAvailableInheritedFlags() bool {
	return c.command.HasAvailableInheritedFlags()
}

// HasAvailableLocalFlags 检查命令是否有可用的本地标志
func (c *Cli) HasAvailableLocalFlags() bool {
	return c.command.HasAvailableLocalFlags()
}

// HasAvailablePersistentFlags 检查命令是否有可用的持久标志
func (c *Cli) HasAvailablePersistentFlags() bool {
	return c.command.HasAvailablePersistentFlags()
}

// HasFlags 检查命令是否定义了任何标志
func (c *Cli) HasFlags() bool {
	return c.command.HasFlags()
}

// HasInheritedFlags 检查命令是否有继承的标志
func (c *Cli) HasInheritedFlags() bool {
	return c.command.HasInheritedFlags()
}

// HasLocalFlags 检查命令是否有本地标志
func (c *Cli) HasLocalFlags() bool {
	return c.command.HasLocalFlags()
}

// HasPersistentFlags 检查命令是否有持久标志
func (c *Cli) HasPersistentFlags() bool {
	return c.command.HasPersistentFlags()
}

// InheritedFlags 返回命令的继承标志集
func (c *Cli) InheritedFlags() *FlagSet {
	return c.command.InheritedFlags()
}

// LocalFlags 返回命令的本地标志集
func (c *Cli) LocalFlags() *FlagSet {
	return c.command.LocalFlags()
}

// LocalNonPersistentFlags 返回命令的非持久本地标志集
func (c *Cli) LocalNonPersistentFlags() *FlagSet {
	return c.command.LocalNonPersistentFlags()
}

// NonInheritedFlags 返回命令的非继承标志集
func (c *Cli) NonInheritedFlags() *FlagSet {
	return c.command.NonInheritedFlags()
}

// PersistentFlags 返回命令的持久标志集
func (c *Cli) PersistentFlags() *FlagSet {
	return c.command.PersistentFlags()
}
