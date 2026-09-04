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
		// 兜底收敛，避免测试挂死；能走到这里说明有 Run 没被 Stop 停掉
		baseCancel()
		<-done
		t.Errorf("%d 组 Run/Stop 并发中，存在未被 Stop 停止的 Run", pairs)
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
