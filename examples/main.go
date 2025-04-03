package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/darkit/zcli"
)

const logo = `
███████╗████████╗ ██████╗  ██████╗ ██╗     
╚══███╔╝╚══██╔══╝██╔═══██╗██╔═══██╗██║     
  ███╔╝    ██║   ██║   ██║██║   ██║██║     
 ███╔╝     ██║   ██║   ██║██║   ██║██║     
███████╗   ██║   ╚██████╔╝╚██████╔╝███████╗
╚══════╝   ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝
`

func main() {
	workDir, _ := os.UserHomeDir()
	app := zcli.NewBuilder().
		WithLogo(logo).WithName("demoapp").WithDisplayName("【演示应用】").WithDescription("这是一个演示应用").WithSystemService(run, stop).
		WithLanguage("zh").WithVersion("1.0.5").WithGitInfo("abc123", "master", "v1.0.5").
		WithWorkDir(workDir).WithEnvVar("ENV", "prod").WithDebug(true).
		Build()

	// 添加全局标志
	app.PersistentFlags().BoolP("debug", "d", false, "启用调试模式")
	app.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	// 添加配置管理命令
	addConfigCommands(app)

	// 执行命令
	if err := app.Execute(); err != nil {
		slog.Error(err.Error())
	}
}

// 服务主函数
func run() {
	slog.Info("服务已启动")

	// 创建定时器
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// 服务主循环 - 简化为只检查isRunning标志
	for range ticker.C {
		slog.Info("服务正在运行...")
	}
}

// 服务停止函数
func stop() {
	slog.Info("服务已停止")
}

// 添加配置管理命令
func addConfigCommands(app *zcli.Cli) {
	configCmd := &zcli.Command{
		Use:   "config",
		Short: "配置管理",
		Run: func(cmd *zcli.Command, args []string) {
			slog.Info("配置管理", "name", cmd.Name(), "args", args)
		},
	}

	showCmd := &zcli.Command{
		Use:                   "show",
		Short:                 "查看当前配置",
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		Run: func(cmd *zcli.Command, args []string) {
			fmt.Println("当前配置:")
			fmt.Println("- 服务名称: demoapp")
			fmt.Println("- 显示名称: 演示应用")
			fmt.Println("- 版本: 1.0.5")
		},
	}

	updateCmd := &zcli.Command{
		Use:                   "update",
		Short:                 "更新配置",
		DisableFlagParsing:    true, // 禁用标志解析
		DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		Run: func(cmd *zcli.Command, args []string) {
			fmt.Println("更新配置...")
		},
	}

	configCmd.AddCommand(showCmd, updateCmd)
	app.AddCommand(configCmd)
}
