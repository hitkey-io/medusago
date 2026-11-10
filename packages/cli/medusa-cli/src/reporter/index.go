// Package reporter mirrors @medusajs/cli/src/reporter: the logger of the
// Medusa CLI and server, as a slog handler.
//
// Like the winston logger upstream: in development (NODE_ENV starting with
// "dev", the default) records are printed with the cli format
// ("info:    message"), otherwise as JSON lines with a timestamp. LOG_LEVEL
// (npm levels, default "http") filters the records; LOG_FILE also appends the
// JSON records to a file.
package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// npmLevels mirrors winston.config.npm.levels.
var npmLevels = map[string]int{"error": 0, "warn": 1, "info": 2, "http": 3, "verbose": 4, "debug": 5, "silly": 6}

func levelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}

var levelColors = map[string]string{"error": "\x1b[31m", "warn": "\x1b[33m", "info": "\x1b[32m", "debug": "\x1b[34m"}

// Handler is the slog handler of the Medusa logger.
type Handler struct {
	mu       *sync.Mutex
	out      io.Writer
	file     io.Writer
	dev      bool
	color    bool
	maxLevel int
	attrs    []slog.Attr
}

// New returns the Medusa logger configured from the environment
// (NODE_ENV, LOG_LEVEL, LOG_FILE).
func New() *slog.Logger {
	nodeEnv := os.Getenv("NODE_ENV")
	if nodeEnv == "" {
		nodeEnv = "development"
	}
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "http"
	}
	max, ok := npmLevels[level]
	if !ok {
		max = npmLevels["http"]
	}
	h := &Handler{mu: &sync.Mutex{}, out: os.Stdout, dev: strings.HasPrefix(nodeEnv, "dev"), maxLevel: max}
	if info, err := os.Stdout.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		h.color = os.Getenv("NO_COLOR") == ""
	}
	if f := os.Getenv("LOG_FILE"); f != "" {
		if file, err := os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			h.file = file
		}
	}
	return slog.New(h)
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return npmLevels[levelName(l)] <= h.maxLevel
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &c
}

// WithGroup implements slog.Handler (groups are flattened).
func (h *Handler) WithGroup(string) slog.Handler { return h }

func attrValue(v slog.Value) any {
	v = v.Resolve()
	if v.Kind() == slog.KindAny {
		if err, ok := v.Any().(error); ok {
			return err.Error()
		}
	}
	return v.Any()
}

// Handle implements slog.Handler.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	name := levelName(r.Level)
	var attrs []slog.Attr
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	jsonLine := func() []byte {
		var b strings.Builder
		level, _ := json.Marshal(name)
		msg, _ := json.Marshal(r.Message)
		b.WriteString(`{"level":` + string(level) + `,"message":` + string(msg))
		for _, a := range attrs {
			k, _ := json.Marshal(a.Key)
			v, err := json.Marshal(attrValue(a.Value))
			if err != nil {
				v, _ = json.Marshal(a.Value.String())
			}
			b.WriteString("," + string(k) + ":" + string(v))
		}
		ts := r.Time
		if ts.IsZero() {
			ts = time.Now()
		}
		b.WriteString(`,"timestamp":"` + ts.Format("2006-01-02 15:04:05") + `"}` + "\n")
		return []byte(b.String())
	}

	var line []byte
	if h.dev {
		label := name + ":"
		if h.color {
			label = levelColors[name] + name + "\x1b[39m:"
		}
		var b strings.Builder
		b.WriteString(label + strings.Repeat(" ", len("verbose")-len(name)+1) + r.Message)
		for _, a := range attrs {
			fmt.Fprintf(&b, " %s=%v", a.Key, attrValue(a.Value))
		}
		b.WriteString("\n")
		line = []byte(b.String())
	} else {
		line = jsonLine()
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(line)
	if h.file != nil {
		_, _ = h.file.Write(jsonLine())
	}
	return err
}
