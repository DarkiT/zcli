package zcli

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
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
			WithSimpleService("service-test", serviceFunc, stopFunc).
			Build()

		if cli.Name() != "service-test" {
			t.Errorf("期望服务名称为 'service-test'，实际为 '%s'", cli.Name())
		}

		// 这里我们无法直接测试服务运行，因为它需要完整的服务管理器
		// 但我们可以验证配置是否正确设置
		if cli.config.Runtime.Run == nil {
			t.Error("期望设置了运行函数")
		}

		if len(cli.config.Runtime.Stop) == 0 {
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
				if cfg.Basic.Name == "" {
					return fmt.Errorf("名称不能为空")
				}
				return nil
			}).
			BuildWithError()

		if err == nil {
			t.Error("期望构建失败，但没有得到错误")
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

// TestServiceInterface 测试服务接口
func TestServiceInterface(t *testing.T) {
	t.Run("SimpleService创建", func(t *testing.T) {
		runCalled := false
		stopCalled := false

		service := NewSimpleService("test-service",
			func(ctx context.Context) error {
				runCalled = true
				// 等待上下文被取消
				<-ctx.Done()
				return ctx.Err()
			},
			func() error {
				stopCalled = true
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

		if !runCalled {
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

		if !stopCalled {
			t.Error("期望调用停止函数")
		}
	})
}

// TestConcurrentServiceManager 测试并发服务管理器
func TestConcurrentServiceManager(t *testing.T) {
	t.Run("基本操作", func(t *testing.T) {
		service := NewSimpleService("concurrent-test",
			func(ctx context.Context) error {
				// 模拟启动工作
				time.Sleep(10 * time.Millisecond)
				// 然后立即返回，表示服务启动完成
				return nil
			},
			func() error {
				return nil
			},
		)

		config := ServiceConfig{
			Name:        "concurrent-test",
			DisplayName: "并发测试服务",
			EnvVars:     make(map[string]string),
		}

		manager := NewConcurrentServiceManager(service, config)

		// 设置较短的超时时间用于测试
		manager.SetStartTimeout(1 * time.Second)
		manager.SetStopTimeout(1 * time.Second)

		// 测试初始状态
		if !manager.IsStopped() {
			t.Error("期望初始状态为停止")
		}

		// 测试启动
		err := manager.Start()
		if err != nil {
			t.Errorf("启动服务失败: %v", err)
		}

		// 等待服务完成
		time.Sleep(100 * time.Millisecond)

		// 验证最终状态（服务应该已经自然结束）
		if !manager.IsStopped() {
			t.Error("期望最终状态为停止")
		}

		// 检查统计信息
		stats := manager.GetStats()
		if stats.StartCount != 1 {
			t.Errorf("期望启动次数为1，实际为%d", stats.StartCount)
		}

		// 注意：由于服务自然结束，停止次数应该是0
		if stats.StopCount != 0 {
			t.Errorf("期望停止次数为0，实际为%d", stats.StopCount)
		}
	})

	t.Run("并发安全", func(t *testing.T) {
		service := NewSimpleService("concurrent-safety-test",
			func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			func() error {
				return nil
			},
		)

		config := ServiceConfig{
			Name:        "concurrent-safety-test",
			DisplayName: "并发安全测试",
			EnvVars:     make(map[string]string),
		}

		manager := NewConcurrentServiceManager(service, config)

		const numGoroutines = 10
		var wg sync.WaitGroup

		// 并发启动
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = manager.Start()
			}()
		}

		// 并发停止
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(10 * time.Millisecond)
				_ = manager.Stop()
			}()
		}

		wg.Wait()

		// 验证最终状态一致
		if !manager.IsStopped() && manager.GetState() != StateError {
			t.Errorf("期望最终状态为停止或错误，实际为%s", manager.GetState())
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

// TestTemplates 测试模板功能
func TestTemplates(t *testing.T) {
	t.Run("FromTemplate", func(t *testing.T) {
		builder := FromTemplate("web-service")

		if builder == nil {
			t.Error("期望得到有效的Builder")
		}

		// 测试验证器是否添加
		_, err := builder.WithName("web-test").BuildWithError()
		if err == nil {
			t.Error("期望验证失败，因为没有设置运行函数")
		}
	})

	t.Run("QuickService创建", func(t *testing.T) {
		cli := QuickService("web-api", "Web API服务", func(ctx context.Context) error {
			return nil
		})

		if cli.Name() != "web-api" {
			t.Errorf("期望服务名称为 'web-api'，实际为 '%s'", cli.Name())
		}
	})
}
