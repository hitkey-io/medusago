package common

import "strings"

// CompressName mirrors compressName from compress-name.ts (limit 58).
func CompressName(name string) string {
	return CompressNameWithLimit(name, 58)
}

// CompressNameWithLimit mirrors compressName with an explicit limit.
func CompressNameWithLimit(name string, limit int) string {
	if len(name) <= limit {
		return strings.ToLower(name)
	}

	hash := strings.ToLower(SimpleHash(name))
	hashLength := len(hash)

	// Remove consecutive duplicates
	parts := strings.Split(strings.ToLower(name), "_")
	deduplicated := make([]string, 0, len(parts))
	for i, p := range parts {
		if i == 0 || p != parts[i-1] {
			deduplicated = append(deduplicated, p)
		}
	}
	result := strings.Join(deduplicated, "_")

	// If still greater than the limit, truncate each part to 4 characters
	if len(result) > limit {
		all := strings.Split(CamelToSnakeCase(name), "_")
		for i, p := range all {
			if len(p) > 4 {
				all[i] = p[:4]
			}
		}
		result = strings.Join(all, "_")
	}

	name = result
	diff := len(name) + hashLength - limit
	if diff > 0 {
		return name[:len(name)-diff] + hash
	}
	return strings.ToLower(name + hash)
}
