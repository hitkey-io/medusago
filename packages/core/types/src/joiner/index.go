// Package joiner mirrors @medusajs/types/src/joiner.
package joiner

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JoinerRelationship mirrors JoinerRelationship.
type JoinerRelationship struct {
	Alias         string         `json:"alias"`
	ForeignKey    string         `json:"foreignKey"`
	PrimaryKey    string         `json:"primaryKey"`
	ServiceName   string         `json:"serviceName"`
	Entity        string         `json:"entity,omitempty"`
	Inverse       bool           `json:"inverse,omitempty"`
	IsList        bool           `json:"isList,omitempty"`
	HasMany       bool           `json:"hasMany,omitempty"`
	DeleteCascade bool           `json:"deleteCascade,omitempty"`
	Args          map[string]any `json:"args,omitempty"`
}

// FieldAlias is either a plain path ("inventory_items.inventory") or an
// object with options.
type FieldAlias struct {
	Path                   string   `json:"path"`
	IsList                 bool     `json:"isList,omitempty"`
	ForwardArgumentsOnPath []string `json:"forwardArgumentsOnPath,omitempty"`
}

// UnmarshalJSON accepts both the string and the object form.
func (f *FieldAlias) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &f.Path)
	}
	type plain FieldAlias
	return strictUnmarshal(data, (*plain)(f))
}

// JoinerServiceConfigAlias mirrors the alias entries of a joiner config.
type JoinerServiceConfigAlias struct {
	Name     StringOrStrings `json:"name"`
	Entity   string          `json:"entity,omitempty"`
	Args     map[string]any  `json:"args,omitempty"`
	Internal json.RawMessage `json:"__internal,omitempty"`
}

// KeyValue is one entry of an OrderedStringMap.
type KeyValue struct {
	Key   string
	Value string
}

// OrderedStringMap is a JSON object of strings that keeps its key order.
type OrderedStringMap []KeyValue

// UnmarshalJSON decodes a JSON object preserving key order.
func (m *OrderedStringMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("expected an object")
	}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return err
		}
		var v string
		if err := dec.Decode(&v); err != nil {
			return err
		}
		*m = append(*m, KeyValue{Key: k.(string), Value: v})
	}
	_, err := dec.Token()
	return err
}

// StringOrStrings holds a value typed `string | string[]` upstream.
type StringOrStrings []string

// UnmarshalJSON accepts both a string and an array of strings.
func (s *StringOrStrings) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var one string
		if err := json.Unmarshal(data, &one); err != nil {
			return err
		}
		*s = StringOrStrings{one}
		return nil
	}
	return json.Unmarshal(data, (*[]string)(s))
}

// JoinerExtension mirrors an `extends` entry of a joiner config.
type JoinerExtension struct {
	ServiceName  string                `json:"serviceName"`
	Entity       string                `json:"entity,omitempty"`
	FieldAlias   map[string]FieldAlias `json:"fieldAlias,omitempty"`
	Relationship JoinerRelationship    `json:"relationship"`
}

// ExtraField is an extra column of a link table.
type ExtraField struct {
	Type         string         `json:"type"`
	DefaultValue *string        `json:"defaultValue,omitempty"`
	Nullable     bool           `json:"nullable,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
}

// NamedExtraField is an ExtraField with its column name.
type NamedExtraField struct {
	Name string
	ExtraField
}

// ExtraFields keeps the declaration order of the extra columns, which drives
// the column order of the link table.
type ExtraFields []NamedExtraField

// UnmarshalJSON decodes a JSON object preserving key order.
func (e *ExtraFields) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("extraFields: expected an object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		var field ExtraField
		if err := strictUnmarshal(raw, &field); err != nil {
			return err
		}
		*e = append(*e, NamedExtraField{Name: tok.(string), ExtraField: field})
	}
	_, err := dec.Token()
	return err
}

// DatabaseConfig is the databaseConfig of a link definition.
type DatabaseConfig struct {
	TableName   string      `json:"tableName,omitempty"`
	IDPrefix    string      `json:"idPrefix,omitempty"`
	ExtraFields ExtraFields `json:"extraFields,omitempty"`
}

// ModuleJoinerConfig mirrors ModuleJoinerConfig (the subset used by links).
type ModuleJoinerConfig struct {
	ServiceName          string                     `json:"serviceName,omitempty"`
	Schema               string                     `json:"schema,omitempty"`
	LinkableKeys         OrderedStringMap           `json:"linkableKeys,omitempty"`
	IDPrefixToEntityName map[string]string          `json:"idPrefixToEntityName,omitempty"`
	IsLink               bool                       `json:"isLink,omitempty"`
	IsReadOnlyLink       bool                       `json:"isReadOnlyLink,omitempty"`
	DatabaseConfig       *DatabaseConfig            `json:"databaseConfig,omitempty"`
	Alias                []JoinerServiceConfigAlias `json:"alias,omitempty"`
	PrimaryKeys          []string                   `json:"primaryKeys,omitempty"`
	Relationships        []JoinerRelationship       `json:"relationships,omitempty"`
	Extends              []JoinerExtension          `json:"extends,omitempty"`
}

// MustParseModuleJoinerConfig decodes a JSON joiner config, rejecting unknown
// keys so that nothing from an upstream definition is silently dropped.
func MustParseModuleJoinerConfig(data string) *ModuleJoinerConfig {
	var cfg ModuleJoinerConfig
	if err := strictUnmarshal([]byte(data), &cfg); err != nil {
		panic(fmt.Sprintf("invalid joiner config: %v", err))
	}
	return &cfg
}

func strictUnmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
