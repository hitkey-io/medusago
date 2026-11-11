// Package migrationscripts mirrors @medusajs/utils/src/modules-sdk/
// migration-scripts: the scripts the module migration commands run for
// modules defined with DML models.
package migrationscripts

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// GenerateMigrationScript is a script built by BuildGenerateMigrationScript.
type GenerateMigrationScript func(logger *slog.Logger)

// uniqueTimestampDelay is the pause after a generated migration (overridden
// by tests).
var uniqueTimestampDelay = time.Second

// SetUniqueTimestampDelay changes the pause after a generated migration and
// returns the previous value (for tests).
func SetUniqueTimestampDelay(d time.Duration) time.Duration {
	prev := uniqueTimestampDelay
	uniqueTimestampDelay = d
	return prev
}

// terminalLine mirrors new Array(process.stdout.columns).join("-").
func terminalLine() string {
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return ""
	}
	columns := 80
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 1 {
		columns = c
	}
	return strings.Repeat("-", columns-1)
}

// BuildGenerateMigrationScript mirrors buildGenerateMigrationScript: the
// script generating the migration of the module models into
// pathToMigrations, with a snapshot named after the module. Errors are
// logged, not returned.
func BuildGenerateMigrationScript(moduleName string, models []*dml.Entity, pathToMigrations string) GenerateMigrationScript {
	return func(logger *slog.Logger) {
		if logger == nil {
			logger = slog.Default()
		}
		logger.Info(terminalLine())
		logger.Info("")
		logger.Info(fmt.Sprintf("MODULE: %s", moduleName))

		result, err := migrations.Generate(migrations.GenerateOptions{
			SnapshotName: ".snapshot-" + common.KebabCase(strings.Replace(moduleName, "Service", "", 1)),
			Path:         pathToMigrations,
			Models:       models,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Failed with error %s", err.Error()))
			return
		}
		if result.FileName != "" {
			logger.Info(fmt.Sprintf("Generated successfully (%s).", result.FileName))
			// MikroORM timestamps have a one second precision: the pause
			// keeps the names of the migrations of several modules unique.
			time.Sleep(uniqueTimestampDelay)
			return
		}
		logger.Info("Skipped. No changes detected in your models.")
	}
}
