package zcli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpWritesToStdout(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("help-out").
		WithDescription("help should go to stdout").
		WithCommand(NewCommand("status", "Show app status", WithCommandRun(func(cmd *Command, args []string) {}))).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	var stdout, stderr bytes.Buffer
	app.SetOut(&stdout)
	app.SetErr(&stderr)
	app.SetArgs([]string{"--help"})
	if err := app.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Show app status") || !strings.Contains(out, "status") {
		t.Fatalf("help stdout missing status command:\n%s", out)
	}
	if strings.Contains(out, "System Commands") {
		t.Fatalf("user status must not be grouped as a system command:\n%s", out)
	}
	if strings.Contains(stderr.String(), "Show app status") {
		t.Fatalf("help leaked to stderr:\n%s", stderr.String())
	}
}

func TestUserStatusIsNotSystemCommand(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("mixed-status").
		WithService(func(ctx context.Context) error { return nil }).
		WithCommand(NewCommand("status", "Application status", WithCommandRun(func(cmd *Command, args []string) {}))).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	var userStatus, serviceStatus bool
	for _, cmd := range app.Commands() {
		if cmd.Name() != "status" {
			continue
		}
		if isFrameworkServiceCommand(cmd) {
			serviceStatus = true
		} else {
			userStatus = true
		}
	}
	if !userStatus || !serviceStatus {
		t.Fatalf("expected both user and service status commands, user=%v service=%v", userStatus, serviceStatus)
	}
}

func TestCompletionCommandIsAvailable(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("comp-app").
		WithCommand(NewCommand("ping", "ping", WithCommandRun(func(cmd *Command, args []string) {}))).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if app.Command().CompletionOptions.DisableDefaultCmd {
		t.Fatal("default completion command should be enabled")
	}

	found := false
	for _, cmd := range app.Commands() {
		if cmd.Name() == "completion" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("completion command should be registered")
	}
}

func TestRootActionAuto(t *testing.T) {
	t.Run("service only defaults to run", func(t *testing.T) {
		app, err := NewBuilder("en").
			WithName("svc-only").
			WithService(func(ctx context.Context) error { return nil }).
			BuildWithError()
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if !app.shouldRunServiceOnRoot() {
			t.Fatal("pure service should default to run")
		}
	})

	t.Run("mixed app shows help on root", func(t *testing.T) {
		ran := false
		app, err := NewBuilder("en").
			WithName("mixed-app").
			WithService(func(ctx context.Context) error {
				ran = true
				return nil
			}).
			WithCommand(NewCommand("inspect", "inspect", WithCommandRun(func(cmd *Command, args []string) {}))).
			BuildWithError()
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if app.shouldRunServiceOnRoot() {
			t.Fatal("mixed app should default to help")
		}

		var stdout bytes.Buffer
		app.SetOut(&stdout)
		app.SetArgs([]string{})
		if err := app.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
		if ran {
			t.Fatal("mixed app must not start service on root")
		}
		if !strings.Contains(stdout.String(), "inspect") {
			t.Fatalf("root help should list user commands:\n%s", stdout.String())
		}
	})

	t.Run("explicit run overrides mixed default", func(t *testing.T) {
		app, err := NewBuilder("en").
			WithName("mixed-run").
			WithRootAction(RootActionRun).
			WithService(func(ctx context.Context) error { return nil }).
			WithCommand(NewCommand("inspect", "inspect", WithCommandRun(func(cmd *Command, args []string) {}))).
			BuildWithError()
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if !app.shouldRunServiceOnRoot() {
			t.Fatal("WithRootAction(run) should start service on root")
		}
	})
}

func TestExportFlagsDedupesOverlappingSets(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("flags-app").
		WithPersistentFlags(func(flags *FlagSet) {
			flags.String("region", "us-east-1", "region")
			flags.Bool("debug", false, "debug")
		}).
		WithFlags(func(flags *FlagSet) {
			flags.Int("port", 8080, "port")
		}).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	sets := app.ExportFlagsForViper("debug")
	if len(sets) != 1 {
		t.Fatalf("expected a single bindable flag set, got %d", len(sets))
	}

	counts := map[string]int{}
	sets[0].VisitAll(func(flag *Flag) {
		counts[flag.Name]++
	})
	if counts["region"] != 1 || counts["port"] != 1 {
		t.Fatalf("flags should appear once, got %#v", counts)
	}
	if counts["debug"] != 0 || counts["help"] != 0 {
		t.Fatalf("excluded/system flags leaked: %#v", counts)
	}
}

func TestMissingExecutableStillBuildsHelp(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("missing-bin").
		WithService(func(ctx context.Context) error { return nil }).
		WithExecutable("/definitely/missing/zcli-test-bin").
		BuildWithError()
	if err != nil {
		t.Fatalf("missing install path must not fail Build: %v", err)
	}

	var stdout bytes.Buffer
	app.SetOut(&stdout)
	app.SetArgs([]string{"--help"})
	if err := app.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(stdout.String(), "missing-bin") {
		t.Fatalf("help missing app name:\n%s", stdout.String())
	}
}

func TestStructuredDependenciesAppendAndSet(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("deps-app").
		WithDependency("network-online.target", DependencyAfter).
		WithStructuredDependencies(
			Dependency{Name: "postgresql.service", Type: DependencyRequire},
			Dependency{Name: "redis.service", Type: DependencyWant},
		).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := len(app.Config().Service().StructuredDeps); got != 3 {
		t.Fatalf("append should keep all dependencies, got %d", got)
	}

	app, err = NewBuilder("en").
		WithName("deps-set").
		WithDependency("keep-me", DependencyAfter).
		SetStructuredDependencies(Dependency{Name: "only-me", Type: DependencyWant}).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	deps := app.Config().Service().StructuredDeps
	if len(deps) != 1 || deps[0].Name != "only-me" {
		t.Fatalf("SetStructuredDependencies should replace, got %#v", deps)
	}
}
