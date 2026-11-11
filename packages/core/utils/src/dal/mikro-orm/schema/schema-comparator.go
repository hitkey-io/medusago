package schema

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
)

// omap is a JavaScript object used as a map: string keys in insertion
// order.
type omap[V any] struct {
	keys []string
	m    map[string]V
}

func (o *omap[V]) set(k string, v V) {
	if o.m == nil {
		o.m = map[string]V{}
	}
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

func (o *omap[V]) get(k string) (V, bool) {
	v, ok := o.m[k]
	return v, ok
}

func (o *omap[V]) has(k string) bool {
	_, ok := o.m[k]
	return ok
}

func (o *omap[V]) del(k string) {
	if _, ok := o.m[k]; !ok {
		return
	}
	delete(o.m, k)
	for i, x := range o.keys {
		if x == k {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			return
		}
	}
}

func (o *omap[V]) values() []V {
	out := make([]V, len(o.keys))
	for i, k := range o.keys {
		out[i] = o.m[k]
	}
	return out
}

func (o *omap[V]) entries() []string { return append([]string(nil), o.keys...) }

func (o *omap[V]) len() int { return len(o.keys) }

// ColumnDifference mirrors the changedColumns entries.
type ColumnDifference struct {
	OldColumnName     string
	FromColumn        *Column
	Column            *Column
	ChangedProperties map[string]bool
}

// TableDifference mirrors MikroORM's TableDifference.
type TableDifference struct {
	Name               string
	AddedColumns       omap[*Column]
	AddedForeignKeys   omap[*ForeignKey]
	AddedIndexes       omap[*Index]
	AddedChecks        omap[*Check]
	ChangedColumns     omap[*ColumnDifference]
	ChangedForeignKeys omap[*ForeignKey]
	ChangedIndexes     omap[*Index]
	ChangedChecks      omap[*Check]
	RemovedColumns     omap[*Column]
	RemovedForeignKeys omap[*ForeignKey]
	RemovedIndexes     omap[*Index]
	RemovedChecks      omap[*Check]
	RenamedColumns     omap[*Column]
	RenamedIndexes     omap[*Index]
	HasChangedComment  bool
	ChangedComment     *string
	FromTable          *Table
	ToTable            *Table
}

// SchemaDifference mirrors MikroORM's SchemaDifference.
type SchemaDifference struct {
	NewTables           omap[*Table]
	RemovedTables       omap[*Table]
	ChangedTables       omap[*TableDifference]
	OrphanedForeignKeys []*ForeignKey
	NewNamespaces       []string
	RemovedNamespaces   []string
	FromSchema          *DatabaseSchema
}

// Compare mirrors SchemaComparator.compare: the operations that change
// fromSchema into toSchema.
func Compare(fromSchema, toSchema *DatabaseSchema, inverseDiff *SchemaDifference) *SchemaDifference {
	diff := &SchemaDifference{FromSchema: fromSchema}
	foreignKeysToTable := omap[[]*ForeignKey]{}
	for _, ns := range toSchema.Namespaces() {
		if fromSchema.HasNamespace(ns) || ns == DefaultSchemaName {
			continue
		}
		if !hasString(diff.NewNamespaces, ns) {
			diff.NewNamespaces = append(diff.NewNamespaces, ns)
		}
	}
	for _, ns := range fromSchema.Namespaces() {
		if toSchema.HasNamespace(ns) || ns == DefaultSchemaName {
			continue
		}
		if !hasString(diff.RemovedNamespaces, ns) {
			diff.RemovedNamespaces = append(diff.RemovedNamespaces, ns)
		}
	}
	for _, table := range toSchema.Tables {
		name := table.ShortestName()
		if !fromSchema.HasTable(name) {
			diff.NewTables.set(name, toSchema.Table(name))
			continue
		}
		var inverse *TableDifference
		if inverseDiff != nil {
			inverse, _ = inverseDiff.ChangedTables.get(name)
		}
		if td := diffTable(fromSchema.Table(name), toSchema.Table(name), inverse); td != nil {
			diff.ChangedTables.set(name, td)
		}
	}
	for _, table := range fromSchema.Tables {
		name := table.ShortestName()
		table = fromSchema.Table(name)
		if !toSchema.HasTable(name) {
			diff.RemovedTables.set(name, table)
		}
		for _, fk := range table.ForeignKeys() {
			list, _ := foreignKeysToTable.get(fk.ReferencedTableName)
			foreignKeysToTable.set(fk.ReferencedTableName, append(list, fk))
		}
	}
	for _, table := range diff.RemovedTables.values() {
		name := table.Name
		if table.Schema != "" {
			name = table.Schema + "." + table.Name
		}
		fks, ok := foreignKeysToTable.get(name)
		if !ok {
			continue
		}
		diff.OrphanedForeignKeys = append(diff.OrphanedForeignKeys, fks...)
		for _, fk := range fks {
			changed, ok := diff.ChangedTables.get(fk.LocalTableName)
			if !ok {
				continue
			}
			for _, key := range changed.RemovedForeignKeys.entries() {
				removed, _ := changed.RemovedForeignKeys.get(key)
				if name != removed.ReferencedTableName {
					continue
				}
				changed.RemovedForeignKeys.del(key)
			}
		}
	}
	return diff
}

// diffTable mirrors SchemaComparator.diffTable (nil when nothing changed).
func diffTable(fromTable, toTable *Table, inverse *TableDifference) *TableDifference {
	changes := 0
	td := &TableDifference{Name: fromTable.ShortestName(), FromTable: fromTable, ToTable: toTable}
	if diffComment(fromTable.Comment, toTable.Comment) {
		td.HasChangedComment = true
		td.ChangedComment = toTable.Comment
		changes++
	}
	for _, column := range toTable.Columns() {
		if fromTable.HasColumn(column.Name) {
			continue
		}
		td.AddedColumns.set(column.Name, column)
		changes++
	}
	for _, column := range fromTable.Columns() {
		if !toTable.HasColumn(column.Name) {
			td.RemovedColumns.set(column.Name, column)
			changes++
			continue
		}
		changed := diffColumn(column, toTable.Column(column.Name), fromTable)
		if len(changed) == 0 {
			continue
		}
		if len(changed) == 1 && changed["generated"] {
			td.AddedColumns.set(column.Name, toTable.Column(column.Name))
			td.RemovedColumns.set(column.Name, column)
			changes++
			continue
		}
		td.ChangedColumns.set(column.Name, &ColumnDifference{
			OldColumnName:     column.Name,
			FromColumn:        column,
			Column:            toTable.Column(column.Name),
			ChangedProperties: changed,
		})
		changes++
	}
	detectColumnRenamings(td, inverse)

	fromIndexes := fromTable.GetIndexes()
	toIndexes := toTable.GetIndexes()
	fromHasPrimary := false
	for _, i := range fromIndexes {
		if i.Primary {
			fromHasPrimary = true
		}
	}
	for _, index := range toIndexes {
		if (index.Primary && fromHasPrimary) || fromTable.HasIndex(index.KeyName) {
			continue
		}
		td.AddedIndexes.set(index.KeyName, index)
		changes++
	}
	for _, index := range fromIndexes {
		if (index.Primary && toTable.PrimaryKey() == nil) || (!index.Primary && !toTable.HasIndex(index.KeyName)) {
			td.RemovedIndexes.set(index.KeyName, index)
			changes++
			continue
		}
		var toIndex *Index
		if index.Primary {
			toIndex = toTable.PrimaryKey()
		} else {
			toIndex = toTable.Index(index.KeyName)
		}
		if !diffIndex(index, toIndex) {
			continue
		}
		td.ChangedIndexes.set(index.KeyName, toIndex)
		changes++
	}
	detectIndexRenamings(td)

	for _, check := range toTable.Checks {
		if fromTable.Check(check.Name) != nil {
			continue
		}
		td.AddedChecks.set(check.Name, check)
		changes++
	}
	for _, check := range fromTable.Checks {
		to := toTable.Check(check.Name)
		if to == nil {
			td.RemovedChecks.set(check.Name, check)
			changes++
			continue
		}
		if !diffExpression(strPtr(check.Expression), strPtr(to.Expression)) {
			continue
		}
		td.ChangedChecks.set(check.Name, to)
		changes++
	}

	fromFKs := omap[*ForeignKey]{}
	for _, fk := range fromTable.ForeignKeys() {
		fromFKs.set(fk.ConstraintName, fk)
	}
	toFKs := omap[*ForeignKey]{}
	for _, fk := range toTable.ForeignKeys() {
		toFKs.set(fk.ConstraintName, fk)
	}
	for _, from := range fromFKs.values() {
		for _, to := range toFKs.values() {
			if !diffForeignKey(from, to, td) {
				fromFKs.del(from.ConstraintName)
				toFKs.del(to.ConstraintName)
			} else if strings.EqualFold(from.ConstraintName, to.ConstraintName) {
				td.ChangedForeignKeys.set(to.ConstraintName, to)
				changes++
				fromFKs.del(from.ConstraintName)
				toFKs.del(to.ConstraintName)
			}
		}
	}
	for _, from := range fromFKs.values() {
		td.RemovedForeignKeys.set(from.ConstraintName, from)
		changes++
	}
	for _, to := range toFKs.values() {
		td.AddedForeignKeys.set(to.ConstraintName, to)
		changes++
	}
	if changes == 0 {
		return nil
	}
	return td
}

func detectColumnRenamings(td *TableDifference, inverse *TableDifference) {
	type pair struct{ removed, added *Column }
	candidates := omap[[]pair]{}
	oldFKs := td.FromTable.ForeignKeys()
	newFKs := td.ToTable.ForeignKeys()
	for _, added := range td.AddedColumns.values() {
		for _, removed := range td.RemovedColumns.values() {
			if len(diffColumn(added, removed, td.FromTable)) != 0 {
				continue
			}
			wasFK, isFK := false, false
			for _, fk := range oldFKs {
				if hasString(fk.ColumnNames, removed.Name) {
					wasFK = true
				}
			}
			for _, fk := range newFKs {
				if hasString(fk.ColumnNames, added.Name) {
					isFK = true
				}
			}
			if wasFK != isFK {
				continue
			}
			if inverse != nil {
				if renamed, ok := inverse.RenamedColumns.get(added.Name); ok && renamed.Name != removed.Name {
					continue
				}
			}
			list, _ := candidates.get(added.Name)
			candidates.set(added.Name, append(list, pair{removed, added}))
		}
	}
	for _, list := range candidates.values() {
		if len(list) != 1 {
			continue
		}
		removed, added := list[0].removed, list[0].added
		if td.RenamedColumns.has(removed.Name) {
			continue
		}
		td.RenamedColumns.set(removed.Name, added)
		td.AddedColumns.del(added.Name)
		td.RemovedColumns.del(removed.Name)
	}
}

func detectIndexRenamings(td *TableDifference) {
	type pair struct{ removed, added *Index }
	candidates := omap[[]pair]{}
	for _, added := range td.AddedIndexes.values() {
		for _, removed := range td.RemovedIndexes.values() {
			if diffIndex(added, removed) {
				continue
			}
			list, _ := candidates.get(added.KeyName)
			candidates.set(added.KeyName, append(list, pair{removed, added}))
		}
	}
	for _, list := range candidates.values() {
		if len(list) != 1 {
			continue
		}
		removed, added := list[0].removed, list[0].added
		if td.RenamedIndexes.has(removed.KeyName) {
			continue
		}
		td.RenamedIndexes.set(removed.KeyName, added)
		td.AddedIndexes.del(added.KeyName)
		td.RemovedIndexes.del(removed.KeyName)
	}
}

func diffForeignKey(key1, key2 *ForeignKey, td *TableDifference) bool {
	if strings.ToLower(strings.Join(key1.ColumnNames, "~")) != strings.ToLower(strings.Join(key2.ColumnNames, "~")) {
		return true
	}
	if strings.ToLower(strings.Join(key1.ReferencedColumnNames, "~")) != strings.ToLower(strings.Join(key2.ReferencedColumnNames, "~")) {
		return true
	}
	if key1.ConstraintName != key2.ConstraintName || key1.ReferencedTableName != key2.ReferencedTableName || key1.DeferMode != key2.DeferMode {
		return true
	}
	// supportsMultipleCascadePaths: true on PostgreSQL
	for _, col := range key1.ColumnNames {
		if cd, ok := td.ChangedColumns.get(col); ok && cd.ChangedProperties["type"] {
			return true
		}
	}
	rule := func(v string) string {
		if v == "" {
			v = "restrict"
		}
		v = strings.Replace(strings.ToLower(v), "no action", "restrict", 1)
		return strings.ReplaceAll(v, `"`, "")
	}
	return rule(key1.UpdateRule) != rule(key2.UpdateRule) || rule(key1.DeleteRule) != rule(key2.DeleteRule)
}

// mapColumnToProperty mirrors SchemaComparator.mapColumnToProperty (with
// autoincrement false, like diffColumn passes the columns).
var (
	columnLengthRe = regexp.MustCompile(`\w+\((\d+)\)`)
)

func mapColumnToProperty(c *Column) columnProp {
	p := columnProp{
		columnTypes:    []string{c.Type},
		length:         c.Length,
		precision:      c.Precision,
		scale:          c.Scale,
		items:          c.EnumItems,
		nativeEnumName: c.NativeEnumName,
	}
	if m := columnLengthRe.FindStringSubmatch(c.Type); m != nil {
		p.length = intPtr(atoi(m[1]))
	}
	if m := precisionScaleR.FindStringSubmatch(c.Type); m != nil {
		p.precision, p.scale = intPtr(atoi(m[1])), intPtr(atoi(m[2]))
	}
	return p
}

// diffColumn mirrors SchemaComparator.diffColumn: the changed properties.
func diffColumn(from, to *Column, fromTable *Table) map[string]bool {
	changed := map[string]bool{}
	fromProp := mapColumnToProperty(from)
	toProp := mapColumnToProperty(to)
	fromType := normalizeColumnType(strings.ToLower(columnTypeOf(from.MappedType, fromProp)), fromProp)
	toType := normalizeColumnType(strings.ToLower(columnTypeOf(to.MappedType, toProp)), toProp)
	ignored := func(key string) bool {
		return hasString(from.IgnoreSchemaChanges, key) || hasString(to.IgnoreSchemaChanges, key)
	}
	// native enums are not supported: fromNativeEnum is always undefined
	if fromType != toType && !ignored("type") && from.Generated == "" && to.Generated == "" {
		if !strings.Contains(toType, ".") && fromTable.Schema != "" && fromTable.Schema != DefaultSchemaName {
			toType = fromTable.Schema + "." + toType
		}
		if fromType != toType {
			changed["type"] = true
		}
	}
	if from.Nullable != to.Nullable && from.Generated == "" && to.Generated == "" {
		changed["nullable"] = true
	}
	if diffExpression(optString(from.Generated), optString(to.Generated)) {
		changed["generated"] = true
	}
	if from.Autoincrement != to.Autoincrement {
		changed["autoincrement"] = true
	}
	// unsigned: not supported by PostgreSQL
	if !ignored("default") && !hasSameDefaultValue(from, to) {
		changed["default"] = true
	}
	if diffComment(from.Comment, to.Comment) {
		changed["comment"] = true
	}
	if from.MappedType != TypeArray && to.MappedType != TypeArray && diffEnumItems(from.EnumItems, to.EnumItems) {
		changed["enumItems"] = true
	}
	if strings.ToLower(from.Extra) != strings.ToLower(to.Extra) && !ignored("extra") {
		changed["extra"] = true
	}
	return changed
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func diffEnumItems(a, b []string) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

func diffComment(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a != nil && b != nil {
		return *a != *b
	}
	if a == nil && *b == "" {
		return false
	}
	if b == nil && *a == "" {
		return false
	}
	return true
}

// diffIndex mirrors SchemaComparator.diffIndex.
func diffIndex(a, b *Index) bool {
	if a.Expression != "" || b.Expression != "" || a.Type == "fulltext" || b.Type == "fulltext" {
		return a.KeyName != b.KeyName
	}
	return !isIndexFulfilledBy(a, b) || !isIndexFulfilledBy(b, a)
}

func isIndexFulfilledBy(a, b *Index) bool {
	if len(a.ColumnNames) != len(b.ColumnNames) {
		return false
	}
	for i := range a.ColumnNames {
		if a.ColumnNames[i] != b.ColumnNames[i] {
			return false
		}
	}
	if !a.Unique && !a.Primary {
		return true
	}
	return a.Primary == b.Primary && a.Unique == b.Unique && a.DeferMode == b.DeferMode
}

var (
	simplifyPrefixRe = regexp.MustCompile(`_\w+'(.*?)'`)
	simplifyCharsRe  = regexp.MustCompile("['\"`()\\n\\[\\]]|::\\w+| +")
)

// diffExpression mirrors SchemaComparator.diffExpression.
func diffExpression(a, b *string) bool {
	simplify := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := simplifyPrefixRe.ReplaceAllString(*s, "$1")
		v = strings.ToLower(simplifyCharsRe.ReplaceAllString(v, ""))
		return &v
	}
	sa, sb := simplify(a), simplify(b)
	if sa == nil || sb == nil {
		return sa != sb
	}
	return *sa != *sb
}

var (
	jsonDefaultPrefixRe = regexp.MustCompile(`^(_\w+\\)?'(.*?)\\?'$`)
	jsonDefaultQuoteRe  = regexp.MustCompile(`^\(?'(.*?)'\)?$`)
)

func parseJSONDefault(def string) any {
	if def == "" {
		return nil
	}
	v := jsonDefaultPrefixRe.ReplaceAllString(def, "$2")
	v = jsonDefaultQuoteRe.ReplaceAllString(v, "$1")
	var out any
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return v
	}
	return out
}

var falsyDefaults = []string{"0", "false", "f", "n", "no", "off"}

// hasSameDefaultValue mirrors SchemaComparator.hasSameDefaultValue.
func hasSameDefaultValue(from, to *Column) bool {
	if from.Default == nil || strings.ToLower(*from.Default) == "null" || strings.HasPrefix(*from.Default, "nextval(") {
		return to.Default == nil || strings.ToLower(*to.Default) == "null"
	}
	if to.MappedType == TypeBoolean {
		toDefault := "undefined"
		if to.Default != nil {
			toDefault = *to.Default
		}
		return !hasString(falsyDefaults, *from.Default) == !hasString(falsyDefaults, toDefault)
	}
	if to.MappedType == TypeJSON {
		toDefault := ""
		if to.Default != nil {
			toDefault = *to.Default
		}
		return reflect.DeepEqual(parseJSONDefault(*from.Default), parseJSONDefault(toDefault))
	}
	if to.MappedType == TypeDateTime && *from.Default != "" && to.Default != nil && *to.Default != "" {
		norm := func(s string) string {
			s = strings.Replace(strings.ToLower(s), "current_timestamp", "now", 1)
			return strings.TrimSuffix(s, "()")
		}
		return norm(*from.Default) == norm(*to.Default)
	}
	if *from.Default != "" && to.Default != nil && *to.Default != "" {
		return strings.EqualFold(*from.Default, *to.Default)
	}
	if to.Default != nil && (*to.Default == "" || *to.Default == "''") {
		return *from.Default == "" || *from.Default == "''"
	}
	// from.default == to.default (loose): to.default is null here
	return to.Default != nil && *from.Default == *to.Default
}
