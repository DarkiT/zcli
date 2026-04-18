package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/darkit/zcli"
)

func main() {
	app, err := zcli.NewBuilder("en").
		WithName("notesctl").
		WithDisplayName("Notes CLI").
		WithDescription("Recommended CLI-only example built with zcli.").
		WithVersion("1.0.0").
		WithValidator(func(cfg *zcli.Config) error {
			if strings.Contains(cfg.Basic().Name, " ") {
				return fmt.Errorf("application name must not contain spaces")
			}
			return nil
		}).
		WithInitHook(func(cmd *zcli.Command, args []string) error {
			profile, err := cmd.Root().PersistentFlags().GetString("profile")
			if err != nil {
				return err
			}
			slog.Info("init hook completed", "profile", profile, "command", cmd.CommandPath(), "args", args)
			return nil
		}).
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(1)
	}

	app.Command().SilenceUsage = true
	app.PersistentFlags().String("profile", "dev", "Runtime profile to load")
	app.PersistentFlags().Bool("debug", false, "Enable debug mode")

	app.AddCommand(
		newStatusCommand(),
		newEchoCommand(),
	)

	if err := app.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func newStatusCommand() *zcli.Command {
	return &zcli.Command{
		Use:   "status",
		Short: "Show runtime configuration",
		RunE: func(cmd *zcli.Command, args []string) error {
			profile, err := cmd.Root().PersistentFlags().GetString("profile")
			if err != nil {
				return err
			}
			debug, err := cmd.Root().PersistentFlags().GetBool("debug")
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(
				cmd.OutOrStdout(),
				"profile: %s\ndebug: %t\ncommand: %s\nat: %s\n",
				profile,
				debug,
				cmd.CommandPath(),
				time.Now().Format(time.RFC3339),
			)
			return nil
		},
	}
}

func newEchoCommand() *zcli.Command {
	return &zcli.Command{
		Use:   "echo [message]",
		Short: "Echo a message",
		RunE: func(cmd *zcli.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("message is required")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(args, " "))
			return nil
		},
	}
}
