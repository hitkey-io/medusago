package common

// UnsafePathKeys mirrors UNSAFE_PATH_KEYS: the object keys that let a dotted
// path climb to Object.prototype (`__proto__`, or `constructor` ->
// `prototype`). Walks that build nested objects from user-controlled path
// segments skip them, like upstream.
var UnsafePathKeys = map[string]bool{
	"__proto__":   true,
	"constructor": true,
	"prototype":   true,
}

// IsUnsafePathSegment mirrors isUnsafePathSegment.
func IsUnsafePathSegment(segment string) bool {
	return UnsafePathKeys[segment]
}

// HasUnsafePathSegment reports whether any segment is unsafe
// (segments.some(isUnsafePathSegment)).
func HasUnsafePathSegment(segments []string) bool {
	for _, s := range segments {
		if UnsafePathKeys[s] {
			return true
		}
	}
	return false
}
