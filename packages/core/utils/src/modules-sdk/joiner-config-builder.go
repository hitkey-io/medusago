package modulessdk

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// DefineJoinerConfigOptions mirrors the options of defineJoinerConfig.
type DefineJoinerConfigOptions struct {
	Alias []joiner.JoinerServiceConfigAlias
	// IDPrefixToEntityName defaults to the prefixes of the DML models.
	IDPrefixToEntityName map[string]string
	// Schema defaults to the GraphQL schema generated from the DML models.
	Schema *string
	// Models are the DML models of the module.
	Models []*dml.Entity
	// Entities resolve the related entities of the models' relationships
	// (upstream resolves them through the relationship callbacks); defaults
	// to Models.
	Entities []*dml.Entity
	// ModelNames are the non-DML models ({ name } objects or MikroORM
	// entities) given with the DML models.
	ModelNames   []string
	LinkableKeys joiner.OrderedStringMap
	PrimaryKeys  []string
}

func orderedSet(m *joiner.OrderedStringMap, key, value string) {
	for i, kv := range *m {
		if kv.Key == key {
			(*m)[i].Value = value
			return
		}
	}
	*m = append(*m, joiner.KeyValue{Key: key, Value: value})
}

// DefineJoinerConfig mirrors defineJoinerConfig of joiner-config-builder.ts
// for explicitly given models.
func DefineJoinerConfig(serviceName string, opts DefineJoinerConfigOptions) *joiner.ModuleJoinerConfig {
	// models deduplicated by name (Map semantics: first position, last value)
	var names []string
	byName := map[string]*dml.Entity{}
	for _, m := range opts.Models {
		if _, ok := byName[m.Name]; !ok {
			names = append(names, m.Name)
		}
		byName[m.Name] = m
	}
	models := make([]*dml.Entity, len(names))
	for i, n := range names {
		models[i] = byName[n]
	}
	type namedModel struct {
		name string
	}
	all := make([]namedModel, 0, len(models)+len(opts.ModelNames))
	for _, m := range models {
		all = append(all, namedModel{m.Name})
	}
	seen := map[string]bool{}
	for _, n := range opts.ModelNames {
		if byName[n] != nil || seen[n] {
			continue
		}
		seen[n] = true
		all = append(all, namedModel{n})
	}

	schema := ""
	if opts.Schema != nil {
		schema = *opts.Schema
	} else {
		schema = dml.ToGraphQLSchema(models)
	}
	idPrefix := opts.IDPrefixToEntityName
	if idPrefix == nil {
		idPrefix = BuildIDPrefixToEntityNameFromDmlObjects(models)
	}

	linkableKeys := BuildLinkableKeysFromDmlObjects(models)
	for _, n := range opts.ModelNames {
		if byName[n] != nil {
			continue
		}
		orderedSet(&linkableKeys, strings.ToLower(common.CamelToSnakeCase(n))+"_id", n)
	}
	for _, kv := range opts.LinkableKeys {
		orderedSet(&linkableKeys, kv.Key, kv.Value)
	}

	var primaryKeys []string
	addPK := func(pk string) {
		for _, x := range primaryKeys {
			if x == pk {
				return
			}
		}
		primaryKeys = append(primaryKeys, pk)
	}
	for _, pk := range opts.PrimaryKeys {
		addPK(pk)
	}
	if len(models) > 0 {
		for _, lc := range BuildLinkConfigFromModelObjects(serviceName, models, nil) {
			for _, l := range lc.Linkables {
				addPK(l.PrimaryKey)
			}
		}
	}
	addPK("id")

	entities := opts.Entities
	if entities == nil {
		entities = models
	}
	internal := buildCrossModuleJoinMetadataFromDmlObjects(models, entities)

	var aliases []joiner.JoinerServiceConfigAlias
	for _, a := range opts.Alias {
		out := joiner.JoinerServiceConfigAlias{Name: a.Name, Entity: a.Entity}
		if meta, ok := internal[a.Entity]; ok {
			out.Internal = meta
		}
		suffix, _ := a.Args["methodSuffix"].(string)
		if suffix == "" {
			suffix = common.Pluralize(common.UpperCaseFirst(a.Entity))
		}
		out.Args = map[string]any{"methodSuffix": suffix}
		aliases = append(aliases, out)
	}
	for _, m := range all {
		entityName := common.UpperCaseFirst(m.name)
		overridden := false
		for _, a := range opts.Alias {
			if a.Entity == entityName {
				overridden = true
				break
			}
		}
		if overridden {
			continue
		}
		snake := strings.ToLower(common.CamelToSnakeCase(m.name))
		out := joiner.JoinerServiceConfigAlias{
			Name:   joiner.StringOrStrings{snake, common.Pluralize(snake)},
			Entity: entityName,
			Args:   map[string]any{"methodSuffix": common.Pluralize(entityName)},
		}
		if meta, ok := internal[entityName]; ok {
			out.Internal = meta
		}
		aliases = append(aliases, out)
	}

	return &joiner.ModuleJoinerConfig{
		ServiceName:          serviceName,
		PrimaryKeys:          primaryKeys,
		Schema:               schema,
		IDPrefixToEntityName: idPrefix,
		LinkableKeys:         linkableKeys,
		Alias:                aliases,
	}
}

// buildCrossModuleJoinMetadataFromDmlObjects mirrors the function of the
// same name: the __internal metadata of the aliases, as JSON in upstream key
// order.
func buildCrossModuleJoinMetadataFromDmlObjects(models, entities []*dml.Entity) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for _, model := range models {
		entityName := common.UpperCaseFirst(model.Name)
		_, _, pgSchema, tableName := dml.ParseEntityName(model)
		crossjoinable := []string{}
		add := func(s string) {
			for _, x := range crossjoinable {
				if x == s {
					return
				}
			}
			crossjoinable = append(crossjoinable, s)
		}
		var relations bytes.Buffer
		relCount := 0
		for i := range model.Properties {
			p := &model.Properties[i]
			if p.Relation != nil {
				rel := deriveInternalRelationMetadata(model, p, entities)
				if rel == nil {
					continue
				}
				if relCount > 0 {
					relations.WriteByte(',')
				}
				relCount++
				relations.Write(jsonString(p.Name))
				relations.WriteString(`:{"entity":`)
				relations.Write(jsonString(rel.entity))
				relations.WriteString(`,"foreignKey":`)
				relations.Write(jsonString(rel.foreignKey))
				relations.WriteString(`,"foreignKeyOwner":`)
				relations.Write(jsonString(rel.owner))
				if rel.isList {
					relations.WriteString(`,"isList":true`)
				}
				relations.WriteByte('}')
				if rel.owner == "self" {
					add(rel.foreignKey)
				}
				continue
			}
			if p.Field.Computed {
				continue
			}
			add(p.Name)
		}
		var buf bytes.Buffer
		buf.WriteString(`{"crossjoinable":`)
		cj, _ := json.Marshal(crossjoinable)
		buf.Write(cj)
		buf.WriteString(`,"tableName":`)
		buf.Write(jsonString(tableName))
		if pgSchema != "" {
			buf.WriteString(`,"schema":`)
			buf.Write(jsonString(pgSchema))
		}
		if relCount > 0 {
			buf.WriteString(`,"relations":{`)
			buf.Write(relations.Bytes())
			buf.WriteByte('}')
		}
		buf.WriteByte('}')
		out[entityName] = buf.Bytes()
	}
	return out
}

func jsonString(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

type internalRelation struct {
	entity     string
	foreignKey string
	owner      string
	isList     bool
}

func findModel(models []*dml.Entity, name string) *dml.Entity {
	for _, m := range models {
		if m.Name == name || common.UpperCaseFirst(common.ToCamelCase(m.Name)) == common.UpperCaseFirst(common.ToCamelCase(name)) {
			return m
		}
	}
	return nil
}

// deriveInternalRelationMetadata mirrors the function of the same name.
func deriveInternalRelationMetadata(model *dml.Entity, p *dml.Property, models []*dml.Entity) *internalRelation {
	referenced := findModel(models, p.Relation.Entity)
	if referenced == nil {
		return nil
	}
	related := common.UpperCaseFirst(referenced.Name)
	switch p.Relation.Type {
	case "belongsTo":
		fk := p.Relation.Options.ForeignKeyName
		if fk == "" {
			fk = common.CamelToSnakeCase(p.Name + "Id")
		}
		return &internalRelation{entity: related, foreignKey: fk, owner: "self"}
	case "hasMany":
		mappedBy := p.Relation.MappedBy
		if mappedBy == "" {
			mappedBy = common.CamelToSnakeCase(common.UpperCaseFirst(common.ToCamelCase(model.Name)))
		}
		inverse, ok := referenced.Property(mappedBy)
		if !ok || inverse.Relation == nil || inverse.Relation.Type != "belongsTo" {
			return nil
		}
		fk := inverse.Relation.Options.ForeignKeyName
		if fk == "" {
			fk = common.CamelToSnakeCase(mappedBy + "Id")
		}
		return &internalRelation{entity: related, foreignKey: fk, owner: "target", isList: true}
	}
	return nil
}

// BuildIDPrefixToEntityNameFromDmlObjects mirrors the function of the same
// name.
func BuildIDPrefixToEntityNameFromDmlObjects(models []*dml.Entity) map[string]string {
	out := map[string]string{}
	for _, m := range models {
		id, ok := m.Property("id")
		if !ok || id.Field == nil || id.Field.DataType.Options == nil {
			continue
		}
		if id.Field.DataType.Options.Prefix != "" && (id.Field.PrimaryKey || id.Field.DataType.Name == "id") {
			out[id.Field.DataType.Options.Prefix] = m.Name
		}
	}
	return out
}

// BuildLinkableKeysFromDmlObjects mirrors the function of the same name.
func BuildLinkableKeysFromDmlObjects(models []*dml.Entity) joiner.OrderedStringMap {
	out := joiner.OrderedStringMap{}
	for _, m := range models {
		for _, p := range m.Properties {
			if p.Field == nil || !p.Field.PrimaryKey {
				continue
			}
			orderedSet(&out, strings.ToLower(common.CamelToSnakeCase(m.Name))+"_"+p.Name, common.UpperCaseFirst(m.Name))
		}
	}
	return out
}

// Linkable mirrors an entry of a module's linkable config: what defineLink
// takes to link an entity.
type Linkable struct {
	Linkable    string `json:"linkable"`
	PrimaryKey  string `json:"primaryKey"`
	ServiceName string `json:"serviceName"`
	Field       string `json:"field"`
	Entity      string `json:"entity"`
	// Alias and Filterable are the optional alias and filterable fields of
	// an InputSource given to defineLink.
	Alias      string   `json:"alias,omitempty"`
	Filterable []string `json:"filterable,omitempty"`
}

// ModelLinkConfig is the linkable config of one model: its linkable keys by
// property, the first being what toJSON returns.
type ModelLinkConfig struct {
	Field     string
	Linkables []Linkable
}

// ToJSON mirrors the toJSON of a model link config: its first linkable.
func (c ModelLinkConfig) ToJSON() Linkable {
	if len(c.Linkables) == 0 {
		return Linkable{}
	}
	return c.Linkables[0]
}

// Key returns the linkable of the given property.
func (c ModelLinkConfig) Key(property string) (Linkable, bool) {
	for _, l := range c.Linkables {
		if l.PrimaryKey == property {
			return l, true
		}
	}
	return Linkable{}, false
}

// BuildLinkConfigFromModelObjects mirrors the function of the same name:
// the `linkable` export of a module, keyed by lowerCaseFirst(model name).
func BuildLinkConfigFromModelObjects(serviceName string, models []*dml.Entity, linkableKeys joiner.OrderedStringMap) []*ModelLinkConfig {
	var out []*ModelLinkConfig
	get := func(field string) *ModelLinkConfig {
		for _, c := range out {
			if c.Field == field {
				return c
			}
		}
		c := &ModelLinkConfig{Field: field}
		out = append(out, c)
		return c
	}
	var linkableModels []string
	for _, kv := range linkableKeys {
		linkableModels = append(linkableModels, kv.Value)
	}
	for _, m := range models {
		classLike := common.UpperCaseFirst(m.Name)
		if len(linkableModels) > 0 && !containsString(linkableModels, classLike) {
			continue
		}
		cfg := get(common.LowerCaseFirst(m.Name))
		for _, p := range m.Properties {
			if p.Field == nil || !(p.Field.PrimaryKey || p.Field.DataType.Name == "id") {
				continue
			}
			l := Linkable{
				Linkable:    strings.ToLower(common.CamelToSnakeCase(m.Name)) + "_" + p.Name,
				PrimaryKey:  p.Name,
				ServiceName: serviceName,
				Field:       common.LowerCaseFirst(m.Name),
				Entity:      classLike,
			}
			replaced := false
			for i := range cfg.Linkables {
				if cfg.Linkables[i].PrimaryKey == p.Name {
					cfg.Linkables[i] = l
					replaced = true
				}
			}
			if !replaced {
				cfg.Linkables = append(cfg.Linkables, l)
			}
		}
	}
	for _, kv := range linkableKeys {
		snake := common.CamelToSnakeCase(common.ToCamelCase(kv.Value))
		prop := strings.Replace(kv.Key, snake+"_", "", 1)
		cfg := get(common.LowerCaseFirst(kv.Value))
		if _, ok := cfg.Key(prop); ok {
			continue
		}
		cfg.Linkables = append(cfg.Linkables, Linkable{
			Linkable:    kv.Key,
			PrimaryKey:  prop,
			ServiceName: serviceName,
			Field:       common.LowerCaseFirst(kv.Value),
			Entity:      common.UpperCaseFirst(kv.Value),
		})
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
