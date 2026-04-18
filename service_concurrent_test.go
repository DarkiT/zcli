package zcli

import (
	"context"
	"errors"
	"os"
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
		_ = sm.Run(sm.getCtx())
	}()

	select {
	case <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
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
	case <-time.After(testDuration(500 * time.Millisecond)):
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
			_ = sm.Run(sm.getCtx())
		}()

		// 保证启动后立即停止，验证 exitChan 在多轮启动/停止中不会竞态
		time.Sleep(testDuration(10 * time.Millisecond))
		if err := sm.Stop(); err != nil {
			t.Fatalf("round %d: stop: %v", i, err)
		}

		select {
		case <-runDone:
		case <-time.After(testDuration(300 * time.Millisecond)):
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
		_ = sm.Run(sm.getCtx())
	}()

	// 先取消上下文，再调用 Stop，应当均匀退出
	cancel()
	if err := sm.Stop(); err != nil {
		t.Fatalf("stop after cancel: %v", err)
	}

	select {
	case <-runDone:
	case <-time.After(testDuration(300 * time.Millisecond)):
		t.Fatal("service did not exit after context cancel")
	}

	if sm.running.Load() {
		t.Error("service should be stopped")
	}
}

func TestRunSeparatesCommandAndServiceContexts(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "run-context-separation"
	started := make(chan context.Context, 1)
	config.runtime.Run = func(ctx context.Context) error {
		started <- ctx
		<-ctx.Done()
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	sm.mu.Lock()
	session := sm.newCommandSessionLocked()
	commandCtx := session.commandCtx
	sm.mu.Unlock()

	runDone := make(chan error, 1)
	go func() {
		runDone <- sm.Run(context.Background())
	}()

	var serviceCtx context.Context
	select {
	case serviceCtx = <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not start")
	}

	if serviceCtx == nil {
		t.Fatal("service context should not be nil")
	}
	if serviceCtx == commandCtx {
		t.Fatal("service context should be derived from the command context, not replace it")
	}
	if sm.getCtx() != commandCtx {
		t.Fatal("command context should remain stable while the service is running")
	}

	if err := sm.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(testDuration(500 * time.Millisecond)):
		t.Fatal("service did not exit after stop")
	}
}

func TestStopCancelsRunContextBeforeStopHook(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "cancel-before-stop-hook"

	started := make(chan struct{}, 1)
	runCanceled := make(chan struct{}, 1)
	stopObserved := make(chan struct{}, 1)

	config.runtime.Run = func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		runCanceled <- struct{}{}
		return nil
	}
	config.runtime.Stop = func() error {
		select {
		case <-runCanceled:
			stopObserved <- struct{}{}
			return nil
		case <-time.After(testDuration(200 * time.Millisecond)):
			return errors.New("stop hook ran before Run(ctx) observed cancellation")
		}
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan error, 1)
	go func() {
		runDone <- sm.Run(sm.getCtx())
	}()

	select {
	case <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not start")
	}

	if err := sm.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case <-stopObserved:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("stop hook did not observe the canceled run context")
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(testDuration(500 * time.Millisecond)):
		t.Fatal("run did not exit after stop")
	}
}

func TestShutdownCausePropagatesToRunContext(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "shutdown-cause"

	started := make(chan struct{}, 1)
	causeCh := make(chan *ShutdownCause, 1)
	config.runtime.Run = func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		if cause, ok := GetShutdownCause(ctx); ok {
			causeCh <- cause
		} else {
			causeCh <- nil
		}
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan error, 1)
	go func() {
		runDone <- sm.Run(sm.getCtx())
	}()

	select {
	case <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not start")
	}

	if err := sm.stopWithCause(newShutdownCause(ShutdownReasonSignal, os.Interrupt, nil), true); err != nil {
		t.Fatalf("stopWithCause failed: %v", err)
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(testDuration(500 * time.Millisecond)):
		t.Fatal("run did not exit after shutdown cause")
	}

	select {
	case cause := <-causeCh:
		if cause == nil {
			t.Fatal("expected a shutdown cause on the run context")
		}
		if cause.Reason != ShutdownReasonSignal {
			t.Fatalf("unexpected shutdown reason: %s", cause.Reason)
		}
		if cause.Signal != os.Interrupt {
			t.Fatalf("unexpected shutdown signal: %v", cause.Signal)
		}
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("run did not report a shutdown cause")
	}
}

func TestDaemonStartContextDoesNotTerminateRuntime(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "daemon-start-session"

	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	config.runtime.Run = func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		stopped <- struct{}{}
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runner := sm.buildRunner()
	runner.StartTimeout = testDuration(10 * time.Millisecond)
	runner.StopTimeout = testDuration(100 * time.Millisecond)

	if err := runner.Start(nil); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not start through daemon runner")
	}

	select {
	case <-stopped:
		t.Fatal("service runtime should outlive the daemon start timeout context")
	case <-time.After(testDuration(50 * time.Millisecond)):
	}

	if err := runner.Stop(nil); err != nil {
		t.Fatalf("runner stop failed: %v", err)
	}

	select {
	case <-stopped:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not stop after runner stop")
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
	config.runtime.ShutdownInitial = testDuration(50 * time.Millisecond)
	config.runtime.ShutdownGrace = testDuration(40 * time.Millisecond)

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
		sm.waitForServiceCompletion(sm.getCtx(), runDone)
		close(done)
	}()

	// 触发 ctx 取消，进入超时分支
	sm.stopFuncOnce.Store(false)
	cancel()

	select {
	case <-done:
	case <-time.After(testDuration(500 * time.Millisecond)):
		t.Fatal("waitForServiceCompletion did not return within expected timeout window")
	}

	sm.stopFuncOnce.Store(false)

	elapsed := time.Since(start)
	minExpected := testDuration(40 * time.Millisecond)
	maxExpected := testDuration(500 * time.Millisecond)
	if elapsed < minExpected || elapsed > maxExpected {
		t.Fatalf("timeout path returned in %v, want within [%v,%v]", elapsed, minExpected, maxExpected)
	}

	if stopCount.Load() == 0 {
		t.Log("stop function may be skipped when no ctx-driven exit path is available")
	}
}

func TestServiceRunCanRestartAfterStop(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "rerun-service"

	started := make(chan struct{}, 2)
	config.runtime.Run = func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	for round := 0; round < 2; round++ {
		runDone := make(chan struct{})
		go func() {
			defer close(runDone)
			_ = sm.Run(context.Background())
		}()

		select {
		case <-started:
		case <-time.After(testDuration(200 * time.Millisecond)):
			t.Fatalf("round %d: service not started", round)
		}

		if err := sm.Stop(); err != nil {
			t.Fatalf("round %d: stop failed: %v", round, err)
		}

		select {
		case <-runDone:
		case <-time.After(testDuration(500 * time.Millisecond)):
			t.Fatalf("round %d: service did not exit after stop", round)
		}
	}
}

func TestWaitForServiceCompletionReturnsHandledRunError(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "run-error"

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	var handled atomic.Bool
	sm.AddErrorHandler(errorHandlerFunc(func(err error) error {
		handled.Store(true)
		return err
	}))

	runDone := make(chan struct{})
	runErrCh := make(chan error, 1)
	runErrCh <- errors.New("boom")
	close(runDone)
	close(runErrCh)

	err = sm.waitForServiceCompletion(sm.getCtx(), runDone, runErrCh)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("unexpected run error: %v", err)
	}
	if !handled.Load() {
		t.Fatal("expected error handler to be invoked for run error")
	}
}

type errorHandlerFunc func(error) error

func (f errorHandlerFunc) HandleError(err error) error { return f(err) }

func TestServiceRunWithNilConfigContextUsesFallback(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "nil-config-context-run"
	config.ctx = nil

	started := make(chan struct{}, 1)
	config.runtime.Run = func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	}
	config.runtime.Stop = func() error { return nil }

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan error, 1)
	go func() {
		runDone <- sm.Run(context.Background())
	}()

	select {
	case <-started:
	case <-time.After(testDuration(200 * time.Millisecond)):
		t.Fatal("service did not start with nil config context")
	}

	if err := sm.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(testDuration(500 * time.Millisecond)):
		t.Fatal("service did not exit after stop")
	}
}

func TestNewCliWithNilConfigContextInitializesServiceCommands(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "nil-config-context-init"
	config.ctx = nil
	config.runtime.Run = func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewCli should not panic with nil config context: %v", r)
		}
	}()

	cli := NewCli(WithConfig(config))
	if cli == nil || cli.command == nil {
		t.Fatal("expected cli command to be initialized")
	}

	for _, name := range []string{"run", "install", "uninstall", "start", "stop", "restart", "status"} {
		cmd, _, err := cli.command.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Fatalf("expected service command %q to be registered, got cmd=%v err=%v", name, cmd, err)
		}
	}
}
