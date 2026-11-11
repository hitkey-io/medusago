package dml

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// This file exposes what the MikroORM schema generator reads from the
// entity metadata of a module (DatabaseSchema.fromMetadata): the columns,
// foreign keys, indexes and checks of each table, including the pivot
// tables MikroORM defines for many to many relationships. It extends the
// derivation of resolve-meta.go with the @Index/@Check decorators applied by
// create-mikro-orm-entity.ts (apply-indexes.ts, apply-checks.ts and the
// indexes of define-property.ts / define-relationship.ts).

// SchemaIndex is an entity index (EntityMetadata.indexes): the DML always
// declares them as SQL expressions.
type SchemaIndex struct {
	Name       string
	Expression string
}

// SchemaCheck is a check constraint (EntityMetadata.checks).
type SchemaCheck struct {
	Name       string
	Expression string
}

// SchemaProperty is the part of an entity property the schema generator
// reads.
type SchemaProperty struct {
	Name        string
	Kind        string
	FieldNames  []string
	ColumnTypes []string
	Nullable    bool
	Primary     bool
	Persist     bool
	Enum        bool
	Items       []any
	DefaultRaw  *string
	// Autoincrement is nil when the property does not set it.
	Autoincrement *bool
	// Unsigned mirrors MetadataDiscovery.initUnsigned: false for primary
	// keys and relations, nil (undefined) otherwise.
	Unsigned *bool
	Owner    bool
	Unique   bool
	// relations
	ReferencedColumnNames      []string
	ReferencedTableName        string
	TargetClass                string
	TargetSchema               string
	DeleteRule                 string
	UpdateRule                 string
	Cascade                    []string
	CreateForeignKeyConstraint bool
}

// SchemaEntity is the metadata of one table.
type SchemaEntity struct {
	ClassName   string
	Collection  string
	Schema      string
	CompositePK bool
	PivotTable  bool
	// Props is meta.props: the primary keys first, then the other
	// properties in definition order.
	Props   []*SchemaProperty
	Indexes []SchemaIndex
	Checks  []SchemaCheck
}

// Property returns the property with the given name.
func (e *SchemaEntity) Property(name string) *SchemaProperty {
	for _, p := range e.Props {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// SchemaMetadata derives the metadata of the given entities (the models of
// one module) and of the pivot tables MikroORM defines for them, in the
// order of MetadataStorage.getAll() (mapDiscoveredEntities: class names in
// descending localeCompare order).
func SchemaMetadata(entities []*Entity) ([]*SchemaEntity, error) {
	d := &deriver{entities: map[string]*Entity{}, byName: map[string]*ormMeta{}}
	for _, e := range entities {
		d.entities[parseEntityName(e).modelName] = e
	}
	for _, e := range entities {
		m, err := d.createEntity(e)
		if err != nil {
			return nil, err
		}
		d.metas = append(d.metas, m)
		d.byName[m.className] = m
	}
	if err := d.discover(); err != nil {
		return nil, err
	}
	all := append(append([]*ormMeta{}, d.metas...), d.pivots...)
	sort.SliceStable(all, func(i, j int) bool { return LocaleCompare(all[j].className, all[i].className) < 0 })
	out := make([]*SchemaEntity, len(all))
	for i, m := range all {
		out[i] = d.schemaEntity(m)
	}
	return out, nil
}

func (d *deriver) schemaEntity(m *ormMeta) *SchemaEntity {
	e := &SchemaEntity{
		ClassName:   m.className,
		Collection:  m.tableName(),
		Schema:      m.schema,
		CompositePK: m.compositePK,
		PivotTable:  m.pivotTable,
		Indexes:     m.indexes,
		Checks:      m.checks,
	}
	var primary, rest []*SchemaProperty
	for _, p := range m.props {
		sp := &SchemaProperty{
			Name:                       p.name,
			Kind:                       p.kind,
			FieldNames:                 p.fieldNames,
			ColumnTypes:                p.columnTypes,
			Nullable:                   p.nullable != nil && *p.nullable,
			Primary:                    p.primary,
			Persist:                    p.persist == nil || *p.persist,
			Enum:                       p.enum,
			Items:                      p.items,
			DefaultRaw:                 p.defaultRaw,
			Autoincrement:              p.autoincrement,
			Owner:                      p.owner != nil && *p.owner,
			Unique:                     p.unique != nil && *p.unique,
			ReferencedColumnNames:      p.referencedColumnNames,
			ReferencedTableName:        p.referencedTableName,
			DeleteRule:                 p.deleteRule,
			UpdateRule:                 p.updateRule,
			Cascade:                    p.cascade,
			CreateForeignKeyConstraint: p.createForeignKeyConstraint,
		}
		switch {
		case p.kind == "m:1" || p.kind == "1:1":
			sp.Unsigned = boolPtr(false)
		case p.primary:
			sp.Unsigned = boolPtr(false)
		}
		if p.kind != "scalar" {
			sp.TargetClass = p.typ
			if t := d.byName[p.typ]; t != nil {
				sp.TargetSchema = t.schema
			}
		}
		if p.primary {
			primary = append(primary, sp)
		} else {
			rest = append(rest, sp)
		}
	}
	e.Props = append(primary, rest...)
	return e
}

// createPsqlIndexStatement mirrors createPsqlIndexStatementHelper: the name
// and the expression of an index.
func createPsqlIndexStatement(name, qualifiedName string, columns []string, typ, where string, unique bool) (string, string) {
	tokens := strings.Split(strings.ReplaceAll(qualifiedName, `"`, ""), ".")
	tableName := qualifiedName
	tableReference := `"` + qualifiedName + `"`
	if len(tokens) > 1 {
		tableName = strings.Join(tokens[1:], ".")
		tableReference = `"` + tokens[0] + `"."` + tableName + `"`
	}
	if name == "" {
		name = "IDX_" + tableName + "_" + strings.Join(columns, "_")
		if unique {
			name += "_unique"
		}
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = `"` + c + `"`
	}
	typeStr := ""
	if typ != "" {
		typeStr = " USING " + typ
	}
	optionsStr := ""
	if where != "" {
		optionsStr = " WHERE " + where
	}
	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}
	return name, fmt.Sprintf(`CREATE %sINDEX IF NOT EXISTS "%s" ON %s%s (%s)%s`, uniqueStr, name, tableReference, typeStr, strings.Join(quoted, ", "), optionsStr)
}

// applyIndexes mirrors applyIndexes (apply-indexes.ts): the indexes of a
// field (.index() / .unique()).
func (m *ormMeta) applyIndexes(tableName string, p *Property) {
	for _, idx := range p.Field.Indexes {
		name, expr := createPsqlIndexStatement(idx.Name, tableName, []string{p.Name}, "", "deleted_at IS NULL", idx.Type == "unique")
		m.indexes = append(m.indexes, SchemaIndex{Name: name, Expression: expr})
	}
}

// applyEntityIndexes mirrors applyEntityIndexes (apply-indexes.ts) with
// validateIndexFields (build-indexes.ts).
func (m *ormMeta) applyEntityIndexes(tableName string, indexes []Index) error {
	for _, idx := range indexes {
		if len(idx.On) == 0 {
			return fmt.Errorf(`"on" is a required property when applying indexes on a DML entity`)
		}
		var invalid []string
		for _, f := range idx.On {
			if m.prop(f) == nil {
				invalid = append(invalid, f)
			}
		}
		if len(invalid) > 0 {
			return fmt.Errorf("Cannot apply indexes on fields (%s) for model %s", strings.Join(invalid, ", "), m.className)
		}
		name, expr := createPsqlIndexStatement(idx.Name, tableName, idx.On, idx.Type, idx.Where, idx.Unique)
		m.indexes = append(m.indexes, SchemaIndex{Name: name, Expression: expr})
	}
	return nil
}

// applyChecks mirrors applyChecks (apply-checks.ts) with
// MetadataDiscovery.initCheckConstraints (unnamed checks are named
// <table>_check).
func (m *ormMeta) applyChecks(checks []Check) {
	for _, c := range checks {
		name := c.Name
		if name == "" {
			name = m.tableName() + "_check"
		}
		m.checks = append(m.checks, SchemaCheck{Name: name, Expression: c.Expression})
	}
}

// LocaleCompare approximates String.prototype.localeCompare (ICU root
// collation) for identifiers: characters compare case-insensitively first
// (punctuation before digits before letters), then lowercase sorts before
// uppercase.
func LocaleCompare(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	for i := 0; i < len(ra) && i < len(rb); i++ {
		if c := comparePrimary(ra[i], rb[i]); c != 0 {
			return c
		}
	}
	if len(ra) != len(rb) {
		if len(ra) < len(rb) {
			return -1
		}
		return 1
	}
	for i := range ra {
		if ra[i] == rb[i] {
			continue
		}
		la, lb := unicode.IsLower(ra[i]), unicode.IsLower(rb[i])
		if la && !lb {
			return -1
		}
		if lb && !la {
			return 1
		}
		if ra[i] < rb[i] {
			return -1
		}
		return 1
	}
	return 0
}

// punctuationOrder is the ICU root order of the ASCII punctuation and
// symbols.
const punctuationOrder = " _-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$"

func primaryWeight(r rune) (int, rune) {
	switch {
	case unicode.IsLetter(r):
		return 3, unicode.ToLower(r)
	case unicode.IsDigit(r):
		return 2, r
	}
	if i := strings.IndexRune(punctuationOrder, r); i >= 0 {
		return 1, rune(i)
	}
	return 1, r + 128
}

func comparePrimary(a, b rune) int {
	ca, wa := primaryWeight(a)
	cb, wb := primaryWeight(b)
	if ca != cb {
		if ca < cb {
			return -1
		}
		return 1
	}
	if wa != wb {
		if wa < wb {
			return -1
		}
		return 1
	}
	return 0
}
