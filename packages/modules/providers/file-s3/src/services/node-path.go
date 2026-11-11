package services

import "strings"

// Ports of Node's path.posix.normalize and path.posix.parse (the file keys
// are built with them).

// normalizeString mirrors normalizeString (path resolution without the
// leading slash).
func normalizeString(p string, allowAboveRoot bool) string {
	res := ""
	lastSegmentLength := 0
	lastSlash := -1
	dots := 0
	var code byte
	for i := 0; i <= len(p); i++ {
		if i < len(p) {
			code = p[i]
		} else if code == '/' {
			break
		} else {
			code = '/'
		}
		if code == '/' {
			if lastSlash == i-1 || dots == 1 {
				// NOOP
			} else if dots == 2 {
				if len(res) < 2 || lastSegmentLength != 2 || res[len(res)-1] != '.' || res[len(res)-2] != '.' {
					if len(res) > 2 {
						lastSlashIndex := strings.LastIndex(res, "/")
						if lastSlashIndex == -1 {
							res = ""
							lastSegmentLength = 0
						} else {
							res = res[:lastSlashIndex]
							lastSegmentLength = len(res) - 1 - strings.LastIndex(res, "/")
						}
						lastSlash = i
						dots = 0
						continue
					} else if len(res) != 0 {
						res = ""
						lastSegmentLength = 0
						lastSlash = i
						dots = 0
						continue
					}
				}
				if allowAboveRoot {
					if len(res) > 0 {
						res += "/.."
					} else {
						res = ".."
					}
					lastSegmentLength = 2
				}
			} else {
				if len(res) > 0 {
					res += "/" + p[lastSlash+1:i]
				} else {
					res = p[lastSlash+1 : i]
				}
				lastSegmentLength = i - lastSlash - 1
			}
			lastSlash = i
			dots = 0
		} else if code == '.' && dots != -1 {
			dots++
		} else {
			dots = -1
		}
	}
	return res
}

// posixNormalize mirrors path.posix.normalize.
func posixNormalize(p string) string {
	if p == "" {
		return "."
	}
	isAbsolute := p[0] == '/'
	trailingSeparator := p[len(p)-1] == '/'
	p = normalizeString(p, !isAbsolute)
	if p == "" {
		if isAbsolute {
			return "/"
		}
		if trailingSeparator {
			return "./"
		}
		return "."
	}
	if trailingSeparator {
		p += "/"
	}
	if isAbsolute {
		return "/" + p
	}
	return p
}

// parsedPath mirrors the result of path.posix.parse.
type parsedPath struct {
	Root, Dir, Base, Ext, Name string
}

// posixParse mirrors path.posix.parse.
func posixParse(p string) parsedPath {
	ret := parsedPath{}
	if p == "" {
		return ret
	}
	isAbsolute := p[0] == '/'
	start := 0
	if isAbsolute {
		ret.Root = "/"
		start = 1
	}
	startDot := -1
	startPart := 0
	end := -1
	matchedSlash := true
	i := len(p) - 1
	preDotState := 0
	for ; i >= start; i-- {
		code := p[i]
		if code == '/' {
			if !matchedSlash {
				startPart = i + 1
				break
			}
			continue
		}
		if end == -1 {
			matchedSlash = false
			end = i + 1
		}
		if code == '.' {
			if startDot == -1 {
				startDot = i
			} else if preDotState != 1 {
				preDotState = 1
			}
		} else if startDot != -1 {
			preDotState = -1
		}
	}
	if end != -1 {
		s := startPart
		if startPart == 0 && isAbsolute {
			s = 1
		}
		if startDot == -1 || preDotState == 0 || (preDotState == 1 && startDot == end-1 && startDot == startPart+1) {
			ret.Base = p[s:end]
			ret.Name = ret.Base
		} else {
			ret.Name = p[s:startDot]
			ret.Base = p[s:end]
			ret.Ext = p[startDot:end]
		}
	}
	if startPart > 0 {
		ret.Dir = p[:startPart-1]
	} else if isAbsolute {
		ret.Dir = "/"
	}
	return ret
}

// sanitizeFilePath mirrors sanitizeFilePath: posix slashes, no leading
// slashes, "." and ".." segments removed.
func sanitizeFilePath(filePath string) string {
	clean := strings.ReplaceAll(filePath, "\\", "/")
	clean = strings.TrimLeft(clean, "/")
	normalized := posixNormalize(clean)
	var out []string
	for _, seg := range strings.Split(normalized, "/") {
		if seg != ".." && seg != "." {
			out = append(out, seg)
		}
	}
	return strings.Join(out, "/")
}

// encodeURIComponent mirrors the JavaScript function.
func encodeURIComponent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 {
			c := byte(r)
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || strings.IndexByte("-_.!~*'()", c) >= 0 {
				b.WriteByte(c)
				continue
			}
		}
		for _, c := range []byte(string(r)) {
			b.WriteString("%")
			b.WriteString(strings.ToUpper(hexByte(c)))
		}
	}
	return b.String()
}

func hexByte(c byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[c>>4], digits[c&0xf]})
}
