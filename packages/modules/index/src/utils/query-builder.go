package utils

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	andOperator = "$and"
	orOperator  = "$or"
	notOperator = "$not"
)

// OperatorMap mirrors OPERATOR_MAP.
var OperatorMap = map[string]string{
	"$eq":    "=",
	"$lt":    "<",
	"$gt":    ">",
	"$lte":   "<=",
	"$gte":   ">=",
	"$ne":    "!=",
	"$in":    "IN",
	"$nin":   "NOT IN",
	"$is":    "IS",
	"$like":  "LIKE",
	"$ilike": "ILIKE",
}

func escapeJsonPathString(val string) string {
	val = strings.ReplaceAll(val, `\`, `\\`)
	val = strings.ReplaceAll(val, `"`, `\"`)
	return strings.ReplaceAll(val, `'`, `\'`)
}

// jsString mirrors String(value) / template literal interpolation.
func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return jsNumberString(t)
	case int:
		return strconv.Itoa(t)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			if e != nil {
				parts[i] = jsString(e)
			}
		}
		return strings.Join(parts, ",")
	case *dal.Record:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}

func buildSafeJsonPathQuery(field, operator string, value any) string {
	jsonPathOperator := operator
	caseInsensitiveFlag := ""
	switch {
	case operator == "=":
		jsonPathOperator = "=="
	case strings.Contains(strings.ToUpper(operator), "LIKE"):
		jsonPathOperator = "like_regex"
		if strings.ToUpper(operator) == "ILIKE" {
			caseInsensitiveFlag = ` flag "i"`
		}
	case operator == "IS":
		jsonPathOperator = "=="
	case operator == "IS NOT":
		jsonPathOperator = "!="
	}
	rendered := ""
	if s, ok := value.(string); ok {
		val := s
		if jsonPathOperator == "like_regex" {
			val = strings.ReplaceAll(strings.ReplaceAll(val, "%", ".*"), "_", ".")
		}
		rendered = `"` + escapeJsonPathString(val) + `"`
	} else {
		rendered = jsString(value)
	}
	return "$." + field + " " + jsonPathOperator + " " + rendered + caseInsensitiveFlag
}

// jsonStringify mirrors JSON.stringify for filter values.
func jsonStringify(v any) string {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return "null"
		}
		return jsNumberString(t)
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = jsonStringify(e)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *dal.Record:
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range t.Keys() {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, _ := dal.MarshalValue(k)
			b.Write(kb)
			b.WriteByte(':')
			e, _ := t.Get(k)
			b.WriteString(jsonStringify(e))
		}
		b.WriteByte('}')
		return b.String()
	}
	out, err := dal.MarshalValue(v)
	if err != nil {
		return "null"
	}
	return string(out)
}

// QueryOptions mirrors QueryOptions.
type QueryOptions struct {
	Skip    any
	Take    any
	OrderBy *dal.Record
}

// QueryFormat mirrors QueryFormat.
type QueryFormat struct {
	Select    *dal.Record
	Where     *dal.Record
	JoinWhere *dal.Record
}

// QueryBuilder mirrors QueryBuilder.
type QueryBuilder struct {
	searchVectorColumnName string
	structure              *dal.Record
	selector               QueryFormat
	options                QueryOptions
	schema                 *SchemaObjectRepresentation
	allSchemaFields        map[string]bool
	rawFilters             *dal.Record
	requestedFields        *dal.Record
	idsOnly                bool
}

// NewQueryBuilder mirrors new QueryBuilder(args). rawFilters is
// rawConfig.filters.
func NewQueryBuilder(schema *SchemaObjectRepresentation, selector QueryFormat, options QueryOptions, rawFilters, requestedFields *dal.Record, idsOnly bool) *QueryBuilder {
	return &QueryBuilder{
		searchVectorColumnName: "document_tsv",
		structure:              selector.Select,
		selector:               selector,
		options:                options,
		schema:                 schema,
		allSchemaFields:        schema.AllFields(),
		rawFilters:             rawFilters,
		requestedFields:        requestedFields,
		idsOnly:                idsOnly,
	}
}

func isLogicalOperator(key string) bool {
	return key == andOperator || key == orOperator || key == notOperator
}

func (q *QueryBuilder) getStructureKeys(structure any) []string {
	var keys []string
	seen := map[string]bool{}
	var collect func(obj any)
	collect = func(obj any) {
		r, ok := obj.(*dal.Record)
		if !ok {
			return
		}
		for _, key := range r.Keys() {
			v, _ := r.Get(key)
			if isLogicalOperator(key) {
				if arr, ok := v.([]any); ok {
					for _, item := range arr {
						collect(item)
					}
				}
			} else if key != "entity" {
				if !seen[key] {
					seen[key] = true
					keys = append(keys, key)
				}
			}
		}
	}
	collect(structure)
	return keys
}

func (q *QueryBuilder) getEntity(path string, throwWhenNotFound bool) (*SchemaPropertiesMapEntry, error) {
	e, ok := q.schema.PropertiesMap[path]
	if !ok {
		if !throwWhenNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("Could not find entity for path: %s. It might not be indexed.", path)
	}
	return e, nil
}

func (q *QueryBuilder) getGraphQLType(path, field string) (string, error) {
	if isLogicalOperator(field) {
		return "JSON", nil
	}
	e, err := q.getEntity(path, true)
	if err != nil {
		return "", err
	}
	fields, ok := q.schema.Types[e.Ref.Entity]
	if !ok {
		return "", errors.New("Cannot read properties of undefined (reading '_fields')")
	}
	fieldType, ok := fields[field]
	if !ok {
		return "", fmt.Errorf("Field %s is not indexed.", field)
	}
	isArray := strings.HasPrefix(fieldType, "[")
	current := strings.NewReplacer("[", "", "]", "", "!", "").Replace(fieldType)
	if isArray {
		return current + "[]", nil
	}
	return current, nil
}

var leadingInt = regexp.MustCompile(`^\s*[+-]?\d+`)
var leadingFloat = regexp.MustCompile(`^\s*[+-]?(?:Infinity|(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?)`)

func jsParseInt(v any) any {
	s := jsString(v)
	m := leadingInt.FindString(s)
	if m == "" {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(m), 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func jsParseFloat(v any) any {
	if f, ok := v.(float64); ok {
		return f
	}
	s := jsString(v)
	m := strings.TrimSpace(leadingFloat.FindString(s))
	if m == "" {
		return math.NaN()
	}
	switch m {
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func jsTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0 && !math.IsNaN(t)
	case int:
		return t != 0
	}
	return true
}

// jsDateISO mirrors new Date(val).toISOString().
func jsDateISO(v any) (any, error) {
	var t time.Time
	ok := false
	switch x := v.(type) {
	case string:
		t, ok = common.ParseJSDate(x)
	case float64:
		if !math.IsNaN(x) {
			t, ok = time.UnixMilli(int64(x)).UTC(), true
		}
	case time.Time:
		t, ok = x, true
	case bool:
		if x {
			t, ok = time.UnixMilli(1).UTC(), true
		} else {
			t, ok = time.UnixMilli(0).UTC(), true
		}
	case nil:
		t, ok = time.UnixMilli(0).UTC(), true
	}
	if !ok {
		return nil, errors.New("Invalid time value")
	}
	return dal.FormatJSDate(t), nil
}

func convertValue(graphqlType string, v any) (any, error) {
	switch graphqlType {
	case "Int":
		return jsParseInt(v), nil
	case "Float":
		return jsParseFloat(v), nil
	case "String", "ID":
		return jsString(v), nil
	case "Boolean":
		return jsTruthy(v), nil
	case "Date", "DateTime":
		return jsDateISO(v)
	case "Time":
		return jsDateISO("1970-01-01T" + jsString(v) + "Z")
	}
	return v, nil
}

func (q *QueryBuilder) transformValueToType(path string, field []string, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	fullPath := append([]string{path}, field...)
	prop := fullPath[len(fullPath)-1]
	fieldPath := strings.Join(fullPath[:len(fullPath)-1], ".")
	t, err := q.getGraphQLType(fieldPath, prop)
	if err != nil {
		return nil, err
	}
	graphqlType := strings.Replace(t, "[]", "", 1)
	if arr, ok := value.([]any); ok {
		out := make([]any, len(arr))
		for i, v := range arr {
			c, err := convertValue(graphqlType, v)
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	}
	return convertValue(graphqlType, value)
}

type pgCastType struct {
	cast         string
	defaultValue string
}

func (q *QueryBuilder) getPostgresCastType(path string, field []string) (pgCastType, error) {
	castMap := map[string]string{"Int": "::int", "Float": "::double precision", "Boolean": "::boolean", "Date": "::timestamp", "DateTime": "::timestamp", "Time": "::time", "": ""}
	defaults := map[string]string{"Int": "0", "Float": "0", "Boolean": "false", "Date": "1970-01-01 00:00:00", "DateTime": "1970-01-01 00:00:00", "Time": "00:00:00", "": ""}
	fullPath := append([]string{path}, field...)
	prop := fullPath[len(fullPath)-1]
	fieldPath := strings.Join(fullPath[:len(fullPath)-1], ".")
	t, err := q.getGraphQLType(fieldPath, prop)
	if err != nil {
		return pgCastType{}, err
	}
	isList := strings.HasSuffix(t, "[]")
	t = strings.Replace(t, "[]", "", 1)
	cast := castMap[t]
	if isList {
		cast += "[]"
	}
	return pgCastType{cast: cast, defaultValue: defaults[t]}, nil
}

type aliasMap struct {
	m     map[string]string
	index int
}

func newAliasMap() *aliasMap { return &aliasMap{m: map[string]string{}} }

func (a *aliasMap) has(key string) bool {
	v, ok := a.m[key]
	return ok && v != ""
}

// at returns the alias of a path, "undefined" when missing (JS template).
func (a *aliasMap) at(key string) string {
	if v, ok := a.m[key]; ok {
		return v
	}
	return "undefined"
}

func (q *QueryBuilder) getShortAlias(am *aliasMap, alias string, level int) string {
	if am.has(alias) {
		return am.m[alias]
	}
	v := "t_" + strconv.Itoa(am.index)
	am.index++
	if level > 0 {
		v += "_" + strconv.Itoa(level)
	}
	am.m[alias] = v
	return v
}

func pathJoin(parentPath, key string) string {
	if parentPath != "" {
		return parentPath + "." + key
	}
	return key
}

func (q *QueryBuilder) parseWhere(am *aliasMap, obj *dal.Record, builder *knexBuilder, parentPath string) error {
	getPathAndField := func(key string) ([]string, string) {
		path := strings.Split(pathJoin(parentPath, key), ".")
		field := []string{path[len(path)-1]}
		path = path[:len(path)-1]
		for !am.has(strings.Join(path, ".")) && len(path) > 0 {
			field = append([]string{path[len(path)-1]}, field...)
			path = path[:len(path)-1]
		}
		return field, strings.Join(path, ".")
	}
	getPathOperation := func(attr string, path []string, value any) (string, error) {
		partial := path
		if len(path) > 1 {
			partial = path[:len(path)-1]
		}
		val, err := q.transformValueToType(attr, partial, value)
		if err != nil {
			return "", err
		}
		var result any = val
		for i := len(path) - 1; i >= 0; i-- {
			r := dal.NewRecord()
			r.Set(path[i], result)
			result = r
		}
		return jsonStringify(result), nil
	}

	var firstErr error
	fail := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	for _, key := range obj.Keys() {
		if firstErr != nil {
			break
		}
		pathAsArray := strings.Split(pathJoin(parentPath, key), ".")
		fieldOrLogicalOperator := pathAsArray[len(pathAsArray)-1]
		pathAsArray = pathAsArray[:len(pathAsArray)-1]
		value, _ := obj.Get(key)

		if isLogicalOperator(fieldOrLogicalOperator) {
			if _, isArr := value.([]any); !isArr {
				value = []any{value}
			}
		}
		arr, isArr := value.([]any)
		rec, isRec := value.(*dal.Record)
		sub := strings.Join(pathAsArray, ".")

		group := func(cond any) func(*knexBuilder) error {
			return func(sb *knexBuilder) error {
				if c, ok := cond.(*dal.Record); ok {
					return q.parseWhere(am, c, sb, sub)
				}
				return nil
			}
		}
		switch {
		case fieldOrLogicalOperator == andOperator && isArr:
			builder.Where(func(qb *knexBuilder) error {
				for _, cond := range arr {
					qb.AndWhere(group(cond))
				}
				return nil
			})
		case fieldOrLogicalOperator == orOperator && isArr:
			builder.Where(func(qb *knexBuilder) error {
				for _, cond := range arr {
					qb.OrWhere(group(cond))
				}
				return nil
			})
		case fieldOrLogicalOperator == notOperator && (isArr || isRec):
			builder.WhereNot(func(qb *knexBuilder) error {
				if isArr {
					for _, cond := range arr {
						qb.AndWhere(group(cond))
					}
					return nil
				}
				return q.parseWhere(am, rec, qb, sub)
			})
		case isRec && !isLogicalOperator(fieldOrLogicalOperator):
			currentPath := pathJoin(parentPath, key)
			hasOperators := false
			for _, sk := range rec.Keys() {
				if _, ok := OperatorMap[sk]; ok {
					hasOperators = true
				}
			}
			if !hasOperators {
				if err := q.parseWhere(am, rec, builder, currentPath); err != nil {
					fail(err)
				}
				continue
			}
			field, attr := getPathAndField(key)
			for _, subKey := range rec.Keys() {
				operator, ok := OperatorMap[subKey]
				if !ok {
					fail(fmt.Errorf("Unsupported operator: %s", subKey))
					break
				}
				nested := strings.Repeat("->?", len(field)-1)
				raw, _ := rec.Get(subKey)
				subValue, err := q.transformValueToType(attr, field, raw)
				if err != nil {
					fail(err)
					break
				}
				var val []any
				if operator == "IN" || operator == "NOT IN" {
					switch sv := subValue.(type) {
					case []any:
						val = sv
					case nil:
						val = nil
					default:
						val = []any{sv}
					}
				} else {
					val = []any{subValue}
				}
				if operator == "=" && subValue == nil {
					operator = "IS"
				} else if operator == "!=" && subValue == nil {
					operator = "IS NOT"
				}
				hasID := field[len(field)-1] == "id"
				targetField := field[len(field)-1]
				alias := am.at(attr)
				switch operator {
				case "=":
					if hasID {
						builder.WhereRaw(alias+".id = ?", subValue)
					} else {
						op, err := getPathOperation(attr, field, subValue)
						if err != nil {
							fail(err)
							break
						}
						builder.WhereRaw(alias+".data @> ?::jsonb", op)
					}
				case "IN", "NOT IN":
					if len(val) == 0 {
						continue
					}
					placeholders := strings.TrimSuffix(strings.Repeat("?,", len(val)), ",")
					isNegated := operator == "NOT IN"
					if hasID {
						op := "IN"
						if isNegated {
							op = "NOT IN"
						}
						builder.WhereRaw(alias+".id "+op+" ("+placeholders+")", val...)
					} else {
						jsonbValues := make([]any, len(val))
						for i, item := range val {
							r := dal.NewRecord()
							r.Set(targetField, item)
							jsonbValues[i] = jsonStringify(r)
						}
						if isNegated {
							builder.WhereRaw("NOT EXISTS (SELECT 1 FROM unnest(ARRAY["+placeholders+"]::JSONB[]) AS v(val) WHERE "+alias+".data"+nested+" @> v.val)", jsonbValues...)
						} else {
							builder.WhereRaw(alias+".data"+nested+" @> ANY(ARRAY["+placeholders+"]::JSONB[])", jsonbValues...)
						}
					}
				default:
					if hasID {
						builder.WhereRaw("("+alias+".id) "+operator+" ?", val...)
					} else {
						builder.WhereRaw(alias+".data"+nested+" @@ ?", buildSafeJsonPathQuery(targetField, operator, val[0]))
					}
				}
			}
		default:
			field, attr := getPathAndField(key)
			nested := strings.Repeat("->?", len(field)-1)
			v, err := q.transformValueToType(attr, field, value)
			if err != nil {
				fail(err)
				break
			}
			alias := am.at(attr)
			hasID := field[len(field)-1] == "id"
			if vs, ok := v.([]any); ok {
				if len(vs) == 0 {
					continue
				}
				placeholders := strings.TrimSuffix(strings.Repeat("?,", len(vs)), ",")
				if hasID {
					builder.WhereRaw(alias+".id IN ("+placeholders+")", vs...)
				} else {
					targetField := field[len(field)-1]
					jsonbValues := make([]any, len(vs))
					for i, item := range vs {
						r := dal.NewRecord()
						r.Set(targetField, item)
						jsonbValues[i] = jsonStringify(r)
					}
					builder.WhereRaw(alias+".data"+nested+" @> ANY(ARRAY["+placeholders+"]::JSONB[])", jsonbValues...)
				}
			} else if v != nil || value == nil {
				// isDefined(value): undefined never reaches here; null is
				// defined and goes through the "=" branch.
				if hasID {
					builder.WhereRaw(alias+".id = ?", v)
				} else {
					op, err := getPathOperation(attr, field, v)
					if err != nil {
						fail(err)
						break
					}
					builder.WhereRaw(alias+".data @> ?::jsonb", op)
				}
			}
		}
	}
	return firstErr
}

type joinEntity struct {
	entity    *SchemaPropertiesMapEntry
	parEntity *SchemaPropertiesMapEntry
	parAlias  string
	alias     string
}

func (q *QueryBuilder) buildQueryParts(structure any, parentAlias string, parentEntity *SchemaPropertiesMapEntry, parentProperty string, aliasPath []string, level int, am *aliasMap) ([]string, error) {
	currentAliasPath := strings.Join(append(append([]string(nil), aliasPath...), parentProperty), ".")
	isSelectableField := q.allSchemaFields[parentProperty]
	entities, _ := q.getEntity(currentAliasPath, false)
	if isSelectableField || entities == nil || entities.Ref.Alias == "" {
		return nil, nil
	}
	mainEntity := entities
	mainAlias := q.getShortAlias(am, strings.ToLower(mainEntity.Ref.Entity), level)

	var allEntities []joinEntity
	parentRef := ""
	if parentEntity != nil {
		parentRef = parentEntity.Ref.Entity
	}
	if entities.ShortCutOf == "" {
		allEntities = append(allEntities, joinEntity{entity: entities, parEntity: parentEntity, parAlias: parentAlias, alias: mainAlias})
	} else {
		intermediateAlias := strings.Split(entities.ShortCutOf, ".")
		for i, x := len(intermediateAlias)-1, 0; i >= 0; i, x = i-1, x+1 {
			intermediateEntity, _ := q.getEntity(strings.Join(intermediateAlias, "."), false)
			if intermediateEntity == nil {
				break
			}
			intermediateAlias = intermediateAlias[:len(intermediateAlias)-1]
			if intermediateEntity.Ref.Entity == parentRef {
				break
			}
			parentIntermediateEntity, err := q.getEntity(strings.Join(intermediateAlias, "."), true)
			if err != nil {
				return nil, err
			}
			alias := q.getShortAlias(am, strings.ToLower(intermediateEntity.Ref.Entity), level) + "_" + strconv.Itoa(x)
			parAlias := parentAlias
			if parentIntermediateEntity.Ref.Entity != parentRef {
				parAlias = q.getShortAlias(am, strings.ToLower(parentIntermediateEntity.Ref.Entity), level) + "_" + strconv.Itoa(x+1)
			}
			if x == 0 {
				am.m[currentAliasPath] = alias
			}
			allEntities = append([]joinEntity{{entity: intermediateEntity, parEntity: parentIntermediateEntity, parAlias: parAlias, alias: alias}}, allEntities...)
		}
	}

	var queryParts []string
	for _, join := range allEntities {
		joinBuilder := newKnexBuilder()
		alias := join.alias
		am.m[currentAliasPath] = alias
		if level > 0 {
			cName := NormalizeTableName(join.entity.Ref.Entity)
			joinTable := "cat_" + cName + " AS " + alias
			if join.entity.IsInverse || join.parEntity.IsInverse {
				pName := strings.ToLower(join.entity.Ref.Entity + join.parEntity.Ref.Entity)
				pivotTable := GetPivotTableName(pName)
				joinBuilder.LeftJoin(pivotTable+" AS "+alias+"_ref", alias+"_ref.child_id", join.parAlias+".id")
				joinBuilder.LeftJoin(joinTable, alias+".id", alias+"_ref.parent_id")
			} else {
				pName := strings.ToLower(join.parEntity.Ref.Entity + join.entity.Ref.Entity)
				pivotTable := GetPivotTableName(pName)
				joinBuilder.LeftJoin(pivotTable+" AS "+alias+"_ref", alias+"_ref.parent_id", join.parAlias+".id")
				joinBuilder.LeftJoin(joinTable, alias+".id", alias+"_ref.child_id")
			}
			joinWhere := q.selector.JoinWhere
			if joinWhere != nil {
				for _, key := range joinWhere.Keys() {
					k := strings.Split(key, ".")
					curPath := strings.Join(k[:len(k)-1], ".")
					if curPath != currentAliasPath {
						continue
					}
					rel, _ := q.getEntity(curPath, false)
					if rel == nil || rel.Ref.Entity != join.entity.Ref.Entity {
						continue
					}
					v, _ := joinWhere.Get(key)
					w := dal.NewRecord()
					w.Set(key, v)
					if err := q.parseWhere(am, w, joinBuilder, ""); err != nil {
						return nil, err
					}
					break
				}
			}
			sql, err := joinBuilder.ToQuery()
			if err != nil {
				return nil, err
			}
			sql = strings.Replace(sql, "select * ", "", 1)
			sql = strings.Replace(sql, "where", "and", 1)
			queryParts = append(queryParts, sql)
		}
	}

	rec, _ := structure.(*dal.Record)
	for _, child := range q.getStructureKeys(structure) {
		childStructure := get(rec, child)
		parts, err := q.buildQueryParts(childStructure, mainAlias, mainEntity, child, append(append([]string(nil), aliasPath...), parentProperty), level+1, am)
		if err != nil {
			return nil, err
		}
		for _, p := range parts {
			if p != "" {
				queryParts = append(queryParts, p)
			}
		}
	}
	return queryParts, nil
}

// selectParts is the ordered select object of buildSelectParts.
type selectParts struct {
	keys   []string
	values map[string]selectPart
}

func (s *selectParts) set(key string, p selectPart) {
	if _, ok := s.values[key]; !ok {
		s.keys = append(s.keys, key)
	}
	p.alias = key
	s.values[key] = p
}

func (s *selectParts) delete(key string) {
	if _, ok := s.values[key]; !ok {
		return
	}
	delete(s.values, key)
	for i, k := range s.keys {
		if k == key {
			s.keys = append(s.keys[:i], s.keys[i+1:]...)
			break
		}
	}
}

func (q *QueryBuilder) buildSelectParts(structure any, parentProperty string, am *aliasMap, aliasPath []string, sp *selectParts) {
	currentAliasPath := strings.Join(append(append([]string(nil), aliasPath...), parentProperty), ".")
	if q.allSchemaFields[parentProperty] {
		parentAliasPath := strings.Join(aliasPath, ".")
		alias := am.at(parentAliasPath)
		sp.delete(parentAliasPath)
		if parentProperty == "id" {
			sp.set(currentAliasPath, selectPart{column: alias + ".id"})
		} else if !q.idsOnly {
			sp.set(currentAliasPath, selectPart{raw: &knexRaw{sql: alias + ".data->'" + parentProperty + "'"}})
		}
		return
	}
	alias, ok := am.m[currentAliasPath]
	if !ok || alias == "" {
		return
	}
	if !q.idsOnly {
		sp.set(currentAliasPath, selectPart{column: alias + ".data"})
	}
	sp.set(currentAliasPath+".id", selectPart{column: alias + ".id"})
	rec, _ := structure.(*dal.Record)
	for _, child := range q.getStructureKeys(structure) {
		q.buildSelectParts(get(rec, child), child, am, append(append([]string(nil), aliasPath...), parentProperty), sp)
	}
}

// transformOrderBy mirrors transformOrderBy.
func (q *QueryBuilder) transformOrderBy(arr []*dal.Record) ([]string, map[string]string, error) {
	var keys []string
	result := map[string]string{}
	var nested func(obj *dal.Record, prefix string) error
	nested = func(obj *dal.Record, prefix string) error {
		ks := obj.Keys()
		if len(ks) == 0 {
			return nil
		}
		if len(ks) > 1 {
			return errors.New("Order by only supports one key per object.")
		}
		key := ks[0]
		value, _ := obj.Get(key)
		if r, ok := value.(*dal.Record); ok {
			return nested(r, prefix+key+".")
		}
		dir := "ASC"
		switch v := value.(type) {
		case string:
			switch strings.ToUpper(v) {
			case "DESC":
				dir = "DESC"
			}
		case bool:
			if !v {
				dir = "DESC"
			}
		case float64:
			if v == -1 {
				dir = "DESC"
			}
		}
		k := prefix + key
		if _, ok := result[k]; !ok {
			keys = append(keys, k)
		}
		result[k] = dir
		return nil
	}
	for _, obj := range arr {
		if err := nested(obj, ""); err != nil {
			return nil, nil, err
		}
	}
	return keys, result, nil
}

func numberOption(v any, fallback int) int {
	switch t := v.(type) {
	case float64:
		if !math.IsNaN(t) {
			return int(t)
		}
	case int:
		return t
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return int(f)
		}
		if strings.TrimSpace(t) == "" {
			return 0
		}
	case nil:
		return fallback
	case bool:
		if t {
			return 1
		}
		return 0
	}
	return fallback
}

// BuildQuery mirrors buildQuery({ hasPagination, hasCount }).
func (q *QueryBuilder) BuildQuery(hasPagination, hasCount bool) (sql string, sqlCount string, err error) {
	selectOnlyStructure := q.selector.Select
	structure := q.requestedFields
	filter := q.selector.Where
	if filter == nil {
		filter = dal.NewRecord()
	}

	var orderObjs []*dal.Record
	if q.options.OrderBy != nil {
		orderObjs = []*dal.Record{q.options.OrderBy}
	}
	orderKeys, orderBy, err := q.transformOrderBy(orderObjs)
	if err != nil {
		return "", "", err
	}
	take := numberOption(q.options.Take, 15)
	skip := numberOption(q.options.Skip, 0)

	rootKeys := q.getStructureKeys(structure)
	rootKey := ""
	if len(rootKeys) > 0 {
		rootKey = rootKeys[0]
	}
	rootStructure := get(structure, rootKey)

	entity, err := q.getEntity(rootKey, true)
	if err != nil {
		return "", "", err
	}
	rootEntity := strings.ToLower(entity.Ref.Entity)
	am := newAliasMap()

	hasTextSearch := false
	var textSearchQuery any
	searchProp := rootKey + ".q"
	if v, ok := filter.Get(searchProp); ok {
		if !jsTruthy(v) {
			filter.Delete(searchProp)
		} else {
			hasTextSearch = true
			textSearchQuery = v
			filter.Delete(searchProp)
		}
	}

	filterSort := dal.NewRecord()
	if q.rawFilters != nil {
		filterSort = spread(UnflattenObjectKeys(q.rawFilters), nil)
	}
	for _, k := range orderKeys {
		filterSort.Set(k, orderBy[k])
	}
	filterSortStructure := get(UnflattenObjectKeys(filterSort), rootKey)
	if filterSortStructure == nil {
		filterSortStructure = dal.NewRecord()
	}

	joinParts, err := q.buildQueryParts(filterSortStructure, "", entity, rootKey, nil, 0, am)
	if err != nil {
		return "", "", err
	}
	rootAlias := am.at(rootKey)

	inner := newKnexBuilder()
	outer := newKnexBuilder()
	inner.Distinct(rootAlias + ".id")

	var orderBySelects []knexRaw
	var orderByClauses []string
	for _, aliasPath := range orderKeys {
		path := strings.Split(aliasPath, ".")
		field := path[len(path)-1]
		attr := strings.Join(path[:len(path)-1], ".")
		alias, hasAlias := am.m[attr]
		direction := orderBy[aliasPath]
		pgType, err := q.getPostgresCastType(attr, []string{field})
		if err != nil {
			return "", "", err
		}
		hasID := field == "id"
		orderExpression := rootAlias + ".id " + direction
		if hasAlias && alias != "" {
			aggregateAlias := `"` + aliasPath + `_agg"`
			aggregateExpression := "(" + alias + ".data->>'" + field + "')" + pgType.cast
			if hasID {
				aggregateExpression = alias + ".id"
			} else {
				fn := "MAX"
				if direction == "ASC" {
					fn = "MIN"
				}
				orderBySelects = append(orderBySelects, knexRaw{sql: fn + "(" + aggregateExpression + ") AS " + aggregateAlias})
				orderExpression = aggregateAlias + " " + direction
			}
			outer.OrderByRaw(aggregateExpression + " " + direction)
		}
		orderByClauses = append(orderByClauses, orderExpression)
	}
	if len(orderBySelects) > 0 {
		inner.SelectRaw(orderBySelects...)
	}
	inner.From("cat_" + NormalizeTableName(rootEntity) + " AS " + q.getShortAlias(am, rootKey, 0))
	for _, p := range joinParts {
		inner.JoinRaw(p)
	}

	if hasTextSearch {
		searchWhereParts := []string{rootAlias + "." + q.searchVectorColumnName + " @@ plainto_tsquery('simple', ?)"}
		for _, part := range joinParts {
			var chunks []string
			for _, c := range strings.Split(part, " as ") {
				chunks = append(chunks, strings.Split(c, " on ")...)
			}
			for _, alias := range chunks {
				if strings.HasPrefix(alias, `"t_`) && !strings.Contains(alias, "_ref") {
					searchWhereParts = append(searchWhereParts, alias+"."+q.searchVectorColumnName+" @@ plainto_tsquery('simple', ?)")
				}
			}
		}
		bindings := make([]any, len(searchWhereParts))
		for i := range bindings {
			bindings[i] = textSearchQuery
		}
		inner.WhereRaw("("+strings.Join(searchWhereParts, " OR ")+")", bindings...)
	}

	if err := q.parseWhere(am, filter, inner, ""); err != nil {
		return "", "", err
	}
	if len(orderBySelects) > 0 {
		inner.GroupBy(rootAlias + ".id")
	}
	if len(orderByClauses) > 0 {
		inner.OrderByRaw(strings.Join(orderByClauses, ", "))
	} else {
		inner.OrderBy(rootAlias+".id", "ASC")
	}

	if hasCount {
		estimate := inner.Clone()
		estimate.ClearSelect().SelectLiteral("1")
		estimate.ClearOrder()
		eq, err := estimate.ToQuery()
		if err != nil {
			return "", "", err
		}
		if sqlCount, err = rawToQuery("SELECT count_estimate(?) AS estimate_count", eq); err != nil {
			return "", "", err
		}
	}

	if hasPagination {
		inner.Limit(take)
		if skip > 0 {
			inner.Offset(skip)
		}
	}

	outer.From("cat_" + NormalizeTableName(rootEntity) + " AS " + q.getShortAlias(am, rootKey, 0))
	innerSQL, err := inner.ToQuery()
	if err != nil {
		return "", "", err
	}
	outer.JoinRaw("INNER JOIN (" + innerSQL + ") AS paginated_ids ON " + rootAlias + ".id = paginated_ids.id")
	if err := q.parseWhere(am, filter, outer, ""); err != nil {
		return "", "", err
	}
	outerParts, err := q.buildQueryParts(rootStructure, "", entity, rootKey, nil, 0, am)
	if err != nil {
		return "", "", err
	}
	for _, p := range outerParts {
		outer.JoinRaw(p)
	}
	sp := &selectParts{values: map[string]selectPart{}}
	q.buildSelectParts(get(selectOnlyStructure, rootKey), rootKey, am, nil, sp)
	parts := make([]selectPart, 0, len(sp.keys))
	for _, k := range sp.keys {
		parts = append(parts, sp.values[k])
	}
	outer.SelectAliased(parts)
	sql, err = outer.ToQuery()
	return sql, sqlCount, err
}

// BuildObjectFromResultset mirrors buildObjectFromResultset over rows
// whose columns are the select aliases.
func (q *QueryBuilder) BuildObjectFromResultset(columns []string, rows [][]any) []any {
	structure := q.structure
	rootKeys := q.getStructureKeys(structure)
	if len(rootKeys) == 0 {
		return []any{}
	}
	rootKey := rootKeys[0]

	type pathDetail struct {
		property   string
		parents    []string
		parentPath string
	}
	maps := map[string]*dal.Record{}
	var mapOrder []string
	isListMap := map[string]bool{}
	referenceMap := map[string]bool{}
	pathDetails := map[string]*pathDetail{}

	var initializeMaps func(structure any, path []string)
	initializeMaps = func(structure any, path []string) {
		currentPath := strings.Join(path, ".")
		entity, _ := q.getEntity(currentPath, false)
		if entity == nil {
			return
		}
		if _, ok := maps[currentPath]; !ok {
			mapOrder = append(mapOrder, currentPath)
		}
		maps[currentPath] = dal.NewRecord()
		if len(path) > 1 {
			property := path[len(path)-1]
			parents := append([]string(nil), path[:len(path)-1]...)
			parentPath := strings.Join(parents, ".")
			if q.allSchemaFields[property] {
				pathDetails[currentPath] = &pathDetail{property, parents, parentPath}
				isListMap[currentPath] = false
				return
			}
			isList := false
			for _, p := range entity.Ref.Parents {
				if p.TargetProp == property {
					isList = p.IsList
					break
				}
			}
			isListMap[currentPath] = isList
			pathDetails[currentPath] = &pathDetail{property, parents, parentPath}
		}
		rec, _ := structure.(*dal.Record)
		for _, key := range q.getStructureKeys(structure) {
			initializeMaps(get(rec, key), append(append([]string(nil), path...), key))
		}
	}
	initializeMaps(get(structure, rootKey), []string{rootKey})

	colIndex := map[string]int{}
	for i, c := range columns {
		colIndex[c] = i
	}
	type column struct{ field, property string }
	columnMap := map[string][]column{}
	for _, property := range columns {
		segs := strings.Split(property, ".")
		field := segs[len(segs)-1]
		parent := strings.Join(segs[:len(segs)-1], ".")
		columnMap[parent] = append(columnMap[parent], column{field, property})
	}
	cell := func(row []any, name string) any {
		if i, ok := colIndex[name]; ok {
			return row[i]
		}
		return nil
	}
	idKey := func(v any) string { return jsString(v) }

	for _, row := range rows {
		for _, path := range mapOrder {
			id := cell(row, path+".id")
			detail := pathDetails[path]
			if detail == nil {
				if _, ok := maps[path].Get(idKey(id)); !ok {
					var obj any = cell(row, path)
					if obj == nil && id != nil {
						r := dal.NewRecord()
						for _, c := range columnMap[path] {
							r.Set(c.field, cell(row, c.property))
						}
						obj = r
					}
					maps[path].Set(idKey(id), obj)
				}
				continue
			}
			key := ""
			current := ""
			for _, p := range detail.parents {
				current += p
				key += jsString(cell(row, current+".id")) + "."
				current += "."
			}
			key += jsString(id)
			if referenceMap[key] {
				continue
			}
			var obj any = cell(row, path)
			if obj == nil && id != nil {
				r := dal.NewRecord()
				for _, c := range columnMap[path] {
					r.Set(c.field, cell(row, c.property))
				}
				obj = r
			}
			maps[path].Set(idKey(id), obj)
			parentObj, _ := get(maps[detail.parentPath], jsString(cell(row, detail.parentPath+".id"))).(*dal.Record)
			if parentObj == nil {
				continue
			}
			isList := isListMap[detail.parentPath+"."+detail.property]
			if isList {
				if _, ok := get(parentObj, detail.property).([]any); !ok {
					parentObj.Set(detail.property, []any{})
				}
			}
			if obj != nil {
				if isList {
					list, _ := get(parentObj, detail.property).([]any)
					parentObj.Set(detail.property, append(list, obj))
				} else {
					parentObj.Set(detail.property, obj)
				}
			}
			referenceMap[key] = true
		}
	}
	root := maps[rootKey]
	out := []any{}
	if root == nil {
		return out
	}
	for _, k := range root.Keys() {
		v, _ := root.Get(k)
		out = append(out, v)
	}
	return out
}
