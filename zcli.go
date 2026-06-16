package zcli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
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

// applyBuilderAssembly 统一收束 Builder 到 App/Cli 的装配顺序。
// 顺序固定为：版本/基础 root → help/UI → init hooks → service commands。
func (c *Cli) applyBuilderAssembly(pendingCmds []*Command, initHooks []InitHook) {
	root := c.Command()
	if root == nil {
		return
	}

	if len(pendingCmds) > 0 {
		c.AddCommand(pendingCmds...)
	}

	c.configureRootCommand(root)
	c.attachInitHooks(root, initHooks)
	c.setupService()
}

// configureRootCommand 为根命令注入 help/UI/completion 等装配能力。
func (c *Cli) configureRootCommand(rootCmd *Command) {
	if rootCmd == nil {
		return
	}
	c.addHelpCommand(rootCmd)

	// 设置控制台颜色支持
	if c.config.basic.NoColor || !isColorSupported() {
		color.NoColor = true
	}

	// 禁用默认的 completion 命令
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// 使用 UI 渲染器覆盖帮助展示
	renderer := newUIRenderer(c)
	rootCmd.SetHelpFunc(renderer.renderHelp)
}

// addRootCommand 兼容旧装配调用，内部统一转向新的 root command 配置入口。
func (c *Cli) addRootCommand(rootCmd *Command) {
	c.configureRootCommand(rootCmd)
}

// attachInitHooks 将 Builder 注册的 init hooks 统一附着到 root command。
func (c *Cli) attachInitHooks(root *Command, hooks []InitHook) {
	if root == nil || len(hooks) == 0 {
		return
	}

	prev := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *Command, args []string) error {
		if prev != nil {
			if err := prev(cmd, args); err != nil {
				return err
			}
		}
		for _, hook := range hooks {
			if err := hook(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *Cli) addHelpCommand(rootCmd *Command) {
	// 创建自定义帮助命令
	helpCmd := &Command{
		Use:   "help",
		Short: c.lang.UI.Help.Command,
		Long:  c.lang.UI.Help.Description,
		RunE: func(cc *Command, args []string) error {
			if len(args) == 0 {
				// 如果没有参数，显示根命令的帮助
				return rootCmd.Help()
			}

			// 查找指定的命令
			cmd, _, err := rootCmd.Find(args)
			if cmd == nil || err != nil {
				// 命令未找到
				_, _ = fmt.Fprintf(
					cc.OutOrStderr(),
					"%s%s\n\n",
					c.colors.Error.Sprint(c.lang.Error.Prefix),
					fmt.Sprintf(c.lang.Error.Help.UnknownTopic, args),
				)
				return rootCmd.Usage()
			}

			// 初始化命令的帮助标志
			cmd.InitDefaultHelpFlag()
			cmd.InitDefaultVersionFlag()

			// 显示找到的命令的帮助信息
			return cmd.Help()
		},
		// 添加命令补全功能
		ValidArgsFunction: func(cc *Command, args []string, toComplete string) ([]Completion, ShellCompDirective) {
			var completions []string

			// 获取当前路径的命令
			cmd, _, e := cc.Root().Find(args)
			if e != nil {
				return nil, ShellCompDirectiveNoFileComp
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

			return completions, ShellCompDirectiveNoFileComp
		},
	}

	// 设置使用模板
	// helpCmd.SetUsageTemplate(fmt.Sprintf(c.lang.Command.HelpUsage, rootCmd.CommandPath()))

	// 配置根命令的帮助选项
	rootCmd.PersistentFlags().BoolP("help", "h", false, c.lang.UI.Help.Description)

	// 禁用自动生成的帮助命令
	rootCmd.SetHelpCommand(helpCmd)

	// 设置帮助函数
	rootCmd.SetHelpFunc(func(cmd *Command, args []string) {
		// 如果是帮助命令本身，使用默认帮助
		if cmd.Name() == "help" {
			_ = cmd.Usage()
			return
		}
		// 否则使用自定义的帮助显示
		c.configureRootCommand(cmd)
	})
}

// showVersion 显示版本信息
func (c *Cli) showVersion(buf *strings.Builder) {
	if c.config.runtime.BuildInfo != nil {
		buf.WriteString(c.colors.Description.Sprint(c.config.runtime.BuildInfo.String()))
	} else if c.config.basic.Version != "" {
		const versionFormat = "Version: %s\n"
		buf.WriteString(c.colors.Description.Sprintf(versionFormat, c.config.basic.Version))
	}
}
