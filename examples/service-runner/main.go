package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/darkit/zcli"
)

type queueWorker struct {
	name         string
	logger       *slog.Logger
	pollInterval time.Duration
	stopCh       chan struct{}
	stopOnce     sync.Once
}

func newQueueWorker(logger *slog.Logger, pollInterval time.Duration) *queueWorker {
	return &queueWorker{
		name:         "queue-worker",
		logger:       logger,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

func (w *queueWorker) Name() string {
	return w.name
}

func (w *queueWorker) Run(ctx context.Context) error {
	w.logger.Info("queue worker started", "interval", w.pollInterval.String())

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("context cancelled, worker exiting")
			return nil
		case <-w.stopCh:
			w.logger.Info("stop requested, worker exiting")
			return nil
		case <-ticker.C:
			w.logger.Info("processed queue batch", "items", 5)
		}
	}
}

func (w *queueWorker) Stop() error {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	w.logger.Info("queue worker stop hook completed")
	return nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	worker := newQueueWorker(logger, 2*time.Second)

	app, err := zcli.NewBuilder("en").
		WithName(worker.Name()).
		WithDisplayName("Queue Worker").
		WithDescription("Recommended ServiceRunner example for non-trivial services.").
		WithVersion("1.0.0").
		WithServiceRunner(worker).
		WithServiceTimeouts(15*time.Second, 20*time.Second).
		WithValidator(func(cfg *zcli.Config) error {
			if cfg.Basic().Name != worker.Name() {
				return fmt.Errorf("builder name must match ServiceRunner name")
			}
			return nil
		}).
		BuildWithError()
	if err != nil {
		logger.Error("build failed", "error", err)
		os.Exit(1)
	}

	if err := app.Execute(); err != nil {
		logger.Error("service command failed", "error", err)
		os.Exit(1)
	}
}
