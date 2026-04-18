package main

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/darkit/zcli"
	"github.com/spf13/pflag"
)

type externalConfig struct {
	flagSets []*pflag.FlagSet
}

func bindFlagSets(flagSets ...*pflag.FlagSet) func(*externalConfig) {
	return func(cfg *externalConfig) {
		cfg.flagSets = flagSets
	}
}

func main() {
	app, err := zcli.NewBuilder("en").
		WithName("bindable-flags").
		WithDisplayName("Bindable Flags Demo").
		WithDescription("Recommended example for exporting zcli flags to other packages.").
		WithVersion("1.0.0").
		BuildWithError()
	if err != nil {
		slog.Error("build failed", "error", err)
		os.Exit(1)
	}

	inspectCmd := newInspectCommand(app)
	setupFlags(app, inspectCmd)
	app.AddCommand(inspectCmd)

	if err := app.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func newInspectCommand(app *zcli.Cli) *zcli.Command {
	return &zcli.Command{
		Use:   "inspect",
		Short: "Show which flags can be exported to another package",
		RunE: func(cmd *zcli.Command, args []string) error {
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
				flagSet.VisitAll(func(flag *pflag.Flag) {
					if flag.Changed {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s = %s\n", flag.Name, flag.Value.String())
					}
				})
			}

			return nil
		},
	}
}

func setupFlags(app *zcli.Cli, inspectCmd *zcli.Command) {
	app.PersistentFlags().String("config", "", "Path to the configuration file")
	app.PersistentFlags().Bool("debug", false, "Enable debug logging")
	app.PersistentFlags().String("region", "us-east-1", "Target region")

	app.Flags().Int("port", 8080, "HTTP port")
	app.Flags().Bool("dry-run", false, "Preview changes without applying them")

	inspectCmd.Flags().String("output", "json", "Output format")
	inspectCmd.Flags().Bool("verbose", false, "Show verbose details")
}
