package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// Rollback mirrors `medusa db:rollback [modules...]`: for each module, the
// last executed migration is reverted (MikroORM migrator.down() without
// options), with the logs of buildRevertMigrationScript.
func Rollback(ctx context.Context, cfg config.ConfigModule, moduleNames []string) error {
	logger := slog.Default()
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := EnsureDbExists(ctx, pool, logger); err != nil {
		return err
	}

	logger.Info("Reverting migrations...")

	enabled := map[string]config.ModuleConfig{}
	var all []string
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		enabled[mc.Key] = mc
		all = append(all, mc.Key)
	}
	var missing []string
	for _, name := range moduleNames {
		if _, ok := enabled[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		terminalSeparator()
		logger.Error(fmt.Sprintf("Cannot revert migrations for unknown module(s) %s", strings.Join(missing, ",")))
		list := make([]string, len(all))
		for i, m := range all {
			list[i] = "          - " + m
		}
		logger.Error("Available modules:\n" + strings.Join(list, "\n"))
		return fmt.Errorf("unknown modules")
	}

	var failures []string
	for _, name := range moduleNames {
		mc := enabled[name]
		terminalSeparator()
		logger.Info("")
		logger.Info(fmt.Sprintf("MODULE: %s", name))
		exp, ok := modules.Registry[mc.Resolve]
		if !ok {
			logger.Info("Skipped. Nothing to revert")
			continue
		}
		reverted, err := migrations.Revert(ctx, pool, name, exp.Migrations, 1)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed with error %s", err.Error()))
			failures = append(failures, err.Error())
			continue
		}
		for _, m := range reverted {
			logger.Info(fmt.Sprintf("  ● Reverting %s", m))
			logger.Info(fmt.Sprintf("  ✔ Reverted %s", m))
		}
		if len(reverted) > 0 {
			logger.Info("Reverted successfully")
		} else {
			logger.Info("Skipped. Nothing to revert")
		}
	}
	if len(failures) > 0 {
		terminalSeparator()
		return fmt.Errorf("Some migrations failed to revert: %s", strings.Join(failures, ", "))
	}
	terminalSeparator()
	logger.Info("Migrations reverted")
	return nil
}
