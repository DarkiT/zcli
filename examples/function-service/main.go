package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/darkit/zcli"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get working directory", "error", err)
		os.Exit(1)
	}

	app, err := zcli.NewBuilder("en").
		WithName("mailer-worker").
		WithDisplayName("Mailer Worker").
		WithDescription("Recommended function-based service example for small applications.").
		WithVersion("1.0.0").
		WithService(runMailerWorker, stopMailerWorker).
		WithWorkDir(workDir).
		WithEnvVar("APP_ENV", "production").
		WithDependency("network-online.target", zcli.DependencyAfter).
		WithServiceTimeouts(15*time.Second, 20*time.Second).
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(1)
	}

	if err := app.Execute(); err != nil {
		slog.Error("service command failed", "error", err)
		os.Exit(1)
	}
}

func runMailerWorker(ctx context.Context) error {
	slog.Info("mailer worker started")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("mailer worker received shutdown signal")
			return nil
		case <-ticker.C:
			slog.Info("processed email batch", "count", 25)
		}
	}
}

func stopMailerWorker() error {
	slog.Info("flushing mailer buffers")
	time.Sleep(150 * time.Millisecond)
	slog.Info("mailer worker stopped cleanly")
	return nil
}
