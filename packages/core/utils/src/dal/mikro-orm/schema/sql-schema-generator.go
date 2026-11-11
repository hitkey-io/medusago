package schema

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// OrderedMetadata mirrors AbstractSchemaGenerator.getOrderedMetadata: the
// entities (in MetadataStorage.getAll() order) sorted by the
// CommitOrderCalculator so referenced tables come first.
func OrderedMetadata(metadata []*dml.SchemaEntity) []*dml.SchemaEntity {
	calc := newCommitOrderCalculator()
	byName := map[string]*dml.SchemaEntity{}
	for _, meta := range metadata {
		calc.addNode(meta.ClassName)
		byName[meta.ClassName] = meta
	}
	for i := len(metadata) - 1; i >= 0; i-- {
		meta := metadata[i]
		for _, prop := range meta.Props {
			calc.discoverProperty(prop, meta.ClassName)
		}
	}
	var out []*dml.SchemaEntity
	for _, cls := range calc.sort() {
		out = append(out, byName[cls])
	}
	return out
}

// TargetSchema mirrors SqlSchemaGenerator.getTargetSchema for the default
// schema.
func TargetSchema(metadata []*dml.SchemaEntity) *DatabaseSchema {
	return FromMetadata(OrderedMetadata(metadata), DefaultSchemaName)
}

// MigrationDiff is the SQL of an update migration.
type MigrationDiff struct {
	Up   string
	Down string
}

// UpdateSchemaMigrationSQL mirrors
// SqlSchemaGenerator.getUpdateSchemaMigrationSQL with a fromSchema (the
// snapshot), wrap false, safe false and dropTables true.
func UpdateSchemaMigrationSQL(fromSchema *DatabaseSchema, metadata []*dml.SchemaEntity) MigrationDiff {
	toSchema := TargetSchema(metadata)
	fromSchema.prune()
	toSchema.prune()
	diffUp := Compare(fromSchema, toSchema, nil)
	diffDown := Compare(toSchema, fromSchema, diffUp)
	return MigrationDiff{Up: diffToSQL(diffUp), Down: diffToSQL(diffDown)}
}

// diffToSQL mirrors SqlSchemaGenerator.diffToSQL (safe false, dropTables
// true, createForeignKeyConstraints true).
func diffToSQL(diff *SchemaDifference) string {
	ret := ""
	for _, ns := range diff.NewNamespaces {
		ret += dumpString("create schema if not exists "+QuoteIdentifier(ns), "\n")
	}
	for _, fk := range diff.OrphanedForeignKeys {
		schemaName, tableName := splitTableName(fk.LocalTableName)
		fk := fk
		ret += dump(createSchemaBuilder(schemaName).alterTable(tableName, func(t *tableBuilder) {
			t.dropForeign(fk.ColumnNames, fk.ConstraintName)
		}), "\n\n")
	}
	for _, t := range diff.NewTables.values() {
		ret += dump(createTable(t), "\n\n")
	}
	for _, t := range diff.NewTables.values() {
		t := t
		ret += dump(createSchemaBuilder(t.Schema).alterTable(t.Name, func(table *tableBuilder) {
			createForeignKeys(table, t, "")
		}), "\n\n")
	}
	for _, t := range diff.RemovedTables.values() {
		ret += dump(dropTable(t.Name, t.Schema), "\n\n")
	}
	for _, td := range diff.ChangedTables.values() {
		for _, b := range preAlterTable(td) {
			ret += dump(b, "\n\n")
		}
	}
	for _, td := range diff.ChangedTables.values() {
		for _, b := range alterTable(td) {
			ret += dump(b, "\n\n")
		}
	}
	for _, td := range diff.ChangedTables.values() {
		for _, b := range postAlterTable(td) {
			ret += dump(b, "\n\n")
		}
	}
	for _, ns := range diff.RemovedNamespaces {
		ret += dumpString("drop schema if exists "+QuoteIdentifier(ns), "\n")
	}
	return ret
}

// dropTable mirrors SqlSchemaGenerator.dropTable (cascade statement).
func dropTable(name, schema string) *schemaBuilder {
	b := createSchemaBuilder(schema).dropTableIfExists(name)
	sql := b.statements()[0]
	return newSchemaBuilder("").raw(sql + " cascade")
}

// createForeignKeys mirrors SqlSchemaGenerator.createForeignKeys.
func createForeignKeys(table *tableBuilder, tableDef *Table, schema string) {
	for _, fk := range tableDef.ForeignKeys() {
		createForeignKey(table, fk, schema)
	}
}

// preAlterTable mirrors SqlSchemaGenerator.preAlterTable.
func preAlterTable(diff *TableDifference) []*schemaBuilder {
	var ret []*schemaBuilder
	if sql := getPreAlterTable(diff); sql != "" {
		ret = append(ret, newSchemaBuilder("").raw(sql))
	}
	schemaName, tableName := splitTableName(diff.Name)
	ret = append(ret, createSchemaBuilder(schemaName).alterTable(tableName, func(table *tableBuilder) {
		for _, fk := range diff.RemovedForeignKeys.values() {
			table.dropForeign(fk.ColumnNames, fk.ConstraintName)
		}
		for _, fk := range diff.ChangedForeignKeys.values() {
			table.dropForeign(fk.ColumnNames, fk.ConstraintName)
		}
	}))
	return ret
}

// postAlterTable mirrors SqlSchemaGenerator.postAlterTable.
func postAlterTable(diff *TableDifference) []*schemaBuilder {
	if sql := getPostAlterTable(diff); sql != "" {
		return []*schemaBuilder{newSchemaBuilder("").raw(sql)}
	}
	return nil
}

// dropIndex mirrors SqlSchemaGenerator.dropIndex.
func dropIndex(table *tableBuilder, index *Index, oldIndexName string) {
	if oldIndexName == "" {
		oldIndexName = index.KeyName
	}
	switch {
	case index.Primary:
		table.dropPrimary(oldIndexName)
	case index.Unique && index.Constraint:
		table.dropUnique(index.ColumnNames, oldIndexName)
	default:
		table.dropIndex(index.ColumnNames, oldIndexName)
	}
}

// alterTable mirrors SqlSchemaGenerator.alterTable (no native enums).
func alterTable(diff *TableDifference) []*schemaBuilder {
	schemaName, tableName := splitTableName(diff.Name)
	var ret []*schemaBuilder
	ret = append(ret, createSchemaBuilder(schemaName).alterTable(tableName, func(table *tableBuilder) {
		for _, index := range diff.RemovedIndexes.values() {
			dropIndex(table, index, "")
		}
		for _, index := range diff.ChangedIndexes.values() {
			dropIndex(table, index, "")
		}
		for _, check := range diff.RemovedChecks.values() {
			table.dropChecks(check.Name)
		}
		for _, check := range diff.ChangedChecks.values() {
			table.dropChecks(check.Name)
		}
		if removed := diff.RemovedColumns.values(); len(removed) > 0 {
			table.rawQuery(getDropColumnsSQL(tableName, removed, schemaName))
		}
	}))
	ret = append(ret, createSchemaBuilder(schemaName).alterTable(tableName, func(table *tableBuilder) {
		for _, column := range diff.AddedColumns.values() {
			col := createTableColumn(table, column, diff.FromTable, nil, true)
			configureColumn(column, col, nil)
			var fk *ForeignKey
			for _, f := range diff.AddedForeignKeys.values() {
				if len(f.ColumnNames) == 1 && f.ColumnNames[0] == column.Name {
					fk = f
					break
				}
			}
			if fk != nil {
				diff.AddedForeignKeys.del(fk.ConstraintName)
				fd := col.references(fk.ReferencedColumnNames[0])
				fd.inTable = getReferencedTableName(fk.ReferencedTableName, "")
				fd.keyName = fk.ConstraintName
				fd.onUpdate = fk.UpdateRule
				fd.onDelete = fk.DeleteRule
				if fk.DeferMode != "" {
					fd.deferrable = fk.DeferMode
				}
			}
		}
		for _, cd := range diff.ChangedColumns.values() {
			if len(cd.ChangedProperties) == 1 && cd.ChangedProperties["comment"] {
				continue
			}
			if len(cd.ChangedProperties) == 1 && cd.ChangedProperties["enumItems"] && cd.Column.NativeEnumName != "" {
				continue
			}
			col := createTableColumn(table, cd.Column, diff.FromTable, cd.ChangedProperties, true)
			if col != nil {
				col.alter()
				configureColumn(cd.Column, col, cd.ChangedProperties)
			}
		}
		for _, cd := range diff.ChangedColumns.values() {
			if cd.ChangedProperties["autoincrement"] {
				table.rawQuery(getAlterColumnAutoincrement(tableName, cd.Column, schemaName))
			}
		}
		for _, cd := range diff.ChangedColumns.values() {
			if !cd.ChangedProperties["comment"] {
				continue
			}
			skip := false
			for _, k := range []string{"type", "nullable", "autoincrement", "unsigned", "default", "enumItems"} {
				if cd.ChangedProperties[k] {
					skip = true
				}
			}
			if skip {
				continue
			}
			table.rawQuery(getChangeColumnCommentSQL(tableName, cd.Column, schemaName))
		}
		for _, old := range diff.RenamedColumns.entries() {
			column, _ := diff.RenamedColumns.get(old)
			table.rawQuery(getRenameColumnSQL(tableName, old, column, schemaName))
		}
		for _, fk := range diff.AddedForeignKeys.values() {
			createForeignKey(table, fk, "")
		}
		for _, fk := range diff.ChangedForeignKeys.values() {
			createForeignKey(table, fk, "")
		}
		for _, index := range diff.AddedIndexes.values() {
			createIndex(table, index, diff.ToTable, false)
		}
		for _, index := range diff.ChangedIndexes.values() {
			createIndex(table, index, diff.ToTable, true)
		}
		for _, old := range diff.RenamedIndexes.entries() {
			index, _ := diff.RenamedIndexes.get(old)
			if index.Unique {
				dropIndex(table, index, old)
				createIndex(table, index, diff.ToTable, false)
			} else {
				table.rawQuery(getRenameIndexSQL(index, old))
			}
		}
		for _, check := range diff.AddedChecks.values() {
			createCheck(table, check)
		}
		for _, check := range diff.ChangedChecks.values() {
			createCheck(table, check)
		}
		if diff.HasChangedComment {
			comment := ""
			if diff.ChangedComment != nil && *diff.ChangedComment != "" {
				q := QuoteValue(*diff.ChangedComment)
				comment = q[1 : len(q)-1]
			}
			table.setComment(comment)
		}
	}))
	return ret
}

// commitOrderCalculator mirrors MikroORM's CommitOrderCalculator.
type commitOrderCalculator struct {
	nodes  omap[*commitNode]
	sorted []string
}

type commitEdge struct {
	from, to string
	weight   int
}

type commitNode struct {
	hash         string
	state        int
	dependencies omap[*commitEdge]
}

func newCommitOrderCalculator() *commitOrderCalculator { return &commitOrderCalculator{} }

func (c *commitOrderCalculator) addNode(hash string) {
	c.nodes.set(hash, &commitNode{hash: hash})
}

func (c *commitOrderCalculator) addDependency(from, to string, weight int) {
	n, _ := c.nodes.get(from)
	n.dependencies.set(to, &commitEdge{from, to, weight})
}

func (c *commitOrderCalculator) discoverProperty(prop *dml.SchemaProperty, entityName string) {
	toOneOwner := (prop.Kind == "1:1" && prop.Owner) || prop.Kind == "m:1"
	// m:n owners always have a pivot entity once discovered
	if !toOneOwner {
		return
	}
	target := prop.TargetClass
	if target == "" || !c.nodes.has(target) {
		return
	}
	weight := 1
	if prop.Nullable || !prop.Persist {
		weight = 0
	}
	c.addDependency(target, entityName, weight)
}

func (c *commitOrderCalculator) sort() []string {
	for _, v := range c.nodes.values() {
		if v.state != 0 {
			continue
		}
		c.visit(v)
	}
	out := make([]string, len(c.sorted))
	for i, h := range c.sorted {
		out[len(c.sorted)-1-i] = h
	}
	return out
}

func (c *commitOrderCalculator) visit(node *commitNode) {
	node.state = 1
	for _, edge := range node.dependencies.values() {
		target, _ := c.nodes.get(edge.to)
		switch target.state {
		case 2:
		case 1:
			c.visitOpenNode(node, target, edge)
		case 0:
			c.visit(target)
		}
	}
	if node.state != 2 {
		node.state = 2
		c.sorted = append(c.sorted, node.hash)
	}
}

func (c *commitOrderCalculator) visitOpenNode(node, target *commitNode, edge *commitEdge) {
	back, ok := target.dependencies.get(node.hash)
	if !ok || back.weight >= edge.weight {
		return
	}
	for _, e := range target.dependencies.values() {
		t, _ := c.nodes.get(e.to)
		if t.state == 0 {
			c.visit(t)
		}
	}
	target.state = 2
	c.sorted = append(c.sorted, target.hash)
}
