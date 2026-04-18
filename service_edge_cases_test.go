package zcli

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// 1) stopFuncOnce 防止重复执行
func TestStopFuncOnce(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "stop-once"
	var count int32
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
	config.runtime.Stop = func() error { atomic.AddInt32(&count, 1); return nil }

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	// 避免调用 daemon.Service 带来的后台 goroutine 竞态，直接跑 sm.Run
	go sm.Run(sm.getCtx())
	time.Sleep(10 * time.Millisecond) // 让 Run 进入阻塞态
	_ = sm.Stop()
	_ = sm.Stop()
	_ = sm.Stop()

	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("stop called %d times, want 1", count)
	}
}

// 2) Interactive 分支直接调用 callStopFunctions
func TestInteractiveCallsStopFunctions(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "interactive"
	var called int32
	config.runtime.Run = func(ctx context.Context) error { return nil }
	config.runtime.Stop = func() error { atomic.AddInt32(&called, 1); return nil }

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	// 与 TestStopFuncOnce 一致，避免 daemon 包产生的竞态，直接走 sm.Run
	go sm.Run(sm.getCtx())
	time.Sleep(10 * time.Millisecond)
	sm.stopExecuted.Store(false)
	sm.callStopFunctions()

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("stop not called via interactive path")
	}
}

// 3) checkPermissions 权限检查
func TestCheckPermissions(t *testing.T) {
	loc := NewServiceLocalizer(GetLanguageManager(), newColors())
	tdir := t.TempDir()

	file := filepath.Join(tdir, "f")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := checkPermissions(file, 0o755, loc); err == nil {
		t.Fatalf("expected exec perm error")
	}
	if err := checkPermissions(file, os.ModeDir|0o700, loc); err == nil {
		t.Fatalf("expected needDir error")
	}

	dir := filepath.Join(tdir, "d")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := checkPermissions(dir, 0o755, loc); err == nil {
		t.Fatalf("expected needFile error")
	}
	if err := checkPermissions(dir, os.ModeDir|0o700, loc); err == nil {
		t.Fatalf("expected write perm error")
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := checkPermissions(dir, os.ModeDir|0o755, loc); err != nil {
		t.Fatalf("unexpected perm error: %v", err)
	}
}
