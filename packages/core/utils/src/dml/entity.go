// Package dml mirrors @medusajs/utils/src/dml: the data model language used
// by modules to describe their entities.
//
// An Entity holds the DML definition as written upstream (properties,
// relationships, indexes, cascades) together with Meta, the resolved
// metadata upstream derives from it (table, columns, foreign keys, pivot
// tables) through its MikroORM entity builder.
package dml

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DataTypeOptions are the options of a property data type.
type DataTypeOptions struct {
	Prefix       string `json:"prefix,omitempty"`
	Searchable   bool   `json:"searchable,omitempty"`
	Translatable bool   `json:"translatable,omitempty"`
	Choices      []any  `json:"choices,omitempty"`
}

// DataType is the data type of a field: id, text, enum, dateTime, json,
// bigNumber, number, boolean, float, serial or array.
type DataType struct {
	Name    string           `json:"name"`
	Options *DataTypeOptions `json:"options,omitempty"`
}

// FieldIndex is an index declared on a single property.
type FieldIndex struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// Field is a scalar property.
type Field struct {
	DataType     DataType        `json:"dataType"`
	Nullable     bool            `json:"nullable"`
	Computed     bool            `json:"computed"`
	PrimaryKey   bool            `json:"primaryKey,omitempty"`
	DefaultValue json.RawMessage `json:"defaultValue,omitempty"`
	Indexes      []FieldIndex    `json:"indexes,omitempty"`
}

// RelationOptions are the options of a relationship.
type RelationOptions struct {
	MappedBy          string `json:"mappedBy,omitempty"`
	ForeignKey        bool   `json:"foreignKey,omitempty"`
	ForeignKeyName    string `json:"foreignKeyName,omitempty"`
	PivotTable        string `json:"pivotTable,omitempty"`
	PivotEntity       string `json:"pivotEntity,omitempty"`
	JoinColumn        string `json:"joinColumn,omitempty"`
	InverseJoinColumn string `json:"inverseJoinColumn,omitempty"`
}

// Relation is a relationship property: belongsTo, hasMany, hasOne,
// hasOneWithFK or manyToMany.
type Relation struct {
	Type       string          `json:"type"`
	Entity     string          `json:"entity"`
	MappedBy   string          `json:"mappedBy,omitempty"`
	Nullable   bool            `json:"nullable"`
	Searchable bool            `json:"searchable"`
	Options    RelationOptions `json:"options"`
}

// Property is one entry of an entity schema; exactly one of Field and
// Relation is set.
type Property struct {
	Name     string    `json:"name"`
	Field    *Field    `json:"field,omitempty"`
	Relation *Relation `json:"relation,omitempty"`
}

// Index is an entity level index.
type Index struct {
	Name   string   `json:"name,omitempty"`
	On     []string `json:"on"`
	Unique bool     `json:"unique,omitempty"`
	Where  string   `json:"where,omitempty"`
	Type   string   `json:"type,omitempty"`
}

// Check is a check constraint.
type Check struct {
	Name       string `json:"name,omitempty"`
	Expression string `json:"expression"`
}

// Cascades lists the relations deleted or detached with the entity.
type Cascades struct {
	Delete []string `json:"delete,omitempty"`
	Detach []string `json:"detach,omitempty"`
}

// MetaProperty is a resolved entity property: a column, a foreign key or a
// relation.
type MetaProperty struct {
	Name                  string          `json:"name"`
	Kind                  string          `json:"kind"`
	Type                  string          `json:"type,omitempty"`
	ColumnType            string          `json:"columnType,omitempty"`
	FieldNames            []string        `json:"fieldNames,omitempty"`
	Nullable              bool            `json:"nullable"`
	Primary               bool            `json:"primary,omitempty"`
	Persist               *bool           `json:"persist,omitempty"`
	Autoincrement         bool            `json:"autoincrement,omitempty"`
	Default               json.RawMessage `json:"default,omitempty"`
	DefaultRaw            string          `json:"defaultRaw,omitempty"`
	Items                 []any           `json:"items,omitempty"`
	Entity                string          `json:"entity,omitempty"`
	MappedBy              string          `json:"mappedBy,omitempty"`
	InversedBy            string          `json:"inversedBy,omitempty"`
	Owner                 bool            `json:"owner,omitempty"`
	MapToPk               bool            `json:"mapToPk,omitempty"`
	DeleteRule            string          `json:"deleteRule,omitempty"`
	PivotTable            string          `json:"pivotTable,omitempty"`
	PivotEntity           string          `json:"pivotEntity,omitempty"`
	JoinColumns           []string        `json:"joinColumns,omitempty"`
	InverseJoinColumns    []string        `json:"inverseJoinColumns,omitempty"`
	ReferencedColumnNames []string        `json:"referencedColumnNames,omitempty"`
}

// EntityMeta is the resolved metadata of an entity.
type EntityMeta struct {
	ClassName  string         `json:"className"`
	Collection string         `json:"collection"`
	PrimaryKey []string       `json:"primaryKeys"`
	Properties []MetaProperty `json:"properties"`
}

// Entity is a DML entity.
type Entity struct {
	Name       string     `json:"name"`
	TableName  string     `json:"tableName"`
	Properties []Property `json:"properties"`
	Indexes    []Index    `json:"indexes,omitempty"`
	Checks     []Check    `json:"checks,omitempty"`
	Cascades   Cascades   `json:"cascades"`
	Meta       EntityMeta `json:"meta"`
}

// Property returns the DML property with the given name.
func (e *Entity) Property(name string) (*Property, bool) {
	for i := range e.Properties {
		if e.Properties[i].Name == name {
			return &e.Properties[i], true
		}
	}
	return nil, false
}

// MetaProperty returns the resolved property with the given name.
func (e *Entity) MetaProperty(name string) (*MetaProperty, bool) {
	for i := range e.Meta.Properties {
		if e.Meta.Properties[i].Name == name {
			return &e.Meta.Properties[i], true
		}
	}
	return nil, false
}

// MustParseEntity decodes a generated entity, rejecting unknown keys so that
// nothing from the upstream definition is silently dropped.
func MustParseEntity(data string) *Entity {
	var e Entity
	dec := json.NewDecoder(bytes.NewReader([]byte(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		panic(fmt.Sprintf("invalid DML entity: %v", err))
	}
	registerTranslatableFields(&e)
	return &e
}
