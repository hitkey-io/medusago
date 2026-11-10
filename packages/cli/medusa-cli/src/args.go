package cli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type optionKind int

const (
	boolOption optionKind = iota
	stringOption
	numberOption
)

// option is a yargs option definition.
type option struct {
	name     string
	aliases  []string
	kind     optionKind
	def      any
	describe string
}

// parsedArgs is the yargs argv of a command.
type parsedArgs struct {
	values     map[string]any
	set        map[string]bool
	positional []string
}

func (a parsedArgs) bool(name string) bool {
	v, _ := a.values[name].(bool)
	return v
}

func (a parsedArgs) string(name string) string {
	switch v := a.values[name].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func (a parsedArgs) number(name string) (float64, bool) {
	switch v := a.values[name].(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

// kebab converts camelCase option names to kebab-case (yargs accepts both).
func kebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('-')
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// globalOptions mirrors the global options of create-cli.ts.
var globalOptions = []option{
	{name: "verbose", kind: boolOption, def: false, describe: "Turn on verbose output"},
	{name: "no-color", aliases: []string{"no-colors"}, kind: boolOption, def: false, describe: "Turn off the color in output"},
	{name: "json", kind: boolOption, def: false, describe: "Turn on the JSON logger"},
	{name: "help", aliases: []string{"h"}, kind: boolOption, def: false, describe: "Show help"},
}

// parseArgs parses args like yargs in strict mode: --flag, --flag=value,
// --no-flag for booleans, -x value, and positionals. Arguments after "--"
// and, when restAfter > 0, everything after the first restAfter positionals
// are kept as positionals.
func parseArgs(args []string, opts []option, restAfter int) (parsedArgs, error) {
	all := append(append([]option{}, globalOptions...), opts...)
	lookup := map[string]*option{}
	for i := range all {
		o := &all[i]
		lookup[o.name] = o
		for _, a := range o.aliases {
			lookup[a] = o
		}
	}
	res := parsedArgs{values: map[string]any{}, set: map[string]bool{}}
	for _, o := range all {
		if o.def != nil {
			res.values[o.name] = o.def
		}
	}
	var unknown []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			res.positional = append(res.positional, args[i+1:]...)
			break
		}
		if restAfter > 0 && len(res.positional) >= restAfter {
			res.positional = append(res.positional, args[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			res.positional = append(res.positional, arg)
			continue
		}
		name := strings.TrimLeft(arg, "-")
		value, hasValue := "", false
		if k, v, ok := strings.Cut(name, "="); ok {
			name, value, hasValue = k, v, true
		}
		name = kebab(name)
		o, ok := lookup[name]
		negated := false
		if !ok && strings.HasPrefix(name, "no-") {
			if no, found := lookup[strings.TrimPrefix(name, "no-")]; found && no.kind == boolOption {
				o, ok, negated = no, true, true
			}
		}
		if !ok {
			unknown = append(unknown, name)
			if !hasValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		res.set[o.name] = true
		switch o.kind {
		case boolOption:
			v := !negated
			if hasValue {
				v = value != "false"
			} else if i+1 < len(args) && (args[i+1] == "true" || args[i+1] == "false") {
				v = args[i+1] == "true"
				i++
			}
			if negated && o.name == "no-color" {
				v = true
			}
			res.values[o.name] = v
		default:
			if !hasValue {
				if i+1 >= len(args) {
					return res, fmt.Errorf("Not enough arguments following: %s", name)
				}
				i++
				value = args[i]
			}
			if o.kind == numberOption {
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					res.values[o.name] = value
					continue
				}
				res.values[o.name] = f
			} else {
				res.values[o.name] = value
			}
		}
	}
	// yargs exposes every option under its name and its aliases
	for _, o := range all {
		v, ok := res.values[o.name]
		if !ok {
			continue
		}
		for _, a := range o.aliases {
			res.values[a] = v
			if res.set[o.name] {
				res.set[a] = true
			}
		}
	}
	if len(unknown) > 0 {
		label := "argument"
		if len(unknown) > 1 {
			label = "arguments"
		}
		return res, fmt.Errorf("Unknown %s: %s", label, strings.Join(unknown, ", "))
	}
	return res, nil
}
