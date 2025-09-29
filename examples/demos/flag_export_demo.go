package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/darkit/zcli"
	"github.com/spf13/pflag"
)

// 模拟外部包的配置结构
type ExternalConfig struct {
	pflagSets []*pflag.FlagSet
}

// 模拟外部包的 WithBindPFlags 选项函数
func WithBindPFlags(flags ...*pflag.FlagSet) func(*ExternalConfig) {
	return func(c *ExternalConfig) {
		c.pflagSets = flags
	}
}

func main() {
	// 创建一个带有多种标志的应用
	app := zcli.NewBuilder("zh").
		WithName("flag-demo").
		WithDescription("重构后的标志导出功能演示").
		WithVersion("2.0.0").
		Build()

	// 添加各种类型的标志
	setupFlags(app)

	// 演示0: 显示系统标志列表
	fmt.Println("=== 演示0: 系统标志检查 ===")
	systemFlags := app.GetSystemFlags()
	sort.Strings(systemFlags) // 排序便于查看
	fmt.Printf("当前排除的系统标志 (%d 个):\n", len(systemFlags))
	for i, flag := range systemFlags {
		if i%5 == 0 && i > 0 {
			fmt.Printf("\n")
		}
		fmt.Printf("%-20s", flag)
	}
	fmt.Printf("\n\n")

	// 演示标志检查功能
	testFlags := []string{"help", "version", "debug", "config", "port"}
	fmt.Println("标志类型检查:")
	for _, flag := range testFlags {
		isSystem := app.IsSystemFlag(flag)
		fmt.Printf("  %-10s -> %s\n", flag, map[bool]string{true: "系统标志", false: "业务标志"}[isSystem])
	}

	// 演示1: 获取所有标志集合
	fmt.Println("\n=== 演示1: 获取所有标志集合 ===")
	allFlagSets := app.GetAllFlagSets()
	fmt.Printf("获取到 %d 个标志集合\n", len(allFlagSets))
	for i, flagSet := range allFlagSets {
		fmt.Printf("标志集合 %d: %d 个标志\n", i+1, countFlags(flagSet))
	}

	// 演示2: 获取可绑定的标志集合（排除系统标志）
	fmt.Println("\n=== 演示2: 智能过滤的标志集合 ===")
	bindableFlagSets := app.GetBindableFlagSets()
	fmt.Printf("获取到 %d 个可绑定标志集合\n", len(bindableFlagSets))
	for i, flagSet := range bindableFlagSets {
		fmt.Printf("可绑定标志集合 %d: ", i+1)
		var flags []string
		flagSet.VisitAll(func(flag *pflag.Flag) {
			flags = append(flags, flag.Name)
		})
		sort.Strings(flags) // 排序便于查看
		for j, flag := range flags {
			if j > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", flag)
		}
		fmt.Println()
	}

	// 演示3: 获取过滤后的单个标志集合
	fmt.Println("\n=== 演示3: 自定义过滤标志集合 ===")
	filteredFlags := app.GetFilteredFlags("debug", "port") // 额外排除 debug 和 port 标志
	fmt.Printf("排除 debug,port 后的标志: ")
	var flags []string
	filteredFlags.VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, flag.Name)
	})
	sort.Strings(flags)
	for i, flag := range flags {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", flag)
	}
	fmt.Println()

	// 演示4: 导出给 Viper 使用
	fmt.Println("\n=== 演示4: 导出给外部包使用 ===")
	viperFlags := app.ExportFlagsForViper("config", "debug") // 排除 config 和 debug
	externalConfig := &ExternalConfig{}
	// 模拟外部包的使用方式
	WithBindPFlags(viperFlags...)(externalConfig)
	fmt.Printf("传递给外部包的标志集合数量: %d\n", len(externalConfig.pflagSets))

	totalFlags := 0
	for i, flagSet := range externalConfig.pflagSets {
		count := countFlags(flagSet)
		totalFlags += count
		fmt.Printf("  集合 %d: %d 个标志\n", i+1, count)
	}
	fmt.Printf("  总计: %d 个可用标志\n", totalFlags)

	// 演示5: 获取标志名称列表
	fmt.Println("\n=== 演示5: 标志名称管理 ===")
	allFlagNames := app.GetFlagNames(true) // 包含继承的标志
	sort.Strings(allFlagNames)
	fmt.Printf("所有标志名称 (%d 个): %v\n", len(allFlagNames), allFlagNames)

	filteredFlagNames := app.GetFilteredFlagNames("debug", "config")
	sort.Strings(filteredFlagNames)
	fmt.Printf("过滤后的标志名称 (%d 个): %v\n", len(filteredFlagNames), filteredFlagNames)

	// 演示6: 高级过滤场景
	fmt.Println("\n=== 演示6: 高级过滤场景 ===")

	// 场景1：只排除帮助相关标志，保留其他系统标志
	helpOnlyFilter := app.GetFilteredFlagNames("help", "h")
	fmt.Printf("只排除帮助标志后剩余: %d 个\n", len(helpOnlyFilter))

	// 场景2：排除所有补全相关标志
	completionFlags := []string{
		"completion", "complete", "completion-bash",
		"completion-zsh", "completion-fish", "completion-powershell",
	}
	noCompletionFlags := app.GetFilteredFlagNames(completionFlags...)
	fmt.Printf("排除补全标志后剩余: %d 个\n", len(noCompletionFlags))

	// 演示7: 实际的命令行解析
	fmt.Println("\n=== 演示7: 实际命令行解析演示 ===")

	// 添加示例命令
	testCmd := &zcli.Command{
		Use:   "test",
		Short: "测试命令",
		Run: func(cmd *zcli.Command, args []string) {
			fmt.Println("执行测试命令")

			// 在命令中获取并显示标志值
			fmt.Println("\n当前标志值:")
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Changed {
					fmt.Printf("  %s = %s\n", flag.Name, flag.Value.String())
				}
			})

			// 显示外部包可用的标志
			fmt.Println("\n可传递给外部包的标志:")
			exportFlags := app.ExportFlagsForViper()
			for i, flagSet := range exportFlags {
				fmt.Printf("  标志集合 %d:\n", i+1)
				flagSet.VisitAll(func(flag *pflag.Flag) {
					if flag.Changed {
						fmt.Printf("    %s = %s (可传递)\n", flag.Name, flag.Value.String())
					}
				})
			}
		},
	}

	// 为测试命令添加本地标志
	testCmd.Flags().StringP("output", "o", "json", "输出格式")
	testCmd.Flags().BoolP("verbose", "V", false, "详细输出")

	app.AddCommand(testCmd)

	// 演示使用方式
	fmt.Println("\n使用示例:")
	fmt.Println("  go run flag_export_demo.go test --output yaml --verbose --config /path/to/config")
	fmt.Println("  go run flag_export_demo.go --help")
	fmt.Println("  go run flag_export_demo.go test --help")

	// 执行应用
	if err := app.Execute(); err != nil {
		log.Printf("执行失败: %v", err)
		os.Exit(1)
	}
}

// setupFlags 设置各种类型的标志
func setupFlags(app *zcli.Cli) {
	// 持久标志（会传递给子命令）
	app.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	app.PersistentFlags().BoolP("debug", "d", false, "启用调试模式")
	app.PersistentFlags().StringP("log-level", "l", "info", "日志级别")

	// 本地标志（仅限根命令）
	app.Flags().StringP("database", "D", "", "数据库连接字符串")
	app.Flags().IntP("port", "p", 8080, "服务端口")
	app.Flags().StringSliceP("tags", "t", nil, "标签列表")
	app.Flags().StringP("output-format", "f", "json", "输出格式")
	app.Flags().BoolP("dry-run", "n", false, "预览模式")
}

// countFlags 计算标志集合中的标志数量
func countFlags(flagSet *pflag.FlagSet) int {
	count := 0
	flagSet.VisitAll(func(*pflag.Flag) {
		count++
	})
	return count
}
