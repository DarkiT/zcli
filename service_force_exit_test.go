package zcli

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	service "github.com/darkit/daemon"
)

type stubSystemInteractive struct{}

func (stubSystemInteractive) String() string                 { return "stub-interactive" }
func (stubSystemInteractive) Detect() bool                   { return true }
func (stubSystemInteractive) Interactive() bool              { return true }
func (stubSystemInteractive) New(service.Interface, *service.Config) (service.Service, error) {
	return nil, nil
}

type stubSystemDaemon struct{}

func (stubSystemDaemon) String() string                 { return "stub-daemon" }
func (stubSystemDaemon) Detect() bool                   { return true }
func (stubSystemDaemon) Interactive() bool              { return false }
func (stubSystemDaemon) New(service.Interface, *service.Config) (service.Service, error) {
	return nil, nil
}

func TestForceExit_DaemonOnly(t *testing.T) {
	t.Run("interactive_skips_force_exit", func(t *testing.T) {
		origSystems := service.AvailableSystems()
		origChosen := service.ChosenSystem()
		service.ChooseSystem(stubSystemInteractive{})
		if origChosen != nil {
			t.Cleanup(func() { service.ChooseSystem(origChosen) })
		} else {
			t.Cleanup(func() { service.ChooseSystem(origSystems...) })
		}

		config := NewConfig()
		config.basic.Name = "force-exit-interactive"
		config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
		config.runtime.Stop = func() error { return nil }
		config.runtime.StopTimeout = 30 * time.Millisecond

		cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
		ctx, cancel := context.WithCancel(context.Background())
		sm, err := newServiceManager(cli, ctx, cancel)
		if err != nil {
			t.Fatalf("newServiceManager: %v", err)
		}

		origExit := exitFunc
		defer func() { exitFunc = origExit }()
		var exitCalled atomic.Bool
		exitFunc = func(int) { exitCalled.Store(true) }

		_ = sm.Stop()
		time.Sleep(2 * config.runtime.StopTimeout)
		if exitCalled.Load() {
			t.Fatal("force exit should be skipped in interactive mode")
		}
	})

	t.Run("daemon_schedules_force_exit", func(t *testing.T) {
		origSystems := service.AvailableSystems()
		origChosen := service.ChosenSystem()
		service.ChooseSystem(stubSystemDaemon{})
		if origChosen != nil {
			t.Cleanup(func() { service.ChooseSystem(origChosen) })
		} else {
			t.Cleanup(func() { service.ChooseSystem(origSystems...) })
		}

		config := NewConfig()
		config.basic.Name = "force-exit-daemon"
		config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
		config.runtime.Stop = func() error { return nil }
		config.runtime.StopTimeout = 30 * time.Millisecond

		cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
		ctx, cancel := context.WithCancel(context.Background())
		sm, err := newServiceManager(cli, ctx, cancel)
		if err != nil {
			t.Fatalf("newServiceManager: %v", err)
		}

		origExit := exitFunc
		defer func() { exitFunc = origExit }()
		exitCh := make(chan int, 1)
		exitFunc = func(code int) { exitCh <- code }

		_ = sm.Stop()
		select {
		case code := <-exitCh:
			if code != 1 {
				t.Fatalf("unexpected exit code: %d", code)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("expected force exit to trigger")
		}
	})
}

func TestForceExit_CancelledBeforeTimeout(t *testing.T) {
	origSystems := service.AvailableSystems()
	origChosen := service.ChosenSystem()
	service.ChooseSystem(stubSystemDaemon{})
	if origChosen != nil {
		t.Cleanup(func() { service.ChooseSystem(origChosen) })
	} else {
		t.Cleanup(func() { service.ChooseSystem(origSystems...) })
	}

	config := NewConfig()
	config.basic.Name = "force-exit-cancel"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }
	config.runtime.Stop = func() error { return nil }
	config.runtime.StopTimeout = 80 * time.Millisecond

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// newServiceManager will use ctx for run
	sm, err := newServiceManager(cli, ctx, cancel)
	if err != nil {
		t.Fatalf("newServiceManager: %v", err)
	}

	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	var exitCalled atomic.Bool
	exitFunc = func(int) { exitCalled.Store(true) }

	go func() {
		err := sm.Run(sm.getCtx())
		if err != nil && !errors.Is(err, context.Canceled) {
			return
		}
	}()

	_ = sm.Stop()
	// allow Run to finish and cancel force exit
	time.Sleep(2 * config.runtime.StopTimeout)
	if exitCalled.Load() {
		t.Fatal("force exit should be cancelled when run finishes")
	}
}
