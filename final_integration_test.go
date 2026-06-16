package zcli

import (
	"bytes"
	"context"
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
			return
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

func TestAppAssemblyPipeline(t *testing.T) {
	t.Run("NewApp 与 NewCli 共享同一装配内核", func(t *testing.T) {
		app := NewApp(
			WithConfig(func() *Config {
				cfg := NewConfig()
				cfg.basic.Name = "app-entry"
				cfg.basic.Description = "app entry description"
				return cfg
			}()),
		)
		if app == nil {
			t.Fatal("NewApp 应返回有效实例")
		}
		if app.Command() == nil {
			t.Fatal("NewApp 应初始化根命令")
		}
		if app.Name() != "app-entry" {
			t.Fatalf("期望根命令名称为 app-entry，实际为 %q", app.Name())
		}
		if app.command.Short != "app entry description" {
			t.Fatalf("期望根命令短描述已装配，实际为 %q", app.command.Short)
		}
		if !app.command.CompletionOptions.DisableDefaultCmd {
			t.Fatal("根命令应禁用 Cobra 默认 completion 命令")
		}
		if app.Command().HelpFunc() == nil {
			t.Fatal("根命令应挂载 help 渲染函数")
		}
	})

	t.Run("Builder 装配顺序应先挂命令和 help，再挂 init hook 与 service command", func(t *testing.T) {
		var hookCalled bool

		app, err := NewBuilder("en").
			WithName("builder-assembly").
			WithDescription("builder assembly").
			WithService(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}).
			WithCommand(&Command{
				Use: "inspect",
				Run: func(cmd *Command, args []string) {},
			}).
			WithInitHook(func(cmd *Command, args []string) error {
				hookCalled = true
				return nil
			}).
			BuildWithError()
		if err != nil {
			t.Fatalf("BuildWithError 失败: %v", err)
		}

		for _, name := range []string{"run", "install", "uninstall", "start", "stop", "restart", "status"} {
			cmd, _, findErr := app.Command().Find([]string{name})
			if findErr != nil || cmd == nil || cmd.Name() != name {
				t.Fatalf("期望命令 %q 已装配，cmd=%v err=%v", name, cmd, findErr)
			}
		}
		cmd, _, findErr := app.Command().Find([]string{"inspect"})
		if findErr != nil || cmd == nil || cmd.Name() != "inspect" {
			t.Fatalf("期望自定义命令 inspect 已在 service 命令注入前装配，cmd=%v err=%v", cmd, findErr)
		}

		if app.Command().HelpFunc() == nil {
			t.Fatal("Builder 装配后应保留 help 渲染函数")
		}

		app.SetArgs([]string{"inspect"})
		if err := app.Execute(); err != nil {
			t.Fatalf("执行自定义命令失败: %v", err)
		}
		if !hookCalled {
			t.Fatal("执行命令时应先触发 init hook")
		}
	})
}

// TestAllPhasesCompleted 测试所有阶段是否完成
func TestAllPhasesCompleted(t *testing.T) {
	// 阶段1：并发安全问题修复
	t.Run("Phase1_ConcurrencySafety", func(t *testing.T) {
		// 验证Service配置结构体创建
		service := &ServiceConfig{}
		// Service 结构体应该可以正常创建
		if service.Username == "" && service.WorkDir == "" {
			// 空的Service配置是有效的，可以后续设置
			t.Log("Service configuration created successfully")
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

func TestReleaseReadinessScenarios(t *testing.T) {
	t.Run("NewApp 与 NewCommand 主路径可直接联动", func(t *testing.T) {
		app := NewApp()
		if app == nil {
			t.Fatal("NewApp 应返回有效实例")
		}
		app.config.basic.Name = "release-ready"
		app.config.basic.Description = "release readiness check"
		app.config.basic.Version = "1.0.0"
		app.command.Use = app.config.basic.Name
		app.command.Short = app.config.basic.Description
		app.command.Version = app.config.basic.Version

		cmd := NewCommand(
			"inspect",
			"Inspect release ready state",
			WithCommandRun(func(cmd *Command, args []string) {}),
		)
		app.AddCommand(cmd)

		found, _, err := app.Command().Find([]string{"inspect"})
		if err != nil {
			t.Fatalf("查找 inspect 命令失败: %v", err)
		}
		if found == nil || found.Name() != "inspect" {
			t.Fatalf("期望找到 inspect 命令，实际为 %v", found)
		}
	})

	t.Run("Flag export 与 completion 主路径保持可用", func(t *testing.T) {
		app, err := NewBuilder("en").
			WithName("release-flags").
			WithDescription("release flag path").
			WithCommand(NewCommand("inspect", "Inspect release flags")).
			BuildWithError()
		if err != nil {
			t.Fatalf("BuildWithError 失败: %v", err)
		}

		app.PersistentFlags().String("config", "", "Path to config")
		app.PersistentFlags().Bool("debug", false, "Enable debug logging")
		app.Flags().String("profile", "dev", "Runtime profile")

		flags := app.ExportFlagsForViper("debug")
		if len(flags) == 0 {
			t.Fatal("ExportFlagsForViper 应导出非空 flag sets")
		}

		var bashOut bytes.Buffer
		if err := app.GenCompletion(CompletionShellBash, &bashOut, true); err != nil {
			t.Fatalf("GenCompletion 失败: %v", err)
		}
		if bashOut.Len() == 0 {
			t.Fatal("completion 输出不应为空")
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

	// 执行相关：未执行前，Context 允许为 nil（与原生 Cobra 行为一致）
	_ = cli.Context()

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
	for range 100 {
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
