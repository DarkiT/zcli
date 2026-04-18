package zcli

import (
	"fmt"

	"github.com/spf13/pflag"
)

// ============================================================================
// 便捷标志导出方法
// 用于导出标志给外部包使用，支持过滤和批量操作
// ============================================================================

// defaultSystemFlags 返回 Cobra 系统默认标志列表
// 这些标志通常不应该传递给外部业务包
func getDefaultSystemFlags() map[string]bool {
	return map[string]bool{
		// 帮助系统
		"help": true,
		"h":    true,

		// 版本系统
		"version": true,
		"v":       true,

		// 补全系统 - 标准补全命令
		"completion": true,
		"complete":   true,

		// 补全系统 - Shell 特定补全
		"completion-bash":       true,
		"completion-zsh":        true,
		"completion-fish":       true,
		"completion-powershell": true,
		"gen-completion":        true,

		// 补全系统 - 内部调试标志
		"__complete":       true,
		"__completeNoDesc": true,
		"no-descriptions":  true,

		// 补全系统 - 传统/兼容性标志
		"bash-completion":       true,
		"zsh-completion":        true,
		"fish-completion":       true,
		"powershell-completion": true,

		// 内部调试和开发标志
		"debug-completion": true,
		"trace-completion": true,

		// 配置系统常见标志（可能由框架自动添加）
		"config-help":     true,
		"print-config":    true,
		"validate-config": true,
	}
}

// flagFilter 统一的标志过滤器
type flagFilter struct {
	excluded map[string]bool
}

// newFlagFilter 创建新的标志过滤器
func newFlagFilter(additionalExcludes ...string) *flagFilter {
	excluded := getDefaultSystemFlags()

	// 添加用户指定的排除标志
	for _, flag := range additionalExcludes {
		excluded[flag] = true
	}

	return &flagFilter{excluded: excluded}
}

// shouldInclude 检查标志是否应该包含
func (f *flagFilter) shouldInclude(flagName string) bool {
	return !f.excluded[flagName]
}

// createFilteredFlagSet 创建过滤后的标志集合
func (f *flagFilter) createFilteredFlagSet(source *FlagSet, name string) *FlagSet {
	filtered := pflag.NewFlagSet(name, pflag.ContinueOnError)

	source.VisitAll(func(flag *pflag.Flag) {
		if f.shouldInclude(flag.Name) {
			filtered.AddFlag(flag)
		}
	})

	return filtered
}

// getExcludedFlags 返回当前排除的标志列表（用于调试）
func (f *flagFilter) getExcludedFlags() []string {
	var excluded []string
	for flag := range f.excluded {
		excluded = append(excluded, flag)
	}
	return excluded
}

// GetAllFlagSets 返回所有标志集合的切片，便于传递给外部包
// 返回顺序：[本地标志, 持久标志, 继承标志]
func (c *Cli) GetAllFlagSets() []*FlagSet {
	var flagSets []*FlagSet

	if c.HasLocalFlags() {
		flagSets = append(flagSets, c.LocalFlags())
	}

	if c.HasPersistentFlags() {
		flagSets = append(flagSets, c.PersistentFlags())
	}

	if c.HasInheritedFlags() {
		flagSets = append(flagSets, c.InheritedFlags())
	}

	return flagSets
}

// GetBindableFlagSets 返回适用于绑定的标志集合，自动排除常见的系统标志
// excludeFlags: 额外需要排除的标志名称
func (c *Cli) GetBindableFlagSets(excludeFlags ...string) []*FlagSet {
	filter := newFlagFilter(excludeFlags...)
	var filteredFlagSets []*FlagSet
	allFlagSets := c.GetAllFlagSets()

	for i, flagSet := range allFlagSets {
		filtered := filter.createFilteredFlagSet(flagSet, fmt.Sprintf("filtered-%d", i))

		// 只有当过滤后的标志集不为空时才添加
		if filtered.HasFlags() {
			filteredFlagSets = append(filteredFlagSets, filtered)
		}
	}

	return filteredFlagSets
}

// GetFilteredFlags 返回过滤后的单个标志集合，包含所有非排除的标志
func (c *Cli) GetFilteredFlags(excludeFlags ...string) *FlagSet {
	filter := newFlagFilter(excludeFlags...)
	return filter.createFilteredFlagSet(c.Flags(), "filtered-all")
}

// ExportFlagsForViper 导出适用于 Viper 绑定的标志集合
// 这是一个便捷方法，返回可以直接用于 WithBindPFlags 的标志数组
func (c *Cli) ExportFlagsForViper(excludeFlags ...string) []*FlagSet {
	return c.GetBindableFlagSets(excludeFlags...)
}

// GetFlagNames 返回所有标志的名称列表
func (c *Cli) GetFlagNames(includeInherited bool) []string {
	var names []string
	nameSet := make(map[string]bool)

	// 收集本地和持久标志
	c.Flags().VisitAll(func(flag *pflag.Flag) {
		if !nameSet[flag.Name] {
			names = append(names, flag.Name)
			nameSet[flag.Name] = true
		}
	})

	// 如果需要，添加继承的标志
	if includeInherited {
		c.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
			if !nameSet[flag.Name] {
				names = append(names, flag.Name)
				nameSet[flag.Name] = true
			}
		})
	}

	return names
}

// GetFilteredFlagNames 返回过滤后的标志名称列表
func (c *Cli) GetFilteredFlagNames(excludeFlags ...string) []string {
	filter := newFlagFilter(excludeFlags...)
	var names []string
	nameSet := make(map[string]bool)

	c.Flags().VisitAll(func(flag *pflag.Flag) {
		if filter.shouldInclude(flag.Name) && !nameSet[flag.Name] {
			names = append(names, flag.Name)
			nameSet[flag.Name] = true
		}
	})

	return names
}

// GetSystemFlags 返回当前被排除的系统标志列表（调试用）
func (c *Cli) GetSystemFlags() []string {
	filter := newFlagFilter()
	return filter.getExcludedFlags()
}

// IsSystemFlag 检查指定标志是否为系统标志
func (c *Cli) IsSystemFlag(flagName string) bool {
	systemFlags := getDefaultSystemFlags()
	return systemFlags[flagName]
}
