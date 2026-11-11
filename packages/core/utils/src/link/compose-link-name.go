// Package link mirrors @medusajs/utils/src/link.
package link

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

var linkNameSuffixRe = regexp.MustCompile(`(?i)(_id|Service)$`)

// ComposeLinkName mirrors composeLinkName from compose-link-name.ts.
func ComposeLinkName(names ...string) string {
	return common.UpperCaseFirst(common.ToPascalCase(ComposeTableName(append(names, "link")...)))
}

// ComposeTableName mirrors composeTableName from compose-link-name.ts.
func ComposeTableName(names ...string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = linkNameSuffixRe.ReplaceAllString(n, "")
	}
	return strings.Join(out, "_")
}
