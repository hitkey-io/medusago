package utils

import (
	"math"
	"regexp"
	"strings"
	"testing"
)

const codeClass = "[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]"

func stripCode(code, prefix string) string {
	return strings.ReplaceAll(code[len(prefix)+1:], "-", "")
}

func TestGenerateCodeFormat(t *testing.T) {
	code := GenerateCode("GIFT", 4)
	if !regexp.MustCompile(`^GIFT(-` + codeClass + `{4}){4}$`).MatchString(code) {
		t.Fatalf("unexpected code %q", code)
	}
}

func TestGenerateCodeWithoutPrefix(t *testing.T) {
	code := GenerateCode("", 2)
	if !regexp.MustCompile(`^` + codeClass + `{4}-` + codeClass + `{4}$`).MatchString(code) {
		t.Fatalf("unexpected code %q", code)
	}
}

func TestGenerateCodeDoesNotRepeatLeadingCharacters(t *testing.T) {
	for _, tc := range []struct {
		prefix   string
		sections int
	}{{"GIFT", 4}, {"SC", 4}, {"GIFT", 8}} {
		length := tc.sections * 4
		usedBytes := int(math.Ceil(float64(length*5) / 8))
		repeatedLength := length - usedBytes
		for i := 0; i < 100; i++ {
			chars := stripCode(GenerateCode(tc.prefix, tc.sections), tc.prefix)
			if len(chars) != length {
				t.Fatalf("length %d, want %d", len(chars), length)
			}
			if chars[usedBytes:] == chars[:repeatedLength] {
				t.Fatalf("%q repeats its leading characters", chars)
			}
		}
	}
}

func TestGenerateCodeOptDefaults(t *testing.T) {
	code := GenerateCodeOpt(nil, nil)
	if !regexp.MustCompile(`^GIFT(-` + codeClass + `{4}){4}$`).MatchString(code) {
		t.Fatalf("unexpected code %q", code)
	}
	code = GenerateCodeOpt("GC", float64(3))
	if !regexp.MustCompile(`^GC(-` + codeClass + `{4}){3}$`).MatchString(code) {
		t.Fatalf("unexpected code %q", code)
	}
}
