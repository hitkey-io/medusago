package common

import (
	"strings"
	"unicode/utf8"
)

// UpperCaseFirst mirrors upperCaseFirst from upper-case-first.ts.
func UpperCaseFirst(str string) string {
	r, size := utf8.DecodeRuneInString(str)
	if size == 0 {
		return str
	}
	return strings.ToUpper(string(r)) + str[size:]
}

// LowerCaseFirst mirrors lowerCaseFirst from lower-case-first.ts.
func LowerCaseFirst(str string) string {
	r, size := utf8.DecodeRuneInString(str)
	if size == 0 {
		return str
	}
	return strings.ToLower(string(r)) + str[size:]
}
