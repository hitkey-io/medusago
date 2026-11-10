package common

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EnvEditor mirrors EnvEditor of env-editor.ts: it edits the ".env" and
// ".env.template" files of an application.
type EnvEditor struct {
	appRoot string
	files   []envFile
}

type envFile struct {
	exists   bool
	contents []string
	filePath string
}

// NewEnvEditor mirrors new EnvEditor(appRoot).
func NewEnvEditor(appRoot string) *EnvEditor { return &EnvEditor{appRoot: appRoot} }

var envLineSplit = regexp.MustCompile(`\r?\n`)

// Load reads the .env and .env.template files (missing files are empty).
func (e *EnvEditor) Load() error {
	e.files = nil
	for _, name := range []string{".env", ".env.template"} {
		p := filepath.Join(e.appRoot, name)
		raw, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			e.files = append(e.files, envFile{filePath: p})
			continue
		}
		if err != nil {
			return err
		}
		e.files = append(e.files, envFile{exists: true, contents: envLineSplit.Split(string(raw), -1), filePath: p})
	}
	return nil
}

// Get returns the raw value of a key of the ".env" file, or false.
func (e *EnvEditor) Get(key string) (string, bool) {
	for _, f := range e.files {
		if !strings.HasSuffix(f.filePath, ".env") {
			continue
		}
		for _, line := range f.contents {
			if strings.HasPrefix(line, key+"=") {
				return strings.SplitN(line, "=", 2)[1], true
			}
		}
		return "", false
	}
	return "", false
}

// Set sets a key in both files. With withEmptyTemplateValue the key is only
// added (empty) to ".env.template" when it is missing there.
func (e *EnvEditor) Set(key, value string, withEmptyTemplateValue bool) {
	for i := range e.files {
		f := &e.files[i]
		idx := -1
		for j, line := range f.contents {
			if strings.HasPrefix(line, key+"=") {
				idx = j
				break
			}
		}
		if withEmptyTemplateValue && strings.HasSuffix(f.filePath, ".env.template") {
			if idx == -1 {
				f.contents = append(f.contents, key+"=")
			}
			continue
		}
		if idx == -1 {
			f.contents = append(f.contents, key+"="+value)
		} else {
			f.contents[idx] = key + "=" + value
		}
	}
}

// Save writes both files back (missing files are created, like upstream).
func (e *EnvEditor) Save() error {
	for _, f := range e.files {
		if err := os.WriteFile(f.filePath, []byte(strings.Join(f.contents, "\n")), 0o644); err != nil {
			return err
		}
	}
	return nil
}
