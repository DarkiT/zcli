package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/darkit/zcli"
)

type configView struct {
	Name            string             `json:"name"`
	DisplayName     string             `json:"display_name"`
	Description     string             `json:"description"`
	Language        string             `json:"language"`
	Service         zcli.ServiceConfig `json:"service"`
	StartTimeout    string             `json:"start_timeout"`
	StopTimeout     string             `json:"stop_timeout"`
	ShutdownInitial string             `json:"shutdown_initial"`
	ShutdownGrace   string             `json:"shutdown_grace"`
}

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get working directory", "error", err)
		os.Exit(1)
	}

	executable, err := os.Executable()
	if err != nil {
		slog.Error("failed to get executable path", "error", err)
		os.Exit(1)
	}

	app, err := zcli.NewBuilder("en").
		WithName("payments-agent").
		WithDisplayName("Payments Agent").
		WithDescription("Advanced service configuration example.").
		WithVersion("1.0.0").
		WithService(runAgent, stopAgent).
		WithWorkDir(workDir).
		WithChRoot(workDir).
		WithExecutable(executable).
		WithArguments("run", "--profile", "production").
		WithServiceUser("svc-payments").
		WithEnvVar("APP_ENV", "production").
		WithEnvVar("LOG_FORMAT", "json").
		WithDependency("network-online.target", zcli.DependencyAfter).
		WithStructuredDependencies(
			zcli.Dependency{Name: "postgresql.service", Type: zcli.DependencyRequire},
			zcli.Dependency{Name: "redis.service", Type: zcli.DependencyWant},
		).
		WithServiceOption("Restart", "on-failure").
		WithServiceOptionsMap(zcli.ServiceOptions{
			"RestartSec": "5s",
			"UMask":      "0027",
		}).
		WithAllowSudoFallback(true).
		WithServiceTimeouts(20*time.Second, 30*time.Second).
		WithServiceConfig(func(cfg *zcli.ServiceConfig) {
			cfg.Options["LimitNOFILE"] = 65535
		}).
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(1)
	}

	app.AddCommand(newInspectCommand(app))

	if err := app.Execute(); err != nil {
		slog.Error("service command failed", "error", err)
		os.Exit(1)
	}
}

func newInspectCommand(app *zcli.Cli) *zcli.Command {
	return &zcli.Command{
		Use:   "inspect",
		Short: "Print the configured service metadata",
		RunE: func(cmd *zcli.Command, args []string) error {
			cfg := app.Config()
			view := configView{
				Name:            cfg.Basic().Name,
				DisplayName:     cfg.Basic().DisplayName,
				Description:     cfg.Basic().Description,
				Language:        cfg.Basic().Language,
				Service:         cfg.Service(),
				StartTimeout:    cfg.Runtime().StartTimeout.String(),
				StopTimeout:     cfg.Runtime().StopTimeout.String(),
				ShutdownInitial: cfg.Runtime().ShutdownInitial.String(),
				ShutdownGrace:   cfg.Runtime().ShutdownGrace.String(),
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(view)
		},
	}
}

func runAgent(ctx context.Context) error {
	slog.Info("payments agent started")
	<-ctx.Done()
	slog.Info("payments agent received shutdown signal")
	return nil
}

func stopAgent() error {
	slog.Info("payments agent stop hook completed")
	return nil
}
