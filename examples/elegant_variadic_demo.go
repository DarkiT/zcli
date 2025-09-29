//go:build ignore

// 优雅的可变参数演示
// 展示 func(...context.Context) 的魅力：既向下兼容又支持现代最佳实践

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

// 演示函数1: 向下兼容模式 - 忽略context参数
func legacyStyleService(ctxs ...context.Context) {
	slog.Info("=== 向下兼容模式 ===")
	slog.Info("函数签名: func(ctxs ...context.Context)")
	slog.Info("调用方式: 可以忽略传入的context参数")

	// 向下兼容：用户可以完全忽略context参数
	count := 0
	for count < 6 {
		slog.Info("兼容模式服务运行中", "count", count+1)
		time.Sleep(time.Second)
		count++
	}

	slog.Info("兼容模式服务完成（用户自行控制生命周期）")
}

// 演示函数2: 现代最佳实践 - 使用第一个context
func modernStyleService(ctxs ...context.Context) {
	slog.Info("=== 现代最佳实践模式 ===")
	slog.Info("函数签名: func(ctxs ...context.Context)")
	slog.Info("调用方式: 使用第一个context参数实现优雅停止")

	// 检查是否有context传入
	if len(ctxs) == 0 {
		slog.Warn("没有context传入，使用默认行为")
		return
	}

	// 使用第一个context
	ctx := ctxs[0]

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("收到context停止信号，现代服务优雅退出")
			return
		case <-ticker.C:
			slog.Info("现代服务运行中，支持优雅停止...")
		}
	}
}

// 演示函数3: 高级用法 - 处理多个context
func advancedStyleService(ctxs ...context.Context) {
	slog.Info("=== 高级用法模式 ===")
	slog.Info("函数签名: func(ctxs ...context.Context)")
	slog.Info("调用方式: 可以处理多个context，实现复杂的控制逻辑")

	if len(ctxs) == 0 {
		slog.Info("无context参数，运行基础模式")
		time.Sleep(2 * time.Second)
		return
	}

	// 使用第一个context作为主要的生命周期控制
	mainCtx := ctxs[0]

	// 如果有多个context，可以用于不同的控制信号
	slog.Info("接收到的context数量", "count", len(ctxs))

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-mainCtx.Done():
			slog.Info("主context信号：高级服务优雅退出")
			return
		case <-ticker.C:
			slog.Info("高级服务运行中，支持复杂的控制逻辑...")
		}
	}
}

func cleanupFunc() {
	slog.Info("执行清理工作...")
	time.Sleep(100 * time.Millisecond)
	slog.Info("清理完成")
}

func createApp(mode string) *zcli.Cli {
	var serviceFunc func(...context.Context)
	var appName, desc, version string

	switch mode {
	case "legacy":
		serviceFunc = legacyStyleService
		appName = "elegant-legacy"
		desc = "展示向下兼容的优雅设计"
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
		desc = "展示优雅的可变参数设计"
		version = "1.0.0"
	}

	return zcli.NewBuilder("zh").
		WithName(appName).
		WithDisplayName(fmt.Sprintf("【%s】", desc)).
		WithDescription(desc).
		WithLogo(logo).
		WithVersion(version).
		WithSystemService(serviceFunc, cleanupFunc). // 🌟 优雅的统一API
		Build()
}

func printIntro() {
	fmt.Println("🌟 优雅的可变参数设计演示")
	fmt.Println()
	fmt.Println("✨ 设计亮点:")
	fmt.Println("  • 单一API: WithSystemService(func(...context.Context), ...func())")
	fmt.Println("  • 向下兼容: 现有的func()逻辑可以忽略context参数")
	fmt.Println("  • 现代最佳实践: 新代码可以使用context实现优雅停止")
	fmt.Println("  • 类型安全: 编译时检查，避免用户混淆")
	fmt.Println("  • 扩展性: 支持未来的多context高级用法")
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
