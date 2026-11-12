package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	storeCors  = "http://localhost:8000"
	adminCors  = "http://localhost:5173,http://localhost:9000"
	adminEmail = "admin@medusa-test.com"
	docsCors   = "https://docs.medusajs.com"
	redisURL   = "redis://localhost:6379"
)

var authCors = strings.Join([]string{adminCors, storeCors, docsCors}, ",")

type prepareOptions struct {
	projectName     string
	directory       string
	db              *dbSetup
	seed            bool
	spinner         *spinner
	skipDB          bool
	migrations      bool
	nextjsDirectory string
	verbose         bool
	version         string
	packageManager  string
}

var inviteTokenRe = regexp.MustCompile(`Invite token: (?P<token>\S+)`)

// prepareProject mirrors prepare-project.ts for a Go backend: the .env of the
// backend, the module dependencies (go mod download), the migrations and the
// admin invite; the storefront gets the publishable API key.
func prepareProject(ctx context.Context, o prepareOptions) (string, error) {
	backend := getBackendDirectory(o.directory)

	onboarding := "default"
	if o.nextjsDirectory != "" {
		onboarding = "nextjs"
	}
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	env := fmt.Sprintf("MEDUSA_ADMIN_ONBOARDING_TYPE=%s\nSTORE_CORS=%s\nADMIN_CORS=%s\nAUTH_CORS=%s\nREDIS_URL=%s\nJWT_SECRET=supersecret\nCOOKIE_SECRET=supersecret\nAUTH_MFA_ENCRYPTION_KEY=%s",
		onboarding, storeCors, adminCors, authCors, redisURL, hex.EncodeToString(key))
	dbURL := ""
	if !o.skipDB && o.db != nil {
		dbURL = o.db.connectionString
		if o.db.dbName != "" {
			env += "\nDB_NAME=" + o.db.dbName
			dbURL = strings.Replace(dbURL, o.db.dbName, "$DB_NAME", 1)
		}
		env += "\nDATABASE_URL=" + dbURL
	}
	if o.nextjsDirectory != "" {
		env += "\nMEDUSA_ADMIN_ONBOARDING_NEXTJS_DIRECTORY=" + o.nextjsDirectory
	}
	f, err := os.OpenFile(filepath.Join(backend, ".env"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(env)
	f.Close()
	if err != nil {
		return "", err
	}

	o.spinner.title("Installing dependencies...")
	if o.version != "" {
		if _, err := execute(ctx, backend, nil, o.verbose, "go", "get", "github.com/hitkey-io/medusago@"+o.version); err != nil {
			return "", err
		}
	}
	if _, err := execute(ctx, backend, nil, o.verbose, "go", "mod", "download"); err != nil {
		return "", err
	}
	if o.nextjsDirectory != "" {
		if _, err := execute(ctx, o.nextjsDirectory, nil, o.verbose, storefrontPM(o.packageManager), "install"); err != nil {
			return "", err
		}
	}
	// one build of the backend runs the medusa commands below
	bin, err := buildBackend(ctx, backend, o.verbose)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(bin))
	o.spinner.fact("Installed Dependencies")

	inviteToken := ""
	if !o.skipDB && o.migrations {
		o.spinner.title("Running Migrations...")
		out, migrateErr := execute(ctx, backend, nil, o.verbose, bin, "db:migrate")
		if o.db != nil && o.db.conn != nil {
			var n int
			qerr := o.db.conn.QueryRow(ctx, `SELECT count(tablename) from pg_tables WHERE tablename = 'mikro_orm_migrations'`).Scan(&n)
			if (qerr != nil || n == 0) && (migrateErr != nil || out != "") {
				return "", fmt.Errorf("An error occurred while running migrations: %s", strings.TrimSpace(out))
			}
		} else if migrateErr != nil {
			return "", fmt.Errorf("An error occurred while running migrations: %s", strings.TrimSpace(out))
		}
		o.spinner.fact("Ran Migrations")

		// the seed option is declared but not applied upstream (v2.22.6)
		out, _ = execute(ctx, backend, nil, o.verbose, bin, "user", "-e", adminEmail, "--invite")
		if m := inviteTokenRe.FindStringSubmatch(out); m != nil {
			inviteToken = m[1]
		}
	}

	if o.nextjsDirectory != "" && o.db != nil && o.db.conn != nil {
		var token string
		if err := o.db.conn.QueryRow(ctx, `SELECT token FROM "api_key" WHERE type = 'publishable'`).Scan(&token); err == nil {
			envPath := filepath.Join(o.nextjsDirectory, ".env.local")
			if _, err := os.Stat(envPath); err != nil {
				envPath = filepath.Join(o.nextjsDirectory, ".env.template")
			}
			if content, err := os.ReadFile(envPath); err == nil {
				updated := strings.Replace(string(content), "NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=", "NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY="+token, 1)
				_ = os.WriteFile(envPath, []byte(updated), 0o644)
			}
		}
	}
	o.spinner.fact("Finished Preparation")
	return inviteToken, nil
}

// buildBackend compiles the backend once into a temporary directory.
func buildBackend(ctx context.Context, backend string, verbose bool) (string, error) {
	dir, err := os.MkdirTemp("", "create-medusago-app-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "medusa")
	if _, err := execute(ctx, backend, nil, verbose, "go", "build", "-o", bin, "."); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return bin, nil
}
