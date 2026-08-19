package zcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	service "github.com/darkit/daemon"
)

func TestCreateServiceConfig_DefaultStopTimeoutNotLeaked(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "timeout-leak"
	config.runtime.StartTimeout = 0
	config.runtime.RestartTimeout = 0

	sm := &sManager{
		commands:  &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()},
		localizer: NewServiceLocalizer(GetLanguageManager(), newColors()),
	}

	svcConfig, err := sm.createServiceConfig()
	if err != nil {
		t.Fatalf("createServiceConfig: %v", err)
	}
	if config.runtime.StopTimeout != 20*time.Second {
		t.Fatalf("in-process StopTimeout default changed: %s", config.runtime.StopTimeout)
	}
	if svcConfig.Timeout.Stop != 0 {
		t.Fatalf("default StopTimeout leaked into daemon Config.Timeout.Stop: %s", svcConfig.Timeout.Stop)
	}
}

func TestCreateServiceConfig_ExplicitStopTimeoutCopied(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("timeout-explicit").
		WithServiceTimeouts(11*time.Second, 22*time.Second).
		WithRestartTimeout(33 * time.Second).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	sm := &sManager{
		commands:  app,
		localizer: NewServiceLocalizer(GetLanguageManager(), app.colors),
	}
	svcConfig, err := sm.createServiceConfig()
	if err != nil {
		t.Fatalf("createServiceConfig: %v", err)
	}
	if svcConfig.Timeout != (service.TimeoutConfig{Start: 11 * time.Second, Stop: 22 * time.Second, Restart: 33 * time.Second}) {
		t.Fatalf("explicit timeouts not copied: %#v", svcConfig.Timeout)
	}
}

func TestExecuteContext_ReturnsAssemblyError(t *testing.T) {
	want := errors.New("assemble app failed")
	app := &Cli{
		config:      NewConfig(),
		command:     &Command{Use: "broken"},
		assemblyErr: want,
	}

	if err := app.ExecuteContext(context.Background()); !errors.Is(err, want) {
		t.Fatalf("ExecuteContext ignored assemblyErr: %v", err)
	}
	if _, err := app.ExecuteC(); !errors.Is(err, want) {
		t.Fatalf("ExecuteC ignored assemblyErr: %v", err)
	}
	if _, err := app.ExecuteContextC(context.Background()); !errors.Is(err, want) {
		t.Fatalf("ExecuteContextC ignored assemblyErr: %v", err)
	}
}

func TestStartRestartUninstall_FailedStateProceeds(t *testing.T) {
	stub := &fakeDaemonService{
		status:    service.StatusUnknown,
		statusErr: errors.New("service in failed state"),
	}
	sm := newTestServiceManager(t, stub)

	if err := sm.newStartCmd().RunE(sm.newStartCmd(), nil); err != nil {
		t.Fatalf("start failed-state: %v", err)
	}
	if stub.startCalled != 1 {
		t.Fatalf("start should proceed on failed unit, got %d", stub.startCalled)
	}

	if err := sm.newRestartCmd().RunE(sm.newRestartCmd(), nil); err != nil {
		t.Fatalf("restart failed-state: %v", err)
	}
	if stub.restartCalled != 1 || stub.stopCalled != 0 || stub.startCalled != 1 {
		t.Fatalf("restart should call Restart once, got restart=%d stop=%d start=%d", stub.restartCalled, stub.stopCalled, stub.startCalled)
	}

	if err := sm.newUninstallCmd().RunE(sm.newUninstallCmd(), nil); err != nil {
		t.Fatalf("uninstall failed-state: %v", err)
	}
	if stub.uninstallCalled != 1 {
		t.Fatalf("uninstall should proceed on failed unit, got %d", stub.uninstallCalled)
	}

	if err := sm.newStatusCmd().RunE(sm.newStatusCmd(), nil); err != nil {
		t.Fatalf("status failed-state should be informational, got %v", err)
	}
}

func TestRestartCommand_UsesDaemonRestart(t *testing.T) {
	stub := &fakeDaemonService{status: service.StatusRunning}
	sm := newTestServiceManager(t, stub)
	if err := sm.newRestartCmd().RunE(sm.newRestartCmd(), nil); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if stub.restartCalled != 1 {
		t.Fatalf("expected daemon Restart(), got %d", stub.restartCalled)
	}
	if stub.stopCalled != 0 || stub.startCalled != 0 {
		t.Fatalf("restart must not hand-roll Stop+Start, stop=%d start=%d", stub.stopCalled, stub.startCalled)
	}
}

func TestShouldForceInstall_ReadsServiceOptions(t *testing.T) {
	stub := &fakeDaemonService{status: service.StatusStopped}
	sm := newTestServiceManager(t, stub)
	sm.commands.config.service.Options[service.OptionForceInstall] = true
	prevGOOS := serviceManagerGOOS
	serviceManagerGOOS = "windows"
	t.Cleanup(func() { serviceManagerGOOS = prevGOOS })
	sm.config = &service.Config{Name: "test-service"}

	if !sm.shouldForceInstall() {
		t.Fatal("OptionForceInstall on ServiceConfig should enable Force")
	}

	if err := sm.newInstallCmd().RunE(sm.newInstallCmd(), nil); err != nil {
		t.Fatalf("forced install: %v", err)
	}
	if stub.installCalled != 1 {
		t.Fatalf("expected Install, got %d", stub.installCalled)
	}
	if !sm.config.Force || !optionEnabled(sm.config.Option, service.OptionForceInstall) {
		t.Fatalf("live daemon config was not forced: %#v", sm.config)
	}
}

func TestRunArguments_NilServiceConfig(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	sm.commands.config.service = nil
	if got := sm.runArguments(nil); got != nil {
		t.Fatalf("nil ServiceConfig should keep incoming args, got %#v", got)
	}
	if got := sm.runArguments([]string{"run"}); len(got) != 1 || got[0] != "run" {
		t.Fatalf("explicit args should pass through, got %#v", got)
	}
}

func TestCheckPermissions_AllowsReadOnlyExecutable(t *testing.T) {
	loc := NewServiceLocalizer(GetLanguageManager(), nil)
	dir := t.TempDir()
	bin := filepath.Join(dir, "app")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(bin, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := checkPermissions(bin, 0o555, loc); err != nil {
		t.Fatalf("read-only executable should pass r-x check: %v", err)
	}
	if err := checkPermissions(bin, 0o755, loc); err == nil {
		t.Fatal("0o755 still requires owner-write and should fail on 0555")
	}
}

func TestCreateServiceConfig_ValidateBeforeMissingPath(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "bad name"
	config.service.Executable = "/definitely/missing/zcli-test-bin"

	sm := &sManager{
		commands:  &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()},
		localizer: NewServiceLocalizer(GetLanguageManager(), newColors()),
	}
	_, err := sm.createServiceConfig()
	if err == nil || !errors.Is(err, service.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName before path errors, got %v", err)
	}
}
