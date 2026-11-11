package schema

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// Column mirrors a MikroORM Column of a DatabaseTable.
type Column struct {
	Name          string
	Type          string
	MappedType    string
	Unsigned      bool
	Autoincrement bool
	Primary       bool
	Nullable      bool
	Unique        bool
	Length        *int
	Precision     *int
	Scale         *int
	Default       *string
	Comment       *string
	// EnumItems is nil when undefined (JSON []).
	EnumItems           []string
	Generated           string
	NativeEnumName      string
	Extra               string
	IgnoreSchemaChanges []string
}

func (c *Column) clone() *Column {
	cp := *c
	return &cp
}

// Index mirrors an IndexDef.
type Index struct {
	KeyName     string
	ColumnNames []string
	Composite   bool
	Constraint  bool
	Primary     bool
	Unique      bool
	Type        string
	Expression  string
	DeferMode   string
	Options     jsonNode
	// fromProperty: the index of a property marked index/unique (key
	// order of the object DatabaseTable.addColumnFromProperty pushes).
	fromProperty bool
}

// Check mirrors a CheckDef.
type Check struct {
	Name       string
	Expression string
	Definition string
	ColumnName string
}

// ForeignKey mirrors a ForeignKey.
type ForeignKey struct {
	ConstraintName        string
	ColumnNames           []string
	LocalTableName        string
	ReferencedColumnNames []string
	ReferencedTableName   string
	DeleteRule            string
	UpdateRule            string
	DeferMode             string
}

// Table mirrors DatabaseTable.
type Table struct {
	Name    string
	Schema  string
	Comment *string

	columns     []*Column
	Indexes     []*Index
	Checks      []*Check
	foreignKeys []*ForeignKey
	// NativeEnums is kept as read (always {} for DML models).
	NativeEnums jsonNode
}

func newTable(name, schema string) *Table {
	return &Table{Name: name, Schema: schema, NativeEnums: jsonObject{}}
}

// Columns mirrors getColumns (insertion order).
func (t *Table) Columns() []*Column { return t.columns }

// Column mirrors getColumn.
func (t *Table) Column(name string) *Column {
	for _, c := range t.columns {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// HasColumn mirrors hasColumn.
func (t *Table) HasColumn(name string) bool { return t.Column(name) != nil }

// addColumn mirrors `this.columns[name] = column` (an existing key keeps
// its position).
func (t *Table) addColumn(c *Column) {
	for i, x := range t.columns {
		if x.Name == c.Name {
			t.columns[i] = c
			return
		}
	}
	t.columns = append(t.columns, c)
}

// ForeignKeys mirrors getForeignKeys (insertion order).
func (t *Table) ForeignKeys() []*ForeignKey { return t.foreignKeys }

func (t *Table) foreignKey(name string) *ForeignKey {
	for _, fk := range t.foreignKeys {
		if fk.ConstraintName == name {
			return fk
		}
	}
	return nil
}

func (t *Table) setForeignKey(fk *ForeignKey) {
	for i, x := range t.foreignKeys {
		if x.ConstraintName == fk.ConstraintName {
			t.foreignKeys[i] = fk
			return
		}
	}
	t.foreignKeys = append(t.foreignKeys, fk)
}

// GetIndexes mirrors getIndexes (Utils.removeDuplicates: deep equality).
func (t *Table) GetIndexes() []*Index {
	var out []*Index
	for _, idx := range t.Indexes {
		dup := false
		for _, o := range out {
			if indexEquals(o, idx) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, idx)
		}
	}
	return out
}

func indexEquals(a, b *Index) bool {
	return a.KeyName == b.KeyName && strings.Join(a.ColumnNames, "\x00") == strings.Join(b.ColumnNames, "\x00") &&
		len(a.ColumnNames) == len(b.ColumnNames) && a.Composite == b.Composite && a.Constraint == b.Constraint &&
		a.Primary == b.Primary && a.Unique == b.Unique && a.Type == b.Type && a.Expression == b.Expression &&
		a.DeferMode == b.DeferMode && stringify(a.Options) == stringify(b.Options)
}

// Index mirrors getIndex.
func (t *Table) Index(name string) *Index {
	for _, idx := range t.Indexes {
		if idx.KeyName == name {
			return idx
		}
	}
	return nil
}

// HasIndex mirrors hasIndex.
func (t *Table) HasIndex(name string) bool { return t.Index(name) != nil }

// Check mirrors getCheck.
func (t *Table) Check(name string) *Check {
	for _, c := range t.Checks {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// PrimaryKey mirrors getPrimaryKey.
func (t *Table) PrimaryKey() *Index {
	for _, idx := range t.Indexes {
		if idx.Primary {
			return idx
		}
	}
	return nil
}

// ShortestName mirrors getShortestName.
func (t *Table) ShortestName() string {
	if t.Schema == "" || strings.HasPrefix(t.Name, t.Schema+".") {
		return t.Name
	}
	return t.Schema + "." + t.Name
}

func hasString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// addColumnFromProperty mirrors DatabaseTable.addColumnFromProperty.
func (t *Table) addColumnFromProperty(prop *dml.SchemaProperty, meta *dml.SchemaEntity, lengths map[*dml.SchemaProperty]*int) {
	for idx, field := range prop.FieldNames {
		typ := ""
		if idx < len(prop.ColumnTypes) {
			typ = prop.ColumnTypes[idx]
		}
		mapped := MappedType(typ)
		if prop.Enum {
			mapped = TypeEnum
		}
		var precision, scale *int
		length := lengths[prop]
		if mapped == TypeDecimal {
			if m := precisionScaleR.FindStringSubmatch(typ); m != nil {
				precision, scale = intPtr(atoi(m[1])), intPtr(atoi(m[2]))
				length = nil
			}
		}
		if length == nil && typ != "" {
			length = inferLengthFromColumnType(typ)
			if length == nil {
				length = defaultLength(mapped)
			}
			lengths[prop] = length
		}
		primary := prop.Primary
		autoincrement := !meta.CompositePK && primary && prop.Kind == "scalar" && isNumericColumn(mapped)
		if prop.Autoincrement != nil {
			autoincrement = *prop.Autoincrement
		}
		nullable := prop.Nullable
		if existing := t.Column(field); existing != nil {
			nullable = existing.Nullable
		}
		col := &Column{
			Name:          field,
			Type:          typ,
			MappedType:    mapped,
			Autoincrement: autoincrement,
			Primary:       primary,
			Nullable:      nullable,
			Length:        length,
			Precision:     precision,
			Scale:         scale,
		}
		// unsigned: prop.unsigned && isNumericColumn, then ??= autoincrement
		if prop.Unsigned == nil {
			col.Unsigned = autoincrement
		}
		allStrings := true
		for _, it := range prop.Items {
			if _, ok := it.(string); !ok {
				allStrings = false
			}
		}
		if prop.Items != nil && allStrings {
			col.EnumItems = make([]string, len(prop.Items))
			for i, it := range prop.Items {
				col.EnumItems[i] = it.(string)
			}
		}
		col.Default = normalizeDefaultValue(prop.DefaultRaw, length)
		t.addColumn(col)
	}
	if prop.Kind == "m:1" || prop.Kind == "1:1" {
		constraintName := IndexName(t.Name, prop.FieldNames, "foreign")
		schema := prop.TargetSchema
		if schema == "" {
			schema = DefaultSchemaName
		}
		if strings.Contains(prop.ReferencedTableName, ".") {
			schema = ""
		}
		if prop.CreateForeignKeyConstraint {
			referenced := prop.ReferencedTableName
			if schema != "" {
				referenced = schema + "." + referenced
			}
			fk := &ForeignKey{
				ConstraintName:        constraintName,
				ColumnNames:           prop.FieldNames,
				LocalTableName:        t.ShortestName(),
				ReferencedColumnNames: prop.ReferencedColumnNames,
				ReferencedTableName:   referenced,
			}
			cascade := hasString(prop.Cascade, "remove") || hasString(prop.Cascade, "all")
			if prop.DeleteRule != "" || cascade || prop.Nullable {
				switch {
				case prop.DeleteRule != "":
					fk.DeleteRule = prop.DeleteRule
				case cascade:
					fk.DeleteRule = "cascade"
				default:
					fk.DeleteRule = "set null"
				}
			}
			if prop.UpdateRule != "" {
				fk.UpdateRule = prop.UpdateRule
			}
			if hasString(prop.Cascade, "persist") || hasString(prop.Cascade, "all") {
				// supportsMultipleCascadePaths: always true on PostgreSQL
				if fk.UpdateRule == "" {
					fk.UpdateRule = "cascade"
				}
			}
			t.setForeignKey(fk)
		}
	}
	if prop.Unique && !(prop.Primary && !meta.CompositePK) {
		constraint := true
		for _, f := range prop.FieldNames {
			if strings.Contains(f, ".") {
				constraint = false
			}
		}
		t.Indexes = append(t.Indexes, &Index{
			ColumnNames:  prop.FieldNames,
			Composite:    len(prop.FieldNames) > 1,
			KeyName:      IndexName(t.Name, prop.FieldNames, "unique"),
			Constraint:   constraint,
			Unique:       true,
			fromProperty: true,
		})
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// addIndex mirrors DatabaseTable.addIndex for the indexes of DML entities
// (expressions) and the primary key.
func (t *Table) addIndex(meta *dml.SchemaEntity, name, expression string, properties []string, typ string) {
	var columns []string
	for _, p := range properties {
		if prop := meta.Property(p); prop != nil {
			for _, f := range prop.FieldNames {
				if !hasString(columns, f) {
					columns = append(columns, f)
				}
			}
			continue
		}
		if !hasString(columns, p) {
			columns = append(columns, p)
		}
	}
	if len(columns) == 0 && expression == "" {
		return
	}
	if name == "" {
		name = IndexName(t.Name, columns, typ)
	}
	constraint := typ != "index"
	for _, c := range columns {
		if strings.Contains(c, ".") {
			constraint = false
		}
	}
	if columns == nil {
		columns = []string{}
	}
	t.Indexes = append(t.Indexes, &Index{
		KeyName:     name,
		ColumnNames: columns,
		Composite:   len(columns) > 1,
		Constraint:  constraint,
		Primary:     typ == "primary",
		Unique:      typ != "index",
		Expression:  expression,
	})
}

// toJSON mirrors DatabaseTable.toJSON.
func (t *Table) toJSON() jsonObject {
	cols := jsonObject{}
	for _, c := range t.columns {
		o := jsonObject{}
		o.set("name", c.Name)
		o.set("type", c.Type)
		o.set("unsigned", c.Unsigned)
		o.set("autoincrement", c.Autoincrement)
		o.set("primary", c.Primary)
		o.set("nullable", c.Nullable)
		o.set("unique", c.Unique)
		o.set("length", c.Length)
		o.set("precision", c.Precision)
		o.set("scale", c.Scale)
		o.set("default", c.Default)
		o.set("comment", c.Comment)
		items := c.EnumItems
		if items == nil {
			items = []string{}
		}
		o.set("enumItems", items)
		if c.MappedType != typeFullText {
			o.set("mappedType", c.MappedType)
		}
		if c.Generated != "" {
			o.set("generated", c.Generated)
		}
		if c.NativeEnumName != "" {
			o.set("nativeEnumName", c.NativeEnumName)
		}
		if c.Extra != "" {
			o.set("extra", c.Extra)
		}
		if c.IgnoreSchemaChanges != nil {
			o.set("ignoreSchemaChanges", c.IgnoreSchemaChanges)
		}
		cols.set(c.Name, o)
	}
	out := jsonObject{}
	out.set("columns", cols)
	out.set("name", t.Name)
	if t.Schema != "" {
		out.set("schema", t.Schema)
	}
	indexes := []jsonNode{}
	for _, idx := range t.Indexes {
		indexes = append(indexes, idx.toJSON())
	}
	out.set("indexes", indexes)
	checks := []jsonNode{}
	for _, c := range t.Checks {
		o := jsonObject{}
		o.set("name", c.Name)
		o.set("expression", c.Expression)
		o.set("definition", c.Definition)
		if c.ColumnName != "" {
			o.set("columnName", c.ColumnName)
		}
		checks = append(checks, o)
	}
	out.set("checks", checks)
	fks := jsonObject{}
	for _, fk := range t.foreignKeys {
		o := jsonObject{}
		o.set("constraintName", fk.ConstraintName)
		o.set("columnNames", fk.ColumnNames)
		o.set("localTableName", fk.LocalTableName)
		o.set("referencedColumnNames", fk.ReferencedColumnNames)
		o.set("referencedTableName", fk.ReferencedTableName)
		if fk.DeleteRule != "" {
			o.set("deleteRule", fk.DeleteRule)
		}
		if fk.UpdateRule != "" {
			o.set("updateRule", fk.UpdateRule)
		}
		if fk.DeferMode != "" {
			o.set("deferMode", fk.DeferMode)
		}
		fks.set(fk.ConstraintName, o)
	}
	out.set("foreignKeys", fks)
	out.set("nativeEnums", t.NativeEnums)
	if t.Comment != nil {
		out.set("comment", *t.Comment)
	}
	return out
}

func (idx *Index) toJSON() jsonObject {
	o := jsonObject{}
	if idx.fromProperty {
		o.set("columnNames", idx.ColumnNames)
		o.set("composite", idx.Composite)
		o.set("keyName", idx.KeyName)
		o.set("constraint", idx.Constraint)
		o.set("primary", idx.Primary)
		o.set("unique", idx.Unique)
		if idx.DeferMode != "" {
			o.set("deferMode", idx.DeferMode)
		}
		return o
	}
	o.set("keyName", idx.KeyName)
	o.set("columnNames", idx.ColumnNames)
	o.set("composite", idx.Composite)
	o.set("constraint", idx.Constraint)
	o.set("primary", idx.Primary)
	o.set("unique", idx.Unique)
	if idx.Type != "" {
		o.set("type", idx.Type)
	}
	if idx.Expression != "" {
		o.set("expression", idx.Expression)
	}
	if idx.Options != nil {
		o.set("options", idx.Options)
	}
	if idx.DeferMode != "" {
		o.set("deferMode", idx.DeferMode)
	}
	return o
}
