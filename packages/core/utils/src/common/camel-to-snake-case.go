package common

import (
	"regexp"
	"strings"
)

var camelBoundaryRe = regexp.MustCompile(`([a-z])([A-Z])`)

// CamelToSnakeCase mirrors camelToSnakeCase from camel-to-snake-case.ts.
func CamelToSnakeCase(s string) string {
	return strings.ToLower(camelBoundaryRe.ReplaceAllString(s, "${1}_${2}"))
}
