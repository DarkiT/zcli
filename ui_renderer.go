package zcli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// uiRenderer 负责UI渲染逻辑
type uiRenderer struct {
	cli    *Cli
	colors *colors
	lang   *Language
}

// newUIRenderer 创建UI渲染器
func newUIRenderer(cli *Cli) *uiRenderer {
	return &uiRenderer{
		cli:    cli,
		colors: cli.colors,
		lang:   cli.lang,
	}
}

// renderHelp 渲染帮助信息
func (r *uiRenderer) renderHelp(cc *cobra.Command, args []string) {
	var buf strings.Builder
	defer buf.Reset()

	buf.Grow(4096)
	cmdPath := getCommandPath(cc)

	// 渲染各个部分
	r.renderLogo(&buf, cc)
	r.renderDescription(&buf, cc)
	r.renderUsage(&buf, cc, cmdPath)
	r.renderOptions(&buf, cc)
	r.renderCommands(&buf, cc)
	r.renderExamples(&buf, cc)
	r.renderHelpHint(&buf, cc, cmdPath)

	_, _ = fmt.Fprint(cc.OutOrStderr(), buf.String())
}

// renderLogo 渲染Logo部分
func (r *uiRenderer) renderLogo(buf *strings.Builder, cc *cobra.Command) {
	// Logo只在根命令显示
	if r.cli.config.Basic.Logo != "" && cc.Parent() == nil {
		buf.WriteString(separator)
		buf.WriteString(r.colors.Logo.Sprint(strings.TrimSpace(r.cli.config.Basic.Logo)))

		// Version
		if r.cli.command.Version != "" {
			buf.WriteString(r.colors.Logo.Sprintf(" %s %s", r.lang.UI.Version.Label, strings.TrimLeft(r.cli.command.Version, "v")))
		}
		buf.WriteString(separator)
		buf.WriteString(separator)
	}
}

// renderDescription 渲染描述部分
func (r *uiRenderer) renderDescription(buf *strings.Builder, cc *cobra.Command) {
	if cc.Long != "" {
		buf.WriteString(r.colors.Description.Sprint(wordWrap(cc.Long, 80)))
		buf.WriteString(separator)
	} else if cc.Short != "" {
		buf.WriteString(r.colors.Description.Sprint(cc.Short))
		buf.WriteString(separator)
	}
}

// renderUsage 渲染使用方法部分
func (r *uiRenderer) renderUsage(buf *strings.Builder, cc *cobra.Command, cmdPath string) {
	buf.WriteString(separator)
	buf.WriteString(r.colors.Usage.Sprintf("%s:", r.lang.UI.Commands.Usage))
	buf.WriteString(separator)

	// 只有当命令有标志时才显示 [参数]
	if cc.HasAvailableLocalFlags() {
		buf.WriteString(indent)
		buf.WriteString(r.colors.Info.Sprintf("%s [%s]", cmdPath, r.lang.UI.Commands.Flags))
		buf.WriteString(separator)
	}

	// 如果有子命令，显示带命令的用法
	if cc.HasAvailableSubCommands() {
		buf.WriteString(indent)
		if cc.HasAvailableLocalFlags() {
			buf.WriteString(r.colors.Info.Sprintf("%s [%s] [%s]", cmdPath, "command", r.lang.UI.Commands.Flags))
		} else {
			buf.WriteString(r.colors.Info.Sprintf("%s [%s]", cmdPath, "command"))
		}
		buf.WriteString(separator)
	}
}

// renderOptions 渲染选项部分
func (r *uiRenderer) renderOptions(buf *strings.Builder, cc *cobra.Command) {
	if !cc.HasAvailableLocalFlags() {
		return
	}

	buf.WriteString(separator)
	buf.WriteString(r.colors.OptionsTitle.Sprintf("%s", r.lang.UI.Commands.Options))
	buf.WriteString(separator)

	flags := cc.LocalFlags()
	flags.VisitAll(func(f *pflag.Flag) {
		r.renderFlag(buf, f)
	})
}

// renderFlag 渲染单个标志
func (r *uiRenderer) renderFlag(buf *strings.Builder, f *pflag.Flag) {
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

	buf.WriteString(r.colors.Option.Sprint(flagLine))
	buf.WriteString(r.colors.OptionDesc.Sprint(f.Usage))
	if f.DefValue != "" && f.DefValue != "false" {
		buf.WriteString(r.colors.OptionDefault.Sprintf(" "+r.lang.UI.Commands.DefaultValue, f.DefValue))
	}
	buf.WriteString(separator)
}

// renderCommands 渲染命令部分
func (r *uiRenderer) renderCommands(buf *strings.Builder, cc *cobra.Command) {
	if !cc.HasAvailableSubCommands() {
		return
	}

	buf.WriteString(separator)
	buf.WriteString(r.colors.CommandsTitle.Sprintf("%s", r.lang.UI.Commands.AvailableCommands))
	buf.WriteString(separator)

	// 分组处理命令
	normalCmds, systemCmds := r.groupCommands(cc)

	// 显示普通命令
	r.renderCommandGroup(buf, normalCmds, false)

	// 显示系统命令（如果存在）
	if len(normalCmds) > 0 && len(systemCmds) > 0 {
		buf.WriteString(separator)
		buf.WriteString(r.colors.CommandsTitle.Sprintf("%s", r.lang.UI.Commands.SystemCommands))
		buf.WriteString(separator)
	}
	r.renderCommandGroup(buf, systemCmds, true)
}

// groupCommands 对命令进行分组
func (r *uiRenderer) groupCommands(cc *cobra.Command) ([]*cobra.Command, []*cobra.Command) {
	var normalCmds, systemCmds []*cobra.Command

	for _, cmd := range cc.Commands() {
		if cmd.IsAvailableCommand() || cmd.Name() == "help" {
			if _, isSystem := systemCmdOrder[cmd.Name()]; isSystem {
				systemCmds = append(systemCmds, cmd)
			} else {
				normalCmds = append(normalCmds, cmd)
			}
		}
	}

	// 排序
	sort.Slice(normalCmds, func(i, j int) bool {
		return len(normalCmds[i].Name()) < len(normalCmds[j].Name())
	})

	sort.Slice(systemCmds, func(i, j int) bool {
		return systemCmdOrder[systemCmds[i].Name()] < systemCmdOrder[systemCmds[j].Name()]
	})

	return normalCmds, systemCmds
}

// renderCommandGroup 渲染命令组
func (r *uiRenderer) renderCommandGroup(buf *strings.Builder, cmds []*cobra.Command, isSystem bool) {
	for _, cmd := range cmds {
		cmdLine := indent + r.colors.SubCommand.Sprintf("%-*s", spacing-len(indent), cmd.Name())
		buf.WriteString(cmdLine)
		buf.WriteString(r.colors.CommandDesc.Sprint(cmd.Short))
		buf.WriteString(separator)
	}
}

// renderExamples 渲染示例部分
func (r *uiRenderer) renderExamples(buf *strings.Builder, cc *cobra.Command) {
	if !cc.HasExample() {
		return
	}

	buf.WriteString(separator)
	buf.WriteString(r.colors.ExamplesTitle.Sprintf("%s:", r.lang.UI.Commands.Examples))
	buf.WriteString(separator)

	examples := strings.Split(cc.Example, separator)
	for _, example := range examples {
		if example = strings.TrimSpace(example); example != "" {
			if strings.HasPrefix(example, "$ ") {
				// 命令示例
				buf.WriteString(indent)
				buf.WriteString(r.colors.Example.Sprint(example))
			} else {
				// 说明文字
				buf.WriteString(r.colors.ExampleDesc.Sprint(example))
			}
			buf.WriteString(separator)
		}
	}
}

// renderHelpHint 渲染帮助提示部分
func (r *uiRenderer) renderHelpHint(buf *strings.Builder, cc *cobra.Command, cmdPath string) {
	if !cc.HasAvailableSubCommands() {
		return
	}

	buf.WriteString(separator)
	hint := fmt.Sprintf(r.lang.UI.Help.Usage, cmdPath)
	// 如果是子命令提示则删除 [command]
	if cc.Parent() != nil {
		hint = strings.ReplaceAll(hint, " [command]", "")
	}
	buf.WriteString(r.colors.Hint.Sprint(hint))
	buf.WriteString(separator)
}
