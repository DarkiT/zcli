package zcli

import (
	"context"
	"io"
)

// ============================================================================
// 输入输出管理方法
// 用于管理命令的输入输出流和打印功能
// ============================================================================

// InOrStdin 返回命令的输入流，默认为标准输入
func (c *Cli) InOrStdin() io.Reader {
	return c.command.InOrStdin()
}

// ErrOrStderr 返回命令的错误输出流，默认为标准错误
func (c *Cli) ErrOrStderr() io.Writer {
	return c.command.ErrOrStderr()
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

// ============================================================================
// 配置和模板方法
// 用于配置命令的行为、模板和处理函数
// ============================================================================

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

// ErrPrefix 返回错误消息前缀
func (c *Cli) ErrPrefix() string {
	return c.command.ErrPrefix()
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
//
// Deprecated: Use SetOut and/or SetErr instead
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
