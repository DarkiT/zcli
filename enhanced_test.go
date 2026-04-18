package zcli

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestEnhancedBuilderAPI 测试增强的Builder API
func TestEnhancedBuilderAPI(t *testing.T) {
	t.Run("基本Builder功能", func(t *testing.T) {
		cli := NewBuilder("zh").
			WithName("test-app").
			WithDisplayName("测试应用").
			WithDescription("这是一个测试应用").
			WithVersion("1.0.0").
			Build()

		if cli.Name() != "test-app" {
			t.Errorf("期望应用名称为 'test-app'，实际为 '%s'", cli.Name())
		}
	})

	t.Run("ServiceRunner集成", func(t *testing.T) {
		serviceFunc := func(ctx context.Context) error {
			// 等待一段时间或上下文取消
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		stopFunc := func() error {
			return nil
		}

		cli := NewBuilder("zh").
			WithName("service-test").
			WithService(serviceFunc, stopFunc).
			Build()

		if cli.Name() != "service-test" {
			t.Errorf("期望服务名称为 'service-test'，实际为 '%s'", cli.Name())
		}

		// 这里我们无法直接测试服务运行，因为它需要完整的服务管理器
		// 但我们可以验证配置是否正确设置
		if cli.config.runtime.Run == nil {
			t.Error("期望设置了运行函数")
		}

		if cli.config.runtime.Stop == nil {
			t.Error("期望设置了停止函数")
		}
	})

	t.Run("BuildWithError方法", func(t *testing.T) {
		// 测试正常构建
		cli, err := NewBuilder("zh").
			WithName("valid-app").
			WithDisplayName("有效应用").
			BuildWithError()
		if err != nil {
			t.Errorf("期望构建成功，但得到错误: %v", err)
		}

		if cli == nil {
			t.Error("期望得到有效的CLI实例")
		}

		// 测试验证失败
		_, err = NewBuilder("zh").
			// 缺少名称，应该验证失败
			WithValidator(func(cfg *Config) error {
				if cfg.basic.Name == "" {
					return fmt.Errorf("名称不能为空")
				}
				return nil
			}).
			BuildWithError()

		if err == nil {
			t.Error("期望构建失败，但没有得到错误")
		}
	})

	t.Run("新版服务配置Builder语义", func(t *testing.T) {
		cli, err := NewBuilder("zh").
			WithName("dep-app").
			WithDependencies("postgresql", "redis").
			WithDependency("network.target", DependencyAfter).
			WithServiceOption("Restart", "on-failure").
			WithAllowSudoFallback(true).
			BuildWithError()
		if err != nil {
			t.Fatalf("期望构建成功，但得到错误: %v", err)
		}

		if len(cli.config.service.StructuredDeps) != 3 {
			t.Fatalf("期望生成3个结构化依赖，实际为 %d", len(cli.config.service.StructuredDeps))
		}
		if cli.config.service.StructuredDeps[0].Type != DependencyRequire {
			t.Fatalf("WithDependencies 应默认生成 require 依赖，实际为 %q", cli.config.service.StructuredDeps[0].Type)
		}
		if cli.config.service.Options["Restart"] != "on-failure" {
			t.Fatalf("期望服务选项被写入，实际为 %#v", cli.config.service.Options)
		}
		if !cli.config.service.AllowSudoFallback {
			t.Fatal("期望 AllowSudoFallback 被写入")
		}

		cli, err = NewBuilder("zh").
			WithName("legacy-dep-app").
			WithLegacyDependencies("After=network-online.target").
			BuildWithError()
		if err != nil {
			t.Fatalf("期望构建成功，但得到错误: %v", err)
		}
		if len(cli.config.service.Dependencies) != 1 {
			t.Fatalf("期望保留原生字符串依赖，实际为 %#v", cli.config.service.Dependencies)
		}
		if len(cli.config.service.StructuredDeps) != 0 {
			t.Fatalf("WithLegacyDependencies 不应保留结构化依赖，实际为 %#v", cli.config.service.StructuredDeps)
		}
	})

	t.Run("BuildWithError保留InitHook与延迟命令构建", func(t *testing.T) {
		var hookCalled atomic.Bool

		cli, err, panicked := func() (cli *Cli, err error, panicked bool) {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()

			cli, err = NewBuilder("zh").
				WithCommand(&Command{
					Use: "ping",
					Run: func(cmd *Command, args []string) {},
				}).
				WithName("hook-app").
				WithInitHook(func(cmd *cobra.Command, args []string) error {
					hookCalled.Store(true)
					return nil
				}).
				BuildWithError()
			return
		}()

		if panicked {
			t.Fatal("WithCommand 不应提前触发 Build() panic")
		}
		if err != nil {
			t.Fatalf("期望构建成功，但得到错误: %v", err)
		}

		cli.SetArgs([]string{"ping"})
		if err := cli.Execute(); err != nil {
			t.Fatalf("执行命令失败: %v", err)
		}
		if !hookCalled.Load() {
			t.Fatal("BuildWithError 应附加 init hook")
		}

		found := false
		for _, cmd := range cli.Commands() {
			if cmd.Name() == "ping" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("WithCommand 添加的命令应在构建后可见")
		}
	})

	t.Run("WithServiceRunner nil 不应立即 panic", func(t *testing.T) {
		panicked := false
		var err error

		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()

			_, err = NewBuilder("zh").
				WithName("nil-runner").
				WithServiceRunner(nil).
				BuildWithError()
		}()

		if panicked {
			t.Fatal("WithServiceRunner(nil) 不应立即 panic")
		}
		if err == nil {
			t.Fatal("期望 BuildWithError 返回错误")
		}
	})

	t.Run("WithMousetrapDisabled 不应污染全局", func(t *testing.T) {
		prevPackage := MousetrapHelpText
		prevCobra := cobra.MousetrapHelpText
		defer func() {
			MousetrapHelpText = prevPackage
			cobra.MousetrapHelpText = prevCobra
		}()

		MousetrapHelpText = "GLOBAL"
		cobra.MousetrapHelpText = "GLOBAL"

		cli, err := NewBuilder("zh").
			WithName("mouse-app").
			WithMousetrapDisabled(true).
			WithCommand(&Command{Use: "ping", Run: func(cmd *Command, args []string) {}}).
			BuildWithError()
		if err != nil {
			t.Fatalf("构建失败: %v", err)
		}

		if MousetrapHelpText != "GLOBAL" {
			t.Fatalf("builder 不应修改包级 MousetrapHelpText，got %q", MousetrapHelpText)
		}

		cli.SetArgs([]string{"ping"})
		if err := cli.Execute(); err != nil {
			t.Fatalf("执行失败: %v", err)
		}

		if cobra.MousetrapHelpText != "GLOBAL" {
			t.Fatalf("执行后应恢复 cobra.MousetrapHelpText，got %q", cobra.MousetrapHelpText)
		}
	})

	t.Run("便利性API", func(t *testing.T) {
		// 测试QuickService
		cli := QuickService("quick-test", "快速测试服务", func(ctx context.Context) error {
			return nil
		})

		if cli.Name() != "quick-test" {
			t.Errorf("期望服务名称为 'quick-test'，实际为 '%s'", cli.Name())
		}

		// 测试QuickCLI
		cliTool := QuickCLI("cli-tool", "CLI工具", "这是一个CLI工具")

		if cliTool.Name() != "cli-tool" {
			t.Errorf("期望工具名称为 'cli-tool'，实际为 '%s'", cliTool.Name())
		}
	})
}

type noopErrorHandler struct{}

func (noopErrorHandler) HandleError(err error) error { return err }

func TestWithConfigPreservesErrorHandlers(t *testing.T) {
	src := NewConfig()
	src.runtime.ErrorHandlers = []ErrorHandler{noopErrorHandler{}}

	dst := NewConfig()
	WithConfig(src)(dst)

	if len(dst.runtime.ErrorHandlers) != 1 {
		t.Fatalf("expected 1 error handler after clone, got %d", len(dst.runtime.ErrorHandlers))
	}
}

// TestServiceInterface 测试服务接口
func TestServiceInterface(t *testing.T) {
	t.Run("SimpleService创建", func(t *testing.T) {
		var runCalled atomic.Bool
		var stopCalled atomic.Bool

		service := NewSimpleService("test-service",
			func(ctx context.Context) error {
				runCalled.Store(true)
				// 等待上下文被取消
				<-ctx.Done()
				return ctx.Err()
			},
			func() error {
				stopCalled.Store(true)
				return nil
			},
		)

		if service.Name() != "test-service" {
			t.Errorf("期望服务名称为 'test-service'，实际为 '%s'", service.Name())
		}

		// 测试运行
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 在 goroutine 中运行服务
		runDone := make(chan error, 1)
		go func() {
			runDone <- service.Run(ctx)
		}()

		// 等待服务开始运行
		time.Sleep(10 * time.Millisecond)

		if !runCalled.Load() {
			t.Error("期望调用运行函数")
		}

		// 测试停止
		err := service.Stop()
		if err != nil {
			t.Errorf("服务停止失败: %v", err)
		}

		// 等待服务完全停止
		select {
		case runErr := <-runDone:
			if runErr != nil && runErr != context.Canceled {
				t.Errorf("服务运行失败: %v", runErr)
			}
		case <-time.After(1 * time.Second):
			t.Error("服务停止超时")
		}

		if !stopCalled.Load() {
			t.Error("期望调用停止函数")
		}
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("ServiceError创建", func(t *testing.T) {
		err := NewError(ErrServiceStart).
			Service("test-service").
			Operation("start").
			Message("测试错误").
			Context("test", "value").
			Build()

		if err.Code != ErrServiceStart {
			t.Errorf("期望错误码为 %s，实际为 %s", ErrServiceStart, err.Code)
		}

		if err.Service != "test-service" {
			t.Errorf("期望服务名为 'test-service'，实际为 '%s'", err.Service)
		}

		if err.Context["test"] != "value" {
			t.Error("期望上下文包含test=value")
		}
	})

	t.Run("预定义错误函数", func(t *testing.T) {
		err := ErrServiceAlreadyRunning("test-service")

		if err.Code != ErrServiceRunning {
			t.Errorf("期望错误码为 %s，实际为 %s", ErrServiceRunning, err.Code)
		}

		if err.Service != "test-service" {
			t.Errorf("期望服务名为 'test-service'，实际为 '%s'", err.Service)
		}
	})

	t.Run("错误聚合器", func(t *testing.T) {
		aggregator := NewErrorAggregator()

		aggregator.Add(fmt.Errorf("错误1"))
		aggregator.Add(fmt.Errorf("错误2"))
		aggregator.Add(fmt.Errorf("错误3"))

		if !aggregator.HasErrors() {
			t.Error("期望有错误")
		}

		if aggregator.Count() != 3 {
			t.Errorf("期望3个错误，实际为%d", aggregator.Count())
		}

		errorMsg := aggregator.Error()
		if errorMsg == "" {
			t.Error("期望得到错误消息")
		}
	})
}
