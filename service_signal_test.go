package zcli

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestSignalHandling_SIGINT 验证 SIGINT 可以触发 RunWait 退出
func TestSignalHandling_SIGINT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 单独覆盖")
	}
	testSignalHandling(t, syscall.SIGINT)
}

// TestSignalHandling_SIGTERM 验证 SIGTERM 可以触发 RunWait 退出
func TestSignalHandling_SIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 单独覆盖")
	}
	testSignalHandling(t, syscall.SIGTERM)
}

// TestSignalHandling_Timeout 验证分级关闭预算逻辑（缩短版）
func TestSignalHandling_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 路径单独覆盖")
	}

	config := NewConfig()
	config.basic.Name = "signal-timeout"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
	// 缩短到 100ms + 80ms，验证分层等待逻辑
	config.runtime.ShutdownInitial = 100 * time.Millisecond
	config.runtime.ShutdownGrace = 80 * time.Millisecond

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runDone := make(chan struct{})
	started := make(chan struct{})
	// 用 runtime.Run 作为同步点，确保 Run 完成初始化后再 wait
	sm.commands.config.runtime.Run = func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return nil
	}

	go func() {
		defer close(runDone)
		_ = sm.Run(sm.getCtx())
	}()

	select {
	case <-started:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("service not started")
	}

	waitDone := make(chan struct{})
	go func() {
		sm.waitForServiceCompletion(sm.getCtx(), runDone)
		close(waitDone)
	}()

	// 触发 ctx.Done()
	cancel()

	select {
	case <-waitDone:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout handling did not finish")
	}

	sm.stopExecuted.Store(true)

	if sm.running.Load() {
		_ = sm.Stop()
	}
}

// TestContextMerge_ExternalCancel 验证外部上下文取消可触发运行退出
func TestContextMerge_ExternalCancel(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "ctx-merge-external"

	userCtx, userCancel := context.WithCancel(context.Background())
	config.ctx = userCtx

	runDone := make(chan struct{})
	config.runtime.Run = func(ctx context.Context) error {
		close(runDone)
		<-ctx.Done()
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()
	defer userCancel()

	sm, err := newServiceManager(cli, baseCtx, baseCancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	exitDone := make(chan struct{})
	go func() {
		_ = sm.Run(baseCtx)
		close(exitDone)
	}()

	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("service not started")
	}

	userCancel()

	select {
	case <-exitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("service did not exit on external context cancel")
	}
}

// TestContextMerge_DaemonCancel 验证 daemon 传入上下文取消可触发运行退出
func TestContextMerge_DaemonCancel(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "ctx-merge-daemon"

	userCtx, userCancel := context.WithCancel(context.Background())
	config.ctx = userCtx

	runDone := make(chan struct{})
	config.runtime.Run = func(ctx context.Context) error {
		close(runDone)
		<-ctx.Done()
		return nil
	}

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	baseCtx, baseCancel := context.WithCancel(context.Background())
	defer userCancel()

	sm, err := newServiceManager(cli, baseCtx, baseCancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	exitDone := make(chan struct{})
	go func() {
		_ = sm.Run(baseCtx)
		close(exitDone)
	}()

	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("service not started")
	}

	baseCancel()

	select {
	case <-exitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("service did not exit on daemon context cancel")
	}
}

// TestSignalHandling_Windows 确认 Windows 排除 SIGQUIT，仅 INT/TERM
func TestSignalHandling_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 路径")
	}

	config := NewConfig()
	config.basic.Name = "signal-win"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
	config.runtime.ShutdownInitial = 200 * time.Millisecond
	config.runtime.ShutdownGrace = 200 * time.Millisecond

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runWait, ok := sm.config.Option["RunWait"].(func())
	if !ok {
		t.Fatal("RunWait option missing")
	}

	waitDone := make(chan struct{})
	go func() {
		runWait()
		close(waitDone)
	}()

	signal.Ignore(syscall.SIGINT, syscall.SIGTERM)
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	proc, _ := os.FindProcess(os.Getpid())
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		if err := proc.Signal(sig); err != nil {
			t.Fatalf("send %v: %v", sig, err)
		}
		select {
		case <-waitDone:
			return
		case <-time.After(1 * time.Second):
		}
	}

	t.Fatalf("service did not exit on Windows signals")
}

// 公共的信号触发路径（非 Windows）
func testSignalHandling(t *testing.T, sig os.Signal) {
	config := NewConfig()
	config.basic.Name = "signal-test"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
	config.runtime.ShutdownInitial = 300 * time.Millisecond
	config.runtime.ShutdownGrace = 200 * time.Millisecond

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	runWait, ok := sm.config.Option["RunWait"].(func())
	if !ok {
		t.Fatal("RunWait option missing")
	}

	waitDone := make(chan struct{})
	go func() {
		runWait()
		close(waitDone)
	}()

	proc, _ := os.FindProcess(os.Getpid())
	time.Sleep(10 * time.Millisecond) // 确保 RunWait 完成信号注册
	if err := proc.Signal(sig); err != nil {
		t.Fatalf("send %v: %v", sig, err)
	}

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("service did not exit on %v", sig)
	}
}
