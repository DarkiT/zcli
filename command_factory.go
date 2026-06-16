package zcli

import "github.com/spf13/cobra"

// CommandOption 定义命令构造阶段的加法式配置钩子。
// 它只负责补充命令语义，不替代底层 Cobra 命令对象。
type CommandOption func(cmd *Command)

// NewCommand 创建一个新的命令辅助对象。
// 该函数返回的仍然是同一个 Cobra 原生命令对象，只是用 zcli 词汇提供更顺手的构造入口。
func NewCommand(use, short string, opts ...CommandOption) *Command {
	cmd := &Command{
		Use:   use,
		Short: short,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cmd)
		}
	}

	return cmd
}

// WithCommandAliases 为命令设置别名列表。
func WithCommandAliases(aliases ...string) CommandOption {
	return func(cmd *Command) {
		cmd.Aliases = append([]string(nil), aliases...)
	}
}

// WithCommandLong 为命令设置长描述。
func WithCommandLong(long string) CommandOption {
	return func(cmd *Command) {
		cmd.Long = long
	}
}

// WithCommandExample 为命令设置示例文本。
func WithCommandExample(example string) CommandOption {
	return func(cmd *Command) {
		cmd.Example = example
	}
}

// WithCommandRun 为命令设置 Run 回调。
func WithCommandRun(run func(cmd *Command, args []string)) CommandOption {
	return func(cmd *Command) {
		cmd.Run = run
	}
}

// WithCommandRunE 为命令设置 RunE 回调。
func WithCommandRunE(runE func(cmd *Command, args []string) error) CommandOption {
	return func(cmd *Command) {
		cmd.RunE = runE
	}
}

// WithCommandArgs 为命令设置位置参数校验器。
func WithCommandArgs(args PositionalArgs) CommandOption {
	return func(cmd *Command) {
		cmd.Args = args
	}
}

// WithCommandGroup 为命令设置所属命令组 ID。
func WithCommandGroup(groupID string) CommandOption {
	return func(cmd *Command) {
		cmd.GroupID = groupID
	}
}

// WithCommandValidArgs 为命令设置固定的有效位置参数列表。
func WithCommandValidArgs(args ...Completion) CommandOption {
	return func(cmd *Command) {
		cmd.ValidArgs = completionsToStrings(args)
	}
}

// WithCommandCompletion 为命令设置位置参数补全函数。
func WithCommandCompletion(completion CompletionFunc) CommandOption {
	return func(cmd *Command) {
		cmd.ValidArgsFunction = completion
	}
}

// WithCommandFlags 允许在命令创建阶段配置本地标志。
func WithCommandFlags(configure func(flags *FlagSet)) CommandOption {
	return func(cmd *Command) {
		if configure == nil {
			return
		}
		configure(cmd.Flags())
	}
}

// WithCommandPersistentFlags 允许在命令创建阶段配置持久标志。
func WithCommandPersistentFlags(configure func(flags *FlagSet)) CommandOption {
	return func(cmd *Command) {
		if configure == nil {
			return
		}
		configure(cmd.PersistentFlags())
	}
}

// WithCommandSubcommands 允许在命令创建阶段一次性追加子命令。
func WithCommandSubcommands(children ...*Command) CommandOption {
	return func(cmd *Command) {
		if len(children) == 0 {
			return
		}
		cmd.AddCommand(children...)
	}
}

// ExactArgs 返回“参数数量必须精确匹配”的校验器。
func ExactArgs(n int) PositionalArgs {
	return cobra.ExactArgs(n)
}

// MinimumNArgs 返回“参数数量至少为 n”的校验器。
func MinimumNArgs(n int) PositionalArgs {
	return cobra.MinimumNArgs(n)
}

// MaximumNArgs 返回“参数数量至多为 n”的校验器。
func MaximumNArgs(n int) PositionalArgs {
	return cobra.MaximumNArgs(n)
}

// RangeArgs 返回“参数数量必须在区间内”的校验器。
func RangeArgs(min, max int) PositionalArgs {
	return cobra.RangeArgs(min, max)
}

// NoArgs 返回“不接受任何位置参数”的校验器。
func NoArgs() PositionalArgs {
	return cobra.NoArgs
}

// ArbitraryArgs 返回“接受任意位置参数”的校验器。
func ArbitraryArgs() PositionalArgs {
	return cobra.ArbitraryArgs
}

func completionsToStrings(args []Completion) []string {
	if len(args) == 0 {
		return nil
	}

	converted := make([]string, 0, len(args))
	for _, arg := range args {
		converted = append(converted, string(arg))
	}
	return converted
}
