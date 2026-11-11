package common

import (
	"regexp"
	"strings"
)

var (
	kebabBoundaryRe  = regexp.MustCompile(`([a-z])([A-Z])`)
	kebabSeparatorRe = regexp.MustCompile(`[\s\v_]+`)
)

// KebabCase mirrors kebabCase from to-kebab-case.ts.
func KebabCase(s string) string {
	s = kebabBoundaryRe.ReplaceAllString(s, "${1}-${2}")
	s = kebabSeparatorRe.ReplaceAllString(s, "-")
	return strings.ToLower(s)
}
