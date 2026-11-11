package dml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// ResolveMeta derives the Meta of the given entities, the models of one
// module: what upstream gets from MikroORM.init over the module entities
// converted with toMikroORMEntity (create-mikro-orm-entity.ts,
// define-property.ts, define-relationship.ts, then the MikroORM 6 entity
// schema initialization and metadata discovery). Relationships are resolved
// within the given entities. Entities that already have a Meta (decoded from
// the generated upstream models) are left unchanged unless force is set.
func ResolveMeta(entities []*Entity, force ...bool) error {
	metas, err := deriveMetas(entities)
	if err != nil {
		return err
	}
	for i, e := range entities {
		if e.Meta.ClassName != "" && !(len(force) > 0 && force[0]) {
			continue
		}
		e.Meta = metas[i]
	}
	return nil
}

// DeriveMeta returns the derived metadata of the entities without
// assigning it.
func DeriveMeta(entities []*Entity) ([]EntityMeta, error) {
	return deriveMetas(entities)
}

// ormProp is the subset of a MikroORM EntityProperty the derivation needs.
type ormProp struct {
	name                  string
	kind                  string
	typ                   string
	mappedType            string // custom type class: JsonType, ArrayType, BigNumberNumeric
	columnTypes           []string
	fieldNames            []string
	nullable              *bool
	primary               bool
	persist               *bool
	autoincrement         *bool
	hasDefault            bool
	def                   any
	defaultRaw            *string
	items                 []any
	enum                  bool
	mappedBy              string
	inversedBy            string
	owner                 *bool
	mapToPk               bool
	deleteRule            string
	pivotTable            string
	pivotEntity           string
	joinColumns           []string
	inverseJoinColumns    []string
	referencedColumnNames []string
	formula               bool
	cascade               []string
	fixedOrder            bool
	fixedOrderColumn      string
	// schema generator inputs
	unique                     *bool
	updateRule                 string
	createForeignKeyConstraint bool
	referencedTableName        string
}

type ormMeta struct {
	className  string
	collection string
	props      []*ormProp
	pks        []string
	pivot      bool
	// schema generator inputs: the PostgreSQL schema of the table, the
	// @Index and @Check decorators applied by the DML entity builder, and
	// whether the entity is a pivot table MikroORM defines.
	schema      string
	indexes     []SchemaIndex
	checks      []SchemaCheck
	pivotTable  bool
	compositePK bool
}

func (m *ormMeta) prop(name string) *ormProp {
	for _, p := range m.props {
		if p.name == name {
			return p
		}
	}
	return nil
}

// put mirrors `meta.properties[name] = prop`: an existing key keeps its
// position.
func (m *ormMeta) put(p *ormProp) {
	for i, x := range m.props {
		if x.name == p.name {
			m.props[i] = p
			return
		}
	}
	m.props = append(m.props, p)
}

func (m *ormMeta) moveToEnd(name string) {
	for i, x := range m.props {
		if x.name == name {
			m.props = append(append(m.props[:i:i], m.props[i+1:]...), x)
			return
		}
	}
}

func (m *ormMeta) primaryProps() []*ormProp {
	var out []*ormProp
	for _, pk := range m.pks {
		if p := m.prop(pk); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func boolPtr(b bool) *bool { return &b }

func strPtr(s string) *string { return &s }

var underscoreRe = regexp.MustCompile(`([a-z])([A-Z])`)

// underscore mirrors UnderscoreNamingStrategy.underscore.
func underscore(name string) string {
	return strings.ToLower(underscoreRe.ReplaceAllString(name, "${1}_${2}"))
}

// parsedEntityName mirrors parseEntityName.
type parsedEntityName struct {
	tableName              string
	modelName              string
	pgSchema               string
	tableNameWithoutSchema string
}

func parseEntityName(e *Entity) parsedEntityName {
	tableName := common.CamelToSnakeCase(e.TableName)
	parts := strings.Split(tableName, ".")
	out := parsedEntityName{
		tableName: tableName,
		modelName: common.UpperCaseFirst(common.ToCamelCase(e.Name)),
	}
	if len(parts) > 1 {
		out.pgSchema = parts[0]
		out.tableNameWithoutSchema = strings.Join(parts[1:], ".")
	} else {
		out.tableNameWithoutSchema = parts[0]
	}
	return out
}

// ParseEntityName returns the model name, the table name and the PostgreSQL
// schema of an entity, like parseEntityName.
func ParseEntityName(e *Entity) (modelName, tableName, pgSchema, tableNameWithoutSchema string) {
	p := parseEntityName(e)
	return p.modelName, p.tableName, p.pgSchema, p.tableNameWithoutSchema
}

type deriver struct {
	entities map[string]*Entity
	metas    []*ormMeta
	byName   map[string]*ormMeta
	// pivots are the pivot table entities MikroORM defines for the many to
	// many relationships without a pivot entity.
	pivots []*ormMeta
}

func deriveMetas(entities []*Entity) ([]EntityMeta, error) {
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
	out := make([]EntityMeta, len(d.metas))
	for i, m := range d.metas {
		out[i] = m.export()
	}
	return out, nil
}

func (d *deriver) related(e *Entity, rel *Property) (*Entity, error) {
	target, ok := d.entities[common.UpperCaseFirst(common.ToCamelCase(rel.Relation.Entity))]
	if !ok {
		return nil, fmt.Errorf(`Invalid relationship reference for "%s.%s". Make sure to return a DML entity from the relationship callback`, parseEntityName(e).modelName, rel.Name)
	}
	return target, nil
}

// createEntity mirrors createEntity of create-mikro-orm-entity.ts: the
// decorators defineProperty and defineRelationship apply.
func (d *deriver) createEntity(e *Entity) (*ormMeta, error) {
	pn := parseEntityName(e)
	m := &ormMeta{className: pn.modelName, collection: pn.tableName, schema: pn.pgSchema}
	hasID := false
	for i := range e.Properties {
		p := &e.Properties[i]
		if p.Field != nil {
			if p.Field.DataType.Name == "id" {
				if hasID {
					return nil, fmt.Errorf(`The model %s has multiple id properties. A model can only have one id property`, pn.modelName)
				}
				hasID = true
			}
			defineProperty(m, p)
			if !p.Field.Computed && p.Name == "deleted_at" {
				// SPECIAL_PROPERTIES.deleted_at
				if err := m.applyEntityIndexes(pn.tableName, []Index{{On: []string{"deleted_at"}, Where: "deleted_at IS NULL"}}); err != nil {
					return nil, err
				}
			}
			m.applyIndexes(pn.tableName, p)
			continue
		}
		if err := d.defineRelationship(m, e, p); err != nil {
			return nil, err
		}
	}
	if err := m.applyEntityIndexes(pn.tableName, e.Indexes); err != nil {
		return nil, err
	}
	m.applyChecks(e.Checks)
	// EntitySchema.init: initProperties + initPrimaryKeys
	for _, p := range m.props {
		initSchemaProperty(p)
	}
	for _, p := range m.props {
		if p.primary {
			m.pks = append(m.pks, p.name)
		}
	}
	if len(m.pks) == 1 {
		pk := m.prop(m.pks[0])
		if (pk.typ == "number" || pk.typ == "bigint") && pk.autoincrement == nil {
			pk.autoincrement = boolPtr(true)
		}
	}
	return m, nil
}

// initSchemaProperty mirrors EntitySchema.initProperties for one property.
func initSchemaProperty(p *ormProp) {
	switch p.kind {
	case "1:1":
		if p.owner == nil {
			p.owner = boolPtr(p.inversedBy != "" || p.mappedBy == "")
		}
		if *p.owner && p.mappedBy != "" && p.inversedBy == "" {
			p.inversedBy, p.mappedBy = p.mappedBy, ""
		}
		syncJoinColumns(p)
	case "m:1":
		p.owner = boolPtr(true)
		syncJoinColumns(p)
	case "m:n":
		if (p.owner == nil || !*p.owner) && p.mappedBy == "" {
			p.owner = boolPtr(true)
		}
		if p.owner != nil && *p.owner && p.mappedBy != "" && p.inversedBy == "" {
			p.inversedBy, p.mappedBy = p.mappedBy, ""
		}
	}
	// addOneToOne: unique defaults to owner; addManyToOne and owning
	// addOneToOne/addManyToMany: createForeignKeyConstraint defaults to true
	switch p.kind {
	case "1:1":
		if p.unique == nil {
			p.unique = boolPtr(*p.owner)
		}
		if *p.owner {
			p.createForeignKeyConstraint = true
		}
	case "m:1":
		p.createForeignKeyConstraint = true
	case "m:n":
		if p.owner != nil && *p.owner {
			p.createForeignKeyConstraint = true
		}
	}
	if p.kind != "scalar" && p.cascade == nil {
		p.cascade = []string{"persist"}
	}
	if p.formula && p.persist == nil {
		p.persist = boolPtr(false)
	}
}

func syncJoinColumns(p *ormProp) {
	if p.joinColumns != nil && p.fieldNames == nil {
		p.fieldNames = p.joinColumns
	}
	if p.fieldNames != nil && p.joinColumns == nil {
		p.joinColumns = p.fieldNames
	}
}

var columnTypes = map[string]string{
	"boolean":   "boolean",
	"dateTime":  "timestamptz",
	"number":    "integer",
	"bigNumber": "numeric",
	"float":     "real",
	"serial":    "number",
	"text":      "text",
	"json":      "jsonb",
	"array":     "array",
}

var propertyTypes = map[string]string{
	"boolean":   "boolean",
	"dateTime":  "date",
	"number":    "number",
	"bigNumber": "number",
	"float":     "number",
	"serial":    "number",
	"text":      "string",
	"json":      "any",
	"array":     "string[]",
}

func decodeDefault(raw json.RawMessage) any {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	_ = dec.Decode(&v)
	return v
}

// defineProperty mirrors define-property.ts.
func defineProperty(m *ormMeta, p *Property) {
	f := p.Field
	if f.Computed {
		return
	}
	name := p.Name
	withDefault := func(op *ormProp) *ormProp {
		if len(f.DefaultValue) > 0 && string(f.DefaultValue) != "null" {
			op.hasDefault = true
			op.def = decodeDefault(f.DefaultValue)
		}
		return op
	}
	switch name {
	case "created_at", "updated_at":
		m.put(&ormProp{name: name, kind: "scalar", columnTypes: []string{"timestamptz"}, typ: "date", nullable: boolPtr(false), fieldNames: []string{name}, defaultRaw: strPtr("now()")})
		return
	case "deleted_at":
		m.put(&ormProp{name: name, kind: "scalar", columnTypes: []string{"timestamptz"}, typ: "date", nullable: boolPtr(true), fieldNames: []string{name}})
		return
	}
	switch f.DataType.Name {
	case "bigNumber":
		m.put(withDefault(&ormProp{name: name, kind: "scalar", typ: "BigNumberNumeric", mappedType: "BigNumberNumeric", columnTypes: []string{"numeric"}, nullable: boolPtr(f.Nullable), fieldNames: []string{name}}))
		return
	case "array":
		m.put(withDefault(&ormProp{name: name, kind: "scalar", typ: "ArrayType", mappedType: "ArrayType", fieldNames: []string{name}, nullable: boolPtr(f.Nullable)}))
		return
	case "enum":
		var choices []any
		if f.DataType.Options != nil {
			choices = f.DataType.Options.Choices
		}
		typ := "string"
		if len(choices) > 0 {
			if _, ok := choices[0].(string); !ok {
				typ = "number"
			}
		}
		m.put(withDefault(&ormProp{name: name, kind: "scalar", enum: true, items: choices, nullable: boolPtr(f.Nullable), fieldNames: []string{name}, typ: typ}))
		return
	case "id":
		m.put(&ormProp{name: name, kind: "scalar", primary: f.PrimaryKey, columnTypes: []string{"text"}, typ: "string", nullable: boolPtr(false), fieldNames: []string{name}})
		return
	case "json":
		op := &ormProp{name: name, kind: "scalar", columnTypes: []string{"jsonb"}, typ: "any", nullable: boolPtr(f.Nullable), fieldNames: []string{name}}
		if len(f.DefaultValue) > 0 && string(f.DefaultValue) != "null" {
			var buf bytes.Buffer
			_ = json.Compact(&buf, f.DefaultValue)
			op.hasDefault = true
			op.def = buf.String()
		}
		m.put(op)
		return
	case "serial":
		m.put(&ormProp{name: name, kind: "scalar", primary: f.PrimaryKey, autoincrement: boolPtr(true), typ: "number", nullable: boolPtr(f.Nullable), fieldNames: []string{name}})
		return
	case "float":
		m.put(withDefault(&ormProp{name: name, kind: "scalar", columnTypes: []string{"real"}, typ: "number", nullable: boolPtr(f.Nullable), fieldNames: []string{name}}))
		return
	}
	columnType := columnTypes[f.DataType.Name]
	propertyType := propertyTypes[f.DataType.Name]
	if f.PrimaryKey {
		m.put(&ormProp{name: name, kind: "scalar", primary: true, columnTypes: []string{columnType}, typ: propertyType, nullable: boolPtr(false), fieldNames: []string{name}})
		return
	}
	m.put(withDefault(&ormProp{name: name, kind: "scalar", columnTypes: []string{columnType}, typ: propertyType, nullable: boolPtr(f.Nullable), fieldNames: []string{name}}))
}

// assign mirrors the relationship decorators: Object.assign(existing ?? {},
// { name, kind }, options).
func (m *ormMeta) assign(p *ormProp) {
	if existing := m.prop(p.name); existing != nil {
		merged := *existing
		merged.kind = p.kind
		merged.typ = p.typ
		mergeProp(&merged, p)
		m.put(&merged)
		return
	}
	m.put(p)
}

func mergeProp(dst, src *ormProp) {
	if src.columnTypes != nil {
		dst.columnTypes = src.columnTypes
	}
	if src.fieldNames != nil {
		dst.fieldNames = src.fieldNames
	}
	if src.nullable != nil {
		dst.nullable = src.nullable
	}
	if src.persist != nil {
		dst.persist = src.persist
	}
	if src.mappedBy != "" {
		dst.mappedBy = src.mappedBy
	}
	if src.inversedBy != "" {
		dst.inversedBy = src.inversedBy
	}
	if src.owner != nil {
		dst.owner = src.owner
	}
	if src.mapToPk {
		dst.mapToPk = true
	}
	if src.deleteRule != "" {
		dst.deleteRule = src.deleteRule
	}
	if src.pivotTable != "" {
		dst.pivotTable = src.pivotTable
	}
	if src.pivotEntity != "" {
		dst.pivotEntity = src.pivotEntity
	}
	if src.joinColumns != nil {
		dst.joinColumns = src.joinColumns
	}
	if src.inverseJoinColumns != nil {
		dst.inverseJoinColumns = src.inverseJoinColumns
	}
	if src.cascade != nil {
		dst.cascade = src.cascade
	}
	if src.formula {
		dst.formula = true
	}
	if src.unique != nil {
		dst.unique = src.unique
	}
}

func foreignKeyName(rel *Property) string {
	if rel.Relation.Options.ForeignKeyName != "" {
		return rel.Relation.Options.ForeignKeyName
	}
	return common.CamelToSnakeCase(rel.Name + "Id")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// hasMappedBy mirrors `"mappedBy" in relationship`.
func hasMappedBy(rel *Relation) bool {
	return rel.Options.MappedBy != "" || rel.MappedBy != "" || rel.Options.MappedByUndefined
}

func relMappedBy(rel *Relation) string {
	if rel.MappedBy != "" {
		return rel.MappedBy
	}
	return rel.Options.MappedBy
}

// defineRelationship mirrors defineRelationship of define-relationship.ts.
func (d *deriver) defineRelationship(m *ormMeta, e *Entity, p *Property) error {
	related, err := d.related(e, p)
	if err != nil {
		return err
	}
	rel := p.Relation
	rn := parseEntityName(related)
	switch rel.Type {
	case "hasOne":
		shouldRemove := contains(e.Cascades.Delete, p.Name)
		mappedBy := underscore(m.className)
		if hasMappedBy(rel) {
			mappedBy = relMappedBy(rel)
		}
		op := &ormProp{name: p.Name, kind: "1:1", typ: rn.modelName, mappedBy: mappedBy}
		if rel.Nullable {
			op.nullable = boolPtr(true)
		}
		if shouldRemove {
			op.deleteRule = "cascade"
			other, _ := related.Property(mappedBy)
			if !(mappedBy != "" && other != nil && other.Relation != nil && other.Relation.Type == "belongsTo") {
				op.cascade = []string{"persist", "soft-remove"}
			}
		}
		m.assign(op)
	case "hasOneWithFK":
		fk := foreignKeyName(p)
		mappedBy := underscore(m.className)
		if hasMappedBy(rel) {
			mappedBy = relMappedBy(rel)
		}
		op := &ormProp{name: p.Name, kind: "1:1", typ: rn.modelName, fieldNames: []string{fk}, mappedBy: mappedBy, unique: boolPtr(false)}
		if rel.Nullable {
			op.nullable = boolPtr(true)
		}
		if contains(e.Cascades.Delete, p.Name) {
			op.cascade = []string{"persist", "soft-remove"}
		}
		m.assign(op)
		m.put(&ormProp{name: fk, kind: "scalar", typ: "string", columnTypes: []string{"text"}, nullable: boolPtr(rel.Nullable), persist: boolPtr(false), formula: true})
	case "hasMany":
		mappedBy := relMappedBy(rel)
		if mappedBy == "" {
			mappedBy = common.CamelToSnakeCase(m.className)
		}
		op := &ormProp{name: p.Name, kind: "1:m", typ: rn.modelName, mappedBy: mappedBy}
		if contains(e.Cascades.Delete, p.Name) {
			op.cascade = []string{"persist", "soft-remove"}
		}
		m.assign(op)
	case "belongsTo":
		return d.defineBelongsTo(m, e, p, related, rn)
	case "manyToMany":
		return d.defineManyToMany(m, e, p, related, rn)
	}
	return nil
}

func (d *deriver) defineBelongsTo(m *ormMeta, e *Entity, p *Property, related *Entity, rn parsedEntityName) error {
	rel := p.Relation
	mappedBy := relMappedBy(rel)
	if mappedBy == "" {
		mappedBy = common.CamelToSnakeCase(m.className)
	}
	other, _ := related.Property(mappedBy)
	var otherType string
	if other != nil && other.Relation != nil {
		otherType = other.Relation.Type
	}
	shouldCascade := contains(related.Cascades.Delete, mappedBy)
	fk := foreignKeyName(p)
	if other == nil || otherType == "hasMany" || otherType == "manyToMany" {
		detachCascade := relMappedBy(rel) != "" && contains(related.Cascades.Detach, relMappedBy(rel))
		if otherType == "manyToMany" {
			m.put(&ormProp{name: fk, kind: "scalar", typ: "string", columnTypes: []string{"text"}, fieldNames: []string{fk}, nullable: boolPtr(rel.Nullable)})
			op := &ormProp{name: p.Name, kind: "m:1", typ: rn.modelName, nullable: boolPtr(rel.Nullable), persist: boolPtr(false)}
			if shouldCascade || detachCascade {
				op.deleteRule = "cascade"
			}
			m.assign(op)
		} else {
			op := &ormProp{name: fk, kind: "m:1", typ: rn.modelName, columnTypes: []string{"text"}, mapToPk: true, fieldNames: []string{fk}, nullable: boolPtr(rel.Nullable)}
			if shouldCascade {
				op.deleteRule = "cascade"
			}
			m.assign(op)
			m.assign(&ormProp{name: p.Name, kind: "m:1", typ: rn.modelName, fieldNames: []string{fk}, persist: boolPtr(false), nullable: boolPtr(rel.Nullable)})
		}
		return m.applyEntityIndexes(parseEntityName(e).tableName, []Index{{On: []string{fk}, Where: "deleted_at IS NULL"}})
	}
	if otherType == "hasOne" || otherType == "hasOneWithFK" {
		m.put(&ormProp{name: fk, kind: "scalar", columnTypes: []string{"text"}, typ: "string", nullable: boolPtr(rel.Nullable), persist: boolPtr(false), formula: true})
		op := &ormProp{name: p.Name, kind: "1:1", typ: rn.modelName, nullable: boolPtr(rel.Nullable), mappedBy: mappedBy, fieldNames: []string{fk}, owner: boolPtr(true), unique: boolPtr(false)}
		if shouldCascade {
			op.deleteRule = "cascade"
			op.cascade = []string{"persist", "soft-remove"}
		}
		m.assign(op)
		return m.applyEntityIndexes(parseEntityName(e).tableName, []Index{{On: []string{fk}, Where: "deleted_at IS NULL", Unique: true}})
	}
	return fmt.Errorf(`Invalid relationship reference for "%s" on "%s" entity. Make sure to define a hasOne or hasMany relationship`, mappedBy, rn.modelName)
}

// otherSideManyToMany mirrors retrieveOtherSideRelationshipManyToMany.
func (d *deriver) otherSideManyToMany(e *Entity, p *Property, related *Entity) (string, *Property, error) {
	if mb := relMappedBy(p.Relation); mb != "" {
		other, _ := related.Property(mb)
		return mb, other, nil
	}
	self := parseEntityName(e).modelName
	var found []*Property
	for i := range related.Properties {
		rp := &related.Properties[i]
		if rp.Relation == nil || rp.Relation.Type != "manyToMany" {
			continue
		}
		target, ok := d.entities[common.UpperCaseFirst(common.ToCamelCase(rp.Relation.Entity))]
		if !ok {
			return "", nil, fmt.Errorf(`Invalid relationship reference for "%s.%s". Make sure to define the relationship using a factory function`, parseEntityName(related).modelName, rp.Name)
		}
		if parseEntityName(target).modelName == self {
			found = append(found, rp)
		}
	}
	if len(found) > 1 {
		return "", nil, fmt.Errorf(`Invalid relationship reference for "%s.%s". Make sure to set the mappedBy property on one side or the other or both.`, e.Name, p.Name)
	}
	if len(found) == 0 {
		return "", nil, nil
	}
	return found[0].Name, found[0], nil
}

func (d *deriver) defineManyToMany(m *ormMeta, e *Entity, p *Property, related *Entity, rn parsedEntityName) error {
	rel := p.Relation
	mappedBy := relMappedBy(rel)
	otherName, other, err := d.otherSideManyToMany(e, p, related)
	if err != nil {
		return err
	}
	if mappedBy != "" {
		if other == nil {
			return fmt.Errorf(`Missing property "%s" on "%s" entity. Make sure to define it as a relationship`, mappedBy, rn.modelName)
		}
		if other.Relation == nil || other.Relation.Type != "manyToMany" {
			return fmt.Errorf(`Invalid relationship reference for "%s" on "%s" entity. Make sure to define a manyToMany relationship`, mappedBy, rn.modelName)
		}
	} else if other == nil {
		return fmt.Errorf(`Invalid relationship reference for "%s.%s". "mappedBy" should be defined on one side or the other.`, m.className, p.Name)
	}

	var pivotEntityName, pivotTableName string
	if rel.Options.PivotEntity != "" {
		pivot, ok := d.entities[common.UpperCaseFirst(common.ToCamelCase(rel.Options.PivotEntity))]
		if !ok {
			return fmt.Errorf(`Invalid pivotEntity reference for "%s.%s". Make sure to return a DML entity from the pivotEntity callback`, m.className, p.Name)
		}
		pivotEntityName = parseEntityName(pivot).modelName
	}
	tableName := parseEntityName(e).tableNameWithoutSchema
	relatedTableName := rn.tableNameWithoutSchema
	sorted := []string{tableName, relatedTableName}
	sort.Strings(sorted)
	otherOpts := other.Relation.Options
	if pivotEntityName == "" {
		switch {
		case rel.Options.PivotTable != "":
			pivotTableName = rel.Options.PivotTable
		case otherOpts.PivotTable != "":
			pivotTableName = otherOpts.PivotTable
		default:
			pivotTableName = sorted[0] + "_" + common.Pluralize(sorted[1])
		}
	}
	configures := rel.Options.JoinColumn != "" || rel.Options.InverseJoinColumn != "" || rel.Options.PivotTable != ""
	otherConfigures := otherOpts.PivotTable != "" || otherOpts.JoinColumn != "" || otherOpts.InverseJoinColumn != ""
	if configures && otherConfigures {
		return fmt.Errorf(`Invalid relationship reference for "%s.%s". Define "pivotTable", "joinColumn", or "inverseJoinColumn" on only one side of the relationship`, m.className, p.Name)
	}
	var isOwner bool
	switch {
	case configures:
		isOwner = true
	case otherConfigures:
		isOwner = false
	default:
		isOwner = sorted[0] == tableName
	}
	value := mappedBy
	if value == "" {
		value = otherName
	}
	op := &ormProp{name: p.Name, kind: "m:n", typ: rn.modelName, owner: boolPtr(isOwner)}
	if pivotTableName != "" {
		if rn.pgSchema != "" {
			op.pivotTable = rn.pgSchema + "." + pivotTableName
		} else {
			op.pivotTable = pivotTableName
		}
	}
	if pivotEntityName != "" {
		op.pivotEntity = pivotEntityName
	}
	if isOwner {
		op.inversedBy = value
	} else {
		op.mappedBy = value
	}
	if rel.Options.JoinColumn != "" {
		op.joinColumns = []string{rel.Options.JoinColumn}
	}
	if rel.Options.InverseJoinColumn != "" {
		op.inverseJoinColumns = []string{rel.Options.InverseJoinColumn}
	}
	m.assign(op)
	return nil
}

// discover mirrors MetadataDiscovery.processDiscoveredEntities for the
// properties it resolves.
func (d *deriver) discover() error {
	for _, m := range d.metas {
		if pks := m.primaryProps(); len(pks) == 1 && isNumericProp(pks[0]) && pks[0].autoincrement == nil {
			pks[0].autoincrement = boolPtr(true)
		}
	}
	for _, m := range d.metas {
		for _, p := range m.props {
			d.initFieldName(p)
		}
	}
	for _, m := range d.metas {
		for _, p := range m.props {
			d.initCustomType(p)
		}
	}
	for _, m := range d.metas {
		for _, p := range m.props {
			initDefaultValue(p)
			inferTypeFromDefault(p)
			d.initColumnType(p)
		}
	}
	for _, m := range d.metas {
		d.autoWireBidirectional(m)
	}
	for _, m := range d.metas {
		if err := d.processEntity(m); err != nil {
			return err
		}
	}
	return nil
}

func isNumericProp(p *ormProp) bool {
	if len(p.columnTypes) > 0 {
		switch simpleType(p.columnTypes[0]) {
		case "int", "integer", "number", "smallint", "bigint", "tinyint", "serial":
			return true
		}
	}
	return p.typ == "number" || p.typ == "bigint"
}

func simpleType(t string) string {
	t = strings.ToLower(t)
	if i := strings.IndexAny(t, "( "); i >= 0 {
		t = t[:i]
	}
	return t
}

func (d *deriver) target(p *ormProp) *ormMeta { return d.byName[p.typ] }

// initFieldName mirrors MetadataDiscovery.initFieldName.
func (d *deriver) initFieldName(p *ormProp) {
	if len(p.fieldNames) > 0 {
		return
	}
	switch p.kind {
	case "scalar":
		p.fieldNames = []string{underscore(p.name)}
	case "m:1", "1:1":
		p.fieldNames = d.manyToOneFieldName(p, p.name)
	case "m:n":
		if p.owner != nil && *p.owner {
			t := d.target(p)
			for range t.pks {
				p.fieldNames = append(p.fieldNames, underscore(p.name))
			}
		}
	}
}

func (d *deriver) manyToOneFieldName(p *ormProp, name string) []string {
	t := d.target(p)
	var out []string
	for _, pk := range t.primaryProps() {
		d.initFieldName(pk)
		for _, f := range pk.fieldNames {
			out = append(out, underscore(name)+"_"+f)
		}
	}
	return out
}

// initCustomType mirrors MetadataDiscovery.initCustomType for the types the
// DML uses.
func (d *deriver) initCustomType(p *ormProp) {
	if p.mappedType == "" && p.kind == "scalar" && len(p.columnTypes) > 0 && (p.columnTypes[0] == "json" || p.columnTypes[0] == "jsonb") {
		p.mappedType = "JsonType"
	}
	if p.mappedType == "" && (strings.ToLower(p.typ) == "json" || strings.ToLower(p.typ) == "jsonb") {
		p.mappedType = "JsonType"
	}
	if p.mappedType == "" && (strings.ToLower(p.typ) == "array" || strings.HasSuffix(p.typ, "[]")) {
		p.mappedType = "ArrayType"
	}
	if p.mappedType != "" && p.columnTypes == nil {
		switch p.mappedType {
		case "JsonType":
			p.columnTypes = []string{"jsonb"}
		case "ArrayType":
			p.columnTypes = []string{"text[]"}
		case "BigNumberNumeric":
			p.columnTypes = []string{"numeric"}
		}
	}
	if p.mappedType != "" && p.kind == "scalar" && !strings.HasSuffix(p.typ, "[]") {
		p.typ = p.mappedType
	}
	if p.kind == "scalar" && p.columnTypes == nil {
		if ct := scalarColumnType(p); ct != "" {
			p.columnTypes = []string{ct}
		}
	}
}

// scalarColumnType mirrors getMappedType(prop).getColumnType for the
// PostgreSQL platform.
func scalarColumnType(p *ormProp) string {
	if p.enum {
		for _, it := range p.items {
			if _, ok := it.(string); !ok {
				return "smallint"
			}
		}
		return "text"
	}
	switch simpleType(p.typ) {
	case "string", "varchar":
		return "varchar(255)"
	case "text":
		return "text"
	case "number", "int", "integer":
		if p.autoincrement != nil && *p.autoincrement {
			return "serial"
		}
		return "int"
	case "boolean":
		return "boolean"
	case "date", "datetime":
		return "timestamptz"
	case "float":
		return "real"
	case "double":
		return "double precision"
	case "numeric", "decimal":
		return "numeric"
	case "bigint":
		return "bigint"
	}
	return ""
}

// initDefaultValue mirrors MetadataDiscovery.initDefaultValue.
func initDefaultValue(p *ormProp) {
	if p.defaultRaw != nil || !p.hasDefault {
		return
	}
	switch v := p.def.(type) {
	case string:
		p.defaultRaw = strPtr("'" + v + "'")
	case []any:
		if p.mappedType == "ArrayType" {
			p.defaultRaw = strPtr("'" + marshallArray(v) + "'")
		} else {
			p.defaultRaw = strPtr(jsString(v))
		}
	default:
		p.defaultRaw = strPtr(jsString(v))
	}
}

// marshallArray mirrors PostgreSqlPlatform.marshallArray.
func marshallArray(values []any) string {
	parts := make([]string, len(values))
	quoteRe := regexp.MustCompile(`["{},\\]`)
	for i, v := range values {
		s := jsString(v)
		if s == "" || quoteRe.MatchString(s) {
			b, _ := json.Marshal(s)
			s = string(b)
		}
		parts[i] = s
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// jsString mirrors ” + value for JSON values.
func jsString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return strconv.FormatFloat(f, 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			if e == nil {
				parts[i] = ""
			} else {
				parts[i] = jsString(e)
			}
		}
		return strings.Join(parts, ",")
	case map[string]any:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}

// inferTypeFromDefault mirrors MetadataDiscovery.inferTypeFromDefault.
func inferTypeFromDefault(p *ormProp) {
	if (p.defaultRaw == nil && !p.hasDefault) || p.typ != "any" {
		return
	}
	switch p.def.(type) {
	case string:
		p.typ = "string"
	case json.Number, float64:
		p.typ = "number"
	case bool:
		p.typ = "boolean"
	}
}

// initColumnType mirrors MetadataDiscovery.initColumnType.
func (d *deriver) initColumnType(p *ormProp) {
	if p.columnTypes != nil {
		return
	}
	if p.kind == "scalar" {
		if ct := scalarColumnType(p); ct != "" {
			p.columnTypes = []string{ct}
		}
		return
	}
	t := d.target(p)
	if t == nil {
		return
	}
	p.columnTypes = []string{}
	for _, pk := range t.primaryProps() {
		d.initCustomType(pk)
		d.initColumnType(pk)
		cts := pk.columnTypes
		if pk.autoincrement != nil && *pk.autoincrement {
			noAuto := *pk
			noAuto.autoincrement = boolPtr(false)
			noAuto.columnTypes = nil
			if ct := scalarColumnType(&noAuto); ct != "" {
				cts = []string{ct}
			}
		}
		p.columnTypes = append(p.columnTypes, cts...)
	}
}

// autoWireBidirectional mirrors autoWireBidirectionalProperties.
func (d *deriver) autoWireBidirectional(m *ormMeta) {
	for _, p := range m.props {
		if p.kind == "scalar" || (p.owner != nil && *p.owner) || p.mappedBy == "" {
			continue
		}
		t := d.target(p)
		if t == nil {
			continue
		}
		if p2 := t.prop(p.mappedBy); p2 != nil && p2.inversedBy == "" {
			p2.inversedBy = p.name
		}
	}
}

// processEntity mirrors MetadataDiscovery.processEntity.
func (d *deriver) processEntity(m *ormMeta) error {
	m.pks = nil
	for _, p := range m.props {
		if p.primary {
			m.pks = append(m.pks, p.name)
		}
	}
	m.compositePK = len(m.pks) > 1
	if pks := m.primaryProps(); len(pks) == 1 && pks[0].kind != "scalar" && pks[0].deleteRule == "" {
		pks[0].deleteRule = "cascade"
	}
	for _, p := range m.props {
		initNullability(p)
		d.applyNamingStrategy(m, p)
		initDefaultValue(p)
		inferTypeFromDefault(p)
		d.initCustomType(p)
		d.initColumnType(p)
	}
	for _, p := range append([]*ormProp(nil), m.props...) {
		if p.kind == "m:n" && p.owner != nil && *p.owner && p.pivotTable != "" {
			d.definePivotTableEntity(m, p)
		}
	}
	return nil
}

func initNullability(p *ormProp) {
	if p.nullable != nil {
		return
	}
	switch p.kind {
	case "m:1":
		p.nullable = boolPtr(contains(p.cascade, "remove") || contains(p.cascade, "all"))
	case "1:1":
		p.nullable = boolPtr(!(p.owner != nil && *p.owner) || contains(p.cascade, "remove") || contains(p.cascade, "all"))
	default:
		p.nullable = boolPtr(false)
	}
}

func (d *deriver) applyNamingStrategy(m *ormMeta, p *ormProp) {
	if p.fieldNames == nil {
		d.initFieldName(p)
	}
	switch p.kind {
	case "m:n":
		d.initManyToManyFields(m, p)
	case "m:1", "1:1":
		d.initManyToOneFields(p)
	case "1:m":
		d.initOneToManyFields(p)
	}
}

func (d *deriver) pkFieldNames(m *ormMeta) []string {
	var out []string
	for _, pk := range m.primaryProps() {
		d.initFieldName(pk)
		out = append(out, pk.fieldNames...)
	}
	return out
}

func (d *deriver) initManyToOneFields(p *ormProp) {
	t := d.target(p)
	fieldNames := d.pkFieldNames(t)
	if p.referencedTableName == "" {
		p.referencedTableName = t.tableName()
	}
	if p.joinColumns == nil {
		for _, f := range fieldNames {
			p.joinColumns = append(p.joinColumns, underscore(p.name)+"_"+f)
		}
	}
	if p.referencedColumnNames == nil {
		p.referencedColumnNames = fieldNames
	}
}

func (d *deriver) initOneToManyFields(p *ormProp) {
	t := d.target(p)
	if p.joinColumns == nil {
		p.joinColumns = []string{underscore(p.name) + "_id"}
	}
	if p.referencedColumnNames == nil {
		p.referencedColumnNames = d.pkFieldNames(t)
	}
}

func (d *deriver) initManyToManyFields(m *ormMeta, p *ormProp) {
	t := d.target(p)
	pivot := d.byName[p.pivotEntity]
	if p.pivotEntity == "" {
		pivot = nil
	}
	if pivot != nil {
		pivot.pivot = true
		p.pivotTable = pivot.collection
		var pks, fks []*ormProp
		for _, x := range pivot.props {
			if x.primary {
				pks = append(pks, x)
			}
			if x.kind == "m:1" {
				fks = append(fks, x)
			}
		}
		if len(pks) == 1 {
			p.fixedOrder = true
			p.fixedOrderColumn = pks[0].name
		}
		if len(pks) == 2 || len(fks) >= 2 {
			owner := p
			if p.mappedBy != "" {
				owner = t.prop(p.mappedBy)
			}
			first, second := d.ensureCorrectFKOrder(pivot, owner)
			if first != nil {
				if p.joinColumns == nil {
					p.joinColumns = first.fieldNames
				}
				if p.inverseJoinColumns == nil {
					p.inverseJoinColumns = second.fieldNames
				}
			}
		}
	}
	if p.pivotTable == "" && p.owner != nil && *p.owner {
		p.pivotTable = underscore(m.collection) + "_" + underscore(p.name)
	}
	if p.mappedBy != "" {
		p2 := t.prop(p.mappedBy)
		d.initManyToManyFields(t, p2)
		p.pivotTable = p2.pivotTable
		if p2.pivotEntity != "" {
			p.pivotEntity = p2.pivotEntity
		} else {
			p.pivotEntity = p2.pivotTable
		}
		p.fixedOrder = p2.fixedOrder
		p.fixedOrderColumn = p2.fixedOrderColumn
		p.joinColumns = p2.inverseJoinColumns
		p.inverseJoinColumns = p2.joinColumns
	}
	if p.referencedColumnNames == nil {
		p.referencedColumnNames = d.pkFieldNames(m)
	}
	if p.joinColumns == nil {
		for _, ref := range p.referencedColumnNames {
			p.joinColumns = append(p.joinColumns, underscore(m.className)+"_"+ref)
		}
	}
	if p.inverseJoinColumns == nil {
		p.inverseJoinColumns = d.manyToOneFieldName(p, t.className)
	}
}

// ensureCorrectFKOrder mirrors ensureCorrectFKOrderInPivotEntity.
func (d *deriver) ensureCorrectFKOrder(pivot *ormMeta, owner *ormProp) (*ormProp, *ormProp) {
	var pks, fks []*ormProp
	for _, x := range pivot.props {
		if x.primary {
			pks = append(pks, x)
		}
		if x.kind == "m:1" {
			fks = append(fks, x)
		}
	}
	var first, second *ormProp
	switch {
	case len(pks) == 2:
		first, second = pks[0], pks[1]
	case len(fks) >= 2:
		first, second = fks[0], fks[1]
	default:
		return nil, nil
	}
	if owner != nil && owner.typ == first.typ && first.typ != second.typ {
		pivot.moveToEnd(first.name)
		first, second = second, first
	}
	return first, second
}

// definePivotTableEntity mirrors definePivotTableEntity for what it changes
// on the module entities.
func (d *deriver) definePivotTableEntity(m *ormMeta, p *ormProp) {
	if pivot := d.byName[p.pivotEntity]; p.pivotEntity != "" && pivot != nil {
		d.ensureCorrectFKOrder(pivot, p)
		return
	}
	if exists := d.byName[p.pivotTable]; exists != nil {
		p.pivotEntity = exists.className
		return
	}
	for _, pv := range d.pivots {
		if pv.className == p.pivotTable {
			p.pivotEntity = pv.className
			return
		}
	}
	p.pivotEntity = p.pivotTable
	d.pivots = append(d.pivots, d.newPivotTableEntity(m, p))
}

// newPivotTableEntity mirrors the entity definePivotTableEntity creates for
// a many to many relationship without a pivot entity: a table with the two
// foreign keys as a composite primary key.
func (d *deriver) newPivotTableEntity(m *ormMeta, p *ormProp) *ormMeta {
	tableName, schemaName := p.pivotTable, ""
	if i := strings.Index(tableName, "."); i >= 0 {
		schemaName, tableName = tableName[:i], tableName[i+1:]
	}
	if schemaName == "" {
		schemaName = m.schema
	}
	target := d.target(p)
	pivot := &ormMeta{className: p.pivotTable, collection: tableName, schema: schemaName, pivotTable: true, compositePK: true}
	owner := d.definePivotProperty(p, m.className+"_owner", m, true)
	inverse := d.definePivotProperty(p, target.className+"_inverse", target, false)
	pivot.props = []*ormProp{owner, inverse}
	pivot.pks = []string{owner.name, inverse.name}
	return pivot
}

// definePivotProperty mirrors MetadataDiscovery.definePivotProperty.
func (d *deriver) definePivotProperty(p *ormProp, name string, meta *ormMeta, owner bool) *ormProp {
	ret := &ormProp{
		name:                       name,
		typ:                        meta.className,
		kind:                       "m:1",
		cascade:                    []string{"all"},
		primary:                    true,
		autoincrement:              boolPtr(false),
		createForeignKeyConstraint: p.createForeignKeyConstraint,
		owner:                      boolPtr(owner),
	}
	schema := meta.schema
	if schema == "" {
		schema = "public"
	}
	ret.referencedTableName = schema + "." + meta.tableName()
	if owner {
		ret.referencedColumnNames = p.referencedColumnNames
		ret.fieldNames = p.joinColumns
		ret.joinColumns = p.joinColumns
	} else {
		ret.fieldNames = p.inverseJoinColumns
		ret.joinColumns = p.inverseJoinColumns
		ret.referencedColumnNames = d.pkFieldNames(meta)
	}
	d.initColumnType(ret)
	return ret
}

// tableName is the table name without its PostgreSQL schema.
func (m *ormMeta) tableName() string {
	if i := strings.Index(m.collection, "."); i >= 0 {
		return m.collection[i+1:]
	}
	return m.collection
}

func (m *ormMeta) export() EntityMeta {
	out := EntityMeta{ClassName: m.className, Collection: m.collection, PrimaryKey: append([]string{}, m.pks...)}
	if strings.Contains(out.Collection, ".") {
		out.Collection = out.Collection[strings.Index(out.Collection, ".")+1:]
	}
	for _, p := range m.props {
		mp := MetaProperty{
			Name:       p.name,
			Kind:       p.kind,
			Type:       p.typ,
			FieldNames: p.fieldNames,
			Nullable:   p.nullable != nil && *p.nullable,
			Primary:    p.primary,
			Items:      p.items,
			MappedBy:   p.mappedBy,
			InversedBy: p.inversedBy,
			MapToPk:    p.mapToPk,
			DeleteRule: p.deleteRule,
			PivotTable: p.pivotTable,
		}
		if len(p.columnTypes) == 1 {
			mp.ColumnType = p.columnTypes[0]
		}
		if p.persist != nil && !*p.persist {
			mp.Persist = boolPtr(false)
		}
		if p.autoincrement != nil {
			mp.Autoincrement = *p.autoincrement
		}
		if p.hasDefault {
			raw, _ := json.Marshal(p.def)
			mp.Default = raw
		}
		if p.defaultRaw != nil {
			mp.DefaultRaw = *p.defaultRaw
		}
		if p.owner != nil {
			mp.Owner = *p.owner
		}
		if p.kind != "scalar" {
			mp.Entity = p.typ
			mp.PivotEntity = p.pivotEntity
			mp.ReferencedColumnNames = p.referencedColumnNames
		}
		if p.kind == "m:n" {
			mp.JoinColumns = p.joinColumns
			mp.InverseJoinColumns = p.inverseJoinColumns
		}
		out.Properties = append(out.Properties, mp)
	}
	return out
}
