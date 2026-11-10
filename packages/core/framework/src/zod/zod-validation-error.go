package zod

import (
	"fmt"
	"regexp"
	"strings"
)

// Port of zod-validation-error 5.0.0 (v4) fromZodIssue with the default
// options: "Validation error: <Message> at \"<path>\"".

var identifierRe = regexp.MustCompile(`[$_\p{L}\p{Nl}][$\x{200c}\x{200d}\p{L}\p{Nl}\p{Mn}\p{Mc}\p{Nd}\p{Pc}]*`)

func joinPath(path []any) string {
	if len(path) == 1 {
		s := fmt.Sprint(path[0])
		if s == "" {
			return `""`
		}
		return s
	}
	var acc strings.Builder
	for _, key := range path {
		switch k := key.(type) {
		case int:
			fmt.Fprintf(&acc, "[%d]", k)
			continue
		case float64:
			fmt.Fprintf(&acc, "[%v]", k)
			continue
		}
		s := fmt.Sprint(key)
		if strings.Contains(s, `"`) {
			acc.WriteString(`["` + strings.ReplaceAll(s, `"`, `\"`) + `"]`)
			continue
		}
		if !identifierRe.MatchString(s) {
			acc.WriteString(`["` + s + `"]`)
			continue
		}
		if acc.Len() > 0 {
			acc.WriteString(".")
		}
		acc.WriteString(s)
	}
	return acc.String()
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func mapIssue(issue *Issue) string {
	if issue.Code == "invalid_union" && len(issue.Errors) > 0 {
		var msgs []string
		seen := map[string]bool{}
		for _, issues := range issue.Errors {
			var parts []string
			for _, sub := range issues {
				c := *sub
				c.Path = append(append([]any{}, issue.Path...), sub.Path...)
				parts = append(parts, mapIssue(&c))
			}
			m := strings.Join(parts, "; ")
			if !seen[m] {
				seen[m] = true
				msgs = append(msgs, m)
			}
		}
		return strings.Join(msgs, " or ")
	}
	out := titleCase(issue.Message)
	if len(issue.Path) > 0 {
		if len(issue.Path) == 1 {
			if n, ok := issue.Path[0].(int); ok {
				return out + fmt.Sprintf(" at index %d", n)
			}
		}
		out += fmt.Sprintf(` at "%s"`, joinPath(issue.Path))
	}
	return out
}

// FromZodIssue mirrors fromZodIssue(issue).toString().
func FromZodIssue(issue *Issue) string {
	m := mapIssue(issue)
	if m == "" {
		return "Validation error"
	}
	return "Validation error: " + m
}
