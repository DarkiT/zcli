package zcli

import (
	"fmt"
	"os"
)

// App 是 Cli 的语义别名。
// 对外统一使用 App / Command / FlagSet 词汇时，仍保持与现有 Cli 完全相同的对象身份。
type App = Cli

// NewApp 创建一个新的应用实例。
// 它与 NewCli 使用同一套底层装配逻辑，仅提供更贴近上层语义的入口名称。
func NewApp(opts ...Option) *App {
	cfg := NewConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	app := newAppBase(cfg)
	app.applyBuilderAssembly(nil, nil)
	return app
}

func newAppBase(cfg *Config) *App {
	if cfg == nil {
		cfg = NewConfig()
	}

	syncCobraGlobals()

	if cfg.basic.Language != "" {
		if err := SetLanguage(cfg.basic.Language); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "set language failed: %v\n", err)
		}
	}

	app := &Cli{
		config: cfg,
		colors: newColors(),
		lang:   GetLanguageManager().GetPrimary(),
		command: &Command{
			Use:           cfg.basic.Name,
			SilenceErrors: cfg.basic.SilenceErrors,
			SilenceUsage:  cfg.basic.SilenceUsage,
		},
	}

	if app.config.basic.Description != "" {
		app.command.Short = app.config.basic.Description
	}

	app.setupVersion()
	return app
}
