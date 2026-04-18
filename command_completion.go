package zcli

import "io"

// ============================================================================
// 命令行补全方法
// 用于生成各种Shell的命令行补全脚本
// ============================================================================

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

// ============================================================================
// 标志标记和验证方法
// 用于标记标志的特殊属性和验证规则
// ============================================================================

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
