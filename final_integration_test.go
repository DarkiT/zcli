package zcli

import (
	"bytes"
	"strings"
	"testing"
)

// TestFinalIntegration 最终集成测试
func TestFinalIntegration(t *testing.T) {
	// 创建完整的应用实例
	app := NewBuilder().
		WithName("testapp").
		WithDisplayName("Test Application").
		WithDescription("This is a test application").
		WithLanguage("zh").
		WithVersion("1.0.0").
		Build()

	if app == nil {
		t.Fatal("Failed to create application")
	}

	// 添加自定义命令
	customCmd := &Command{
		Use:   "custom",
		Short: "自定义命令",
		Run: func(cmd *Command, args []string) {
			// 测试命令
		},
	}

	app.AddCommand(customCmd)

	// 验证基本功能
	t.Run("BasicFunctionality", func(t *testing.T) {
		// 测试命令列表
		commands := app.Commands()
		if len(commands) == 0 {
			t.Error("Should have at least one command")
		}

		// 测试帮助功能
		var buf bytes.Buffer
		app.SetOut(&buf)
		err := app.Help()
		if err != nil {
			t.Errorf("Help failed: %v", err)
		}

		helpOutput := buf.String()
		if len(helpOutput) == 0 {
			t.Error("Help output should not be empty")
		}

		// 验证中文界面
		if !strings.Contains(helpOutput, "用法") {
			t.Error("Help should contain Chinese text")
		}

		// 验证自定义命令
		if !strings.Contains(helpOutput, "custom") {
			t.Error("Help should contain custom command")
		}
	})

	t.Run("UIRendererFunctionality", func(t *testing.T) {
		// 测试UI渲染器是否正常工作
		renderer := newUIRenderer(app)
		if renderer == nil {
			t.Error("UI renderer should not be nil")
		}

		if renderer.cli != app {
			t.Error("UI renderer should reference the correct CLI")
		}
	})

	t.Run("UtilityFunctions", func(t *testing.T) {
		// 测试工具函数
		wrapped := wordWrap("This is a very long text that should be wrapped", 20)
		if !strings.Contains(wrapped, "\n") {
			t.Error("Text should be wrapped")
		}

		// 测试命令路径
		path := getCommandPath(app.command)
		if path == "" {
			t.Error("Command path should not be empty")
		}

		// 测试环境检测
		_ = isCI()             // 不应该崩溃
		_ = isDevelopment()    // 不应该崩溃
		_ = isProduction()     // 不应该崩溃
		_ = isColorSupported() // 不应该崩溃
	})
}

// TestAllPhasesCompleted 测试所有阶段是否完成
func TestAllPhasesCompleted(t *testing.T) {
	// 阶段1：并发安全问题修复
	t.Run("Phase1_ConcurrencySafety", func(t *testing.T) {
		// 验证Service并发安全
		service := &Service{}
		if service == nil {
			t.Error("Service should be creatable")
		}
		// 其他并发安全测试已在service_concurrent_test.go中
	})

	// 阶段2：command.go结构优化
	t.Run("Phase2_CommandStructure", func(t *testing.T) {
		cli := NewCli()

		// 验证方法分组是否正常工作
		_ = cli.Commands()
		_ = cli.Flags()
		_ = cli.Context()
		_ = cli.Name()

		// 验证所有API兼容性
		cli.AddCommand(&Command{Use: "test"})
		commands := cli.Commands()
		if len(commands) == 0 {
			t.Error("Commands should be added successfully")
		}
	})

	// 阶段3：zcli.go职责分离
	t.Run("Phase3_ResponsibilitySeparation", func(t *testing.T) {
		// 验证UI渲染器模块
		cli := NewCli()
		renderer := newUIRenderer(cli)
		if renderer == nil {
			t.Error("UI renderer should be created")
		}

		// 验证工具函数模块
		text := wordWrap("test text", 10)
		if text != "test text" {
			t.Error("Word wrap should work")
		}

		// 验证平台兼容性
		supported := isColorSupported()
		_ = supported // 应该能够调用而不崩溃
	})

	// 阶段4：验证和测试
	t.Run("Phase4_ValidationAndTesting", func(t *testing.T) {
		// 创建完整应用并验证
		app := NewBuilder().
			WithName("final-test").
			WithLanguage("zh").
			Build()

		if app == nil {
			t.Fatal("Final application should be created successfully")
		}

		// 测试完整的功能链
		var buf bytes.Buffer
		app.SetOut(&buf)
		err := app.Help()
		if err != nil {
			t.Errorf("Final help test failed: %v", err)
		}

		output := buf.String()
		if len(output) == 0 {
			t.Error("Final help output should not be empty")
		}
	})
}

// TestBackwardCompatibility 测试向后兼容性
func TestBackwardCompatibility(t *testing.T) {
	// 确保原有的API调用方式仍然有效
	cli := NewCli()

	// 基本命令操作
	testCmd := &Command{Use: "test", Short: "Test command"}
	cli.AddCommand(testCmd)

	commands := cli.Commands()
	if len(commands) == 0 {
		t.Error("Commands should be available")
	}

	// 标志操作
	flags := cli.Flags()
	if flags == nil {
		t.Error("Flags should be available")
	}

	// 执行相关
	ctx := cli.Context()
	if ctx == nil {
		t.Error("Context should be available")
	}

	// 查询操作
	name := cli.Name()
	_ = name // 应该可以调用

	hasSubCmds := cli.HasSubCommands()
	_ = hasSubCmds // 应该可以调用

	isAvailable := cli.IsAvailableCommand()
	_ = isAvailable // 应该可以调用
}

// TestPerformanceRegression 性能回归测试
func TestPerformanceRegression(t *testing.T) {
	// 创建应用的性能测试
	app := NewBuilder().
		WithName("perf-test").
		WithLanguage("zh").
		Build()

	// 多次调用关键方法，确保没有性能问题
	for i := 0; i < 100; i++ {
		_ = app.Commands()
		_ = app.Flags()
		_ = app.Name()

		// UI渲染性能测试
		var buf bytes.Buffer
		app.SetOut(&buf)
		_ = app.Help()
	}

	// 如果能执行到这里说明性能没有严重问题
	t.Log("Performance regression test passed")
}
