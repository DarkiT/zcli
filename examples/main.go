package main

import (
	"fmt"
	"log"
	"log/slog"
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

type App struct {
	zcli *zcli.Cli
}

func main() {
	var app App

	version := zcli.NewVersionInfo().SetVersion("1.0.5").SetDebug(true).SetBuildTime(time.Now())

	// 创建主命令
	app.zcli = zcli.NewCli(
		zcli.WithName("demoapp"),         // 必需：服务名称
		zcli.WithDisplayName("演示应用"),     // 服务显示名称
		zcli.WithDescription("这是一个演示应用"), // 服务描述
		zcli.WithLanguage("zh"),          // 使用中文
		zcli.WithLogo(logo),              // Logo
		zcli.WithRun(run),                // 必需：服务主函数
		zcli.WithStop(stop),
		zcli.WithBuildInfo(version),
		// zcli.WithVersion("1.0.0"),        // 版本
	)
	app.zcli.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode") // 示例：添加调试模式
	app.zcli.PersistentFlags().StringP("config", "c", "", "Config file path")  // 示例：添加配置文件路径

	// 添加配置管理命令
	app.addConfigCommands()

	// 执行命令
	if err := app.zcli.Execute(); err != nil {
		log.Fatal(err)
	}
}

// 服务主函数
func run() {
	for {
		time.Sleep(time.Second * 5)
		fmt.Println("服务正在运行...")
	}
}

func stop() {
	fmt.Println("服务停止中...")
}

// 添加配置管理命令
func (a *App) addConfigCommands() {
	configCmd := &zcli.Command{
		Use:   "config",
		Short: "配置管理",
		// DisableFlagParsing:    true, // 禁用标志解析
		// DisableFlagsInUseLine: true, // 在使用说明中禁用标志
		Run: func(cmd *zcli.Command, args []string) {
			slog.Info("try to service", "name", cmd.Name(), "args", args)
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
			fmt.Println("- 版本: 1.0.0")
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
	a.zcli.AddCommand(configCmd)
}
