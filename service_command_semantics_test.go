package zcli

import (
	"context"
	"errors"
	"io"
	"testing"

	service "github.com/darkit/daemon"
)

type fakeDaemonService struct {
	status          service.Status
	statusErr       error
	installErr      error
	uninstallErr    error
	startErr        error
	stopErr         error
	restartErr      error
	installCalled   int
	uninstallCalled int
	startCalled     int
	stopCalled      int
	restartCalled   int
}

func (f *fakeDaemonService) Run() error       { return nil }
func (f *fakeDaemonService) Start() error     { f.startCalled++; return f.startErr }
func (f *fakeDaemonService) Stop() error      { f.stopCalled++; return f.stopErr }
func (f *fakeDaemonService) Restart() error   { f.restartCalled++; return f.restartErr }
func (f *fakeDaemonService) Install() error   { f.installCalled++; return f.installErr }
func (f *fakeDaemonService) Uninstall() error { f.uninstallCalled++; return f.uninstallErr }
func (f *fakeDaemonService) Logger(chan<- error) (service.Logger, error) {
	return nil, nil
}

func (f *fakeDaemonService) SystemLogger(chan<- error) (service.Logger, error) {
	return nil, nil
}
func (f *fakeDaemonService) String() string                  { return "fake" }
func (f *fakeDaemonService) Platform() string                { return "test" }
func (f *fakeDaemonService) Status() (service.Status, error) { return f.status, f.statusErr }

func TestInstallCommand_PropagatesStatusErrors(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{
		statusErr: errors.New("query failed"),
	})

	cmd := sm.newInstallCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected install command to surface status error")
	}
	if !IsErrorCode(err, ErrServiceStatus) {
		t.Fatalf("expected ErrServiceStatus, got %v", err)
	}
}

func TestStatusCommand_NotInstalledIsIdempotent(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{
		status:    service.StatusUnknown,
		statusErr: service.ErrNotInstalled,
	})

	cmd := sm.newStatusCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("expected status command to treat not-installed as informational, got %v", err)
	}
}

func TestUninstallCommand_NotInstalledIsIdempotent(t *testing.T) {
	stub := &fakeDaemonService{
		status:    service.StatusUnknown,
		statusErr: service.ErrNotInstalled,
	}
	sm := newTestServiceManager(t, stub)

	cmd := sm.newUninstallCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("expected uninstall command to treat not-installed as informational, got %v", err)
	}
	if stub.uninstallCalled != 0 {
		t.Fatalf("expected uninstall to be skipped when service is absent, got %d call(s)", stub.uninstallCalled)
	}
}

func TestAttachServiceAssemblyInjectsCommandsAndPreservesRootFallback(t *testing.T) {
	config := NewConfig()
	config.basic.Name = "attach-service"
	config.runtime.Run = func(ctx context.Context) error { <-ctx.Done(); return nil }

	rootRunCalled := false
	cli := &Cli{
		config:  config,
		colors:  newColors(),
		lang:    GetLanguageManager().GetPrimary(),
		command: &Command{Use: config.basic.Name},
	}
	cli.command.Run = func(cmd *Command, args []string) {
		rootRunCalled = true
	}

	stub := &fakeDaemonService{}
	localizer := NewServiceLocalizer(GetLanguageManager(), cli.colors)
	localizer.ConfigureOutput(io.Discard, io.Discard, false, false)
	sm := &sManager{
		commands:  cli,
		localizer: localizer,
		service:   stub,
	}

	cli.attachServiceAssembly(sm)

	for _, name := range []string{"run", "install", "uninstall", "start", "stop", "restart", "status"} {
		cmd, _, err := cli.command.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Fatalf("expected service command %q to be injected, cmd=%v err=%v", name, cmd, err)
		}
	}

	if cli.command.Run != nil {
		t.Fatal("root Run should be cleared after service root run strategy is attached")
	}
	if cli.command.RunE == nil {
		t.Fatal("root RunE should be attached for service fallback")
	}

	if err := cli.command.RunE(cli.command, []string{"arg"}); err != nil {
		t.Fatalf("expected original root Run fallback to remain usable, got %v", err)
	}
	if !rootRunCalled {
		t.Fatal("expected original root Run to remain reachable through service root strategy")
	}
}

func newTestServiceManager(t *testing.T, svc service.Service) *sManager {
	t.Helper()

	config := NewConfig()
	config.basic.Name = "test-service"
	cli := &Cli{config: config, colors: newColors(), lang: GetLanguageManager().GetPrimary()}
	cli.command = &Command{}

	localizer := NewServiceLocalizer(GetLanguageManager(), cli.colors)
	localizer.ConfigureOutput(io.Discard, io.Discard, false, false)

	return &sManager{
		commands:  cli,
		localizer: localizer,
		service:   svc,
	}
}
