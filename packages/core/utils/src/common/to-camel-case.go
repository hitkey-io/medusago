package common

import (
	"regexp"
	"strings"
)

var (
	camelCaseRe      = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9]*)(([A-Z][a-z0-9]+)+)$`)
	camelSeparatorRe = regexp.MustCompile(`[^a-zA-Z0-9]+(.)`)
)

// ToCamelCase mirrors toCamelCase from to-camel-case.ts.
func ToCamelCase(str string) string {
	if camelCaseRe.MatchString(str) {
		return str
	}
	return camelSeparatorRe.ReplaceAllStringFunc(strings.ToLower(str), func(m string) string {
		sub := camelSeparatorRe.FindStringSubmatch(m)
		return strings.ToUpper(sub[1])
	})
}
