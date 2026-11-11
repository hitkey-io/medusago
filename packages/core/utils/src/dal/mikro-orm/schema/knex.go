package schema

import (
	"strings"
)

// This file ports the knex 3 schema builder and its PostgreSQL compilers
// (lib/schema/{builder,tablebuilder,columnbuilder,tablecompiler,
// columncompiler}.js, lib/dialects/postgres/schema/pg-*.js) with the
// MikroORM overrides (PostgreSqlTableCompiler.addColumn,
// PostgreSqlColumnCompiler.enu, TableCompiler.prototype.raw), for the calls
// the schema generator makes. Builders compile when they are dumped, into
// the statements of generateDdlCommands.

// wrapIdentifier mirrors the PostgreSQL wrapIdentifierImpl.
func wrapIdentifier(value string) string {
	if value == "*" {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// wrap mirrors formatter.wrap for identifiers: the dot separated segments
// are wrapped one by one.
func wrap(value string) string {
	segments := strings.Split(value, ".")
	for i, s := range segments {
		// wrapString / wrapAsIdentifier trim every segment
		segments[i] = wrapIdentifier(strings.TrimSpace(s))
	}
	return strings.Join(segments, ".")
}

func columnize(cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = wrap(c)
	}
	return strings.Join(out, ", ")
}

// schemaBuilder mirrors knex.schema (with withSchema).
type schemaBuilder struct {
	schema   string
	sequence []func() []string
}

func newSchemaBuilder(schema string) *schemaBuilder {
	return &schemaBuilder{schema: schema}
}

// raw mirrors knex.schema.raw(sql).
func (s *schemaBuilder) raw(sql string) *schemaBuilder {
	s.sequence = append(s.sequence, func() []string { return []string{sql} })
	return s
}

func (s *schemaBuilder) createTable(name string, fn func(t *tableBuilder)) *schemaBuilder {
	s.sequence = append(s.sequence, func() []string {
		t := &tableBuilder{method: "create", schema: s.schema, table: name}
		fn(t)
		return t.compile()
	})
	return s
}

func (s *schemaBuilder) alterTable(name string, fn func(t *tableBuilder)) *schemaBuilder {
	s.sequence = append(s.sequence, func() []string {
		t := &tableBuilder{method: "alter", schema: s.schema, table: name}
		fn(t)
		return t.compile()
	})
	return s
}

func (s *schemaBuilder) dropTableIfExists(name string) *schemaBuilder {
	s.sequence = append(s.sequence, func() []string {
		n := name
		if s.schema != "" {
			n = s.schema + "." + name
		}
		return []string{"drop table if exists " + wrap(n)}
	})
	return s
}

// statements mirrors generateDdlCommands: pre, sql and post statements.
func (s *schemaBuilder) statements() []string {
	var out []string
	for _, f := range s.sequence {
		out = append(out, f()...)
	}
	return out
}

// tableStatement is an entry of the alterTable grouping.
type tableStatement struct {
	method string
	args   []any
}

type checkStatement struct {
	predicate, name string
}

// tableBuilder mirrors TableBuilder.
type tableBuilder struct {
	method  string
	schema  string
	table   string
	columns []*columnBuilder
	alter   []tableStatement
	checks  []checkStatement
	comment *string
}

// foreignData mirrors the foreignData of TableBuilder.foreign.
type foreignData struct {
	columns    []string
	keyName    string
	references []string
	inTable    string
	onUpdate   string
	onDelete   string
	deferrable string
}

func (t *tableBuilder) push(method string, args ...any) {
	t.alter = append(t.alter, tableStatement{method, args})
}

func (t *tableBuilder) column(typ string, args ...any) *columnBuilder {
	c := &columnBuilder{table: t, typ: typ, args: args, method: "add", modifiers: map[string][]any{}}
	t.columns = append(t.columns, c)
	return c
}

func (t *tableBuilder) increments(name string, primaryKey bool) *columnBuilder {
	return t.column("increments", name, primaryKey)
}

func (t *tableBuilder) bigIncrements(name string, primaryKey bool) *columnBuilder {
	return t.column("bigincrements", name, primaryKey)
}

func (t *tableBuilder) enu(name string, items []string) *columnBuilder {
	return t.column("enu", name, items)
}

func (t *tableBuilder) specificType(name, typ string) *columnBuilder {
	return t.column("specifictype", name, typ)
}

// foreign mirrors TableBuilder.foreign.
func (t *tableBuilder) foreign(columns []string, keyName string) *foreignData {
	fd := &foreignData{columns: columns, keyName: keyName}
	t.push("foreign", fd)
	return fd
}

func (t *tableBuilder) primary(columns []string, constraintName string) {
	t.push("primary", columns, constraintName)
}

func (t *tableBuilder) unique(columns []string, indexName, deferrable string) {
	t.push("unique", columns, indexName, deferrable)
}

func (t *tableBuilder) index(columns []string, indexName, typ string) {
	t.push("index", columns, indexName, typ)
}

func (t *tableBuilder) dropPrimary(name string) { t.push("dropPrimary", name) }

func (t *tableBuilder) dropUnique(columns []string, name string) { t.push("dropUnique", columns, name) }

func (t *tableBuilder) dropIndex(columns []string, name string) { t.push("dropIndex", columns, name) }

func (t *tableBuilder) dropForeign(columns []string, name string) {
	t.push("dropForeign", columns, name)
}

// rawQuery mirrors SchemaHelper.pushTableQuery.
func (t *tableBuilder) rawQuery(sql string) { t.push("raw", sql) }

// check mirrors TableBuilder.check (create) and AlterMethods.check.
func (t *tableBuilder) check(predicate, name string) {
	if t.method == "alter" {
		t.push("check", predicate, name)
		return
	}
	t.checks = append(t.checks, checkStatement{predicate, name})
}

func (t *tableBuilder) dropChecks(name string) { t.push("dropChecks", name) }

func (t *tableBuilder) setComment(c string) { t.comment = &c }

// columnBuilder mirrors ColumnBuilder.
type columnBuilder struct {
	table         *tableBuilder
	typ           string
	args          []any
	method        string
	modifiers     map[string][]any
	alterNullable bool
	alterType     bool
	fd            *foreignData
}

func (c *columnBuilder) name() string { return c.args[0].(string) }

func (c *columnBuilder) nullable() *columnBuilder {
	c.modifiers["nullable"] = []any{nil}
	return c
}

func (c *columnBuilder) notNullable() *columnBuilder {
	c.modifiers["nullable"] = []any{false}
	return c
}

// defaultToRaw mirrors defaultTo(knex.raw(sql)); a nil sql is
// defaultTo(null).
func (c *columnBuilder) defaultToRaw(sql *string) *columnBuilder {
	if sql == nil {
		c.modifiers["defaultTo"] = []any{nil}
	} else {
		c.modifiers["defaultTo"] = []any{rawSQL(*sql)}
	}
	return c
}

func (c *columnBuilder) comment(s string) *columnBuilder {
	c.modifiers["comment"] = []any{s}
	return c
}

// alter mirrors .alter() with its defaults.
func (c *columnBuilder) alter() *columnBuilder {
	c.method = "alter"
	c.alterNullable = true
	c.alterType = true
	return c
}

// references mirrors column.references(col).inTable(...): the foreign key
// statement is pushed when references is called.
func (c *columnBuilder) references(column string) *foreignData {
	fd := c.table.foreign([]string{c.name()}, "")
	fd.references = []string{column}
	c.fd = fd
	return fd
}

type rawSQL string

// columnCompiler mirrors ColumnCompiler_PG for one column.
type columnCompiler struct {
	t          *tableCompiler
	c          *columnBuilder
	sequence   []string
	additional []string
	columnType *string
}

func (cc *columnCompiler) isIncrements() bool {
	return strings.Contains(cc.c.typ, "increments")
}

func (cc *columnCompiler) toSQL() []string {
	cc.sequence = append(cc.sequence, cc.compileColumn())
	return append(cc.sequence, cc.additional...)
}

func (cc *columnCompiler) compileColumn() string {
	return wrap(cc.c.name()) + " " + cc.getColumnType() + cc.getModifiers()
}

func (cc *columnCompiler) getColumnType() string {
	if cc.columnType != nil {
		return *cc.columnType
	}
	var t string
	switch cc.c.typ {
	case "increments":
		t = "serial"
		if cc.t.canBeAddPrimaryKey(cc.c.args[1].(bool)) {
			t += " primary key"
		}
	case "bigincrements":
		t = "bigserial"
		if cc.t.canBeAddPrimaryKey(cc.c.args[1].(bool)) {
			t += " primary key"
		}
	case "enu":
		items := cc.c.args[1].([]string)
		vals := make([]string, len(items))
		for i, v := range items {
			vals[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		t = "text check (" + wrap(cc.c.name()) + " in (" + strings.Join(vals, ", ") + "))"
	case "specifictype":
		t = cc.c.args[1].(string)
	}
	cc.columnType = &t
	return t
}

var pgColumnModifiers = []string{"nullable", "defaultTo", "comment"}

func (cc *columnCompiler) getModifiers() string {
	var mods []string
	for _, m := range pgColumnModifiers {
		if cc.isIncrements() && m != "comment" {
			continue
		}
		args, ok := cc.c.modifiers[m]
		if !ok {
			continue
		}
		var v string
		switch m {
		case "nullable":
			v = nullableModifier(args[0])
		case "defaultTo":
			v = defaultToModifier(args[0])
		case "comment":
			cc.commentModifier(args[0].(string))
		}
		if v != "" {
			mods = append(mods, v)
		}
	}
	if len(mods) == 0 {
		return ""
	}
	return " " + strings.Join(mods, " ")
}

func nullableModifier(v any) string {
	if b, ok := v.(bool); ok && !b {
		return "not null"
	}
	return "null"
}

// defaultToModifier mirrors ColumnCompiler.defaultTo with formatDefault for
// raw and null values.
func defaultToModifier(v any) string {
	switch x := v.(type) {
	case nil:
		return "default null"
	case rawSQL:
		return "default " + string(x)
	}
	return ""
}

func (cc *columnCompiler) commentModifier(comment string) {
	escaped := "NULL"
	if comment != "" {
		escaped = "'" + escapeComment(comment) + "'"
	}
	cc.additional = append(cc.additional, "comment on column "+cc.t.tableName()+"."+wrap(cc.c.name())+" is "+escaped)
}

// escapeComment mirrors comment.replace(/(?<!')'(?!')/g, "”").
func escapeComment(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' && (i == 0 || s[i-1] != '\'') && (i == len(s)-1 || s[i+1] != '\'') {
			b.WriteString("''")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// tableCompiler mirrors TableCompiler_PG with PostgreSqlTableCompiler.
type tableCompiler struct {
	b        *tableBuilder
	sequence []string
}

func (t *tableBuilder) compile() []string {
	tc := &tableCompiler{b: t}
	if t.method == "create" {
		tc.create()
	} else {
		tc.alterMethod()
	}
	return tc.sequence
}

func (tc *tableCompiler) push(sql string) { tc.sequence = append(tc.sequence, sql) }

func (tc *tableCompiler) tableName() string {
	name := tc.b.table
	if tc.b.schema != "" {
		name = tc.b.schema + "." + tc.b.table
	}
	return wrap(name)
}

func (tc *tableCompiler) getColumns(method string) []*columnCompiler {
	var out []*columnCompiler
	for _, c := range tc.b.columns {
		if c.method == method {
			out = append(out, &columnCompiler{t: tc, c: c})
		}
	}
	return out
}

func (tc *tableCompiler) primaryStatements() []tableStatement {
	var out []tableStatement
	for _, s := range tc.b.alter {
		if s.method == "primary" {
			out = append(out, s)
		}
	}
	return out
}

func (tc *tableCompiler) canBeAddPrimaryKey(primaryKey bool) bool {
	return primaryKey && len(tc.primaryStatements()) == 0
}

func (tc *tableCompiler) create() {
	cols := tc.getColumns("add")
	var compiled [][]string
	var defs []string
	for _, c := range cols {
		sql := c.toSQL()
		compiled = append(compiled, sql)
		defs = append(defs, sql[0])
	}
	sql := "create table " + tc.tableName() + " (" + strings.Join(defs, ", ") + tc.primaryKeys() + tc.addChecks() + ")"
	tc.push(sql)
	if tc.b.comment != nil {
		tc.push("comment on table " + tc.tableName() + " is '" + *tc.b.comment + "'")
	}
	for _, c := range compiled {
		for _, q := range c[1:] {
			tc.push(q)
		}
	}
	tc.b.comment = nil
	tc.alterTable()
}

func (tc *tableCompiler) primaryKeys() string {
	pks := tc.primaryStatements()
	if len(pks) == 0 {
		return ""
	}
	cols := pks[0].args[0].([]string)
	name := pks[0].args[1].(string)
	if name == "" {
		name = tc.b.table + "_pkey"
	}
	return ", constraint " + wrap(name) + " primary key (" + columnize(cols) + ")"
}

func (tc *tableCompiler) addChecks() string {
	if len(tc.b.checks) == 0 {
		return ""
	}
	parts := make([]string, len(tc.b.checks))
	for i, c := range tc.b.checks {
		prefix := ""
		if c.name != "" {
			prefix = "constraint " + c.name + " "
		}
		parts[i] = prefix + "check (" + c.predicate + ")"
	}
	return ", " + strings.Join(parts, ", ")
}

func (tc *tableCompiler) alterMethod() {
	addCols := tc.getColumns("add")
	var added [][]string
	var addDefs []string
	for _, c := range addCols {
		sql := c.toSQL()
		added = append(added, sql)
		addDefs = append(addDefs, sql[0])
	}
	alterCols := tc.getColumns("alter")
	var altered [][]string
	for _, c := range alterCols {
		altered = append(altered, c.toSQL())
	}
	if len(addDefs) > 0 {
		for i, d := range addDefs {
			addDefs[i] = "add column " + d
		}
		tc.push("alter table " + tc.tableName() + " " + strings.Join(addDefs, ", "))
	}
	for _, c := range alterCols {
		tc.addAlterColumn(c)
	}
	for _, c := range added {
		for _, q := range c[1:] {
			tc.push(q)
		}
	}
	for _, c := range altered {
		for _, q := range c[1:] {
			tc.push(q)
		}
	}
	tc.alterTable()
}

// addAlterColumn mirrors PostgreSqlTableCompiler.addColumn.
func (tc *tableCompiler) addAlterColumn(cc *columnCompiler) {
	table := tc.tableName()
	typ := cc.getColumnType()
	colName := wrapIdentifier(cc.c.name())
	tableRaw := tc.b.table
	if i := strings.LastIndex(tableRaw, "."); i >= 0 {
		tableRaw = tableRaw[i+1:]
	}
	constraintName := tableRaw + "_" + cc.c.name() + "_check"
	defaultTo, hasDefault := cc.c.modifiers["defaultTo"]
	if hasDefault && defaultTo[0] == nil {
		tc.push("alter table " + table + " alter column " + colName + " drop default")
	}
	switch {
	case cc.c.typ == "enu":
		if cc.c.alterType {
			tc.push("alter table " + table + " alter column " + colName + " type text using (" + colName + "::text)")
		}
		// createForeignKeyConstraints: true
		if cc.c.alterNullable {
			tc.push("alter table " + table + ` add constraint "` + constraintName + `" ` + strings.TrimPrefix(typ, "text "))
		}
	case typ == "uuid":
		tc.push("alter table " + table + " alter column " + colName + " drop default")
		tc.push("alter table " + table + " alter column " + colName + " type " + typ + " using (" + colName + "::text::uuid)")
	case cc.c.alterType:
		tc.push("alter table " + table + " alter column " + colName + " type " + typ + " using (" + colName + "::" + typ + ")")
	}
	if hasDefault && defaultTo[0] != nil {
		tc.push("alter table " + table + " alter column " + colName + " set " + defaultToModifier(defaultTo[0]))
	}
	if nullable, ok := cc.c.modifiers["nullable"]; ok {
		if b, isBool := nullable[0].(bool); isBool && !b {
			tc.push("alter table " + table + " alter column " + colName + " set not null")
		} else {
			tc.push("alter table " + table + " alter column " + colName + " drop not null")
		}
	}
}

// alterTable mirrors TableCompiler.alterTable: the statements of the
// alterTable grouping, then the single values (comment).
func (tc *tableCompiler) alterTable() {
	for _, s := range tc.b.alter {
		switch s.method {
		case "raw":
			tc.push(s.args[0].(string))
		case "foreign":
			tc.compileForeign(s.args[0].(*foreignData))
		case "primary":
			if tc.b.method != "create" {
				name := s.args[1].(string)
				if name == "" {
					name = tc.b.table + "_pkey"
				}
				tc.push("alter table " + tc.tableName() + " add constraint " + wrap(name) + " primary key (" + columnize(s.args[0].([]string)) + ")")
			}
		case "unique":
			cols, name, deferrable := s.args[0].([]string), s.args[1].(string), s.args[2].(string)
			if name == "" {
				name = indexCommand("unique", tc.b.table, cols)
			} else {
				name = wrap(name)
			}
			suffix := ""
			if deferrable != "" {
				suffix = " deferrable initially " + deferrable
			}
			tc.push("alter table " + tc.tableName() + " add constraint " + name + " unique (" + columnize(cols) + ")" + suffix)
		case "index":
			cols, name, typ := s.args[0].([]string), s.args[1].(string), s.args[2].(string)
			if name == "" {
				name = indexCommand("index", tc.b.table, cols)
			} else {
				name = wrap(name)
			}
			using := ""
			if typ != "" {
				using = " using " + typ
			}
			tc.push("create index " + name + " on " + tc.tableName() + using + " (" + columnize(cols) + ")")
		case "dropPrimary":
			name := s.args[0].(string)
			if name == "" {
				name = tc.b.table + "_pkey"
			}
			tc.push("alter table " + tc.tableName() + " drop constraint " + wrap(name))
		case "dropUnique", "dropForeign":
			cols, name := s.args[0].([]string), s.args[1].(string)
			if name == "" {
				kind := "unique"
				if s.method == "dropForeign" {
					kind = "foreign"
				}
				name = indexCommand(kind, tc.b.table, cols)
			} else {
				name = wrap(name)
			}
			tc.push("alter table " + tc.tableName() + " drop constraint " + name)
		case "dropIndex":
			cols, name := s.args[0].([]string), s.args[1].(string)
			if name == "" {
				name = indexCommand("index", tc.b.table, cols)
			} else {
				name = wrap(name)
			}
			if tc.b.schema != "" {
				name = wrap(tc.b.schema) + "." + name
			}
			tc.push("drop index " + name)
		case "check":
			pred, name := s.args[0].(string), s.args[1].(string)
			tc.push("alter table " + tc.tableName() + " add constraint " + name + " check(" + pred + ")")
		case "dropChecks":
			tc.push("alter table " + tc.tableName() + " drop constraint " + s.args[0].(string))
		}
	}
	if tc.b.comment != nil {
		tc.push("comment on table " + tc.tableName() + " is '" + *tc.b.comment + "'")
	}
}

func (tc *tableCompiler) compileForeign(fd *foreignData) {
	if fd.inTable == "" || len(fd.references) == 0 {
		return
	}
	keyName := wrap(fd.keyName)
	if fd.keyName == "" {
		keyName = indexCommand("foreign", tc.b.table, fd.columns)
	}
	sql := "alter table " + tc.tableName() + " add constraint " + keyName + " foreign key (" + columnize(fd.columns) +
		") references " + wrap(fd.inTable) + " (" + columnize(fd.references) + ")"
	if fd.onUpdate != "" {
		sql += " on update " + fd.onUpdate
	}
	if fd.onDelete != "" {
		sql += " on delete " + fd.onDelete
	}
	if fd.deferrable != "" {
		sql += " deferrable initially " + strings.ToLower(fd.deferrable) + " "
	}
	tc.push(sql)
}

// indexCommand mirrors TableCompiler._indexCommand.
func indexCommand(typ, table string, columns []string) string {
	t := strings.NewReplacer(".", "_", "-", "_").Replace(table)
	return wrap(strings.ToLower(t + "_" + strings.Join(columns, "_") + "_" + typ))
}
