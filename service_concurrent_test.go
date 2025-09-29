package zcli

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestServiceConcurrentStartStop 测试服务的并发启动和停止操作是否安全
func TestServiceConcurrentStartStop(t *testing.T) {
	// 创建测试用的CLI配置
	config := NewConfig()
	config.Basic.Name = "test-service"
	config.Basic.DisplayName = "Test Service"
	config.Runtime.Run = func(ctxs ...context.Context) {
		// 模拟服务运行，等待一段时间
		time.Sleep(100 * time.Millisecond)
	}

	// 创建CLI实例
	cli := &Cli{
		config: config,
		colors: newColors(),
		lang:   GetLanguageManager().GetPrimary(),
	}

	// 创建服务管理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("Failed to create service manager: %v", err)
	}

	// 并发测试：多个goroutine同时调用Start和Stop
	const numGoroutines = 10
	var wg sync.WaitGroup

	// 启动多个goroutine并发调用Start
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := sm.Start(sm.service)
			expectedError := sm.localizer.GetError("alreadyRunning")
			if err != nil && err.Error() != expectedError {
				t.Errorf("Goroutine %d: Unexpected start error: %v", id, err)
			}
		}(i)
	}

	// 同时启动多个goroutine并发调用Stop
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 稍微延迟，让Start有机会执行
			time.Sleep(10 * time.Millisecond)
			err := sm.Stop(sm.service)
			if err != nil {
				t.Errorf("Goroutine %d: Unexpected stop error: %v", id, err)
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 验证最终状态
	if sm.running.Load() {
		t.Error("Service should not be running after all stop operations")
	}

	if !sm.stopExecuted.Load() {
		t.Error("Stop should have been executed")
	}
}

// TestServiceChannelRaceCondition 专门测试exitChan的竞态条件修复
func TestServiceChannelRaceCondition(t *testing.T) {
	config := NewConfig()
	config.Basic.Name = "race-test-service"
	config.Runtime.Run = func(ctxs ...context.Context) {
		time.Sleep(50 * time.Millisecond)
	}

	cli := &Cli{
		config: config,
		colors: newColors(),
		lang:   GetLanguageManager().GetPrimary(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("Failed to create service manager: %v", err)
	}

	// 进行多轮快速的启动-停止循环，测试channel的竞态条件
	const rounds = 50
	for i := 0; i < rounds; i++ {
		// 启动服务
		err := sm.Start(sm.service)
		expectedError := sm.localizer.GetError("alreadyRunning")
		if err != nil && err.Error() != expectedError {
			t.Fatalf("Round %d: Failed to start service: %v", i, err)
		}

		// 立即停止服务
		err = sm.Stop(sm.service)
		if err != nil {
			t.Fatalf("Round %d: Failed to stop service: %v", i, err)
		}

		// 重置stopExecuted标志以便下一轮测试
		sm.stopExecuted.Store(false)
		sm.running.Store(false)
	}
}

// TestServiceContextCancellation 测试上下文取消时的并发安全性
func TestServiceContextCancellation(t *testing.T) {
	config := NewConfig()
	config.Basic.Name = "context-test-service"
	config.Runtime.Run = func(ctxs ...context.Context) {
		// 模拟长时间运行的服务
		time.Sleep(1 * time.Second)
	}

	cli := &Cli{
		config: config,
		colors: newColors(),
		lang:   GetLanguageManager().GetPrimary(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("Failed to create service manager: %v", err)
	}

	// 启动服务
	err = sm.Start(sm.service)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// 同时从多个goroutine取消上下文和调用Stop
	var wg sync.WaitGroup

	// 取消上下文
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// 同时调用Stop
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		_ = sm.Stop(sm.service)
	}()

	wg.Wait()

	// 等待足够时间确保所有操作完成
	time.Sleep(100 * time.Millisecond)

	// 验证最终状态
	if sm.running.Load() {
		t.Error("Service should not be running after context cancellation")
	}
}
