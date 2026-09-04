package zcli

import (
	"context"
	"sync"
	"testing"
	"time"
)

// 本文件锁定 sManager 的并发契约：既防 data race（-race 下运行），
// 也防「Stop 停止意图被运行启动吞掉」这一静默失效（任何模式都跑）。

// TestStopIntentSurvivesRunStart 验证先于 Run 落地的 Stop 会被后续启动的 Run 尊重。
//
// 失效形态：Stop 取消的是旧 session，Run 重建 session 后继续跑，且调用方拿到的
// Stop 返回值是 nil——服务停不下来但无人知晓。
func TestStopIntentSurvivesRunStart(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "race-stop-intent"
	runCalled := make(chan struct{})
	config.runtime.Run = func(ctx context.Context) error {
		select {
		case <-runCalled:
		default:
			close(runCalled)
		}
		<-ctx.Done()
		return nil
	}

	// session 重建时用的是 config.Context()，兜底取消必须挂在它上面
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	config.ctx = baseCtx

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	sm, err := newServiceManager(cli, baseCtx, baseCancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	// 停止意图先于运行落地
	if err := sm.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = sm.Run(nil)
	}()

	select {
	case <-runDone:
	case <-time.After(testDuration(2 * time.Second)):
		baseCancel()
		<-runDone
		t.Fatal("停止意图被 Run 启动吞掉：Run 在 Stop 之后仍未退出")
	}

	select {
	case <-runCalled:
		t.Error("已请求停止的运行不应再执行用户 RunFunc")
	default:
	}
}

// TestConcurrentRunStopNoDataRace 在同一 sManager 上并发 Run/Stop。
// session / exitChan 的读写若分属不同锁，race detector 会在此暴露。
func TestConcurrentRunStopNoDataRace(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "race-run-stop"
	config.runtime.Run = func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}

	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	config.ctx = baseCtx

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	sm, err := newServiceManager(cli, baseCtx, baseCancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	const pairs = 20
	var wg sync.WaitGroup
	wg.Add(pairs)
	for range pairs {
		go func() {
			defer wg.Done()
			_ = sm.Run(nil)
		}()
	}
	wg.Add(pairs)
	for range pairs {
		go func() {
			defer wg.Done()
			_ = sm.Stop()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testDuration(3 * time.Second)):
		// 兜底收敛，避免测试挂死。此处刻意不断言失败：
		// stopExecuted 是单次的全局标志，一次 Stop 只能兑付一个运行中的服务，
		// 20 个 Run 并发时剩余的 Run 需要各自的 Stop 才会退出——这是既有语义，
		// 本用例的职责是暴露 data race。活性契约由
		// TestStopConcurrentWithRunTerminates 保障。
		baseCancel()
		<-done
	}
}

// TestStopConcurrentWithRunTerminates 验证 Stop 与 Run 同时起步时，单次 Stop
// 必定让 Run 退出——不允许出现「Stop 返回 nil 但服务还在跑」的静默失效。
//
// 失效形态（修复前实测 6/30）：Stop 取消的是旧 session，Run 随后重建 session
// 继续运行，且 Stop 返回值是 nil，调用方无从察觉。
func TestStopConcurrentWithRunTerminates(t *testing.T) {
	const trials = 30
	stuck := 0

	for i := 0; i < trials; i++ {
		config := NewConfig()
		config.basic.Name = "race-stop-terminate"
		config.runtime.Run = func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}

		baseCtx, baseCancel := context.WithCancel(context.Background())
		config.ctx = baseCtx

		cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
		sm, err := newServiceManager(cli, baseCtx, baseCancel)
		if err != nil {
			baseCancel()
			t.Fatalf("trial %d: newServiceManager: %v", i, err)
		}

		gate := make(chan struct{})
		runDone := make(chan struct{})
		go func() {
			defer close(runDone)
			<-gate
			_ = sm.Run(nil)
		}()
		go func() {
			<-gate
			_ = sm.Stop()
		}()
		close(gate)

		select {
		case <-runDone:
		case <-time.After(testDuration(500 * time.Millisecond)):
			stuck++
			baseCancel()
			<-runDone
		}
		baseCancel()
	}

	if stuck > 0 {
		t.Errorf("%d/%d 轮出现 Stop 静默失效：Run 在 Stop 之后仍在运行", stuck, trials)
	}
}

// TestConcurrentErrorHandlerRegistration 验证 AddErrorHandler 与 handleError 可并发。
// errorHandlers 是裸切片，append 与 range 并发即 data race。
func TestConcurrentErrorHandlerRegistration(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "race-error-handler"
	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	sm, err := newServiceManager(cli, baseCtx, baseCancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range iterations {
			sm.AddErrorHandler(noopErrorHandler{})
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range iterations {
			if err := sm.handleError(context.Canceled); err == nil {
				t.Error("handleError 不应吞掉错误")
				return
			}
		}
	}()
	wg.Wait()
}
