// Package zcli 是一个基于cobra的命令行应用框架，提供了更友好的命令行界面和系统服务管理功能。
//
// ZCli的主要特性包括：
//
// 友好的命令行界面:
//   - 支持彩色输出，自动检测终端颜色支持
//   - 自定义Logo显示
//   - 分组显示命令（普通命令和系统命令）
//   - 优化的帮助信息展示
//   - 支持命令补全
//
// 系统服务集成:
//   - 一键集成系统服务管理功能
//   - 支持服务的安装、卸载、启动、停止、重启和状态查询
//   - 支持服务配置的自定义（工作目录、环境变量等）
//   - 跨平台支持 (Windows, Linux, macOS)
//
// 国际化支持:
//   - 内置中文和英文支持
//   - 可扩展的语言包系统
//   - 支持自定义语言包和错误消息
//
// 版本信息管理:
//   - 详细的版本信息展示
//   - 支持构建信息的自定义
//   - 支持调试/发布模式切换
//
// 使用示例:
//
//	package main
//
//	import (
//	    "log/slog"
//	    "os"
//	    "time"
//
//	    "github.com/darkit/zcli"
//	)
//
//	var isRunning = true
//
//	func main() {
//	    workDir, _ := os.UserHomeDir()
//	    app := zcli.NewBuilder("zh").
//	        WithName("myapp").
//	        WithDisplayName("我的应用").
//	        WithDescription("这是一个示例应用").
//	        WithVersion("1.0.0").
//	        WithWorkDir(workDir).
//	        WithSystemService(run, stop).
//	        Build()
//
//	    // 执行应用
//	    if err := app.Execute(); err != nil {
//	        slog.Error(err.Error())
//	    }
//	}
//
//	func run() {
//	    slog.Info("服务已启动")
//	    isRunning = true
//	    ticker := time.NewTicker(time.Second)
//	    defer ticker.Stop()
//
//	    for range ticker.C {
//	        if !isRunning {
//	            break
//	        }
//	        slog.Info("服务正在运行...")
//	    }
//	}
//
//	func stop() {
//	    slog.Warn("服务停止中...")
//	    isRunning = false
//	    slog.Info("服务已停止")
//	}
//
// 主要结构:
//
// Builder: CLI构建器，使用链式调用方式配置应用
//
//	app := zcli.NewBuilder("zh").
//	    WithName("myapp").
//	    WithSystemService(run, stop).
//	    Build()
//
// Cli: 命令行应用对象，封装了cobra.Command
//
//	// 添加命令
//	app.AddCommand(myCommand)
//	// 执行应用
//	app.Execute()
//
// Config: 应用配置，包含基本信息、运行时配置和服务配置
//
// Language: 语言包定义，支持多语言
//
// Command: 命令定义，等同于cobra.Command
//
// 最低Go版本要求: 1.19 (使用了atomic.Bool等特性)
//
// 更多信息和示例请参考: https://github.com/darkit/zcli
package zcli
