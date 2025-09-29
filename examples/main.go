package main

import (
	"context"
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

	// 使用新的context方式创建应用
	app := zcli.NewBuilder("zh").
		WithLogo(logo).
		WithName("demoapp").
		WithDisplayName("【演示应用】").
		WithDescription("这是一个演示优雅服务控制的应用").
		WithSystemService(runService, stopService). // 使用context版本的runService
		WithVersion("1.0.5").
		WithGitInfo("abc123", "master", "v1.0.5").
		WithWorkDir(workDir).
		WithEnvVar("ENV", "prod").
		WithDebug(true).
		Build()

	// 添加全局标志
	app.PersistentFlags().BoolP("debug", "d", false, "启用调试模式")
	app.PersistentFlags().StringP("config", "c", "", "配置文件路径")

	// 添加配置管理命令
	addConfigCommands(app)

	// 执行命令
	if err := app.Execute(); err != nil {
		slog.Error("应用执行失败", "error", err)
		os.Exit(1)
	}
}

// runService 服务主函数 - 使用可变context参数优雅处理生命周期
func runService(ctxs ...context.Context) {
	// 现代最佳实践：使用第一个context
	if len(ctxs) == 0 {
		slog.Info("没有context传入，使用默认行为")
		return
	}
	ctx := ctxs[0]
	slog.Info("服务已启动，等待停止信号...")

	// 创建定时器
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// 服务主循环 - 使用context.Done()优雅处理停止
	for {
		select {
		case <-ctx.Done():
			slog.Info("收到停止信号，准备退出服务循环")
			return
		case <-ticker.C:
			slog.Info("服务正在运行...")
		}
	}
}

// stopService 服务停止函数 - 执行清理工作
func stopService() {
	slog.Info("执行服务清理工作...")

	// 模拟清理工作
	time.Sleep(100 * time.Millisecond)

	slog.Info("服务清理完成，已安全停止")
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
		Use:   "show",
		Short: "查看当前配置",
		Run: func(cmd *zcli.Command, args []string) {
			fmt.Println("当前配置:")
			fmt.Println("- 服务名称: demoapp")
			fmt.Println("- 显示名称: 【演示应用】")
			fmt.Println("- 版本: 1.0.5")
			fmt.Println("- 运行模式: 优雅停止模式")
		},
	}

	updateCmd := &zcli.Command{
		Use:   "update",
		Short: "更新配置",
		Run: func(cmd *zcli.Command, args []string) {
			fmt.Println("更新配置...")
		},
	}

	configCmd.AddCommand(showCmd, updateCmd)
	app.AddCommand(configCmd)
}
