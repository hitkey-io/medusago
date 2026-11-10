package utils

import (
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

var nonTableNameChars = regexp.MustCompile(`[^a-z0-9_]`)

// NormalizeTableName mirrors normalizeTableName.
func NormalizeTableName(name string) string {
	return nonTableNameChars.ReplaceAllString(common.CompressNameWithLimit(strings.ToLower(name), 58), "_")
}

// GetPivotTableName mirrors getPivotTableName.
func GetPivotTableName(tableName string) string {
	return "cat_pivot_" + NormalizeTableName(tableName)
}
