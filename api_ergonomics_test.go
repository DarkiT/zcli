package zcli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuilderSugarAPIs(t *testing.T) {
	t.Run("WithQuickConfig and batch helpers", func(t *testing.T) {
		var hookCalled atomic.Bool

		app, err := NewBuilder("en").
			WithQuickConfig("sugar-app", "Sugar App", "batch construction", "2.0.0").
			WithSilenceUsage(true).
			WithSilenceErrors(true).
			WithNoColor(true).
			WithEnvVars(map[string]string{
				"APP_ENV": "test",
				"":        "ignored",
			}).
			WithPersistentFlags(func(flags *FlagSet) {
				flags.String("profile", "dev", "runtime profile")
			}).
			WithFlags(func(flags *FlagSet) {
				flags.Bool("dry-run", false, "preview only")
			}).
			WithCommands(NewCommand("ping", "ping command", WithCommandRun(func(cmd *Command, args []string) {}))).
			WithCommandFactory(func(app *Cli) *Command {
				return NewCommand("inspect", "inspect app", WithCommandRunE(func(cmd *Command, args []string) error {
					fmt.Fprint(cmd.OutOrStdout(), app.Config().Basic().Name)
					return nil
				}))
			}).
			WithInitHook(func(cmd *Command, args []string) error {
				hookCalled.Store(true)
				return nil
			}).
			BuildWithError()
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if app.Config().Basic().Name != "sugar-app" {
			t.Fatalf("name = %q", app.Config().Basic().Name)
		}
		if !app.Command().SilenceUsage || !app.Command().SilenceErrors {
			t.Fatal("silence flags should be applied to root command")
		}
		if app.Config().Service().EnvVars["APP_ENV"] != "test" {
			t.Fatalf("env vars = %#v", app.Config().Service().EnvVars)
		}
		if _, ok := app.Config().Service().EnvVars[""]; ok {
			t.Fatal("empty env key should be ignored")
		}
		if app.PersistentFlags().Lookup("profile") == nil {
			t.Fatal("persistent flag should be registered during build")
		}
		if app.Flags().Lookup("dry-run") == nil {
			t.Fatal("local flag should be registered during build")
		}

		names := map[string]bool{}
		for _, cmd := range app.Commands() {
			names[cmd.Name()] = true
		}
		if !names["ping"] || !names["inspect"] {
			t.Fatalf("commands = %#v", names)
		}

		app.SetArgs([]string{"inspect"})
		if err := app.Execute(); err != nil {
			t.Fatalf("execute inspect: %v", err)
		}
		if !hookCalled.Load() {
			t.Fatal("init hook should run")
		}
	})

	t.Run("WithBuildTime accepts RFC3339 and rejects invalid values", func(t *testing.T) {
		app, err := NewBuilder("en").
			WithName("time-app").
			WithBuildTime("2026-08-17T02:11:00Z").
			BuildWithError()
		if err != nil {
			t.Fatalf("rfc3339 should parse: %v", err)
		}
		if app.Config().Runtime().BuildInfo == nil || app.Config().Runtime().BuildInfo.BuildTime.IsZero() {
			t.Fatal("build time should be stored")
		}

		_, err = NewBuilder("en").
			WithName("bad-time").
			WithBuildTime("not-a-time").
			BuildWithError()
		if err == nil || !strings.Contains(err.Error(), "invalid build time") {
			t.Fatalf("expected invalid build time error, got %v", err)
		}
	})

	t.Run("nil and negative inputs accumulate at build time", func(t *testing.T) {
		_, err := NewBuilder("en").
			WithName("strict-app").
			WithRuntime(nil).
			WithService(nil).
			WithServiceTimeouts(-1, time.Second).
			WithValidator(nil).
			BuildWithError()
		if err == nil {
			t.Fatal("expected accumulated build errors")
		}

		var buildErr *BuildError
		if !errors.As(err, &buildErr) {
			t.Fatalf("expected BuildError, got %T", err)
		}
		if len(buildErr.Errors) < 3 {
			t.Fatalf("expected multiple build errors, got %#v", buildErr.Errors)
		}
		if !strings.Contains(buildErr.Error(), "2.") {
			t.Fatalf("BuildError should list all errors, got %q", buildErr.Error())
		}
		if ExitCode(err) != ExitUsage {
			t.Fatalf("build errors should map to usage exit code, got %d", ExitCode(err))
		}
	})

	t.Run("Build reuses BuildWithError", func(t *testing.T) {
		app := NewBuilder("en").WithName("panic-free").Build()
		if app.Name() != "panic-free" {
			t.Fatalf("name = %q", app.Name())
		}
	})
}

func TestCommandFactoryOptions(t *testing.T) {
	var preRan atomic.Bool

	cmd := NewCommand(
		"hidden",
		"hidden command",
		WithCommandHidden(true),
		WithCommandSilenceUsage(true),
		WithCommandSilenceErrors(true),
		WithCommandPreRunE(func(cmd *Command, args []string) error {
			preRan.Store(true)
			return nil
		}),
		WithCommandRun(func(cmd *Command, args []string) {}),
	)

	if !cmd.Hidden || !cmd.SilenceUsage || !cmd.SilenceErrors {
		t.Fatal("command options should set hidden/silence flags")
	}

	app := NewBuilder("en").
		WithName("opt-app").
		WithCommand(cmd).
		Build()
	app.SetArgs([]string{"hidden"})
	if err := app.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !preRan.Load() {
		t.Fatal("PreRunE should run")
	}
}

func TestCliRunAndExitCode(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("run-app").
		WithSilenceErrors(true).
		WithSilenceUsage(true).
		WithCommand(NewCommand("ok", "ok", WithCommandRun(func(cmd *Command, args []string) {}))).
		WithCommand(NewCommand("fail", "fail", WithCommandRunE(func(cmd *Command, args []string) error {
			return NewError(ErrRuntime).Operation("run").Message("boom").Build()
		}))).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	app.SetArgs([]string{"ok"})
	if code := app.Run(); code != ExitSuccess {
		t.Fatalf("success exit = %d", code)
	}

	app.SetArgs([]string{"fail"})
	if code := app.Run(); code != ExitFailure {
		t.Fatalf("failure exit = %d", code)
	}

	if code := (*Cli)(nil).Run(); code != ExitFailure {
		t.Fatalf("nil cli exit = %d", code)
	}

	if ExitCode(nil) != ExitSuccess {
		t.Fatal("nil error should be success")
	}
	if ExitCode(ErrConfigValidationFailed(nil)) != ExitUsage {
		t.Fatal("config service error should be usage")
	}
}

func TestErrorChainPreserved(t *testing.T) {
	root := NewError(ErrServiceStop).
		Service("demo").
		Operation("stop").
		Message("failed").
		Build()

	joined := CombineErrors(fmt.Errorf("wrapper: %w", context.Canceled), root)
	if !errors.Is(joined, context.Canceled) {
		t.Fatal("CombineErrors should preserve errors.Is")
	}

	got, ok := GetServiceError(joined)
	if !ok || got.Code != ErrServiceStop {
		t.Fatalf("GetServiceError should unwrap joined errors, got %#v ok=%v", got, ok)
	}
	if !IsServiceError(joined) || !IsErrorCode(joined, ErrServiceStop) {
		t.Fatal("error helpers should follow the error chain")
	}
}

func TestNewCliSharesAppAssembly(t *testing.T) {
	cli := NewCli()
	if cli == nil || cli.Command() == nil {
		t.Fatal("NewCli should return an assembled app")
	}
	cli.AddCommand(NewCommand("ping", "ping", WithCommandRun(func(cmd *Command, args []string) {})))
	if len(cli.Commands()) == 0 {
		t.Fatal("assembled cli should accept commands")
	}
}

func TestWithContextNilUsesBackground(t *testing.T) {
	app, err := NewBuilder("en").
		WithName("ctx-app").
		WithContext(nil).
		BuildWithError()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if app.Config().Context() == nil {
		t.Fatal("nil context should fall back to background")
	}
}
