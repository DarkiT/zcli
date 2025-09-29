package zcli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	indent    = "   " // 基础缩进
	spacing   = 24    // 命令和描述之间的空格数
	separator = "\n"  // 统一使用单个换行作为分隔符
)

// 定义系统命令的固定顺序
var systemCmdOrder = map[string]int{
	"run":       1,
	"start":     2,
	"stop":      3,
	"status":    4,
	"restart":   5,
	"install":   6,
	"uninstall": 7,
}

// addRootCommand 在初始化时设置语言包
func (c *Cli) addRootCommand(rootCmd *cobra.Command) {
	c.addHelpCommand(rootCmd)

	// 设置控制台颜色支持
	if c.config.Basic.NoColor || !isColorSupported() {
		color.NoColor = true
	}

	// 禁用默认的 completion 命令
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// 使用UI渲染器
	renderer := newUIRenderer(c)
	rootCmd.SetHelpFunc(renderer.renderHelp)
}

func (c *Cli) addHelpCommand(rootCmd *cobra.Command) {
	// 创建自定义帮助命令
	helpCmd := &cobra.Command{
		Use:   "help",
		Short: c.lang.UI.Help.Command,
		Long:  c.lang.UI.Help.Description,
		RunE: func(cc *cobra.Command, args []string) error {
			if len(args) == 0 {
				// 如果没有参数，显示根命令的帮助
				return rootCmd.Help()
			}

			// 查找指定的命令
			cmd, _, err := rootCmd.Find(args)
			if cmd == nil || err != nil {
				// 命令未找到
				_, _ = fmt.Fprintf(cc.OutOrStderr(), "%s%s\n\n", c.colors.Error.Sprint(c.lang.Error.Prefix), fmt.Sprintf(c.lang.Error.Help.UnknownTopic, args))
				return rootCmd.Usage()
			}

			// 初始化命令的帮助标志
			cmd.InitDefaultHelpFlag()
			cmd.InitDefaultVersionFlag()

			// 显示找到的命令的帮助信息
			return cmd.Help()
		},
		// 添加命令补全功能
		ValidArgsFunction: func(cc *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var completions []string

			// 获取当前路径的命令
			cmd, _, e := cc.Root().Find(args)
			if e != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if cmd == nil {
				cmd = cc.Root()
			}

			// 收集可用的子命令
			for _, subCmd := range cmd.Commands() {
				if subCmd.IsAvailableCommand() {
					if strings.HasPrefix(subCmd.Name(), toComplete) {
						// 添加命令及其简短描述
						completions = append(completions, fmt.Sprintf("%s\t%s", subCmd.Name(), subCmd.Short))
					}
				}
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
	}

	// 设置使用模板
	// helpCmd.SetUsageTemplate(fmt.Sprintf(c.lang.Command.HelpUsage, rootCmd.CommandPath()))

	// 配置根命令的帮助选项
	rootCmd.PersistentFlags().BoolP("help", "h", false, c.lang.UI.Help.Command)

	// 禁用自动生成的帮助命令
	rootCmd.SetHelpCommand(helpCmd)

	// 设置帮助函数
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// 如果是帮助命令本身，使用默认帮助
		if cmd.Name() == "help" {
			_ = cmd.Usage()
			return
		}
		// 否则使用自定义的帮助显示
		c.addRootCommand(cmd)
	})
}

// showVersion 显示版本信息
func (c *Cli) showVersion(buf *strings.Builder) {
	if c.config.Runtime.BuildInfo != nil {
		buf.WriteString(c.colors.Description.Sprint(c.config.Runtime.BuildInfo.String()))
	} else if c.config.Basic.Version != "" {
		const versionFormat = "Version: %s\n"
		buf.WriteString(c.colors.Description.Sprintf(versionFormat, c.config.Basic.Version))
	}
}
