package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ProjectOptions mirrors the options of create-medusa-app.
type ProjectOptions struct {
	RepoURL           string
	Version           string
	Seed              bool
	SkipDB            bool
	DBURL             string
	Migrations        bool
	Browser           bool
	DirectoryPath     string
	WithNextjsStarter bool
	Verbose           bool
	// PackageManager of the storefront (npm, yarn or pnpm); the backend is
	// a Go module.
	PackageManager string
}

// projectCreator mirrors MedusaProjectCreator.
type projectCreator struct {
	projectName      string
	projectPath      string
	options          ProjectOptions
	db               *dbSetup
	nextjsDirectory  string
	inviteToken      string
	isProjectCreated bool
	printedMessage   bool
	spinner          *spinner
	ctx              context.Context
	cancel           context.CancelFunc
}

// newProjectCreator mirrors ProjectCreatorFactory.create.
func newProjectCreator(args []string, opts ProjectOptions) (*projectCreator, error) {
	if err := checkRequirements(opts); err != nil {
		return nil, err
	}
	name, err := getProjectName(args, opts.DirectoryPath)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &projectCreator{
		projectName: name,
		projectPath: filepath.Join(opts.DirectoryPath, name),
		options:     opts,
		spinner:     newSpinner(opts.Verbose),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (p *projectCreator) create() {
	p.setupProcessManager()
	if err := p.initializeProject(); err != nil {
		p.handleError(err)
		return
	}
	if err := p.setupProject(); err != nil {
		p.handleError(err)
		return
	}
	p.startServices()
}

func (p *projectCreator) initializeProject() error {
	installNextjs := p.options.WithNextjsStarter
	if !installNextjs {
		installNextjs = askForNextjsStarter()
	}
	if installNextjs {
		if err := checkNodeRequirements(); err != nil {
			return err
		}
	}
	if !p.options.SkipDB {
		if err := p.setupDatabase(); err != nil {
			return err
		}
	}

	logMessage("🚀 Starting project setup, this may take a few minutes.", "info")
	p.spinner.start("Setting up project...")

	if err := runCloneRepo(p.ctx, p.projectPath, p.options.RepoURL, p.options.Verbose); err != nil {
		p.spinner.stop()
		return fmt.Errorf("An error occurred while setting up your project: %v", err)
	}
	p.spinner.fact("Created project directory")

	storefront := getStorefrontDirectory(p.projectPath)
	if !installNextjs {
		_ = os.RemoveAll(storefront)
		return nil
	}
	p.nextjsDirectory = storefront
	return installNextjsStarter(storefront, p.spinner)
}

func (p *projectCreator) setupDatabase() error {
	dbName := "medusa-" + slugify(p.projectName)
	db, err := getDbClientAndCredentials(p.ctx, dbName, p.options.DBURL, p.options.Verbose)
	if err != nil {
		return err
	}
	p.db = db
	if p.options.DBURL == "" {
		p.spinner.fact("Creating database...")
		if err := runCreateDb(p.ctx, db); err != nil {
			p.spinner.stop()
			return fmt.Errorf("An error occurred while trying to create your database: %v", err)
		}
		p.spinner.fact(fmt.Sprintf("Database %s created", db.dbName))
	}
	return nil
}

func (p *projectCreator) setupProject() error {
	defer p.db.close()
	token, err := prepareProject(p.ctx, prepareOptions{
		projectName:     p.projectName,
		directory:       p.projectPath,
		db:              p.db,
		seed:            p.options.Seed,
		spinner:         p.spinner,
		skipDB:          p.options.SkipDB,
		migrations:      p.options.Migrations,
		nextjsDirectory: p.nextjsDirectory,
		verbose:         p.options.Verbose,
		version:         p.options.Version,
		packageManager:  p.options.PackageManager,
	})
	if err != nil {
		return err
	}
	p.inviteToken = token
	p.spinner.succeed("Project Prepared")
	return nil
}

func (p *projectCreator) startServices() {
	if p.options.SkipDB || !p.options.Browser {
		p.showSuccessMessage()
		os.Exit(0)
	}
	logMessage("Starting Medusa...", "info")
	startMedusa(p.ctx, getBackendDirectory(p.projectPath))
	if p.nextjsDirectory != "" {
		startNextjsStarter(p.ctx, p.nextjsDirectory, p.options.PackageManager)
	}
	p.isProjectCreated = true
	p.openBrowser()
	// the servers run until the user stops the installer
	<-p.ctx.Done()
	time.Sleep(500 * time.Millisecond)
}

func (p *projectCreator) openBrowser() {
	if err := waitOn(p.ctx, "http://localhost:9000/health"); err != nil {
		return
	}
	url := "http://localhost:9000/app"
	if p.inviteToken != "" {
		url = fmt.Sprintf("http://localhost:9000/app/invite?token=%s&first_run=true", p.inviteToken)
	}
	openURL(url)
}

func (p *projectCreator) handleError(err error) {
	if p.ctx.Err() != nil {
		os.Exit(0)
	}
	p.spinner.stop()
	logError(fmt.Sprintf("An error occurred: %v", err))
}

func (p *projectCreator) showSuccessMessage() {
	nl := "\n"
	command := "go run . develop"
	msg := fmt.Sprintf("Change to the `%s` directory to explore your Medusa project.%s%sStart your Medusa application again with the following command:%s%s%s%s",
		p.projectName, nl, nl, nl, nl, "cd apps/backend && "+command, nl+nl)
	if p.inviteToken != "" {
		msg += fmt.Sprintf("After you start the Medusa application, you can create an admin user with the URL http://localhost:9000/app/invite?token=%s&first_run=true%s", p.inviteToken, nl+nl)
	}
	if p.nextjsDirectory != "" {
		msg += fmt.Sprintf("The storefront is available in the `%s` directory. Change to that directory and start it with the following command:%s%s%s", p.nextjsDirectory, nl+nl, storefrontCommand(p.options.PackageManager, "dev"), nl+nl)
	}
	msg += "Check out the MedusaGo documentation to start your development:" + nl + nl + "https://github.com/hitkey-io/medusago" + nl + nl +
		"Star us on GitHub if you like what we're building:" + nl + nl + "https://github.com/hitkey-io/medusago/stargazers"
	fmt.Println(box(msg, ""))
}

// setupProcessManager mirrors setupProcessManager: on SIGINT/SIGTERM the
// servers stop and the success message is shown once.
func (p *projectCreator) setupProcessManager() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		p.spinner.stop()
		p.db.close()
		created := p.isProjectCreated
		p.cancel()
		if !p.printedMessage && created {
			p.printedMessage = true
			p.showSuccessMessage()
		}
		if !created {
			os.Exit(0)
		}
	}()
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '~':
			b.WriteRune(r)
			dash = false
		case r == '-' || r == ' ':
			if !dash {
				b.WriteRune('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
