package dml

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// This file mirrors entity-builder.ts and the property/relationship classes
// of @medusajs/utils/src/dml: the `model` API a module uses to define its
// data models in Go. The result is the same Entity the generated (upstream)
// models are decoded into; its Meta is resolved with ResolveMeta.

// PropertyBuilder is a schema property: a field or a relationship.
type PropertyBuilder interface {
	parse(name string) Property
}

// SchemaEntry is one named property of a model schema (the schema object of
// model.define, in declaration order).
type SchemaEntry struct {
	Name     string
	Property PropertyBuilder
}

// Schema is a model schema.
type Schema []SchemaEntry

// FieldBuilder mirrors the property classes (TextProperty, NumberProperty,
// ...) and their modifiers.
type FieldBuilder struct {
	field Field
}

func (b *FieldBuilder) parse(name string) Property {
	f := b.field
	f.Indexes = append([]FieldIndex(nil), b.field.Indexes...)
	if b.field.DataType.Options != nil {
		opts := *b.field.DataType.Options
		f.DataType.Options = &opts
	}
	return Property{Name: name, Field: &f}
}

func (b *FieldBuilder) clone() *FieldBuilder {
	c := *b
	c.field.Indexes = append([]FieldIndex(nil), b.field.Indexes...)
	if b.field.DataType.Options != nil {
		opts := *b.field.DataType.Options
		c.field.DataType.Options = &opts
	}
	return &c
}

// Nullable mirrors .nullable().
func (b *FieldBuilder) Nullable() *FieldBuilder {
	c := b.clone()
	c.field.Nullable = true
	return c
}

// Computed mirrors .computed().
func (b *FieldBuilder) Computed() *FieldBuilder {
	c := b.clone()
	c.field.Computed = true
	return c
}

// PrimaryKey mirrors .primaryKey().
func (b *FieldBuilder) PrimaryKey() *FieldBuilder {
	c := b.clone()
	c.field.PrimaryKey = true
	return c
}

// Index mirrors .index(name?).
func (b *FieldBuilder) Index(name ...string) *FieldBuilder {
	c := b.clone()
	c.field.Indexes = append(c.field.Indexes, FieldIndex{Name: firstString(name), Type: "index"})
	return c
}

// Unique mirrors .unique(name?).
func (b *FieldBuilder) Unique(name ...string) *FieldBuilder {
	c := b.clone()
	c.field.Indexes = append(c.field.Indexes, FieldIndex{Name: firstString(name), Type: "unique"})
	return c
}

// Default mirrors .default(value).
func (b *FieldBuilder) Default(value any) *FieldBuilder {
	c := b.clone()
	raw, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("dml: invalid default value: %v", err))
	}
	c.field.DefaultValue = raw
	return c
}

// Searchable mirrors .searchable() (text, number and autoincrement).
func (b *FieldBuilder) Searchable() *FieldBuilder {
	c := b.clone()
	if c.field.DataType.Options == nil {
		c.field.DataType.Options = &DataTypeOptions{}
	}
	c.field.DataType.Options.Searchable = true
	return c
}

// Translatable mirrors .translatable() (text).
func (b *FieldBuilder) Translatable() *FieldBuilder {
	c := b.clone()
	if c.field.DataType.Options == nil {
		c.field.DataType.Options = &DataTypeOptions{}
	}
	c.field.DataType.Options.Translatable = true
	return c
}

func firstString(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func newField(name string, opts *DataTypeOptions) *FieldBuilder {
	return &FieldBuilder{field: Field{DataType: DataType{Name: name, Options: opts}}}
}

// IdOptions are the options of model.id().
type IdOptions struct {
	Prefix string
}

// Id mirrors model.id().
func Id(options ...IdOptions) *FieldBuilder {
	o := &DataTypeOptions{}
	if len(options) > 0 {
		o.Prefix = options[0].Prefix
	}
	return newField("id", o)
}

// Text mirrors model.text().
func Text() *FieldBuilder { return newField("text", &DataTypeOptions{}) }

// Boolean mirrors model.boolean().
func Boolean() *FieldBuilder { return newField("boolean", nil) }

// Number mirrors model.number().
func Number() *FieldBuilder { return newField("number", &DataTypeOptions{}) }

// BigNumber mirrors model.bigNumber().
func BigNumber() *FieldBuilder { return newField("bigNumber", nil) }

// Float mirrors model.float().
func Float() *FieldBuilder { return newField("float", nil) }

// Autoincrement mirrors model.autoincrement().
func Autoincrement() *FieldBuilder { return newField("serial", &DataTypeOptions{}) }

// Array mirrors model.array().
func Array() *FieldBuilder { return newField("array", nil) }

// DateTime mirrors model.dateTime().
func DateTime() *FieldBuilder { return newField("dateTime", nil) }

// JSON mirrors model.json().
func JSON() *FieldBuilder { return newField("json", nil) }

// Enum mirrors model.enum(values).
func Enum(values ...any) *FieldBuilder {
	return newField("enum", &DataTypeOptions{Choices: values})
}

// RelationshipOptions mirrors RelationshipOptions and ManyToManyOptions.
type RelationshipOptions struct {
	// MappedBy is the property on the other side of the relationship.
	MappedBy string
	// ForeignKey (hasOne only) makes it a hasOneWithFK relationship.
	ForeignKey        bool
	ForeignKeyName    string
	PivotTable        string
	PivotEntity       string
	JoinColumn        string
	InverseJoinColumn string
	// MappedByUndefined mirrors { mappedBy: undefined }: the relationship
	// has no other side.
	MappedByUndefined bool
}

// RelationBuilder mirrors the relationship classes.
type RelationBuilder struct {
	rel Relation
}

func (b *RelationBuilder) parse(name string) Property {
	r := b.rel
	return Property{Name: name, Relation: &r}
}

// Nullable mirrors .nullable().
func (b *RelationBuilder) Nullable() *RelationBuilder {
	c := *b
	c.rel.Nullable = true
	return &c
}

// Searchable mirrors .searchable().
func (b *RelationBuilder) Searchable() *RelationBuilder {
	c := *b
	c.rel.Searchable = true
	return &c
}

func newRelation(typ, entity string, options []RelationshipOptions) *RelationBuilder {
	var o RelationshipOptions
	if len(options) > 0 {
		o = options[0]
	}
	return &RelationBuilder{rel: Relation{
		Type:     typ,
		Entity:   entity,
		MappedBy: o.MappedBy,
		Options: RelationOptions{
			MappedBy:          o.MappedBy,
			ForeignKey:        o.ForeignKey,
			ForeignKeyName:    o.ForeignKeyName,
			PivotTable:        o.PivotTable,
			PivotEntity:       o.PivotEntity,
			JoinColumn:        o.JoinColumn,
			InverseJoinColumn: o.InverseJoinColumn,
			MappedByUndefined: o.MappedByUndefined,
		},
	}}
}

// HasOne mirrors model.hasOne(() => Entity, options); with ForeignKey it is
// a hasOneWithFK relationship. The related entity is given by its name.
func HasOne(entity string, options ...RelationshipOptions) *RelationBuilder {
	if len(options) > 0 && options[0].ForeignKey {
		return newRelation("hasOneWithFK", entity, options)
	}
	return newRelation("hasOne", entity, options)
}

// BelongsTo mirrors model.belongsTo(() => Entity, options).
func BelongsTo(entity string, options ...RelationshipOptions) *RelationBuilder {
	return newRelation("belongsTo", entity, options)
}

// HasMany mirrors model.hasMany(() => Entity, options).
func HasMany(entity string, options ...RelationshipOptions) *RelationBuilder {
	return newRelation("hasMany", entity, options)
}

// ManyToMany mirrors model.manyToMany(() => Entity, options).
func ManyToMany(entity string, options ...RelationshipOptions) *RelationBuilder {
	return newRelation("manyToMany", entity, options)
}

var implicitProperties = []string{"created_at", "updated_at", "deleted_at"}

// DefineConfig mirrors the object form of model.define's first argument.
type DefineConfig struct {
	Name      string
	TableName string
}

// Define mirrors model.define(nameOrConfig, schema): nameOrConfig is a
// string (the table name) or a DefineConfig. The schema gets the raw_
// properties of its big numbers and the implicit created_at, updated_at and
// deleted_at properties.
func Define(nameOrConfig any, schema Schema) *Entity {
	var implicit []string
	for _, e := range schema {
		for _, p := range implicitProperties {
			if e.Name == p {
				implicit = append(implicit, p)
			}
		}
	}
	if len(implicit) > 0 {
		panic(fmt.Sprintf(`Cannot define field(s) "%s" as they are implicitly defined on every model`, strings.Join(implicit, ",")))
	}

	var name, tableName string
	switch c := nameOrConfig.(type) {
	case string:
		name, tableName = c, c
	case DefineConfig:
		if c.TableName == "" {
			panic(fmt.Sprintf(`Missing "tableName" property in the config object for "%s" entity`, c.Name))
		}
		name, tableName = c.Name, c.TableName
		if name == "" {
			name = c.TableName
		}
	default:
		panic("dml: Define takes a name or a DefineConfig")
	}
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		name = strings.Join(parts[1:], ".")
	}

	e := &Entity{
		Name:      common.UpperCaseFirst(common.ToCamelCase(name)),
		TableName: tableName,
	}
	for _, entry := range schema {
		e.Properties = append(e.Properties, entry.Property.parse(entry.Name))
	}
	// createBigNumberProperties
	for _, entry := range schema {
		p := entry.Property.parse(entry.Name)
		if p.Field == nil || p.Field.DataType.Name != "bigNumber" {
			continue
		}
		raw := &Field{DataType: DataType{Name: "json"}, Nullable: p.Field.Nullable, Computed: p.Field.Computed}
		if len(p.Field.DefaultValue) > 0 {
			raw.DefaultValue = bigNumberRawDefault(p.Field.DefaultValue)
		}
		e.Properties = append(e.Properties, Property{Name: "raw_" + entry.Name, Field: raw})
	}
	// createDefaultProperties
	e.Properties = append(e.Properties,
		Property{Name: "created_at", Field: &Field{DataType: DataType{Name: "dateTime"}}},
		Property{Name: "updated_at", Field: &Field{DataType: DataType{Name: "dateTime"}}},
		Property{Name: "deleted_at", Field: &Field{DataType: DataType{Name: "dateTime"}, Nullable: true}},
	)
	registerTranslatableFields(e)
	return e
}

// bigNumberRawDefault mirrors the default of a raw_ property: the raw value
// of new BigNumber(default) with trimmed zeros.
func bigNumberRawDefault(def json.RawMessage) json.RawMessage {
	var v any
	_ = json.Unmarshal(def, &v)
	bn, err := totals.NewBigNumber(v)
	if err != nil {
		panic(fmt.Sprintf("dml: invalid big number default: %v", err))
	}
	raw := bn.TrimmedRaw()
	out, _ := json.Marshal(struct {
		Value     string `json:"value"`
		Precision int    `json:"precision"`
	}{raw.Value, raw.Precision})
	return out
}

// WithCascades mirrors .cascades(options).
func (e *Entity) WithCascades(c Cascades) *Entity {
	for _, rel := range c.Delete {
		if p, ok := e.Property(rel); ok && p.Relation != nil && p.Relation.Type == "belongsTo" {
			panic(fmt.Sprintf(`Cannot cascade delete "%s" relationship(s) from "%s" entity. Child to parent cascades are not allowed`, rel, e.Name))
		}
	}
	e.Cascades = c
	return e
}

// WithIndexes mirrors .indexes(indexes) for indexes whose where condition is
// given as SQL: the condition gets the soft delete scope
// (transformIndexWhere) unless WhereNull is set.
func (e *Entity) WithIndexes(indexes []Index) *Entity {
	out := make([]Index, len(indexes))
	for i, idx := range indexes {
		if idx.WhereNull {
			idx.Where = ""
		} else {
			idx.Where = transformWhere(idx.Where)
		}
		idx.WhereNull = false
		out[i] = idx
	}
	e.Indexes = out
	return e
}

// transformWhere mirrors transformWhere (build-indexes.ts).
func transformWhere(where string) string {
	lower := strings.ToLower(where)
	includesDeletable := strings.Contains(lower, "deleted_at is null") || strings.Contains(lower, "deleted_at is not null")
	if where != "" && !includesDeletable {
		where += " AND deleted_at IS NULL"
	}
	if where == "" {
		where = "deleted_at IS NULL"
	}
	return where
}

// WithChecks mirrors .checks(checks) for checks given as SQL expressions.
func (e *Entity) WithChecks(checks []Check) *Entity {
	e.Checks = checks
	return e
}
