//go:build ignore

// 优雅的可变参数演示
// 展示 func(...context.Context)的魅力：既向下兼容又支持现代最佳实践

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
🌟 优雅的可变参数设计
 _____ _                   _   
| ____| | ___  __ _  __ _ | |_ 
|  _| | |/ _ \/ _' |/ _' || __|
| |___| |  __/ (_| | (_| || |_ 
|_____|_|\___|\__, |\__,_| \__|
              |___/            
`

// 演示函数1: 简单模式 - 不使用context的优雅停止
func legacyStyleService(ctx context.Context) error {
	slog.Info("=== 简单模式 ===")
	slog.Info("函数签名: func(ctx context.Context) error")
	slog.Info("调用方式: 简单的定时运行模式")

	// 简单模式：运行固定次数
	count := 0
	for count < 6 {
		slog.Info("简单模式服务运行中", "count", count+1)
		time.Sleep(time.Second)
		count++
	}

	slog.Info("简单模式服务完成")
	return nil
}

// 演示函数2: 现代最佳实践 - 使用context实现优雅停止
func modernStyleService(ctx context.Context) error {
	slog.Info("=== 现代最佳实践模式 ===")
	slog.Info("函数签名: func(ctx context.Context) error")
	slog.Info("调用方式: 使用context参数实现优雅停止")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("收到context停止信号，现代服务优雅退出")
			return nil
		case <-ticker.C:
			slog.Info("现代服务运行中，支持优雅停止...")
		}
	}
}

// 演示函数3: 高级用法 - 带超时控制的服务
func advancedStyleService(ctx context.Context) error {
	slog.Info("=== 高级用法模式 ===")
	slog.Info("函数签名: func(ctx context.Context) error")
	slog.Info("调用方式: 实现带超时控制的复杂逻辑")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("主context信号：高级服务优雅退出")
			return nil
		case <-ticker.C:
			slog.Info("高级服务运行中，支持复杂的控制逻辑...")
		}
	}
}

func cleanupFunc() error {
	slog.Info("执行清理工作...")
	time.Sleep(100 * time.Millisecond)
	slog.Info("清理完成")
	return nil
}

func createApp(mode string) *zcli.Cli {
	var serviceFunc func(context.Context) error
	var appName, desc, version string

	switch mode {
	case "legacy":
		serviceFunc = legacyStyleService
		appName = "elegant-legacy"
		desc = "展示简单模式的优雅设计"
		version = "1.0.0-legacy"
	case "modern":
		serviceFunc = modernStyleService
		appName = "elegant-modern"
		desc = "展示现代最佳实践的优雅设计"
		version = "1.0.0-modern"
	case "advanced":
		serviceFunc = advancedStyleService
		appName = "elegant-advanced"
		desc = "展示高级用法的优雅设计"
		version = "1.0.0-advanced"
	default:
		serviceFunc = modernStyleService
		appName = "elegant-demo"
		desc = "展示优雅的服务设计"
		version = "1.0.0"
	}

	return zcli.NewBuilder("zh").
		WithName(appName).
		WithDisplayName(fmt.Sprintf("【%s】", desc)).
		WithDescription(desc).
		WithLogo(logo).
		WithVersion(version).
		WithService(serviceFunc, cleanupFunc). // 使用新的 WithService API
		Build()
}

func printIntro() {
	fmt.Println("🌟 统一的服务API设计演示")
	fmt.Println()
	fmt.Println("✨ 设计亮点:")
	fmt.Println("  • 统一API: WithService(func(ctx context.Context) error, func() error)")
	fmt.Println("  • 标准签名: 符合Go语言惯例，支持错误返回")
	fmt.Println("  • 优雅停止: 通过context实现服务生命周期管理")
	fmt.Println("  • 类型安全: 编译时类型检查，避免错误使用")
	fmt.Println("  • 易于理解: 清晰的函数签名，降低学习成本")
	fmt.Println()
	fmt.Println("📋 运行方式:")
	fmt.Println("  go run elegant_variadic_demo.go [mode] run")
	fmt.Println("  mode可选: legacy | modern | advanced")
	fmt.Println()
}

func main() {
	printIntro()

	// 解析运行模式
	mode := "modern" // 默认现代模式
	if len(os.Args) > 1 && os.Args[1] != "run" {
		mode = os.Args[1]
		// 移除模式参数，避免传递给应用
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	app := createApp(mode)

	// 显示当前模式
	fmt.Printf("🎯 当前模式: %s\n\n", mode)

	// 执行应用
	if err := app.Execute(); err != nil {
		slog.Error("应用执行失败", "error", err)
		os.Exit(1)
	}
}
