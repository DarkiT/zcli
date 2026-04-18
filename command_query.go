package zcli

import "github.com/spf13/cobra"

// ============================================================================
// 查询和状态方法
// 用于查询命令的状态和属性信息
// ============================================================================

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
func (c *Cli) InitDefaultCompletionCmd(args ...string) {
	c.command.InitDefaultCompletionCmd(args...)
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

// DisplayName 返回用于帮助文案展示的命令名称
func (c *Cli) DisplayName() string {
	return c.command.DisplayName()
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
func (c *Cli) RegisterFlagCompletionFunc(
	flagName string,
	f func(*Command, []string, string) ([]string, ShellCompDirective),
) error {
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

// Find 在命令树中查找匹配的命令
func (c *Cli) Find(args []string) (*Command, []string, error) {
	return c.command.Find(args)
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

// Done 返回一个通道，当服务应该停止时会关闭
// 这为用户提供了优雅处理服务生命周期的方式
func (c *Cli) Done() <-chan struct{} {
	// 如果有服务管理器，返回其上下文的Done通道
	ctx := c.Context()
	return ctx.Done()
}

// SetServiceRunning 设置服务运行状态（内部使用）
// 用于在服务启动时传递正确的上下文
func (c *Cli) SetServiceRunning(running bool) {
	// 预留接口，用于将来的服务状态管理
}

// Command 返回底层的 cobra.Command 指针
// 用于需要直接操作 Cobra 原生 API 的高级场景
//
// 示例：
//
//	cmd := app.Command()
//	cmd.AddCommand(customCmd)
//	cmd.PreRun = func(cmd *cobra.Command, args []string) {
//	    // 自定义预运行逻辑
//	}
func (c *Cli) Command() *cobra.Command {
	return c.command
}

// Config 返回配置的副本（用于调试）
// 注意：返回的是副本，修改不会影响原配置
func (c *Cli) Config() Config {
	clone := Config{ctx: c.config.ctx}

	clone.basic = cloneBasicPtr(c.config.basic)
	clone.service = cloneServicePtr(c.config.service)
	clone.runtime = cloneRuntimePtr(c.config.runtime)

	return clone
}
