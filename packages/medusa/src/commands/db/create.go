package db

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/pg"
)

// CreateOptions mirrors the arguments of dbCreate.
type CreateOptions struct {
	// DB is the --db flag (empty: DB_NAME of .env or medusa-<directory>).
	DB          string
	Directory   string
	Interactive bool
	Logger      *slog.Logger
}

var (
	slugifyRemove     = regexp.MustCompile(`[^\w\s$*_+~.()'"!\-:@]+`)
	slugifySeparators = regexp.MustCompile(`[-\s]+`)
)

// slugify mirrors slugify(value) with the default options (no locale
// character map: non ASCII characters are removed).
func slugify(value string) string {
	var b strings.Builder
	for _, r := range value {
		ch := string(r)
		if ch == "-" {
			ch = " "
		}
		b.WriteString(slugifyRemove.ReplaceAllString(ch, ""))
	}
	return slugifySeparators.ReplaceAllString(strings.TrimSpace(b.String()), "-")
}

// stdinIsTerminal reports whether prompts can be shown.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// promptInput mirrors @inquirer/input with a default value.
func promptInput(message, defaultValue string) string {
	fmt.Fprintf(os.Stdout, "? %s (%s) ", message, defaultValue)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue
	}
	return line
}

// Create mirrors dbCreate: it creates the database named by --db, DB_NAME or
// medusa-<directory name> on the PostgreSQL server of DATABASE_URL, and
// records DB_NAME in the .env file. It never exits the process.
func Create(ctx context.Context, opts CreateOptions) bool {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	dbName := opts.DB

	envEditor := common.NewEnvEditor(opts.Directory)
	if err := envEditor.Load(); err != nil {
		logger.Error(err.Error())
		return false
	}

	// Upstream reads DATABASE_URL from the .env file only; the process
	// environment is accepted as well.
	dbConnectionString, ok := envEditor.Get("DATABASE_URL")
	if !ok || dbConnectionString == "" {
		dbConnectionString = os.Getenv("DATABASE_URL")
	}
	if dbConnectionString == "" {
		logger.Error(`Missing "DATABASE_URL" inside the .env file. The value is required to connect to the PostgreSQL server`)
		return false
	}

	if dbName == "" {
		defaultValue, ok := envEditor.Get("DB_NAME")
		if !ok {
			defaultValue = "medusa-" + slugify(filepath.Base(opts.Directory))
		}
		if opts.Interactive && stdinIsTerminal() {
			dbName = promptInput("Enter the database name", defaultValue)
		} else {
			dbName = defaultValue
		}
	}

	connConfig, err := pg.ParseConnectionString(dbConnectionString)
	if err != nil {
		logger.Error("Unable to establish database connection because of the following error")
		logger.Error(err.Error())
		return false
	}

	// Connect without a database name (the server default, the user name),
	// then fall back to the "postgres" database.
	withoutDB := connConfig.Copy()
	withoutDB.Database = ""
	client, err := pg.CreateClient(ctx, withoutDB)
	if err != nil {
		withPostgres := connConfig.Copy()
		withPostgres.Database = "postgres"
		client, err = pg.CreateClient(ctx, withPostgres)
	}
	if err != nil {
		logger.Error("Unable to establish database connection because of the following error")
		logger.Error(err.Error())
		return false
	}
	defer client.Close(ctx)

	logger.Info(fmt.Sprintf(`Connection established with the database "%s"`, dbName))
	exists, err := pg.DbExists(ctx, client, dbName)
	if err != nil {
		logger.Error(err.Error())
		return false
	}
	if exists {
		logger.Info(fmt.Sprintf(`Database "%s" already exists`, dbName))
		envEditor.Set("DB_NAME", dbName, true)
		if err := envEditor.Save(); err != nil {
			logger.Error(err.Error())
			return false
		}
		logger.Info(fmt.Sprintf(`Updated .env file with "DB_NAME=%s"`, dbName))
		return true
	}

	if err := pg.CreateDb(ctx, client, dbName); err != nil {
		logger.Error(err.Error())
		return false
	}
	logger.Info(fmt.Sprintf(`Created database "%s"`, dbName))

	envEditor.Set("DB_NAME", dbName, false)
	if err := envEditor.Save(); err != nil {
		logger.Error(err.Error())
		return false
	}
	logger.Info(fmt.Sprintf(`Updated .env file with "DB_NAME=%s"`, dbName))
	return true
}
