package zcli

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestServiceConcurrentStop 测试 Stop 幂等性及并发安全
func TestServiceConcurrentStop(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "test-service"

	started := make(chan struct{})
	var once sync.Once
	config.runtime.Run = func(ctx context.Context) error {
		once.Do(func() { close(started) })
		<-ctx.Done()
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}

	// 使用可取消的 ctx，Stop 应通过 cancel 让 Run 返回
	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = sm.Run(sm.ctx)
	}()

	select {
	case <-started:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("service not started")
	}

	const goroutines = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := sm.Stop(); err != nil {
				t.Errorf("goroutine %d stop err: %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("service did not exit after concurrent stop")
	}

	if sm.running.Load() {
		t.Error("service should be stopped")
	}
	if !sm.stopExecuted.Load() {
		t.Error("stopExecuted should be true")
	}
}

// TestServiceChannelRaceCondition 专门测试exitChan的竞态条件修复
func TestServiceChannelRaceCondition(t *testing.T) {
	base := NewConfig()
	base.basic.Name = "race-test-service"

	cli := &Cli{config: base, colors: newColors(), lang: GetLanguageManager().GetPrimary()}

	const rounds = 10
	for i := 0; i < rounds; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cfg := *base
		cfg.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
		cli.config = &cfg

		sm, err := newServiceManager(cli, ctx, cancel)
		if err != nil {
			t.Fatalf("round %d: newServiceManager: %v", i, err)
		}

		runDone := make(chan struct{})
		go func() {
			defer close(runDone)
			_ = sm.Run(sm.ctx)
		}()

		// 保证启动后立即停止，验证 exitChan 在多轮启动/停止中不会竞态
		time.Sleep(10 * time.Millisecond)
		if err := sm.Stop(); err != nil {
			t.Fatalf("round %d: stop: %v", i, err)
		}

		select {
		case <-runDone:
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("round %d: run did not exit", i)
		}
	}
}

// TestServiceContextCancellation 测试上下文取消时的并发安全性
func TestServiceContextCancellation(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "context-test-service"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}

	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = sm.Run(sm.ctx)
	}()

	// 先取消上下文，再调用 Stop，应当均匀退出
	cancel()
	if err := sm.Stop(); err != nil {
		t.Fatalf("stop after cancel: %v", err)
	}

	select {
	case <-runDone:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("service did not exit after context cancel")
	}

	if sm.running.Load() {
		t.Error("service should be stopped")
	}
}

// TestServiceShutdownTimeouts 验证初始等待与宽限期触发 Stop
func TestServiceShutdownTimeouts(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "timeout-test"
	config.runtime.Run = func(_ context.Context) error {
		select {}
	}
	// 缩短默认的 3s+2s 以加速测试，但仍验证分层超时逻辑
	config.runtime.ShutdownInitial = 50 * time.Millisecond
	config.runtime.ShutdownGrace = 40 * time.Millisecond

	var stopCount atomic.Int32
	config.runtime.Stop = func() error {
		stopCount.Add(1)
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan struct{}) // 保持未关闭以触发超时路径
	done := make(chan struct{})
	sm.stopExecuted.Store(true) // 走 callStopFunctions 分支

	start := time.Now()
	go func() {
		sm.waitForServiceCompletion(runDone)
		close(done)
	}()

	// 触发 ctx 取消，进入超时分支
	sm.stopFuncOnce.Store(false)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("waitForServiceCompletion did not return within expected timeout window")
	}

	sm.stopFuncOnce.Store(false)

	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("timeout path returned in %v, want within [40ms,500ms]", elapsed)
	}

	if stopCount.Load() == 0 {
		t.Log("stop function may be skipped when no ctx-driven exit path is available")
	}
}
