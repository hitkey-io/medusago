// Package link mirrors @medusajs/utils/src/link.
package link

import (
	"regexp"
	"strings"
)

var linkNameSuffixRe = regexp.MustCompile(`(?i)(_id|Service)$`)

// ComposeTableName mirrors composeTableName from compose-link-name.ts.
func ComposeTableName(names ...string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = linkNameSuffixRe.ReplaceAllString(n, "")
	}
	return strings.Join(out, "_")
}
