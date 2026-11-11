package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// DatabaseSchema mirrors MikroORM's DatabaseSchema.
type DatabaseSchema struct {
	Name       string
	Tables     []*Table
	namespaces []string
	// NativeEnums is kept as read (always {} for DML models).
	NativeEnums    jsonNode
	nativeEnumKeys []string
}

func newDatabaseSchema(name string) *DatabaseSchema {
	return &DatabaseSchema{Name: name, NativeEnums: jsonObject{}}
}

func (s *DatabaseSchema) addNamespace(ns string) {
	if !hasString(s.namespaces, ns) {
		s.namespaces = append(s.namespaces, ns)
	}
}

// Namespaces mirrors getNamespaces.
func (s *DatabaseSchema) Namespaces() []string { return s.namespaces }

// HasNamespace mirrors hasNamespace.
func (s *DatabaseSchema) HasNamespace(ns string) bool { return hasString(s.namespaces, ns) }

// addTable mirrors DatabaseSchema.addTable.
func (s *DatabaseSchema) addTable(name, schema string) *Table {
	ns := schema
	if ns == "" {
		ns = s.Name
	}
	t := newTable(name, ns)
	t.NativeEnums = s.NativeEnums
	s.Tables = append(s.Tables, t)
	if ns != "" {
		s.addNamespace(ns)
	}
	return t
}

// Table mirrors getTable.
func (s *DatabaseSchema) Table(name string) *Table {
	for _, t := range s.Tables {
		if t.Name == name || t.Schema+"."+t.Name == name {
			return t
		}
	}
	return nil
}

// HasTable mirrors hasTable.
func (s *DatabaseSchema) HasTable(name string) bool { return s.Table(name) != nil }

func (s *DatabaseSchema) hasNativeEnum(key string) bool { return hasString(s.nativeEnumKeys, key) }

// FromMetadata mirrors DatabaseSchema.fromMetadata (DML entities define no
// native enums, embeddables or skipped columns).
func FromMetadata(metadata []*dml.SchemaEntity, schemaName string) *DatabaseSchema {
	s := newDatabaseSchema(schemaName)
	for _, meta := range metadata {
		ns := meta.Schema
		if ns == "" {
			ns = DefaultSchemaName
		}
		t := s.addTable(meta.Collection, ns)
		lengths := map[*dml.SchemaProperty]*int{}
		for _, prop := range meta.Props {
			if shouldHaveColumn(prop) {
				t.addColumnFromProperty(prop, meta, lengths)
			}
		}
		for _, idx := range removeDuplicateIndexes(meta.Indexes) {
			t.addIndex(meta, idx.Name, idx.Expression, nil, "index")
		}
		var pks []string
		for _, p := range meta.Props {
			if p.Primary {
				pks = append(pks, p.Name)
			}
		}
		t.addIndex(meta, "", "", pks, "primary")
		for _, c := range meta.Checks {
			t.Checks = append(t.Checks, &Check{
				Name:       c.Name,
				Expression: c.Expression,
				Definition: "check ((" + c.Expression + "))",
			})
		}
	}
	return s
}

// removeDuplicateIndexes mirrors the Utils.removeDuplicates of
// EntityMetadata.sync.
func removeDuplicateIndexes(indexes []dml.SchemaIndex) []dml.SchemaIndex {
	var out []dml.SchemaIndex
	for _, idx := range indexes {
		dup := false
		for _, o := range out {
			if o == idx {
				dup = true
			}
		}
		if !dup {
			out = append(out, idx)
		}
	}
	return out
}

// shouldHaveColumn mirrors DatabaseSchema.shouldHaveColumn.
func shouldHaveColumn(prop *dml.SchemaProperty) bool {
	if !prop.Persist || len(prop.ColumnTypes) == 0 {
		return false
	}
	return prop.Kind == "scalar" || prop.Kind == "m:1" || (prop.Kind == "1:1" && prop.Owner)
}

// prune mirrors DatabaseSchema.prune without a schema nor wildcard schema
// tables: only the namespaces of ignored tables go.
func (s *DatabaseSchema) prune() {
	var kept []string
	for _, ns := range s.namespaces {
		used := false
		for _, t := range s.Tables {
			if t.Schema == ns {
				used = true
			}
		}
		if used {
			kept = append(kept, ns)
		}
	}
	s.namespaces = kept
}

// JSON returns the snapshot file content (DatabaseSchema.toJSON written by
// fs-extra writeJSON with 2 spaces).
func (s *DatabaseSchema) JSON() string {
	o := jsonObject{}
	ns := s.namespaces
	if ns == nil {
		ns = []string{}
	}
	o.set("namespaces", ns)
	o.set("name", s.Name)
	tables := []jsonNode{}
	for _, t := range s.Tables {
		tables = append(tables, t.toJSON())
	}
	o.set("tables", tables)
	o.set("nativeEnums", s.NativeEnums)
	return stringify(o) + "\n"
}

// EmptySnapshot is the snapshot Migrations.ensureSnapshot writes when none
// exists.
const EmptySnapshot = "{\n  \"tables\": [],\n  \"namespaces\": []\n}"

// FromSnapshot mirrors Migrator.getSchemaFromSnapshot.
func FromSnapshot(data []byte) (*DatabaseSchema, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	node, err := decodeOrdered(dec)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot: %w", err)
	}
	root, ok := node.(jsonObject)
	if !ok {
		return nil, fmt.Errorf("invalid snapshot: not an object")
	}
	s := newDatabaseSchema(DefaultSchemaName)
	for _, f := range root {
		switch f.key {
		case "name":
			s.Name = asString(f.value)
		case "namespaces":
			for _, n := range asArray(f.value) {
				s.addNamespace(asString(n))
			}
		case "nativeEnums":
			s.NativeEnums = f.value
			if obj, ok := f.value.(jsonObject); ok {
				for _, e := range obj {
					s.nativeEnumKeys = append(s.nativeEnumKeys, e.key)
				}
			}
		case "tables":
			for _, tn := range asArray(f.value) {
				t, err := tableFromJSON(tn)
				if err != nil {
					return nil, err
				}
				s.Tables = append(s.Tables, t)
			}
		}
	}
	return s, nil
}

func tableFromJSON(node jsonNode) (*Table, error) {
	obj, ok := node.(jsonObject)
	if !ok {
		return nil, fmt.Errorf("invalid snapshot table")
	}
	t := &Table{NativeEnums: jsonObject{}}
	for _, f := range obj {
		switch f.key {
		case "name":
			t.Name = asString(f.value)
		case "schema":
			t.Schema = asString(f.value)
		case "comment":
			if f.value != nil {
				c := asString(f.value)
				t.Comment = &c
			}
		case "nativeEnums":
			t.NativeEnums = f.value
		case "columns":
			cols, _ := f.value.(jsonObject)
			for _, cf := range cols {
				t.addColumn(columnFromJSON(cf.value))
			}
		case "indexes":
			for _, in := range asArray(f.value) {
				t.Indexes = append(t.Indexes, indexFromJSON(in))
			}
		case "checks":
			for _, cn := range asArray(f.value) {
				co, _ := cn.(jsonObject)
				c := &Check{}
				for _, x := range co {
					switch x.key {
					case "name":
						c.Name = asString(x.value)
					case "expression":
						c.Expression = asString(x.value)
					case "definition":
						c.Definition = asString(x.value)
					case "columnName":
						c.ColumnName = asString(x.value)
					}
				}
				t.Checks = append(t.Checks, c)
			}
		case "foreignKeys":
			fks, _ := f.value.(jsonObject)
			for _, x := range fks {
				fo, _ := x.value.(jsonObject)
				fk := &ForeignKey{}
				for _, y := range fo {
					switch y.key {
					case "constraintName":
						fk.ConstraintName = asString(y.value)
					case "columnNames":
						fk.ColumnNames = asStrings(y.value)
					case "localTableName":
						fk.LocalTableName = asString(y.value)
					case "referencedColumnNames":
						fk.ReferencedColumnNames = asStrings(y.value)
					case "referencedTableName":
						fk.ReferencedTableName = asString(y.value)
					case "deleteRule":
						fk.DeleteRule = asString(y.value)
					case "updateRule":
						fk.UpdateRule = asString(y.value)
					case "deferMode":
						fk.DeferMode = asString(y.value)
					}
				}
				// `foreignKeys[key]`: the snapshot key is the constraint
				// name for the files MikroORM writes
				t.foreignKeys = append(t.foreignKeys, fk)
			}
		}
	}
	return t, nil
}

func columnFromJSON(node jsonNode) *Column {
	obj, _ := node.(jsonObject)
	c := &Column{MappedType: TypeUnknown}
	for _, f := range obj {
		switch f.key {
		case "name":
			c.Name = asString(f.value)
		case "type":
			c.Type = asString(f.value)
		case "unsigned":
			c.Unsigned = asBool(f.value)
		case "autoincrement":
			c.Autoincrement = asBool(f.value)
		case "primary":
			c.Primary = asBool(f.value)
		case "nullable":
			c.Nullable = asBool(f.value)
		case "unique":
			c.Unique = asBool(f.value)
		case "length":
			c.Length = asInt(f.value)
		case "precision":
			c.Precision = asInt(f.value)
		case "scale":
			c.Scale = asInt(f.value)
		case "default":
			c.Default = asScalarString(f.value)
		case "comment":
			c.Comment = asScalarString(f.value)
		case "enumItems":
			c.EnumItems = asStrings(f.value)
		case "mappedType":
			c.MappedType = knownMappedType(asString(f.value))
		case "generated":
			c.Generated = asString(f.value)
		case "nativeEnumName":
			c.NativeEnumName = asString(f.value)
		case "extra":
			c.Extra = asString(f.value)
		case "ignoreSchemaChanges":
			c.IgnoreSchemaChanges = asStrings(f.value)
		}
	}
	return c
}

// knownMappedType mirrors `Type.getType(t[columns[col].mappedType] ??
// UnknownType)`.
func knownMappedType(name string) string {
	switch name {
	case TypeDate, TypeTime, TypeDateTime, TypeBigInt, TypeBlob, TypeUint8Array, TypeArray, TypeEnumArray, TypeEnum,
		TypeJSON, TypeInteger, TypeSmallInt, TypeTinyInt, TypeMediumInt, TypeFloat, TypeDouble, TypeBoolean,
		TypeDecimal, TypeCharacter, TypeString, TypeUUID, TypeText, TypeInterval, TypeUnknown:
		return name
	}
	return TypeUnknown
}

func indexFromJSON(node jsonNode) *Index {
	obj, _ := node.(jsonObject)
	idx := &Index{}
	for i, f := range obj {
		if i == 0 && f.key == "columnNames" {
			idx.fromProperty = true
		}
		switch f.key {
		case "keyName":
			idx.KeyName = asString(f.value)
		case "columnNames":
			idx.ColumnNames = asStrings(f.value)
		case "composite":
			idx.Composite = asBool(f.value)
		case "constraint":
			idx.Constraint = asBool(f.value)
		case "primary":
			idx.Primary = asBool(f.value)
		case "unique":
			idx.Unique = asBool(f.value)
		case "type":
			idx.Type = asString(f.value)
		case "expression":
			idx.Expression = asString(f.value)
		case "deferMode":
			idx.DeferMode = asString(f.value)
		case "options":
			idx.Options = f.value
		}
	}
	if idx.ColumnNames == nil {
		idx.ColumnNames = []string{}
	}
	return idx
}

func asString(n jsonNode) string {
	switch x := n.(type) {
	case string:
		return x
	case jsonNumber:
		return string(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return ""
}

func asScalarString(n jsonNode) *string {
	if n == nil {
		return nil
	}
	s := asString(n)
	return &s
}

func asBool(n jsonNode) bool {
	switch x := n.(type) {
	case bool:
		return x
	case string:
		return x != ""
	case jsonNumber:
		return x != "0"
	}
	return false
}

func asInt(n jsonNode) *int {
	if x, ok := n.(jsonNumber); ok {
		v := 0
		neg := false
		for i, c := range string(x) {
			if i == 0 && c == '-' {
				neg = true
				continue
			}
			if c < '0' || c > '9' {
				break
			}
			v = v*10 + int(c-'0')
		}
		if neg {
			v = -v
		}
		return &v
	}
	return nil
}

func asArray(n jsonNode) []jsonNode {
	a, _ := n.([]jsonNode)
	return a
}

func asStrings(n jsonNode) []string {
	arr := asArray(n)
	if arr == nil {
		if n == nil {
			return nil
		}
		return []string{}
	}
	out := make([]string, len(arr))
	for i, x := range arr {
		out[i] = asString(x)
	}
	return out
}
