package zcli

import (
	"context"
	"io"
	"strings"

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

	cmd := &Cli{
		config: cfg,
		colors: newColors(),
		lang:   getServiceLanguage(cfg.Basic.Language),
		command: &cobra.Command{
			SilenceErrors: true, // 禁止打印错误
			SilenceUsage:  true, // 禁止打印使用说明
		},
	}

	// 如果写了服务描述则把服务描述作为命令描述
	if cmd.config.Basic.Description != "" {
		cmd.command.Short = cmd.config.Basic.Description
	}

	// 添加版本标志
	if cmd.config.Runtime.BuildInfo != nil || cmd.config.Basic.Version != "" {
		// 优先使用 BuildInfo.Version，如果需要可以被 Basic.Version 覆盖
		version := ""
		if cmd.config.Runtime.BuildInfo != nil {
			version = cmd.config.Runtime.BuildInfo.Version
		}
		if cmd.config.Basic.Version != "" {
			version = cmd.config.Basic.Version       // 允许覆盖
			if cmd.config.Runtime.BuildInfo != nil { // 如果有 BuildInfo 则同步更新
				cmd.config.Runtime.BuildInfo.Version = cmd.config.Basic.Version // 同步更新 BuildInfo
			}
		}

		cmd.command.Version = version
		cmd.command.Flags().BoolP("version", "v", false, cmd.lang.Command.VersionDesc)
	}

	// 如果有构建信息，重写版本命令
	if cmd.config.Runtime.BuildInfo != nil {
		var buf strings.Builder
		defer buf.Reset()
		cmd.showVersion(&buf)
		cmd.command.SetVersionTemplate(buf.String())
	}

	// 只有同时设置了 Name 和 Run 函数才初始化服务
	if cmd.config.Basic.Name != "" && cmd.config.Runtime.Run != nil {
		// 如果配置了服务名称和启动函数则初始化服务
		cmd.initService()
		// 如果设置了启动函数则添加默认的启动命令
		cmd.command.Run = func(_ *cobra.Command, args []string) {
			cmd.config.Runtime.Run()
		}
	}

	// 添加根命令
	cmd.addRootCommand(cmd.command)
	return cmd
}

// 基础命令方法

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

// ContainsGroup 检查是否包含指定ID的命令组
func (c *Cli) ContainsGroup(groupID string) bool {
	return c.command.ContainsGroup(groupID)
}

// Context 返回命令的上下文
// 如果未设置，则返回 background context
func (c *Cli) Context() context.Context {
	return c.command.Context()
}

// 执行相关方法

// Execute 执行命令
// 这是启动应用程序的主要入口点
func (c *Cli) Execute() error {
	return c.command.Execute()
}

// ExecuteC 执行命令并返回选中的命令
// 主要用于需要访问选中命令的场景
func (c *Cli) ExecuteC() (*Command, error) {
	return c.command.ExecuteC()
}

// ExecuteContext 在指定的上下文中执行命令
// 可用于传递取消信号或超时控制
func (c *Cli) ExecuteContext(ctx context.Context) error {
	return c.command.ExecuteContext(ctx)
}

// ExecuteContextC 在指定上下文中执行命令并返回选中的命令
func (c *Cli) ExecuteContextC(ctx context.Context) (*Command, error) {
	return c.command.ExecuteContextC(ctx)
}

// 标志相关方法

// Flag 获取指定名称的命令行标志
func (c *Cli) Flag(name string) *Flag {
	return c.command.Flag(name)
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

// Shell补全相关方法

// GenBashCompletion 生成 Bash 补全脚本并写入指定的 Writer
func (c *Cli) GenBashCompletion(w io.Writer) error {
	return c.command.GenBashCompletion(w)
}

// GenBashCompletionFile 生成 Bash 补全脚本并保存到指定文件
func (c *Cli) GenBashCompletionFile(filename string) error {
	return c.command.GenBashCompletionFile(filename)
}

// GenBashCompletionFileV2 生成新版本的 Bash 补全脚本并保存到文件
// includeDesc 参数控制是否包含描述信息
func (c *Cli) GenBashCompletionFileV2(filename string, includeDesc bool) error {
	return c.command.GenBashCompletionFileV2(filename, includeDesc)
}

// GenBashCompletionV2 生成新版本的 Bash 补全脚本并写入 Writer
func (c *Cli) GenBashCompletionV2(w io.Writer, includeDesc bool) error {
	return c.command.GenBashCompletionV2(w, includeDesc)
}

// GenFishCompletion 生成 Fish shell 补全脚本并写入 Writer
func (c *Cli) GenFishCompletion(w io.Writer, includeDesc bool) error {
	return c.command.GenFishCompletion(w, includeDesc)
}

// GenFishCompletionFile 生成 Fish shell 补全脚本并保存到文件
func (c *Cli) GenFishCompletionFile(filename string, includeDesc bool) error {
	return c.command.GenFishCompletionFile(filename, includeDesc)
}

// GenPowerShellCompletion 生成 PowerShell 补全脚本并写入 Writer
func (c *Cli) GenPowerShellCompletion(w io.Writer) error {
	return c.command.GenPowerShellCompletion(w)
}

// GenPowerShellCompletionFile 生成 PowerShell 补全脚本并保存到文件
func (c *Cli) GenPowerShellCompletionFile(filename string) error {
	return c.command.GenPowerShellCompletionFile(filename)
}

// GenPowerShellCompletionFileWithDesc 生成包含描述的 PowerShell 补全脚本并保存到文件
func (c *Cli) GenPowerShellCompletionFileWithDesc(filename string) error {
	return c.command.GenPowerShellCompletionFileWithDesc(filename)
}

// GenPowerShellCompletionWithDesc 生成包含描述的 PowerShell 补全脚本并写入 Writer
func (c *Cli) GenPowerShellCompletionWithDesc(w io.Writer) error {
	return c.command.GenPowerShellCompletionWithDesc(w)
}

// GenZshCompletion 生成 Zsh 补全脚本并写入 Writer
func (c *Cli) GenZshCompletion(w io.Writer) error {
	return c.command.GenZshCompletion(w)
}

// GenZshCompletionFile 生成 Zsh 补全脚本并保存到文件
func (c *Cli) GenZshCompletionFile(filename string) error {
	return c.command.GenZshCompletionFile(filename)
}

// GenZshCompletionFileNoDesc 生成不包含描述的 Zsh 补全脚本并保存到文件
func (c *Cli) GenZshCompletionFileNoDesc(filename string) error {
	return c.command.GenZshCompletionFileNoDesc(filename)
}

// GenZshCompletionNoDesc 生成不包含描述的 Zsh 补全脚本并写入 Writer
func (c *Cli) GenZshCompletionNoDesc(w io.Writer) error {
	return c.command.GenZshCompletionNoDesc(w)
}

// 标志标记相关方法

// MarkFlagCustom 为指定标志添加自定义补全函数
func (c *Cli) MarkFlagCustom(name string, f string) error {
	return c.command.MarkFlagCustom(name, f)
}

// MarkFlagDirname 标记指定标志接受目录名作为参数
func (c *Cli) MarkFlagDirname(name string) error {
	return c.command.MarkFlagDirname(name)
}

// MarkFlagFilename 标记指定标志接受文件名作为参数
func (c *Cli) MarkFlagFilename(name string, extensions ...string) error {
	return c.command.MarkFlagFilename(name, extensions...)
}

// MarkFlagRequired 标记指定标志为必需
func (c *Cli) MarkFlagRequired(name string) error {
	return c.command.MarkFlagRequired(name)
}

// MarkFlagsMutuallyExclusive 标记多个标志互斥
func (c *Cli) MarkFlagsMutuallyExclusive(flagNames ...string) {
	c.command.MarkFlagsMutuallyExclusive(flagNames...)
}

// MarkFlagsOneRequired 标记多个标志中必须指定一个
func (c *Cli) MarkFlagsOneRequired(flagNames ...string) {
	c.command.MarkFlagsOneRequired(flagNames...)
}

// MarkFlagsRequiredTogether 标记多个标志必须同时使用
func (c *Cli) MarkFlagsRequiredTogether(flagNames ...string) {
	c.command.MarkFlagsRequiredTogether(flagNames...)
}

// MarkPersistentFlagDirname 标记指定的持久标志接受目录名作为参数
func (c *Cli) MarkPersistentFlagDirname(name string) error {
	return c.command.MarkPersistentFlagDirname(name)
}

// MarkPersistentFlagFilename 标记指定的持久标志接受文件名作为参数
func (c *Cli) MarkPersistentFlagFilename(name string, extensions ...string) error {
	return c.command.MarkPersistentFlagFilename(name, extensions...)
}

// MarkPersistentFlagRequired 标记指定的持久标志为必需
func (c *Cli) MarkPersistentFlagRequired(name string) error {
	return c.command.MarkPersistentFlagRequired(name)
}

// MarkZshCompPositionalArgumentFile 标记 Zsh 位置参数接受文件
func (c *Cli) MarkZshCompPositionalArgumentFile(argPosition int, patterns ...string) error {
	return c.command.MarkZshCompPositionalArgumentFile(argPosition, patterns...)
}

// MarkZshCompPositionalArgumentWords 标记 Zsh 位置参数接受指定的词列表
func (c *Cli) MarkZshCompPositionalArgumentWords(argPosition int, words ...string) error {
	return c.command.MarkZshCompPositionalArgumentWords(argPosition, words...)
}

// 输入输出相关方法

// InOrStdin 返回命令的输入流，默认为标准输入
func (c *Cli) InOrStdin() io.Reader {
	return c.command.InOrStdin()
}

// OutOrStderr 返回命令的错误输出流，默认为标准错误
func (c *Cli) OutOrStderr() io.Writer {
	return c.command.OutOrStderr()
}

// OutOrStdout 返回命令的标准输出流，默认为标准输出
func (c *Cli) OutOrStdout() io.Writer {
	return c.command.OutOrStdout()
}

// Print 打印到命令的标准输出
func (c *Cli) Print(i ...interface{}) {
	c.command.Print(i...)
}

// PrintErr 打印到命令的错误输出
func (c *Cli) PrintErr(i ...interface{}) {
	c.command.PrintErr(i...)
}

// PrintErrf 格式化打印到命令的错误输出
func (c *Cli) PrintErrf(format string, i ...interface{}) {
	c.command.PrintErrf(format, i...)
}

// PrintErrln 打印到命令的错误输出并换行
func (c *Cli) PrintErrln(i ...interface{}) {
	c.command.PrintErrln(i...)
}

// Printf 格式化打印到命令的标准输出
func (c *Cli) Printf(format string, i ...interface{}) {
	c.command.Printf(format, i...)
}

// Println 打印到命令的标准输出并换行
func (c *Cli) Println(i ...interface{}) {
	c.command.Println(i...)
}

// 设置相关方法

// SetArgs 设置命令的参数
func (c *Cli) SetArgs(a []string) {
	c.command.SetArgs(a)
}

// SetCompletionCommandGroupID 设置补全命令的组ID
func (c *Cli) SetCompletionCommandGroupID(groupID string) {
	c.command.SetCompletionCommandGroupID(groupID)
}

// SetContext 设置命令的上下文
func (c *Cli) SetContext(ctx context.Context) {
	c.command.SetContext(ctx)
}

// SetErr 设置命令的错误输出流
func (c *Cli) SetErr(newErr io.Writer) {
	c.command.SetErr(newErr)
}

// SetErrPrefix 设置错误消息的前缀
func (c *Cli) SetErrPrefix(s string) {
	c.command.SetErrPrefix(s)
}

// SetFlagErrorFunc 设置处理标志错误的自定义函数
func (c *Cli) SetFlagErrorFunc(f func(*Command, error) error) {
	c.command.SetFlagErrorFunc(f)
}

// SetGlobalNormalizationFunc 设置全局标志名称规范化函数
func (c *Cli) SetGlobalNormalizationFunc(n func(f *FlagSet, name string) NormalizedName) {
	c.command.SetGlobalNormalizationFunc(n)
}

// SetHelpCommand 设置自定义的帮助命令
func (c *Cli) SetHelpCommand(cmd *Command) {
	c.command.SetHelpCommand(cmd)
}

// SetHelpCommandGroupID 设置帮助命令的组ID
func (c *Cli) SetHelpCommandGroupID(groupID string) {
	c.command.SetHelpCommandGroupID(groupID)
}

// SetHelpFunc 设置自定义的帮助函数
func (c *Cli) SetHelpFunc(f func(*Command, []string)) {
	c.command.SetHelpFunc(f)
}

// SetHelpTemplate 设置帮助信息的模板
func (c *Cli) SetHelpTemplate(s string) {
	c.command.SetHelpTemplate(s)
}

// SetIn 设置命令的输入流
func (c *Cli) SetIn(newIn io.Reader) {
	c.command.SetIn(newIn)
}

// SetOut 设置命令的标准输出流
func (c *Cli) SetOut(newOut io.Writer) {
	c.command.SetOut(newOut)
}

// SetOutput 设置命令的输出流（同时影响标准输出和错误输出）
func (c *Cli) SetOutput(output io.Writer) {
	c.command.SetOutput(output)
}

// SetUsageFunc 设置自定义的使用说明函数
func (c *Cli) SetUsageFunc(f func(*Command) error) {
	c.command.SetUsageFunc(f)
}

// SetUsageTemplate 设置使用说明的模板
func (c *Cli) SetUsageTemplate(s string) {
	c.command.SetUsageTemplate(s)
}

// SetVersionTemplate 设置版本信息的模板
func (c *Cli) SetVersionTemplate(s string) {
	c.command.SetVersionTemplate(s)
}

// 其他功能性方法

// GlobalNormalizationFunc 返回全局标志名称规范化函数
func (c *Cli) GlobalNormalizationFunc() func(f *FlagSet, name string) NormalizedName {
	return c.command.GlobalNormalizationFunc()
}

// Groups 返回命令的所有组
func (c *Cli) Groups() []*Group {
	return c.command.Groups()
}

// HasAlias 检查是否存在指定的别名
func (c *Cli) HasAlias(s string) bool {
	return c.command.HasAlias(s)
}

// HasAvailableSubCommands 检查是否有可用的子命令
func (c *Cli) HasAvailableSubCommands() bool {
	return c.command.HasAvailableSubCommands()
}

// HasExample 检查是否有示例
func (c *Cli) HasExample() bool {
	return c.command.HasExample()
}

// HasHelpSubCommands 检查是否有帮助子命令
func (c *Cli) HasHelpSubCommands() bool {
	return c.command.HasHelpSubCommands()
}

// HasParent 检查是否有父命令
func (c *Cli) HasParent() bool {
	return c.command.HasParent()
}

// HasSubCommands 检查是否有子命令
func (c *Cli) HasSubCommands() bool {
	return c.command.HasSubCommands()
}

// Help 显示帮助信息
func (c *Cli) Help() error {
	return c.command.Help()
}

// HelpFunc 返回帮助函数
func (c *Cli) HelpFunc() func(*Command, []string) {
	return c.command.HelpFunc()
}

// HelpTemplate 返回帮助信息模板
func (c *Cli) HelpTemplate() string {
	return c.command.HelpTemplate()
}

// InitDefaultCompletionCmd 初始化默认的补全命令
func (c *Cli) InitDefaultCompletionCmd() {
	c.command.InitDefaultCompletionCmd()
}

// InitDefaultHelpCmd 初始化默认的帮助命令
func (c *Cli) InitDefaultHelpCmd() {
	c.command.InitDefaultHelpCmd()
}

// InitDefaultHelpFlag 初始化默认的帮助标志
func (c *Cli) InitDefaultHelpFlag() {
	c.command.InitDefaultHelpFlag()
}

// InitDefaultVersionFlag 初始化默认的版本标志
func (c *Cli) InitDefaultVersionFlag() {
	c.command.InitDefaultVersionFlag()
}

// IsAdditionalHelpTopicCommand 检查是否是额外的帮助主题命令
func (c *Cli) IsAdditionalHelpTopicCommand() bool {
	return c.command.IsAdditionalHelpTopicCommand()
}

// IsAvailableCommand 检查命令是否可用
func (c *Cli) IsAvailableCommand() bool {
	return c.command.IsAvailableCommand()
}

// Name 返回命令名称
func (c *Cli) Name() string {
	return c.command.Name()
}

// NameAndAliases 返回命令名称及其所有别名
func (c *Cli) NameAndAliases() string {
	return c.command.NameAndAliases()
}

// NamePadding 返回名称的填充长度
func (c *Cli) NamePadding() int {
	return c.command.NamePadding()
}

// Parent 返回父命令
func (c *Cli) Parent() *Command {
	return c.command.Parent()
}

// ParseFlags 解析命令行参数中的标志
func (c *Cli) ParseFlags(args []string) error {
	return c.command.ParseFlags(args)
}

// RegisterFlagCompletionFunc 注册标志的补全函数
func (c *Cli) RegisterFlagCompletionFunc(flagName string, f func(*Command, []string, string) ([]string, ShellCompDirective)) error {
	return c.command.RegisterFlagCompletionFunc(flagName, f)
}

// RemoveCommand 移除指定的子命令
func (c *Cli) RemoveCommand(cmds ...*Command) {
	c.command.RemoveCommand(cmds...)
}

// ResetCommands 重置所有子命令
func (c *Cli) ResetCommands() {
	c.command.ResetCommands()
}

// ResetFlags 重置所有标志
func (c *Cli) ResetFlags() {
	c.command.ResetFlags()
}

// Root 返回根命令
func (c *Cli) Root() *Command {
	return c.command.Root()
}

// Runnable 检查命令是否可运行
func (c *Cli) Runnable() bool {
	return c.command.Runnable()
}

// SuggestionsFor 返回针对给定输入的建议
func (c *Cli) SuggestionsFor(typedName string) []string {
	return c.command.SuggestionsFor(typedName)
}

// Traverse 遍历命令树查找匹配的命令
func (c *Cli) Traverse(args []string) (*Command, []string, error) {
	return c.command.Traverse(args)
}

// Usage 显示使用说明
func (c *Cli) Usage() error {
	return c.command.Usage()
}

// UsageFunc 返回使用说明函数
func (c *Cli) UsageFunc() func(*Command) error {
	return c.command.UsageFunc()
}

// UsagePadding 返回使用说明的填充长度
func (c *Cli) UsagePadding() int {
	return c.command.UsagePadding()
}

// UsageString 返回使用说明字符串
func (c *Cli) UsageString() string {
	return c.command.UsageString()
}

// UsageTemplate 返回使用说明模板
func (c *Cli) UsageTemplate() string {
	return c.command.UsageTemplate()
}

// UseLine 返回命令的使用行
func (c *Cli) UseLine() string {
	return c.command.UseLine()
}

// ValidateArgs 验证命令参数
func (c *Cli) ValidateArgs(args []string) error {
	return c.command.ValidateArgs(args)
}

// ValidateFlagGroups 验证标志组
func (c *Cli) ValidateFlagGroups() error {
	return c.command.ValidateFlagGroups()
}

// ValidateRequiredFlags 验证必需的标志
func (c *Cli) ValidateRequiredFlags() error {
	return c.command.ValidateRequiredFlags()
}

// VersionTemplate 返回版本信息模板
func (c *Cli) VersionTemplate() string {
	return c.command.VersionTemplate()
}

// VisitParents 访问所有父命令
func (c *Cli) VisitParents(fn func(*Command)) {
	c.command.VisitParents(fn)
}
