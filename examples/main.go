package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
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

	// 使用新的服务配置方式
	app := zcli.NewBuilder("zh").
		WithLogo(logo).
		WithName("demoapp").
		WithDisplayName("【演示应用】").
		WithDescription("这是一个演示优雅服务控制的应用").
		WithService(runService, stopService). // 使用新的 WithService 方法
		WithServiceTimeouts(0, 5*time.Second). // 演示强制退出（停止超时）
		WithShutdownTimeouts(1*time.Second, 1*time.Second).
		WithVersion("1.0.5").
		WithGitInfo("89447cf2c914ea19d06c30d155d1f6202dbdc54c ", "master", "v1.0.5").
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

// runService 服务主函数 - 标准签名：func(ctx context.Context) error
// 此示例故意忽略 ctx，用于验证无法优雅退出的场景
func runService(_ context.Context) error {
	slog.Info("服务启动中，初始化资源...")

	initTimer := time.NewTimer(200 * time.Millisecond)
	<-initTimer.C

	jobs := make(chan int, 8)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for jobID := range jobs {
			time.Sleep(150 * time.Millisecond)
			slog.Info("处理任务", "id", jobID)
		}
	}()

	producer := time.NewTicker(500 * time.Millisecond)
	defer producer.Stop()

	jobID := 0
	for range producer.C {
		jobID++
		select {
		case jobs <- jobID:
		default:
			slog.Warn("任务队列已满，丢弃任务", "id", jobID)
		}
	}

	// 模拟无响应退出的服务：不处理 ctx，不关闭 jobs，不等待 wg
	// 该 return 不会触发，因为 producer.C 永不结束
	return nil
}

// stopService 服务停止函数 - 标准签名：func() error
func stopService() error {
	slog.Info("执行服务清理工作...")

	// 模拟清理工作
	time.Sleep(100 * time.Millisecond)

	slog.Info("服务清理完成，已安全停止")
	return nil
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
