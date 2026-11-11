package common

import (
	"regexp"
	"strings"
)

var pascalWordRe = regexp.MustCompile(`(^\w|_\w)`)

// ToPascalCase mirrors toPascalCase from to-pascal-case.ts.
func ToPascalCase(s string) string {
	return pascalWordRe.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ToUpper(strings.ReplaceAll(m, "_", ""))
	})
}
