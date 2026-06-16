package zcli

import (
	"context"
	"testing"
	"time"
)

func BenchmarkBuildWithErrorCLI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app, err := NewBuilder("en").
			WithName("bench-cli").
			WithDescription("benchmark cli builder").
			WithCommand(NewCommand(
				"inspect",
				"Inspect runtime state",
				WithCommandRun(func(cmd *Command, args []string) {}),
			)).
			BuildWithError()
		if err != nil {
			b.Fatalf("BuildWithError failed: %v", err)
		}
		if app == nil || app.Command() == nil {
			b.Fatal("expected valid cli instance")
		}
	}
}

func BenchmarkBuildWithErrorService(b *testing.B) {
	run := func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}
	stop := func() error { return nil }

	for i := 0; i < b.N; i++ {
		app, err := NewBuilder("en").
			WithName("bench-service").
			WithDescription("benchmark service builder").
			WithService(run, stop).
			WithServiceTimeouts(15*time.Second, 20*time.Second).
			BuildWithError()
		if err != nil {
			b.Fatalf("BuildWithError failed: %v", err)
		}
		if app == nil || app.Command() == nil {
			b.Fatal("expected valid service cli instance")
		}
	}
}

func BenchmarkNewCommandSugar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := NewCommand(
			"inspect [target]",
			"Inspect a target",
			WithCommandAliases("show"),
			WithCommandLong("Inspect target with additive zcli sugar."),
			WithCommandArgs(ExactArgs(1)),
			WithCommandValidArgs("service-a", "service-b"),
			WithCommandRun(func(cmd *Command, args []string) {}),
			WithCommandFlags(func(flags *FlagSet) {
				flags.Bool("verbose", false, "Enable verbose output")
			}),
		)
		if cmd == nil || cmd.Flags() == nil {
			b.Fatal("expected valid command")
		}
	}
}

func BenchmarkExportFlagsForViper(b *testing.B) {
	app, err := NewBuilder("en").
		WithName("bench-flags").
		WithDescription("benchmark flag export").
		BuildWithError()
	if err != nil {
		b.Fatalf("BuildWithError failed: %v", err)
	}

	app.PersistentFlags().String("config", "", "Path to config")
	app.PersistentFlags().Bool("debug", false, "Enable debug logging")
	app.PersistentFlags().String("region", "us-east-1", "Target region")
	app.Flags().Int("port", 8080, "HTTP port")
	app.Flags().Bool("dry-run", false, "Preview changes without applying them")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flagSets := app.ExportFlagsForViper("debug")
		if len(flagSets) == 0 {
			b.Fatal("expected exported flag sets")
		}
	}
}
