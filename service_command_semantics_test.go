package zcli

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
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

	for _, name := range []string{"run", "install", "reinstall", "uninstall", "start", "stop", "restart", "status"} {
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

func TestReinstallCommand_UsesDaemonFallback(t *testing.T) {
	stub := &fakeDaemonService{status: service.StatusStopped}
	sm := newTestServiceManager(t, stub)

	cmd := sm.newReinstallCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if stub.uninstallCalled != 1 || stub.installCalled != 1 {
		t.Fatalf("expected uninstall+install fallback, got uninstall=%d install=%d", stub.uninstallCalled, stub.installCalled)
	}
}

func TestInstallCommand_ForceOverwritesExisting(t *testing.T) {
	stub := &fakeDaemonService{status: service.StatusStopped}
	sm := newTestServiceManager(t, stub)
	sm.commands.config.service.Force = true

	cmd := sm.newInstallCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("forced install: %v", err)
	}
	if stub.installCalled != 1 {
		t.Fatalf("expected Install to run when Force is set, got %d", stub.installCalled)
	}
}

func TestInstallCommand_AlreadyExistsSkipsInstall(t *testing.T) {
	stub := &fakeDaemonService{status: service.StatusStopped}
	sm := newTestServiceManager(t, stub)

	cmd := sm.newInstallCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("install existing: %v", err)
	}
	if stub.installCalled != 0 {
		t.Fatalf("expected install to be skipped when service exists, got %d", stub.installCalled)
	}
}

func TestWrapServiceError_MapsDaemonSentinels(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})

	cases := []struct {
		name string
		err  error
		code ErrorCode
	}{
		{name: "permission", err: service.ErrPermission, code: ErrPermission},
		{name: "already installed", err: service.ErrAlreadyInstalled, code: ErrServiceInstall},
		{name: "invalid name", err: service.ErrInvalidName, code: ErrConfigInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := sm.wrapServiceError(tc.err, ErrServiceInstall, "install")
			if !IsErrorCode(wrapped, tc.code) {
				t.Fatalf("expected %s, got %v", tc.code, wrapped)
			}
			if !errors.Is(wrapped, tc.err) {
				t.Fatalf("expected errors.Is to keep %v", tc.err)
			}
		})
	}
}

func TestBuilder_RejectsInvalidDaemonServiceName(t *testing.T) {
	_, err := NewBuilder("en").
		WithName("bad name").
		WithService(func(ctx context.Context) error { return nil }).
		BuildWithError()
	if err == nil || !strings.Contains(err.Error(), "path separators") {
		t.Fatalf("expected invalid name error, got %v", err)
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

// TestInstallCommands_NonInterspersedArgs 验证 install/reinstall 关闭 flag 与
// positional 的交错解析，使 `install client --a=111 --b=22` 在第一个非 flag
// token 之后原样透传，同时不破坏 root persistent flag 的解析。
func TestInstallCommands_NonInterspersedArgs(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	for _, makeCmd := range []func() *Command{sm.newInstallCmd, sm.newReinstallCmd} {
		cmd := makeCmd()
		if err := cmd.Flags().Parse([]string{"client", "--a=111", "--b=22"}); err != nil {
			t.Fatalf("parse: %v", err)
		}
		got := cmd.Flags().Args()
		want := []string{"client", "--a=111", "--b=22"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("service args not passed through verbatim: got %v, want %v", got, want)
		}
	}
}

// TestInstallCommand_CobraPassesRawArgs 端到端验证 cobra 层把 install 之后的
// 全部 token（含未注册 flag）原样交给 RunE。
func TestInstallCommand_CobraPassesRawArgs(t *testing.T) {
	app, err := NewBuilder("en").
		WithQuickConfig("probe-app", "Probe", "probe", "0.1.0").
		WithService(func(ctx context.Context) error { return nil }).
		BuildWithError()
	if err != nil {
		t.Fatal(err)
	}
	installCmd, _, _ := app.Command().Find([]string{"install"})
	if installCmd == nil {
		t.Fatal("install command not found")
	}
	var got []string
	installCmd.RunE = func(_ *Command, args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	app.Command().SetArgs([]string{"install", "client", "--a=111", "--b=22"})
	if err := app.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := []string{"client", "--a=111", "--b=22"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RunE received %v, want %v", got, want)
	}
}

// TestApplyInstallArguments_ReplacesServiceArgs 验证 CLI 参数整体替换为服务自启参数，
// 并强制重建 service 实例（让 Install 使用新参数）。
func TestApplyInstallArguments_ReplacesServiceArgs(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	if err := sm.applyInstallArguments([]string{"client", "--a=111", "--b=22"}); err != nil {
		t.Fatalf("applyInstallArguments: %v", err)
	}
	if sm.config == nil {
		t.Fatal("expected config to be initialized")
	}
	want := []string{"client", "--a=111", "--b=22"}
	if !reflect.DeepEqual(sm.config.Arguments, want) {
		t.Fatalf("Arguments = %v, want %v", sm.config.Arguments, want)
	}
	if sm.service != nil {
		t.Fatal("expected service reset so Install rebuilds with new args")
	}
}

// TestApplyInstallArguments_OverridesBuildTimeArgs 验证整体替换语义：
// 构建期 WithArguments 配置在 install 带参时被覆盖。
func TestApplyInstallArguments_OverridesBuildTimeArgs(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	sm.commands.config.service.Arguments = []string{"legacy", "--keep"}
	cfg, err := sm.createServiceConfig()
	if err != nil {
		t.Fatalf("createServiceConfig: %v", err)
	}
	sm.config = cfg
	if err := sm.applyInstallArguments([]string{"client", "--a=111"}); err != nil {
		t.Fatalf("applyInstallArguments: %v", err)
	}
	want := []string{"client", "--a=111"}
	if !reflect.DeepEqual(sm.config.Arguments, want) {
		t.Fatalf("Arguments = %v, want %v（应整体替换构建期参数）", sm.config.Arguments, want)
	}
}

// TestApplyInstallArguments_EmptyArgsKeepConfig 验证不带参数时不动 config/service。
func TestApplyInstallArguments_EmptyArgsKeepConfig(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	if err := sm.applyInstallArguments(nil); err != nil {
		t.Fatalf("applyInstallArguments(nil): %v", err)
	}
	if sm.config != nil || sm.service == nil {
		t.Fatal("expected config/service untouched for empty args")
	}
}
