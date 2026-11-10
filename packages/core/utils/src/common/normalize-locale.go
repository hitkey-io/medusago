package common

import (
	"regexp"
	"strings"
)

var localeScriptSubtag = regexp.MustCompile(`^[a-zA-Z]{4}$`)

func localeUpperCaseFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// NormalizeLocale mirrors normalizeLocale: a BCP 47 language tag with a
// lower cased language, a title cased script and an upper cased region.
func NormalizeLocale(locale string) string {
	segments := strings.Split(locale, "-")
	switch len(segments) {
	case 1:
		return strings.ToLower(segments[0])
	case 2:
		if localeScriptSubtag.MatchString(segments[1]) {
			return strings.ToLower(segments[0]) + "-" + localeUpperCaseFirst(strings.ToLower(segments[1]))
		}
		return strings.ToLower(segments[0]) + "-" + strings.ToUpper(segments[1])
	case 3:
		return strings.ToLower(segments[0]) + "-" + localeUpperCaseFirst(strings.ToLower(segments[1])) + "-" + strings.ToUpper(segments[2])
	}
	return locale
}

// ArrayDifference mirrors arrayDifference: the distinct elements of main
// missing from differing, in order.
func ArrayDifference(main, differing []string) []string {
	exclude := map[string]bool{}
	for _, d := range differing {
		exclude[d] = true
	}
	seen := map[string]bool{}
	out := []string{}
	for _, m := range main {
		if seen[m] {
			continue
		}
		seen[m] = true
		if !exclude[m] {
			out = append(out, m)
		}
	}
	return out
}
