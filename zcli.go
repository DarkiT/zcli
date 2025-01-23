package zcli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	indent    = "   " // 基础缩进
	spacing   = 24    // 命令和描述之间的空格数
	separator = "\n"  // 统一使用单个换行作为分隔符
)

// 定义系统命令的固定顺序
var systemCmdOrder = map[string]int{
	"start":     1,
	"stop":      2,
	"status":    3,
	"restart":   4,
	"install":   5,
	"uninstall": 6,
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

	rootCmd.SetHelpFunc(func(cc *cobra.Command, args []string) {
		var buf strings.Builder
		defer buf.Reset()

		buf.Grow(4096)

		cmdPath := getCommandPath(cc)

		// Logo
		if c.config.Basic.Logo != "" && cc.Parent() == nil {
			buf.WriteString(separator)
			buf.WriteString(c.colors.Logo.Sprint(strings.TrimSpace(c.config.Basic.Logo)))
			// Version
			if c.command.Version != "" {
				buf.WriteString(c.colors.Logo.Sprintf(" %s %s", c.lang.Command.Version, strings.TrimLeft(c.command.Version, "v")))
			}
			buf.WriteString(separator)
			buf.WriteString(separator)
		}

		// Description
		if cc.Long != "" {
			buf.WriteString(c.colors.Description.Sprint(wordWrap(cc.Long, 80)))
			buf.WriteString(separator)
		} else if cc.Short != "" {
			buf.WriteString(c.colors.Description.Sprint(cc.Short))
			buf.WriteString(separator)
		}

		// Usage
		buf.WriteString(separator)
		buf.WriteString(c.colors.Usage.Sprintf("%s:", c.lang.Command.Usage))
		buf.WriteString(separator)

		// 只有当命令有标志时才显示 [参数]
		if cc.HasAvailableLocalFlags() {
			buf.WriteString(indent)
			buf.WriteString(c.colors.Info.Sprintf("%s [%s]", cmdPath, c.lang.Command.Flags))
			buf.WriteString(separator)
		}

		// 如果有子命令，显示带命令的用法
		if cc.HasAvailableSubCommands() {
			buf.WriteString(indent)
			if cc.HasAvailableLocalFlags() {
				buf.WriteString(c.colors.Info.Sprintf("%s [%s] [%s]", cmdPath, c.lang.Command.Command, c.lang.Command.Flags))
			} else {
				buf.WriteString(c.colors.Info.Sprintf("%s [%s]", cmdPath, c.lang.Command.Command))
			}
			buf.WriteString(separator)
		}

		// Options
		if cc.HasAvailableLocalFlags() {
			buf.WriteString(separator)
			buf.WriteString(c.colors.OptionsTitle.Sprintf("%s", c.lang.Command.Options))
			buf.WriteString(separator)
			flags := cc.LocalFlags()
			flags.VisitAll(func(f *pflag.Flag) {
				flagLine := indent
				if f.Shorthand != "" {
					flagLine += fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
				} else {
					flagLine += fmt.Sprintf("    --%s", f.Name)
				}

				padding := spacing - len(flagLine) + len(indent) - 3
				if padding > 0 {
					flagLine += strings.Repeat(" ", padding)
				}

				buf.WriteString(c.colors.Option.Sprint(flagLine))
				buf.WriteString(c.colors.OptionDesc.Sprint(f.Usage))
				if f.DefValue != "" && f.DefValue != "false" {
					buf.WriteString(c.colors.OptionDefault.Sprintf(" "+c.lang.Command.DefaultValue, f.DefValue))
				}
				buf.WriteString(separator)
			})
		}

		// Command
		if cc.HasAvailableSubCommands() {
			buf.WriteString(separator)
			buf.WriteString(c.colors.CommandsTitle.Sprintf("%s", c.lang.Command.AvailableCommands))
			buf.WriteString(separator)

			// 对命令进行分组
			normalCmds := make([]*cobra.Command, 0)
			systemCmds := make([]*cobra.Command, 0)

			for _, cmd := range cc.Commands() {
				if cmd.IsAvailableCommand() || cmd.Name() == "help" {
					// 检查是否是系统命令
					if _, isSystem := systemCmdOrder[cmd.Name()]; isSystem {
						systemCmds = append(systemCmds, cmd)
					} else {
						normalCmds = append(normalCmds, cmd)
					}
				}
			}

			// 对普通命令按名称长度排序
			sort.Slice(normalCmds, func(i, j int) bool {
				return len(normalCmds[i].Name()) < len(normalCmds[j].Name())
			})

			// 显示普通命令
			for _, cmd := range normalCmds {
				cmdLine := indent + c.colors.SubCommand.Sprintf("%-*s", spacing-len(indent), cmd.Name())
				buf.WriteString(cmdLine)
				buf.WriteString(c.colors.CommandDesc.Sprint(cmd.Short))
				buf.WriteString(separator)
			}

			// 如果同时存在普通命令和系统命令，添加一个分隔行
			if len(normalCmds) > 0 && len(systemCmds) > 0 {
				buf.WriteString(separator)
				buf.WriteString(c.colors.CommandsTitle.Sprintf("%s", c.lang.Command.SystemCommands))
				buf.WriteString(separator)
			}

			// 对系统命令按预定义顺序排序
			sort.Slice(systemCmds, func(i, j int) bool {
				return systemCmdOrder[systemCmds[i].Name()] < systemCmdOrder[systemCmds[j].Name()]
			})

			// 显示系统命令
			for _, cmd := range systemCmds {
				cmdLine := indent + c.colors.SubCommand.Sprintf("%-*s", spacing-len(indent), cmd.Name())
				buf.WriteString(cmdLine)
				buf.WriteString(c.colors.CommandDesc.Sprint(cmd.Short))
				buf.WriteString(separator)
			}
		}

		// Examples
		if cc.HasExample() {
			buf.WriteString(separator)
			buf.WriteString(c.colors.ExamplesTitle.Sprintf("%s:", c.lang.Command.Examples))
			buf.WriteString(separator)
			examples := strings.Split(cc.Example, separator)
			for _, example := range examples {
				if example = strings.TrimSpace(example); example != "" {
					if strings.HasPrefix(example, "$ ") {
						// 命令示例
						buf.WriteString(indent)
						buf.WriteString(c.colors.Example.Sprint(example))
					} else {
						// 说明文字
						buf.WriteString(c.colors.ExampleDesc.Sprint(example))
					}
					buf.WriteString(separator)
				}
			}
		}

		// Help hint
		if cc.HasAvailableSubCommands() {
			buf.WriteString(separator)
			hint := fmt.Sprintf(c.lang.Command.HelpUsage, cmdPath)
			// 如果是子命令提示则删除  [command]
			if cc.Parent() != nil {
				hint = strings.ReplaceAll(hint, " [command]", "")
			}
			buf.WriteString(c.colors.Hint.Sprint(hint))
			buf.WriteString(separator)
		}

		_, _ = fmt.Fprint(cc.OutOrStderr(), buf.String())
	})
}

func (c *Cli) addHelpCommand(rootCmd *cobra.Command) {
	// 创建自定义帮助命令
	helpCmd := &cobra.Command{
		Use:   "help",
		Short: c.lang.Command.HelpCommand,
		Long:  c.lang.Command.HelpDesc,
		RunE: func(cc *cobra.Command, args []string) error {
			if len(args) == 0 {
				// 如果没有参数，显示根命令的帮助
				return rootCmd.Help()
			}

			// 查找指定的命令
			cmd, _, err := rootCmd.Find(args)
			if cmd == nil || err != nil {
				// 命令未找到
				_, _ = fmt.Fprintf(cc.OutOrStderr(), "%s%s\n\n", c.colors.Error.Sprint(c.lang.Error), fmt.Sprintf(c.lang.Error.UnknownHelpTopic, args))
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
	rootCmd.PersistentFlags().BoolP("help", "h", false, c.lang.Command.HelpCommand)

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

// wordWrap 将文本按指定宽度换行
func wordWrap(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) <= width {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
}

// isColorSupported 检查终端是否支持彩色输出
func isColorSupported() bool {
	// 1. 首先检查是否明确禁用了颜色
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	// 2. 检查是否在CI环境中
	if os.Getenv("CI") != "" {
		// 检查常见的CI环境
		ciEnvs := []string{
			"GITHUB_ACTIONS",
			"GITLAB_CI",
			"TRAVIS",
			"CIRCLECI",
			"JENKINS_URL",
			"TEAMCITY_VERSION",
		}
		for _, env := range ciEnvs {
			if os.Getenv(env) != "" {
				return true
			}
		}
	}

	// 3. 检查 COLORTERM 环境变量
	if os.Getenv("COLORTERM") != "" {
		return true
	}

	// 4. 检查终端类型
	term := os.Getenv("TERM")
	if term != "" {
		colorTerms := []string{
			"xterm",
			"vt100",
			"color",
			"ansi",
			"cygwin",
			"linux",
		}
		for _, cterm := range colorTerms {
			if strings.Contains(term, cterm) {
				return true
			}
		}
	}

	// 5. 平台特定检查
	if !isWindowsColorSupported() {
		return false
	}

	// 6. 检查是否是标准终端
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	}

	return false
}

// 获取完整的命令路径
func getCommandPath(cc *cobra.Command) string {
	base := filepath.Base(os.Args[0])
	if cc.Name() == "" {
		return base
	}
	// 如果是子命令且有父命令
	if cc.Parent() != nil {
		// 返回 "可执行文件名 子命令名"
		return fmt.Sprintf("%s %s", base, cc.Name())
	}

	return cc.CommandPath()
}
