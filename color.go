package zcli

import "github.com/fatih/color"

type colors struct {
	// 基础元素
	Title       *color.Color
	Version     *color.Color
	Description *color.Color
	Logo        *color.Color

	// 状态信息
	Success *color.Color
	Error   *color.Color
	Info    *color.Color
	Warning *color.Color
	Debug   *color.Color

	// 命令相关
	Usage         *color.Color
	Command       *color.Color
	CommandDesc   *color.Color
	CommandsTitle *color.Color
	SubCommand    *color.Color

	// 选项相关
	OptionsTitle  *color.Color
	Option        *color.Color
	OptionDesc    *color.Color
	OptionDefault *color.Color

	// 示例相关
	ExamplesTitle *color.Color
	Example       *color.Color
	ExampleDesc   *color.Color

	// 其他元素
	Group     *color.Color
	Hint      *color.Color
	BuildInfo *color.Color
	Separator *color.Color
}

func newColors() *colors {
	return &colors{
		Title:       color.New(color.FgMagenta, color.Bold),  // 标题
		Version:     color.New(color.FgWhite, color.Bold),    // 版本号显示
		Description: color.New(color.FgHiYellow, color.Bold), // 命令描述文本
		Logo:        color.New(color.FgCyan),                 // Logo, ASCII 艺术字

		// 状态信息
		Success: color.New(color.FgGreen, color.Bold), // 成功提示、确认信息
		Error:   color.New(color.FgRed, color.Bold),   // 错误提示、警告信息
		Info:    color.New(color.FgWhite),             // 普通信息、默认文本
		Warning: color.New(color.FgYellow),            // 警告信息、注意事项
		Debug:   color.New(color.FgHiBlack),           // 调试信息、详细日志

		// 命令相关
		Usage:         color.New(color.FgGreen, color.Bold), // "Usage:" 标题
		Command:       color.New(color.FgGreen),             // 命令名称
		CommandDesc:   color.New(color.FgWhite, color.Bold), // 命令描述
		CommandsTitle: color.New(color.FgBlue, color.Bold),  // "Commands:" 标题
		SubCommand:    color.New(color.FgHiGreen),           // 子命令名称

		// 选项相关
		OptionsTitle:  color.New(color.FgMagenta, color.Bold), // "Options:" 标题
		Option:        color.New(color.FgCyan),                // 选项名称 (-h, --help)
		OptionDesc:    color.New(color.FgWhite, color.Bold),   // 选项描述
		OptionDefault: color.New(color.FgHiBlack),             // 选项默认值

		// 示例相关
		ExamplesTitle: color.New(color.FgGreen, color.Bold), // "Examples:" 标题
		Example:       color.New(color.FgYellow),            // 示例命令
		ExampleDesc:   color.New(color.FgHiWhite),           // 示例描述

		// 其他元素
		Group:     color.New(color.FgHiMagenta, color.Bold), // 命令分组标题
		Hint:      color.New(color.FgHiBlue, color.Bold),    // 提示信息、帮助文本
		BuildInfo: color.New(color.FgYellow),                // 构建信息、环境信息
		Separator: color.New(color.FgHiBlack),               // 分隔符、边框
	}
}
