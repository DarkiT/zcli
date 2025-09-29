// API兼容性演示
// 演示如何使用新的 WithSystemService 方法的不同调用方式
//
// 运行示例:
//   go run api_compatibility_demo.go            # 使用现代API (推荐)
//   go run api_compatibility_demo.go legacy     # 使用传统无参数方式

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/darkit/zcli"
)

const compatibilityLogo = `
 █████╗ ██████╗ ██╗
██╔══██╗██╔══██╗██║
███████║██████╔╝██║
██╔══██║██╔═══╝ ██║
██║  ██║██║     ██║
╚═╝  ╚═╝╚═╝     ╚═╝ 兼容性演示
`

// 传统API示例：无参数函数（向下兼容）
func legacyServiceMain(ctxs ...context.Context) {
	slog.Info("=== 传统API演示 ===")
	slog.Info("使用无context参数的服务函数风格")

	// 传统风格：忽略context参数，自定义循环
	count := 0
	for count < 8 {
		slog.Info("传统服务运行中", "count", count+1)
		time.Sleep(time.Second)
		count++

		// 传统方式需要自行检查停止条件
		// 这里简单模拟运行有限次数
	}

	slog.Info("传统服务完成（无优雅停止机制）")
}

// 现代API示例：使用context参数（推荐方式）
func modernServiceMain(ctxs ...context.Context) {
	slog.Info("=== 现代API演示 ===")
	slog.Info("使用context参数的服务函数，支持优雅停止")

	// 现代最佳实践：使用第一个context
	if len(ctxs) == 0 {
		slog.Info("没有context传入，使用默认行为")
		return
	}
	ctx := ctxs[0]

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("收到停止信号，现代服务优雅退出")
			return
		case <-ticker.C:
			slog.Info("现代服务运行中，按Ctrl+C可以优雅停止...")
		}
	}
}

func stopFunction() {
	slog.Info("执行清理工作...")
	time.Sleep(100 * time.Millisecond)
	slog.Info("清理完成")
}

func createLegacyApp() *zcli.Cli {
	fmt.Println("\n🔄 创建传统兼容模式应用...")
	fmt.Println("使用 WithSystemService() 方法，传统无参数调用风格")

	return zcli.NewBuilder("zh").
		WithName("legacy-demo").
		WithDisplayName("传统API兼容演示").
		WithDescription("演示向下兼容的传统API用法").
		WithLogo(compatibilityLogo).
		WithVersion("1.0.0-legacy").
		WithSystemService(legacyServiceMain, stopFunction). // 使用统一的方法
		Build()
}

func createModernApp() *zcli.Cli {
	fmt.Println("\n🚀 创建现代最佳实践应用...")
	fmt.Println("使用 WithSystemService() 方法，现代context风格")

	return zcli.NewBuilder("zh").
		WithName("modern-demo").
		WithDisplayName("现代API最佳实践演示").
		WithDescription("演示支持优雅停止的现代API用法").
		WithLogo(compatibilityLogo).
		WithVersion("1.0.0-modern").
		WithSystemService(modernServiceMain, stopFunction). // 使用统一的方法
		Build()
}

func printUsageInfo() {
	fmt.Println("📋 优雅可变参数设计说明:")
	fmt.Println()
	fmt.Println("统一API:")
	fmt.Println("   WithSystemService(func(...context.Context), ...func())")
	fmt.Println()
	fmt.Println("支持三种调用方式:")
	fmt.Println()
	fmt.Println("1. 传统风格 (向下兼容):")
	fmt.Println("   func serviceName(ctxs ...context.Context) {")
	fmt.Println("       // 忽略ctxs参数，使用传统循环逻辑")
	fmt.Println("   }")
	fmt.Println("   - 向下兼容现有代码")
	fmt.Println("   - 无需修改现有逻辑")
	fmt.Println()
	fmt.Println("2. 现代风格 (推荐):")
	fmt.Println("   func serviceName(ctxs ...context.Context) {")
	fmt.Println("       ctx := ctxs[0]  // 使用第一个context")
	fmt.Println("       select { case <-ctx.Done(): return }  // 优雅停止")
	fmt.Println("   }")
	fmt.Println("   - 支持优雅停止 (context.Done())")
	fmt.Println("   - 符合Go最佳实践")
	fmt.Println()
	fmt.Println("3. 高级扩展 (未来):")
	fmt.Println("   func serviceName(ctxs ...context.Context) {")
	fmt.Println("       mainCtx, cancelCtx := ctxs[0], ctxs[1]  // 多context")
	fmt.Println("   }")
	fmt.Println("   - 支持多种控制机制")
	fmt.Println()
	fmt.Println("💡 关键优势：单一API、类型安全、完全兼容、易于扩展")
	fmt.Println()
}

func main() {
	printUsageInfo()

	var app *zcli.Cli

	// 根据命令行参数决定使用哪种调用风格
	if len(os.Args) > 1 && os.Args[1] == "legacy" {
		app = createLegacyApp()
		// 移除已处理的参数，避免传递给应用
		os.Args = append(os.Args[:1], os.Args[2:]...)
	} else {
		app = createModernApp()
	}

	// 执行应用
	if err := app.Execute(); err != nil {
		slog.Error("应用执行失败", "error", err)
		os.Exit(1)
	}
}
