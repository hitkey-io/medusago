package schema

import (
	"regexp"
	"strings"
)

// This file mirrors SchemaHelper with the PostgreSqlSchemaHelper overrides.

// createSchemaBuilder mirrors SchemaHelper.createSchemaBuilder.
func createSchemaBuilder(schema string) *schemaBuilder {
	if schema != "" && schema != DefaultSchemaName {
		return newSchemaBuilder(schema)
	}
	return newSchemaBuilder("")
}

// splitTableName mirrors SchemaHelper.splitTableName.
func splitTableName(name string) (schemaName, tableName string) {
	parts := strings.Split(name, ".")
	tableName = parts[len(parts)-1]
	if len(parts) > 1 {
		schemaName = parts[len(parts)-2]
	}
	return
}

// getReferencedTableName mirrors SchemaHelper.getReferencedTableName.
func getReferencedTableName(referencedTableName, schema string) string {
	schemaName, tableName := splitTableName(referencedTableName)
	if schemaName != "" {
		schema = schemaName
	}
	if schema == "" {
		schema = DefaultSchemaName
	}
	if schemaName == "*" {
		return schema + "." + strings.TrimPrefix(referencedTableName, "*.")
	}
	if schemaName == "" || schemaName == DefaultSchemaName {
		return tableName
	}
	return schemaName + "." + tableName
}

// hasNonDefaultPrimaryKeyName mirrors SchemaHelper.hasNonDefaultPrimaryKeyName.
func hasNonDefaultPrimaryKeyName(t *Table) bool {
	pk := t.PrimaryKey()
	if pk == nil {
		return false
	}
	return pk.KeyName != DefaultPrimaryName(t.Name)
}

// createTableColumn mirrors PostgreSqlSchemaHelper.createTableColumn (no
// native enums, no generated columns). changed is nil when the table is
// created or the column added.
func createTableColumn(table *tableBuilder, column *Column, fromTable *Table, changed map[string]bool, alter bool) *columnBuilder {
	pk := fromTable.PrimaryKey()
	composite := pk != nil && pk.Composite
	primaryKey := column.Primary && changed == nil && !hasNonDefaultPrimaryKeyName(fromTable)
	if column.Autoincrement && column.Generated == "" && !composite && changed == nil {
		if column.MappedType == TypeBigInt {
			return table.bigIncrements(column.Name, primaryKey)
		}
		return table.increments(column.Name, primaryKey)
	}
	allStrings := column.EnumItems != nil
	if changed != nil && column.MappedType == TypeEnum && allStrings {
		checkName := namingIndexName(fromTable.Name, []string{column.Name}, "check")
		if changed["enumItems"] {
			items := make([]string, len(column.EnumItems))
			for i, v := range column.EnumItems {
				items[i] = QuoteValue(v)
			}
			table.check(QuoteIdentifier(column.Name)+" in ("+strings.Join(items, ", ")+")", QuoteIdentifier(checkName))
		}
		switch {
		case changed["type"]:
			return table.specificType(column.Name, column.Type)
		case changed["default"]:
			// defaultTo(column.default) is replaced by the raw default of
			// configureColumn
			return table.specificType(column.Name, column.Type)
		case changed["nullable"]:
			c := table.specificType(column.Name, column.Type)
			if column.Nullable {
				return c.nullable()
			}
			return c.notNullable()
		}
		return nil
	}
	if column.MappedType == TypeEnum && allStrings {
		return table.enu(column.Name, column.EnumItems)
	}
	if changed != nil && strings.Contains(column.Type, "serial") {
		column.Type = strings.Replace(column.Type, "serial", "int", 1)
	}
	return table.specificType(column.Name, column.Type)
}

// configureColumn mirrors PostgreSqlSchemaHelper.configureColumn.
func configureColumn(column *Column, col *columnBuilder, changed map[string]bool) {
	guard := func(key string) bool { return changed == nil || changed[key] }
	if column.Nullable && guard("nullable") {
		col.nullable()
	}
	if !column.Nullable && guard("nullable") {
		col.notNullable()
	}
	if column.Comment != nil && *column.Comment != "" && changed == nil {
		col.comment(*column.Comment)
	}
	configureColumnDefault(column, col, changed)
}

// configureColumnDefault mirrors SchemaHelper.configureColumnDefault.
func configureColumnDefault(column *Column, col *columnBuilder, changed map[string]bool) {
	if changed != nil {
		if changed["default"] {
			col.defaultToRaw(column.Default)
		}
		return
	}
	if column.Default != nil && *column.Default != "null" {
		col.defaultToRaw(column.Default)
	}
}

// createTable mirrors SchemaHelper.createTable.
func createTable(tableDef *Table) *schemaBuilder {
	return createSchemaBuilder(tableDef.Schema).createTable(tableDef.Name, func(table *tableBuilder) {
		for _, column := range tableDef.Columns() {
			col := createTableColumn(table, column, tableDef, nil, true)
			configureColumn(column, col, nil)
		}
		for _, index := range tableDef.GetIndexes() {
			createPrimary := true
			for _, c := range tableDef.Columns() {
				if c.Autoincrement && c.Primary {
					createPrimary = false
				}
			}
			createPrimary = createPrimary || hasNonDefaultPrimaryKeyName(tableDef)
			createIndex(table, index, tableDef, createPrimary)
		}
		for _, check := range tableDef.Checks {
			table.check(check.Expression, check.Name)
		}
		if tableDef.Comment != nil && *tableDef.Comment != "" {
			table.setComment(strings.TrimSuffix(strings.TrimPrefix(QuoteValue(*tableDef.Comment), "'"), "'"))
		}
	})
}

// createForeignKey mirrors SchemaHelper.createForeignKey
// (createForeignKeyConstraints is true).
func createForeignKey(table *tableBuilder, fk *ForeignKey, schema string) {
	fd := table.foreign(fk.ColumnNames, fk.ConstraintName)
	fd.references = fk.ReferencedColumnNames
	fd.inTable = getReferencedTableName(fk.ReferencedTableName, schema)
	fd.keyName = fk.ConstraintName
	// supportsMultipleCascadePaths: true on PostgreSQL
	if fk.UpdateRule != "" {
		fd.onUpdate = fk.UpdateRule
	}
	if fk.DeleteRule != "" {
		fd.onDelete = fk.DeleteRule
	}
	if fk.DeferMode != "" {
		fd.deferrable = fk.DeferMode
	}
}

// createIndex mirrors SchemaHelper.createIndex.
func createIndex(table *tableBuilder, index *Index, tableDef *Table, createPrimary bool) {
	if index.Primary && !createPrimary {
		return
	}
	switch {
	case index.Expression != "":
		table.rawQuery(index.Expression)
	case index.Primary:
		keyName := ""
		if hasNonDefaultPrimaryKeyName(tableDef) {
			keyName = index.KeyName
		}
		table.primary(index.ColumnNames, keyName)
	case index.Unique:
		table.unique(index.ColumnNames, index.KeyName, index.DeferMode)
	case index.Type == "fulltext":
		cols := make([]string, len(index.ColumnNames))
		for i, c := range index.ColumnNames {
			cols[i] = QuoteIdentifier(c)
		}
		name := tableDef.Name
		if tableDef.Schema != "" {
			name = tableDef.Schema + "." + tableDef.Name
		}
		if len(index.ColumnNames) == 1 && tableDef.Column(index.ColumnNames[0]) != nil && tableDef.Column(index.ColumnNames[0]).Type == "tsvector" {
			table.rawQuery("create index " + QuoteIdentifier(index.KeyName) + " on " + QuoteIdentifier(name) + " using gin(" + cols[0] + ")")
			return
		}
		table.rawQuery("create index " + QuoteIdentifier(index.KeyName) + " on " + QuoteIdentifier(name) + " using gin(to_tsvector('simple', " + strings.Join(cols, " || ' ' || ") + "))")
	default:
		table.index(index.ColumnNames, index.KeyName, index.Type)
	}
}

// createCheck mirrors SchemaHelper.createCheck.
func createCheck(table *tableBuilder, check *Check) {
	table.check(check.Expression, check.Name)
}

// getPreAlterTable mirrors PostgreSqlSchemaHelper.getPreAlterTable.
func getPreAlterTable(diff *TableDifference) string {
	var ret []string
	schemaName, tableName := splitTableName(diff.Name)
	name := tableName
	if schemaName != "" && schemaName != DefaultSchemaName {
		name = schemaName + "." + tableName
	}
	quoted := QuoteIdentifier(name)
	for _, col := range diff.ChangedColumns.values() {
		if col.FromColumn.MappedType != TypeEnum {
			continue
		}
		if col.FromColumn.NativeEnumName == "" && col.Column.NativeEnumName != "" && col.FromColumn.Default != nil && *col.FromColumn.Default != "" {
			ret = append(ret, "alter table "+quoted+` alter column "`+col.Column.Name+`" drop default`)
		}
		if col.FromColumn.NativeEnumName != "" && col.Column.NativeEnumName == "" && col.FromColumn.Default != nil && *col.FromColumn.Default != "" {
			ret = append(ret, "alter table "+quoted+` alter column "`+col.Column.Name+`" drop default`)
		}
		if col.FromColumn.NativeEnumName == "" && (col.ChangedProperties["enumItems"] || col.Column.NativeEnumName != "") {
			ret = append(ret, "alter table "+quoted+` drop constraint if exists "`+tableName+"_"+col.Column.Name+`_check"`)
		}
	}
	for _, col := range diff.ChangedColumns.values() {
		if col.ChangedProperties["type"] && col.FromColumn.Type == "uuid" {
			ret = append(ret, "alter table "+quoted+` alter column "`+col.Column.Name+`" type text using ("`+col.Column.Name+`"::text)`)
		}
	}
	return strings.Join(ret, ";\n")
}

// getPostAlterTable mirrors PostgreSqlSchemaHelper.getPostAlterTable.
func getPostAlterTable(diff *TableDifference) string {
	var ret []string
	schemaName, tableName := splitTableName(diff.Name)
	name := tableName
	if schemaName != "" && schemaName != DefaultSchemaName {
		name = schemaName + "." + tableName
	}
	quoted := QuoteIdentifier(name)
	for _, col := range diff.ChangedColumns.values() {
		if col.FromColumn.MappedType != TypeEnum {
			continue
		}
		hasDefault := col.Column.Default != nil && *col.Column.Default != ""
		if col.FromColumn.NativeEnumName == "" && col.Column.NativeEnumName != "" && hasDefault {
			ret = append(ret, "alter table "+quoted+` alter column "`+col.Column.Name+`" set default `+*col.Column.Default)
		}
		if col.FromColumn.NativeEnumName != "" && col.Column.NativeEnumName == "" && hasDefault {
			ret = append(ret, "alter table "+quoted+` alter column "`+col.Column.Name+`" set default `+*col.Column.Default)
		}
	}
	return strings.Join(ret, ";\n")
}

// getAlterColumnAutoincrement mirrors
// PostgreSqlSchemaHelper.getAlterColumnAutoincrement.
func getAlterColumnAutoincrement(tableName string, column *Column, schemaName string) string {
	var ret []string
	name := tableName
	if schemaName != "" && schemaName != DefaultSchemaName {
		name = schemaName + "." + tableName
	}
	if column.Autoincrement {
		seq := IndexName(tableName, []string{column.Name}, "sequence")
		ret = append(ret, "create sequence if not exists "+QuoteIdentifier(seq))
		ret = append(ret, "select setval('"+seq+"', (select max("+QuoteIdentifier(column.Name)+") from "+QuoteIdentifier(name)+"))")
		ret = append(ret, "alter table "+QuoteIdentifier(name)+" alter column "+QuoteIdentifier(column.Name)+" set default nextval('"+seq+"')")
	} else if column.Default == nil {
		ret = append(ret, "alter table "+QuoteIdentifier(name)+" alter column "+QuoteIdentifier(column.Name)+" drop default")
	}
	return strings.Join(ret, ";\n")
}

// getChangeColumnCommentSQL mirrors
// PostgreSqlSchemaHelper.getChangeColumnCommentSQL.
func getChangeColumnCommentSQL(tableName string, to *Column, schemaName string) string {
	n := tableName
	if schemaName != "" && schemaName != DefaultSchemaName {
		n = schemaName + "." + tableName
	}
	value := "null"
	if to.Comment != nil && *to.Comment != "" {
		value = QuoteValue(*to.Comment)
	}
	return "comment on column " + QuoteIdentifier(n) + `."` + to.Name + `" is ` + value
}

// getRenameColumnSQL mirrors SchemaHelper.getRenameColumnSQL.
func getRenameColumnSQL(tableName, oldColumnName string, to *Column, schemaName string) string {
	ref := ""
	if schemaName != "" && schemaName != DefaultSchemaName {
		ref = `"` + schemaName + `".`
	}
	return "alter table " + ref + QuoteIdentifier(tableName) + " rename column " + QuoteIdentifier(oldColumnName) + " to " + QuoteIdentifier(to.Name)
}

// getRenameIndexSQL mirrors PostgreSqlSchemaHelper.getRenameIndexSQL.
func getRenameIndexSQL(index *Index, oldIndexName string) string {
	return "alter index " + QuoteIdentifier(oldIndexName) + " rename to " + QuoteIdentifier(index.KeyName)
}

// getDropColumnsSQL mirrors SchemaHelper.getDropColumnsSQL.
func getDropColumnsSQL(tableName string, columns []*Column, schemaName string) string {
	n := tableName
	if schemaName != "" && schemaName != DefaultSchemaName {
		n = schemaName + "." + tableName
	}
	drops := make([]string, len(columns))
	for i, c := range columns {
		drops[i] = "drop column " + QuoteIdentifier(c.Name)
	}
	return "alter table " + QuoteIdentifier(n) + " " + strings.Join(drops, ", ")
}

var (
	pragmaRe    = regexp.MustCompile(`(?i)pragma table_.+`)
	blankLineRe = regexp.MustCompile(`\n\n+`)
)

// dump mirrors SchemaHelper.dump for a builder.
func dump(b *schemaBuilder, appendStr string) string {
	queries := b.statements()
	if len(queries) == 0 {
		return ""
	}
	d := strings.Join(queries, ";\n") + ";" + appendStr
	tmp := strings.TrimSpace(blankLineRe.ReplaceAllString(pragmaRe.ReplaceAllString(d, ""), "\n"))
	if tmp == "" {
		return ""
	}
	return tmp + appendStr
}

// dumpString mirrors SchemaHelper.dump for a string.
func dumpString(sql, appendStr string) string {
	if sql == "" {
		return ""
	}
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	return sql + appendStr
}
