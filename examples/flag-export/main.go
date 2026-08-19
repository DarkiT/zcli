package main

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/darkit/zcli"
)

type externalConfig struct {
	flagSets []*zcli.FlagSet
}

func bindFlagSets(flagSets ...*zcli.FlagSet) func(*externalConfig) {
	return func(cfg *externalConfig) {
		cfg.flagSets = flagSets
	}
}

func main() {
	app, err := zcli.NewBuilder("en").
		WithQuickConfig("bindable-flags", "Bindable Flags Demo", "Recommended example for exporting zcli flags to other packages.", "1.0.0").
		WithPersistentFlags(func(flags *zcli.FlagSet) {
			flags.String("config", "", "Path to the configuration file")
			flags.Bool("debug", false, "Enable debug logging")
			flags.String("region", "us-east-1", "Target region")
		}).
		WithFlags(func(flags *zcli.FlagSet) {
			flags.Int("port", 8080, "HTTP port")
			flags.Bool("dry-run", false, "Preview changes without applying them")
		}).
		WithCommandFactory(func(app *zcli.Cli) *zcli.Command {
			return newInspectCommand(app)
		}).
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(zcli.ExitUsage)
	}

	os.Exit(app.Run())
}

func newInspectCommand(app *zcli.Cli) *zcli.Command {
	return zcli.NewCommand(
		"inspect",
		"Show which flags can be exported to another package",
		zcli.WithCommandFlags(func(flags *zcli.FlagSet) {
			flags.String("output", "json", "Output format")
			flags.Bool("verbose", false, "Show verbose details")
		}),
		zcli.WithCommandRunE(func(cmd *zcli.Command, args []string) error {
			forwarded := app.ExportFlagsForViper("debug")
			external := &externalConfig{}
			bindFlagSets(forwarded...)(external)

			systemFlags := app.GetSystemFlags()
			sort.Strings(systemFlags)

			filteredNames := app.GetFilteredFlagNames("debug")
			sort.Strings(filteredNames)

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "system flags (%d): %v\n", len(systemFlags), systemFlags)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "bindable flag names: %v\n", filteredNames)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported flag sets: %d\n", len(external.flagSets))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "changed values that would be forwarded:")

			for _, flagSet := range external.flagSets {
				flagSet.VisitAll(func(flag *zcli.Flag) {
					if flag.Changed {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s = %s\n", flag.Name, flag.Value.String())
					}
				})
			}

			return nil
		}),
	)
}
