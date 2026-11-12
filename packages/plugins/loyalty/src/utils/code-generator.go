// Package utils mirrors src/utils of @medusajs/loyalty-plugin.
package utils

import (
	"crypto/rand"
	"strings"
)

// codeCharacters are the characters of a code (similar-looking characters
// such as 0/O and 1/I excluded).
const codeCharacters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateCode mirrors generateCode(prefix, sections): sections groups of 4
// random characters joined by "-", after the prefix unless it is empty.
func GenerateCode(prefix string, sections int) string {
	length := sections * 4
	if length < 0 {
		length = 0
	}
	// one random byte per character: 256 is a multiple of the character set
	// size (32), so the modulo introduces no bias
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(err)
	}
	var code strings.Builder
	for _, b := range randomBytes {
		code.WriteByte(codeCharacters[int(b)%len(codeCharacters)])
	}
	s := code.String()
	var chunks []string
	for i := 0; i < len(s); i += 4 {
		end := i + 4
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	if prefix != "" {
		return prefix + "-" + strings.Join(chunks, "-")
	}
	return strings.Join(chunks, "-")
}

// GenerateCodeOpt is GenerateCode with the default parameters of
// generateCode applied to undefined values (nil): prefix "GIFT", 4
// sections. It takes values read from the plugin options as they are.
func GenerateCodeOpt(prefix, sections any) string {
	p := "GIFT"
	if prefix != nil {
		if s, ok := prefix.(string); ok {
			p = s
		}
	}
	n := 4
	switch v := sections.(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		n = int(v)
	}
	return GenerateCode(p, n)
}
