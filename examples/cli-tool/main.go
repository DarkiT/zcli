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
		WithQuickConfig("notesctl", "Notes CLI", "Recommended CLI-only example built with zcli.", "1.0.0").
		WithSilenceUsage(true).
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
		WithPersistentFlags(func(flags *zcli.FlagSet) {
			flags.String("profile", "dev", "Runtime profile to load")
			flags.Bool("debug", false, "Enable debug mode")
		}).
		WithCommands(
			newStatusCommand(),
			newEchoCommand(),
		).
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(zcli.ExitUsage)
	}

	os.Exit(app.Run())
}

func newStatusCommand() *zcli.Command {
	return zcli.NewCommand(
		"status",
		"Show runtime configuration",
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
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
		}),
	)
}

func newEchoCommand() *zcli.Command {
	return zcli.NewCommand(
		"echo [message]",
		"Echo a message",
		zcli.WithCommandSilenceUsage(true),
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("message is required")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(args, " "))
			return nil
		}),
	)
}
