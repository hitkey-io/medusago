package common

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// knownEnvironments mirrors KNOWN_ENVIRONMENTS of load-env.ts.
var knownEnvironments = []string{"staging", "production", "test"}

// LoadEnv mirrors loadEnv(environment, envDir): it loads ".env.<environment>"
// (for staging, production and test) and ".env" from envDir into the process
// environment, like `expand(dotenv.config({ path: files, quiet: true }))`:
// variables already set are kept, the first file defining a key wins, and
// values are expanded with dotenv-expand semantics. Errors are ignored.
func LoadEnv(environment, envDir string) {
	files := []string{filepath.Join(envDir, ".env")}
	if slices.Contains(knownEnvironments, environment) {
		files = []string{filepath.Join(envDir, ".env."+environment), filepath.Join(envDir, ".env")}
	}

	parsed := newOrderedEnv()
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		file := ParseDotenv(string(raw))
		for _, k := range file.keys {
			// dotenv.populate without override: the first file wins.
			if _, ok := parsed.values[k]; !ok {
				parsed.set(k, file.values[k])
			}
		}
	}
	for _, k := range parsed.keys {
		if _, ok := os.LookupEnv(k); !ok {
			_ = os.Setenv(k, parsed.values[k])
		}
	}
	expandDotenv(parsed)
}

// ConfigDotenv mirrors dotenv.config({ path }) without expansion: the keys of
// the file that are not set yet are added to the process environment. The
// medusa CLI entry point (cli.js) does this with the ".env" of the working
// directory before running a command.
func ConfigDotenv(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	parsed := ParseDotenv(string(raw))
	for _, k := range parsed.keys {
		if _, ok := os.LookupEnv(k); !ok {
			_ = os.Setenv(k, parsed.values[k])
		}
	}
}

// DotenvValues keeps the key order of the parsed files (JavaScript object
// insertion order).
type DotenvValues struct {
	keys   []string
	values map[string]string
}

func newOrderedEnv() *DotenvValues { return &DotenvValues{values: map[string]string{}} }

func (e *DotenvValues) set(k, v string) {
	if _, ok := e.values[k]; !ok {
		e.keys = append(e.keys, k)
	}
	e.values[k] = v
}

// dotenvLine mirrors the LINE regular expression of dotenv's parser.
var dotenvLine = regexp.MustCompile(`(?m)(?:^|^)[ \t\f\v\r\n]*(?:export[ \t\f\v\r\n]+)?([\w.-]+)(?:[ \t\f\v\r\n]*=[ \t\f\v\r\n]*?|:[ \t\f\v\r\n]+?)([ \t\f\v\r\n]*'(?:\\'|[^'])*'|[ \t\f\v\r\n]*"(?:\\"|[^"])*"|[ \t\f\v\r\n]*` + "`(?:\\\\`|[^`])*`" + `|[^#\r\n]+)?[ \t\f\v\r\n]*(?:#.*)?(?:$|$)`)

// ParseDotenv mirrors dotenv.parse (the default parser).
func ParseDotenv(src string) *DotenvValues {
	out := newOrderedEnv()
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	for _, m := range dotenvLine.FindAllStringSubmatch(src, -1) {
		key := m[1]
		value := strings.TrimSpace(m[2])
		if len(value) >= 2 {
			q := value[0]
			if (q == '\'' || q == '"' || q == '`') && value[len(value)-1] == q {
				value = value[1 : len(value)-1]
				if q == '"' {
					value = strings.ReplaceAll(value, `\n`, "\n")
					value = strings.ReplaceAll(value, `\r`, "\r")
				}
			}
		}
		out.set(key, value)
	}
	return out
}

// Keys returns the parsed keys in file order.
func (e *DotenvValues) Keys() []string { return e.keys }

// Get returns a parsed value.
func (e *DotenvValues) Get(k string) (string, bool) {
	v, ok := e.values[k]
	return v, ok
}

// expandDotenv mirrors dotenv-expand 11 expand({ parsed }) over the process
// environment.
func expandDotenv(parsed *DotenvValues) {
	for _, key := range parsed.keys {
		value := parsed.values[key]
		if env, ok := os.LookupEnv(key); ok {
			if env == parsed.values[key] {
				value = interpolateEnv(value, parsed)
			} else {
				value = env
			}
		} else {
			value = interpolateEnv(value, parsed)
		}
		parsed.values[key] = strings.ReplaceAll(value, `\$`, "$")
	}
	for _, key := range parsed.keys {
		_ = os.Setenv(key, parsed.values[key])
	}
}

func isEnvKeyChar(c byte) bool {
	return c == '_' || c == '.' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// interpolateEnv mirrors interpolate() of dotenv-expand: it replaces every
// match of DOTENV_SUBSTITUTION_REGEX
// (\\)?(\$)(?!\()(\{?)([\w.]+)(?::?-(default))?(\}?) in one pass.
func interpolateEnv(value string, parsed *DotenvValues) string {
	var b strings.Builder
	i := 0
	for i < len(value) {
		start := i
		escaped := false
		if value[i] == '\\' && i+1 < len(value) && value[i+1] == '$' {
			escaped = true
			i++
		}
		if value[i] != '$' {
			b.WriteByte(value[start])
			i = start + 1
			continue
		}
		j := i + 1
		if j < len(value) && value[j] == '(' {
			b.WriteString(value[start : i+1])
			i++
			continue
		}
		if j < len(value) && value[j] == '{' {
			j++
		}
		keyStart := j
		for j < len(value) && isEnvKeyChar(value[j]) {
			j++
		}
		if j == keyStart {
			// No key: the regular expression does not match here.
			b.WriteString(value[start : i+1])
			i++
			continue
		}
		key := value[keyStart:j]
		defaultValue := ""
		hasDefault := false
		k := j
		if k < len(value) && value[k] == ':' {
			k++
		}
		if k < len(value) && value[k] == '-' {
			d := k + 1
			end := scanEnvDefault(value, d)
			if end > d {
				defaultValue = value[d:end]
				hasDefault = true
				j = end
			}
		}
		if j < len(value) && value[j] == '}' {
			j++
		}
		match := value[start:j]
		i = j
		if escaped {
			b.WriteString(match[1:])
			continue
		}
		b.WriteString(resolveEnvKey(key, value, defaultValue, hasDefault, parsed))
	}
	return b.String()
}

// scanEnvDefault returns the end of a default value: characters other than
// "}" and ${...} groups (nested up to three levels in the upstream pattern).
func scanEnvDefault(s string, i int) int {
	for i < len(s) {
		if s[i] == '}' {
			return i
		}
		if s[i] == '$' && i+1 < len(s) && s[i+1] == '{' {
			depth := 0
			j := i + 1
			for j < len(s) {
				if s[j] == '{' {
					depth++
				} else if s[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			if j >= len(s) {
				return len(s)
			}
			i = j + 1
			continue
		}
		i++
	}
	return i
}

func resolveEnvKey(key, value, defaultValue string, hasDefault bool, parsed *DotenvValues) string {
	if env := os.Getenv(key); env != "" {
		if p, ok := parsed.values[key]; ok && env == p {
			return env
		}
		return interpolateEnv(env, parsed)
	}
	if p := parsed.values[key]; p != "" {
		if p != value {
			return interpolateEnv(p, parsed)
		}
	}
	if hasDefault && defaultValue != "" {
		if strings.HasPrefix(defaultValue, "$") {
			return interpolateEnv(defaultValue, parsed)
		}
		return defaultValue
	}
	return ""
}
