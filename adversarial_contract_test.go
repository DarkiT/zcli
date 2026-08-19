package zcli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

// ===== install/reinstall 带参透传对抗测试 =====

// TestAdversarial_InstallPassesAnyTokenVerbatim 验证 SetInterspersed(false) 下，
// 第一个非 flag token（通常是服务子命令）之后的任意 token 都原样进入服务参数，
// 不被解析/剥离/报错。注意：install 命令自身位置上的未知 flag 仍由 pflag 报错，
// 这是有意保留的——install 前导 flag 仍属 install 命令本身。
func TestAdversarial_InstallPassesAnyTokenVerbatim(t *testing.T) {
	cases := []struct {
		name string
		args []string // install 之后的部分（首个 token 为服务子命令）
	}{
		{"user-scenario", []string{"client", "--a=111", "--b=22"}},
		{"double-dash-kept", []string{"client", "--", "--a=111"}},
		{"short-unknown-flag", []string{"client", "-x"}},
		{"empty-string-token", []string{"client", ""}},
		{"spaces-and-quotes", []string{"client", `--a="x y"`, "k=v"}},
		{"equals-form-flag", []string{"client", "--a=111"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, err := NewBuilder("en").
				WithQuickConfig("adversarial-app", "Adv", "adv", "0.1.0").
				WithService(func(ctx context.Context) error { return nil }).
				BuildWithError()
			if err != nil {
				t.Fatal(err)
			}
			installCmd, _, _ := app.Command().Find([]string{"install"})
			var got []string
			installCmd.RunE = func(_ *Command, args []string) error {
				got = append([]string(nil), args...)
				return nil
			}
			full := append([]string{"install"}, c.args...)
			app.Command().SetArgs(full)
			if err := app.Execute(); err != nil {
				t.Fatalf("execute %v: %v", full, err)
			}
			if !reflect.DeepEqual(got, c.args) {
				t.Fatalf("args=%v, want verbatim %v", got, c.args)
			}
		})
	}
}

// TestAdversarial_InstallHelpReturnsNilWithoutSideEffects 验证 cobra 在
// `install --help` 时自动响应帮助：返回 nil、不 panic、不触碰安装逻辑。
func TestAdversarial_InstallHelpReturnsNilWithoutSideEffects(t *testing.T) {
	app, err := NewBuilder("en").
		WithQuickConfig("adv-help", "Adv", "adv", "0.1.0").
		WithService(func(ctx context.Context) error { return nil }).
		BuildWithError()
	if err != nil {
		t.Fatal(err)
	}
	for _, argv := range [][]string{{"install", "--help"}, {"install", "-h"}} {
		app.Command().SetArgs(argv)
		if err := app.Execute(); err != nil {
			t.Fatalf("execute %v: %v", argv, err)
		}
	}
}

// TestAdversarial_HelpNotLeadingIsPassthrough 验证帮助 flag 只在 install 命令
// 自身位置生效；`client --help` 中的 --help 应作为服务参数透传。
func TestAdversarial_HelpNotLeadingIsPassthrough(t *testing.T) {
	app, err := NewBuilder("en").
		WithQuickConfig("adv-helppos", "Adv", "adv", "0.1.0").
		WithService(func(ctx context.Context) error { return nil }).
		BuildWithError()
	if err != nil {
		t.Fatal(err)
	}
	installCmd, _, _ := app.Command().Find([]string{"install"})
	var got []string
	installCmd.RunE = func(_ *Command, args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	app.Command().SetArgs([]string{"install", "client", "--help"})
	if err := app.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := []string{"client", "--help"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%v, want %v（--help 非首位应透传）", got, want)
	}
}

// TestAdversarial_RootPersistentFlagBeforeInstallStillParses 验证 root 级
// persistent flag 出现在 install 之前时仍被正常解析，install 只收到其后的 token。
// 值需从命令的 merged flags（cmd.Flags()）读取——cobra 把父级 persistent flag
// 合并进当前命令解析，root 的 PersistentFlags() 保持默认。
func TestAdversarial_RootPersistentFlagBeforeInstallStillParses(t *testing.T) {
	app, err := NewBuilder("en").
		WithQuickConfig("adv-rootflag", "Adv", "adv", "0.1.0").
		WithService(func(ctx context.Context) error { return nil }).
		WithPersistentFlags(func(flags *FlagSet) {
			flags.String("profile", "dev", "runtime profile")
		}).
		BuildWithError()
	if err != nil {
		t.Fatal(err)
	}
	installCmd, _, _ := app.Command().Find([]string{"install"})
	var got []string
	var profile string
	installCmd.RunE = func(_ *Command, args []string) error {
		got = append([]string(nil), args...)
		profile, err = installCmd.Flags().GetString("profile")
		return err
	}
	app.Command().SetArgs([]string{"--profile=prod", "install", "client"})
	if err := app.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if profile != "prod" {
		t.Fatalf("root persistent --profile not parsed: %q", profile)
	}
	if !reflect.DeepEqual(got, []string{"client"}) {
		t.Fatalf("install args = %v, want [client]", got)
	}
}

// TestAdversarial_RootFlagAfterInstallIsServiceArg 验证新语义边界：
// root 级 flag 出现在 install 之后时不再被 root 解析，而是作为服务参数透传。
func TestAdversarial_RootFlagAfterInstallIsServiceArg(t *testing.T) {
	app, err := NewBuilder("en").
		WithQuickConfig("adv-rootflag2", "Adv", "adv", "0.1.0").
		WithService(func(ctx context.Context) error { return nil }).
		WithPersistentFlags(func(flags *FlagSet) {
			flags.String("profile", "dev", "runtime profile")
		}).
		BuildWithError()
	if err != nil {
		t.Fatal(err)
	}
	installCmd, _, _ := app.Command().Find([]string{"install"})
	var got []string
	installCmd.RunE = func(_ *Command, args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	app.Command().SetArgs([]string{"install", "client", "--profile=dev"})
	if err := app.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := []string{"client", "--profile=dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%v, want %v（install 后的 root flag 应作为服务参数）", got, want)
	}
}

// TestAdversarial_ApplyInstallArguments_InvalidNameReturnsError 验证 config 缺失且
// createServiceConfig 校验失败时返回错误而非 panic。
func TestAdversarial_ApplyInstallArguments_InvalidNameReturnsError(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	sm.commands.config.basic.Name = "bad name" // 触发 ErrInvalidName
	if err := sm.applyInstallArguments([]string{"client"}); err == nil {
		t.Fatal("expected error when config creation fails")
	}
}

// TestAdversarial_ApplyInstallArguments_CopiesArgs 验证参数写入时深拷贝，
// 调用方后续修改原切片不影响已保存的服务参数。
func TestAdversarial_ApplyInstallArguments_CopiesArgs(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	args := []string{"a", "b"}
	if err := sm.applyInstallArguments(args); err != nil {
		t.Fatal(err)
	}
	args[0] = "mutated"
	if sm.config.Arguments[0] != "a" {
		t.Fatalf("Arguments aliased caller slice: %v", sm.config.Arguments)
	}
}

// TestAdversarial_ApplyInstallArguments_MultipleCallsReplace 验证多次调用是整体
// 替换而非累积，且不 panic。
func TestAdversarial_ApplyInstallArguments_MultipleCallsReplace(t *testing.T) {
	sm := newTestServiceManager(t, &fakeDaemonService{})
	if err := sm.applyInstallArguments([]string{"v1"}); err != nil {
		t.Fatal(err)
	}
	if err := sm.applyInstallArguments([]string{"v2", "--x"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"v2", "--x"}
	if !reflect.DeepEqual(sm.config.Arguments, want) {
		t.Fatalf("Arguments = %v, want %v", sm.config.Arguments, want)
	}
}
