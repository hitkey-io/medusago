package commands

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"
)

// DevelopOptions mirrors the arguments of `medusa develop`.
type DevelopOptions struct {
	Directory string
	// StartArgs are forwarded to the `start` command of the dev server
	// (host, port, ...).
	StartArgs []string
	// PollInterval is the file system polling interval (default 500ms).
	PollInterval time.Duration
}

// watchIgnored mirrors the chokidar `ignored` list of develop.ts: dot files,
// node_modules, dist, static, private, src/admin and .medusa.
func watchIgnored(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") && seg != "." {
			return true
		}
	}
	for _, p := range []string{"node_modules", "dist", "static", "private", "src/admin", ".medusa"} {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	// Directories named node_modules anywhere (chokidar matches the name).
	return slices.Contains(strings.Split(rel, "/"), "node_modules")
}

type fileStamp struct {
	mod  time.Time
	size int64
}

func snapshot(dir string) map[string]fileStamp {
	out := map[string]fileStamp{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if rel != "." && watchIgnored(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			out[rel] = fileStamp{mod: info.ModTime(), size: info.Size()}
		}
		return nil
	})
	return out
}

type fileChange struct {
	action string // created, modified, removed
	file   string
}

func diffSnapshots(before, after map[string]fileStamp) []fileChange {
	var changes []fileChange
	for f, s := range after {
		prev, ok := before[f]
		switch {
		case !ok:
			changes = append(changes, fileChange{"created", f})
		case prev != s:
			changes = append(changes, fileChange{"modified", f})
		}
	}
	for f := range before {
		if _, ok := after[f]; !ok {
			changes = append(changes, fileChange{"removed", f})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].file < changes[j].file })
	return changes
}

// devServer forks the `start` command like develop.ts. For a project that is
// a Go module (go.mod in the directory) the project binary is rebuilt with
// `go build` before every start, so changes to its Go sources are picked up;
// otherwise the running medusa binary itself is started again (configuration
// changes only).
type devServer struct {
	dir       string
	startArgs []string
	logger    *slog.Logger
	binary    string
	goModule  bool
	cmd       *exec.Cmd
	exited    chan struct{}
}

const logSource = "[Watcher]"

func (s *devServer) build() error {
	if !s.goModule {
		return nil
	}
	cmd := exec.Command("go", "build", "-o", s.binary, ".")
	cmd.Dir = s.dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *devServer) start() {
	if err := s.build(); err != nil {
		s.logger.Error(fmt.Sprintf("%s Dev server failed to start", logSource), "error", err)
		s.logger.Info(fmt.Sprintf("%s The server will restart automatically after your changes", logSource))
		return
	}
	cmd := exec.Command(s.binary, append([]string{"start"}, s.startArgs...)...)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), "NODE_ENV=development")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		s.logger.Error(fmt.Sprintf("%s Dev server failed to start", logSource), "error", err)
		s.logger.Info(fmt.Sprintf("%s The server will restart automatically after your changes", logSource))
		return
	}
	exited := make(chan struct{})
	s.cmd, s.exited = cmd, exited
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
}

func (s *devServer) stop() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = s.cmd.Process.Kill()
	} else {
		_ = s.cmd.Process.Signal(syscall.SIGINT)
	}
	select {
	case <-s.exited:
	case <-time.After(15 * time.Second):
		_ = s.cmd.Process.Kill()
		<-s.exited
	}
	s.cmd = nil
}

// Develop mirrors develop.ts: it starts the server with NODE_ENV=development
// and restarts it whenever a file of the project changes.
func Develop(opts DevelopOptions) error {
	logger := slog.Default()
	logger.Info(fmt.Sprintf("%s Using standard dev server (restart on file change)", logSource))

	dir := opts.Directory
	interval := opts.PollInterval
	if interval == 0 {
		interval = 500 * time.Millisecond
	}

	s := &devServer{dir: dir, startArgs: opts.StartArgs, logger: logger}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		s.goModule = true
		s.binary = filepath.Join(dir, ".medusa", "develop", "medusa")
		if runtime.GOOS == "windows" {
			s.binary += ".exe"
		}
	} else {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		s.binary = exe
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s.start()
	state := snapshot(dir)
	logger.Info(fmt.Sprintf("%s Watching filesystem to reload dev server on file change", logSource))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.stop()
			return nil
		case <-ticker.C:
			next := snapshot(dir)
			changes := diffSnapshots(state, next)
			state = next
			if len(changes) == 0 {
				continue
			}
			started := time.Now()
			for _, c := range changes {
				logger.Info(fmt.Sprintf("%s %s %s %s: restarting dev server", logSource, c.action, c.file, c.action))
			}
			s.stop()
			s.start()
			logger.Info(fmt.Sprintf("%s Reloaded in %.2fms", logSource, float64(time.Since(started).Microseconds())/1000))
		}
	}
}
