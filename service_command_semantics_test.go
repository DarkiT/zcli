package zcli

import (
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
