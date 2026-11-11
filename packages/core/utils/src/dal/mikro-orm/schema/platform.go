// Package schema ports the parts of MikroORM 6 (@mikro-orm/knex schema,
// @mikro-orm/postgresql) and knex 3 (PostgreSQL schema compiler) that
// `medusa db:generate` runs upstream to turn the module models into a
// migration: DatabaseSchema/DatabaseTable (built from the entity metadata or
// read from a snapshot file), SchemaComparator and SqlSchemaGenerator with
// the PostgreSqlSchemaHelper. Only PostgreSQL is supported, and only what
// the entity metadata of DML models can produce (no native enums, no
// generated columns, no comments).
package schema

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

// DefaultSchemaName mirrors PostgreSqlPlatform.getDefaultSchemaName.
const DefaultSchemaName = "public"

// Mapped types: the keys of MikroORM's `t` map (the names snapshots store).
const (
	TypeDate        = "date"
	TypeTime        = "time"
	TypeDateTime    = "datetime"
	TypeBigInt      = "bigint"
	TypeBlob        = "blob"
	TypeUint8Array  = "uint8array"
	TypeArray       = "array"
	TypeEnumArray   = "enumArray"
	TypeEnum        = "enum"
	TypeJSON        = "json"
	TypeInteger     = "integer"
	TypeSmallInt    = "smallint"
	TypeTinyInt     = "tinyint"
	TypeMediumInt   = "mediumint"
	TypeFloat       = "float"
	TypeDouble      = "double"
	TypeBoolean     = "boolean"
	TypeDecimal     = "decimal"
	TypeCharacter   = "character"
	TypeString      = "string"
	TypeUUID        = "uuid"
	TypeText        = "text"
	TypeInterval    = "interval"
	TypeUnknown     = "unknown"
	typeFullText    = "fulltext"
	defaultDateTime = 6
)

var simpleTypeRe = regexp.MustCompile(`[^(), ]+`)

// extractSimpleType mirrors Platform.extractSimpleType.
func extractSimpleType(t string) string {
	return simpleTypeRe.FindString(strings.ToLower(t))
}

var pgTypeAliases = map[string]string{
	"int2":              "smallint",
	"smallserial":       "smallint",
	"int":               "integer",
	"int4":              "integer",
	"serial":            "integer",
	"serial4":           "integer",
	"int8":              "bigint",
	"bigserial":         "bigint",
	"serial8":           "bigint",
	"numeric":           "decimal",
	"bool":              "boolean",
	"real":              "float",
	"float4":            "float",
	"float8":            "double",
	"timestamp":         "datetime",
	"timestamptz":       "datetime",
	"bytea":             "blob",
	"jsonb":             "json",
	"character varying": "varchar",
	"bpchar":            "character",
}

// MappedType mirrors PostgreSqlPlatform.getMappedType: the mapped type of a
// column type, by its key in MikroORM's `t` map.
func MappedType(columnType string) string {
	simple := extractSimpleType(columnType)
	if simple == "tsvector" {
		return typeFullText
	}
	t := columnType
	if alias, ok := pgTypeAliases[simple]; ok {
		t = alias
	}
	if strings.HasSuffix(t, "[]") {
		return TypeArray
	}
	switch extractSimpleType(t) {
	case "character", "char":
		return TypeCharacter
	case "string", "varchar":
		return TypeString
	case "interval":
		return TypeInterval
	case "text":
		return TypeText
	case "int", "number", "integer":
		return TypeInteger
	case "bigint":
		return TypeBigInt
	case "smallint":
		return TypeSmallInt
	case "tinyint":
		return TypeTinyInt
	case "mediumint":
		return TypeMediumInt
	case "float":
		return TypeFloat
	case "double":
		return TypeDouble
	case "decimal", "numeric":
		return TypeDecimal
	case "boolean":
		return TypeBoolean
	case "blob", "buffer":
		return TypeBlob
	case "uint8array":
		return TypeUint8Array
	case "uuid":
		return TypeUUID
	case "date":
		return TypeDate
	case "datetime", "timestamp":
		return TypeDateTime
	case "time":
		return TypeTime
	case "object", "json":
		return TypeJSON
	case "enum":
		return TypeEnum
	}
	return TypeUnknown
}

// isNumericColumn mirrors Platform.isNumericColumn.
func isNumericColumn(mappedType string) bool {
	switch mappedType {
	case TypeInteger, TypeSmallInt, TypeBigInt, TypeTinyInt:
		return true
	}
	return false
}

// defaultLength mirrors Type.getDefaultLength for the types that define it.
func defaultLength(mappedType string) *int {
	switch mappedType {
	case TypeDateTime, TypeInterval, TypeTime:
		return intPtr(defaultDateTime)
	case TypeDate:
		return intPtr(0)
	case TypeString:
		return intPtr(255)
	case TypeCharacter:
		return intPtr(1)
	}
	return nil
}

func intPtr(i int) *int { return &i }

func strPtr(s string) *string { return &s }

// QuoteIdentifier mirrors PostgreSqlPlatform.quoteIdentifier (only the
// first dot separates a schema).
func QuoteIdentifier(id string) string {
	escaped := strings.ReplaceAll(id, `"`, `""`)
	return `"` + strings.Replace(escaped, ".", `"."`, 1) + `"`
}

// QuoteValue mirrors PostgreSqlPlatform.quoteValue for strings
// (pg escapeLiteral).
func QuoteValue(s string) string {
	var b strings.Builder
	hasBackslash := false
	b.WriteByte('\'')
	for _, c := range s {
		switch c {
		case '\'':
			b.WriteString("''")
		case '\\':
			b.WriteString(`\\`)
			hasBackslash = true
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('\'')
	if hasBackslash {
		return " E" + b.String()
	}
	return b.String()
}

// hash mirrors Utils.hash with the md5 hashAlgorithm of the configuration.
func hash(data string, length int) string {
	sum := md5.Sum([]byte(data))
	return hex.EncodeToString(sum[:])[:length]
}

// IndexName mirrors PostgreSqlPlatform.getIndexName over the
// UnderscoreNamingStrategy indexName.
func IndexName(tableName string, columns []string, typ string) string {
	name := namingIndexName(tableName, columns, typ)
	if len(name) > 63 {
		return name[:55-len(typ)] + "_" + hash(name, 5) + "_" + typ
	}
	return name
}

// namingIndexName mirrors AbstractNamingStrategy.indexName.
func namingIndexName(tableName string, columns []string, typ string) string {
	if i := strings.Index(tableName, "."); i >= 0 {
		tableName = tableName[i+1:]
	}
	if typ == "primary" {
		return tableName + "_pkey"
	}
	cols := make([]string, len(columns))
	for i, c := range columns {
		cols[i] = strings.ReplaceAll(c, ".", "_")
	}
	if typ == "sequence" {
		return tableName + "_" + strings.Join(cols, "_") + "_seq"
	}
	if len(cols) > 0 {
		return tableName + "_" + strings.Join(cols, "_") + "_" + typ
	}
	return tableName + "_" + typ
}

// DefaultPrimaryName mirrors PostgreSqlPlatform.getDefaultPrimaryName.
func DefaultPrimaryName(tableName string) string {
	name := tableName + "_pkey"
	if len(name) > 63 {
		return name[:55-len("primary")] + "_" + hash(name, 5) + "_primary"
	}
	return name
}

// columnProp is what Type.getColumnType reads from a property (the
// comparator maps columns to properties with mapColumnToProperty).
type columnProp struct {
	columnTypes    []string
	autoincrement  bool
	length         *int
	precision      *int
	scale          *int
	items          []string
	nativeEnumName string
}

// columnTypeOf mirrors mappedType.getColumnType(prop, platform) for the
// PostgreSQL platform.
func columnTypeOf(mappedType string, p columnProp) string {
	switch mappedType {
	case TypeText:
		return "text"
	case TypeInteger:
		if p.autoincrement {
			return "serial"
		}
		return "int"
	case TypeSmallInt, TypeTinyInt:
		return "smallint"
	case TypeMediumInt:
		return "mediumint"
	case TypeBigInt:
		if p.autoincrement {
			return "bigserial"
		}
		return "bigint"
	case TypeFloat:
		return "real"
	case TypeDouble:
		return "double precision"
	case TypeDecimal:
		return decimalDeclaration(p.precision, p.scale)
	case TypeBoolean:
		return "boolean"
	case TypeDateTime:
		if p.length != nil {
			return "timestamptz(" + strconv.Itoa(*p.length) + ")"
		}
		return "timestamptz"
	case TypeDate:
		if p.length != nil && *p.length != 0 {
			return "date(" + strconv.Itoa(*p.length) + ")"
		}
		return "date"
	case TypeTime:
		return "time(0)"
	case TypeJSON:
		return "jsonb"
	case TypeArray:
		return "text[]"
	case TypeEnum:
		if p.nativeEnumName != "" {
			return p.nativeEnumName
		}
		if len(p.columnTypes) > 0 {
			return p.columnTypes[0]
		}
		return "text"
	case TypeString:
		return varcharDeclaration(p.length)
	case TypeCharacter:
		return charDeclaration(p.length)
	case TypeUUID:
		return "uuid"
	case TypeBlob, TypeUint8Array:
		return "bytea"
	case TypeInterval:
		if p.length != nil {
			return "interval(" + strconv.Itoa(*p.length) + ")"
		}
		return "interval"
	}
	if len(p.columnTypes) > 0 {
		return p.columnTypes[0]
	}
	return "text"
}

func decimalDeclaration(precision, scale *int) string {
	pr, sc := 10, 0
	if precision != nil {
		pr = *precision
	}
	if scale != nil {
		sc = *scale
	}
	return "numeric(" + strconv.Itoa(pr) + "," + strconv.Itoa(sc) + ")"
}

func varcharDeclaration(length *int) string {
	if length != nil && *length == -1 {
		return "varchar"
	}
	l := 255
	if length != nil {
		l = *length
	}
	return "varchar(" + strconv.Itoa(l) + ")"
}

func charDeclaration(length *int) string {
	if length != nil && *length == -1 {
		return "char"
	}
	l := 1
	if length != nil {
		l = *length
	}
	return "char(" + strconv.Itoa(l) + ")"
}

var (
	timetzRe        = regexp.MustCompile(`^timetz\b`)
	timeWithTZRe    = regexp.MustCompile(`^time\s+with\s+time\s+zone\b`)
	whitespaceRunRe = regexp.MustCompile(`\s+`)
)

// normalizeColumnType mirrors PostgreSqlPlatform.normalizeColumnType.
func normalizeColumnType(t string, p columnProp) string {
	simple := extractSimpleType(t)
	switch simple {
	case "int", "int4", "integer":
		return "int"
	case "bigint", "int8":
		return "bigint"
	case "smallint", "int2":
		return "smallint"
	case "boolean", "bool":
		return "boolean"
	case "varchar", "character varying":
		return varcharDeclaration(p.length)
	case "char", "bpchar":
		return charDeclaration(p.length)
	case "decimal", "numeric":
		return decimalDeclaration(p.precision, p.scale)
	case "interval":
		if p.length != nil {
			return "interval(" + strconv.Itoa(*p.length) + ")"
		}
		return "interval"
	case "geometry", "geography":
		return whitespaceRunRe.ReplaceAllString(strings.ToLower(t), "")
	}
	original := strings.ToLower(t)
	if len(p.columnTypes) > 0 {
		original = strings.ToLower(p.columnTypes[0])
	}
	if timetzRe.MatchString(original) || timeWithTZRe.MatchString(original) {
		l := defaultDateTime
		if p.length != nil {
			l = *p.length
		}
		return "timetz(" + strconv.Itoa(l) + ")"
	}
	return strings.ToLower(t)
}

var (
	pgLengthRe      = regexp.MustCompile(`^(\w+(?:\s+\w+)*)\s*(?:\(\s*(\d+)\s*\)|$)`)
	precisionScaleR = regexp.MustCompile(`\w+\((\d+), ?(\d+)\)`)
)

// inferLengthFromColumnType mirrors
// PostgreSqlSchemaHelper.inferLengthFromColumnType.
func inferLengthFromColumnType(t string) *int {
	m := pgLengthRe.FindStringSubmatch(t)
	if m == nil {
		return nil
	}
	if m[2] == "" {
		switch m[1] {
		case "character varying", "varchar", "bpchar", "char", "character":
			return intPtr(-1)
		case "interval", "time", "timestamp", "timestamptz":
			return intPtr(defaultDateTime)
		}
		return nil
	}
	n, _ := strconv.Atoi(m[2])
	return &n
}

// pgDefaultValues mirrors PostgreSqlSchemaHelper.DEFAULT_VALUES.
var pgDefaultValues = map[string]string{
	"now()":                "now()",
	"current_timestamp(?)": "current_timestamp(?)",
	"('now'::text)::timestamp(?) with time zone":    "current_timestamp(?)",
	"('now'::text)::timestamp(?) without time zone": "current_timestamp(?)",
	"null::character varying":                       "null",
	"null::timestamp with time zone":                "null",
	"null::timestamp without time zone":             "null",
}

var (
	castDefaultRe = regexp.MustCompile(`^'(.*)'::(.*)$`)
	lengthParamRe = regexp.MustCompile(`\(\d+\)`)
)

// normalizeDefaultValue mirrors PostgreSqlSchemaHelper.normalizeDefaultValue.
func normalizeDefaultValue(def *string, length *int) *string {
	if def == nil || *def == "" {
		return def
	}
	if m := castDefaultRe.FindStringSubmatch(*def); m != nil {
		if m[2] == "integer" {
			f, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				return strPtr("NaN")
			}
			return strPtr(strconv.FormatFloat(f, 'f', -1, 64))
		}
		return strPtr("'" + m[1] + "'")
	}
	generic := *def
	if loc := lengthParamRe.FindStringIndex(generic); loc != nil {
		generic = generic[:loc[0]] + "(?)" + generic[loc[1]:]
	}
	generic = strings.ToLower(generic)
	norm, ok := pgDefaultValues[generic]
	if !ok {
		return def
	}
	repl := ""
	if length != nil {
		repl = "(" + strconv.Itoa(*length) + ")"
	}
	return strPtr(strings.Replace(norm, "(?)", repl, 1))
}
