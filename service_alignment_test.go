package zcli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	service "github.com/darkit/daemon"
)

func TestCreateServiceConfig_PropagatesDaemonFields(t *testing.T) {
	prevGOOS := serviceManagerGOOS
	prevExecutable := executablePath
	serviceManagerGOOS = "windows"
	executablePath = func() (string, error) { return "/opt/demo/bin/demo", nil }
	t.Cleanup(func() {
		serviceManagerGOOS = prevGOOS
		executablePath = prevExecutable
	})

	config := NewConfig()
	config.basic.Name = "demo-service"
	config.basic.DisplayName = "Demo Service"
	config.basic.Description = "service description"
	config.service.Username = "svc-user"
	config.service.Arguments = []string{"serve", "--config", "app.yaml"}
	config.service.StructuredDeps = []Dependency{
		{Name: "postgresql", Type: DependencyRequire},
	}
	config.service.EnvVars = map[string]string{"ENV": "prod"}
	config.service.Options = ServiceOptions{
		service.OptionRestart: "on-failure",
	}
	config.service.AllowSudoFallback = true

	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	sm := &sManager{
		commands:  cli,
		localizer: NewServiceLocalizer(GetLanguageManager(), cli.colors),
	}

	svcConfig, err := sm.createServiceConfig()
	if err != nil {
		t.Fatalf("createServiceConfig: %v", err)
	}

	if svcConfig.UserName != "svc-user" {
		t.Fatalf("expected service user to be propagated, got %q", svcConfig.UserName)
	}
	if len(svcConfig.StructuredDeps) != 1 || svcConfig.StructuredDeps[0].Name != "postgresql" {
		t.Fatalf("expected structured dependencies to be propagated, got %#v", svcConfig.StructuredDeps)
	}
	if svcConfig.EnvVars["ENV"] != "prod" {
		t.Fatalf("expected env vars to be propagated, got %#v", svcConfig.EnvVars)
	}
	if svcConfig.Option[service.OptionRestart] != "on-failure" {
		t.Fatalf("expected service options to be propagated, got %#v", svcConfig.Option)
	}
	if !svcConfig.AllowSudoFallback {
		t.Fatal("expected AllowSudoFallback to be propagated")
	}
	if got, want := svcConfig.Executable, "/opt/demo/bin/demo"; got != want {
		t.Fatalf("expected executable %q, got %q", want, got)
	}
	if got, want := svcConfig.WorkingDirectory, filepath.Dir("/opt/demo/bin/demo"); got != want {
		t.Fatalf("expected working directory %q, got %q", want, got)
	}
}

func TestBuildRunner_StopInvokesUserStop(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "runner-stop"

	started := make(chan struct{})
	config.runtime.Run = func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return nil
	}

	var stopCount atomic.Int32
	config.runtime.Stop = func() error {
		stopCount.Add(1)
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
	if err := runner.Start(nil); err != nil {
		t.Fatalf("runner.Start: %v", err)
	}

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("service did not start in time")
	}

	if err := runner.Stop(nil); err != nil {
		t.Fatalf("runner.Stop: %v", err)
	}
	if stopCount.Load() != 1 {
		t.Fatalf("expected user stop to be called once, got %d", stopCount.Load())
	}
}

func TestBaseService_CanRestartAfterStop(t *testing.T) {
	svc, err := NewBaseService(ServiceConfig{Name: "base-service"})
	if err != nil {
		t.Fatalf("NewBaseService: %v", err)
	}

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- svc.Run(context.Background())
	}()

	waitForRunning(t, svc)
	if err := svc.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first run returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("first run did not stop in time")
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- svc.Run(context.Background())
	}()

	waitForRunning(t, svc)
	select {
	case err := <-secondDone:
		t.Fatalf("second run returned early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second run returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second run did not stop in time")
	}
}

func TestManagedService_AfterStartFailureStopsRunner(t *testing.T) {
	runDone := make(chan struct{})
	runner := &runnerWrapper{runDone: runDone}

	managed := NewManagedService(runner, lifecycleWithAfterStartError{
		err: errors.New("boom"),
	})

	err := managed.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "after start hook failed") {
		t.Fatalf("expected after-start error, got %v", err)
	}

	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("managed service run did not exit after AfterStart failure")
	}
	if !runner.stopCalled.Load() {
		t.Fatal("expected managed service to call Stop after AfterStart failure")
	}
}

type lifecycleWithAfterStartError struct {
	err error
}

func (lifecycleWithAfterStartError) BeforeStart() error  { return nil }
func (l lifecycleWithAfterStartError) AfterStart() error { return l.err }
func (lifecycleWithAfterStartError) BeforeStop() error   { return nil }
func (lifecycleWithAfterStartError) AfterStop() error    { return nil }

type runnerWrapper struct {
	stopCalled atomic.Bool
	runDone    chan struct{}
}

func (r *runnerWrapper) Run(ctx context.Context) error {
	<-ctx.Done()
	close(r.runDone)
	return ctx.Err()
}

func (r *runnerWrapper) Stop() error {
	r.stopCalled.Store(true)
	return nil
}

func (r *runnerWrapper) Name() string { return "managed" }

func waitForRunning(t *testing.T, svc *BaseService) {
	t.Helper()

	deadline := time.After(500 * time.Millisecond)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()

	for {
		if svc.IsRunning() {
			return
		}

		select {
		case <-deadline:
			t.Fatal("service did not enter running state in time")
		case <-tick.C:
		}
	}
}
