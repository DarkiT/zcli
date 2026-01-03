// API兼容性演示
// 演示如何使用新的 WithService 方法的不同服务模式
//
// 运行示例:
//   go run api_compatibility_demo.go            # 使用现代API (推荐)
//   go run api_compatibility_demo.go legacy     # 使用简单模式

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

// 简单模式示例：不使用context优雅停止
func legacyServiceMain(ctx context.Context) error {
	slog.Info("=== 简单模式演示 ===")
	slog.Info("使用简单的服务函数风格")

	// 简单风格：运行固定次数
	count := 0
	for count < 8 {
		slog.Info("简单服务运行中", "count", count+1)
		time.Sleep(time.Second)
		count++
	}

	slog.Info("简单服务完成")
	return nil
}

// 现代模式示例：使用context参数（推荐方式）
func modernServiceMain(ctx context.Context) error {
	slog.Info("=== 现代模式演示 ===")
	slog.Info("使用context参数的服务函数，支持优雅停止")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("收到停止信号，现代服务优雅退出")
			return nil
		case <-ticker.C:
			slog.Info("现代服务运行中，按Ctrl+C可以优雅停止...")
		}
	}
}

func stopFunction() error {
	slog.Info("执行清理工作...")
	time.Sleep(100 * time.Millisecond)
	slog.Info("清理完成")
	return nil
}

func createLegacyApp() *zcli.Cli {
	fmt.Println("\n🔄 创建简单模式应用...")
	fmt.Println("使用 WithService() 方法，简单运行风格")

	return zcli.NewBuilder("zh").
		WithName("legacy-demo").
		WithDisplayName("简单模式演示").
		WithDescription("演示简单的服务用法").
		WithLogo(compatibilityLogo).
		WithVersion("1.0.0-legacy").
		WithService(legacyServiceMain, stopFunction). // 使用统一的方法
		Build()
}

func createModernApp() *zcli.Cli {
	fmt.Println("\n🚀 创建现代最佳实践应用...")
	fmt.Println("使用 WithService() 方法，现代context风格")

	return zcli.NewBuilder("zh").
		WithName("modern-demo").
		WithDisplayName("现代模式演示").
		WithDescription("演示支持优雅停止的现代用法").
		WithLogo(compatibilityLogo).
		WithVersion("1.0.0-modern").
		WithService(modernServiceMain, stopFunction). // 使用统一的方法
		Build()
}

func printUsageInfo() {
	fmt.Println("📋 统一服务API设计说明:")
	fmt.Println()
	fmt.Println("统一API:")
	fmt.Println("   WithService(func(ctx context.Context) error, func() error)")
	fmt.Println()
	fmt.Println("支持两种服务模式:")
	fmt.Println()
	fmt.Println("1. 简单模式:")
	fmt.Println("   func serviceName(ctx context.Context) error {")
	fmt.Println("       // 简单的运行逻辑，不使用context")
	fmt.Println("       return nil")
	fmt.Println("   }")
	fmt.Println("   - 适合简单的服务场景")
	fmt.Println("   - 不需要优雅停止")
	fmt.Println()
	fmt.Println("2. 现代模式 (推荐):")
	fmt.Println("   func serviceName(ctx context.Context) error {")
	fmt.Println("       select { case <-ctx.Done(): return nil }  // 优雅停止")
	fmt.Println("   }")
	fmt.Println("   - 支持优雅停止 (context.Done())")
	fmt.Println("   - 符合Go最佳实践")
	fmt.Println("   - 更好的资源管理")
	fmt.Println()
	fmt.Println("关键优势：统一API、标准签名、类型安全、易于理解")
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
