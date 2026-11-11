package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// SchemaObjectRepresentationPropertiesToOmit mirrors
// schemaObjectRepresentationPropertiesToOmit (types/index.ts).
var SchemaObjectRepresentationPropertiesToOmit = []string{
	"_schemaPropertiesMap",
	"_serviceNameModuleConfigMap",
	"JSON",
	"DateTime",
	"Date",
	"Time",
}

func omitted(key string) bool {
	for _, k := range SchemaObjectRepresentationPropertiesToOmit {
		if k == key {
			return true
		}
	}
	return false
}

// Relationship is the part of a link joiner relationship the index uses.
type Relationship struct {
	ServiceName string `json:"serviceName"`
	ForeignKey  string `json:"foreignKey"`
}

// ModuleConfig is the part of the joiner config of an indexed entity's
// module the index uses (empty when the entity belongs to no module).
type ModuleConfig struct {
	ServiceName   string         `json:"serviceName"`
	IsLink        bool           `json:"isLink"`
	PrimaryKeys   []string       `json:"primaryKeys"`
	Relationships []Relationship `json:"relationships"`
}

// Parent mirrors SchemaObjectEntityRepresentation["parents"][0].
// IsListUnset records an undefined isList (buildAliasMap defaults it to
// true for shortcuts).
type Parent struct {
	Ref             *SchemaObjectEntityRepresentation
	InSchemaRef     *SchemaObjectEntityRepresentation
	TargetProp      string
	InverseSideProp string
	IsList          bool
	IsListUnset     bool
	IsInverse       bool
}

// SchemaObjectEntityRepresentation mirrors SchemaObjectEntityRepresentation.
type SchemaObjectEntityRepresentation struct {
	Entity       string
	Alias        string
	Fields       []string
	Listeners    []string
	ModuleConfig ModuleConfig
	Parents      []Parent
}

// SchemaPropertiesMapEntry mirrors SchemaPropertiesMap[0].
type SchemaPropertiesMapEntry struct {
	ShortCutOf string
	Ref        *SchemaObjectEntityRepresentation
	IsInverse  bool
	IsList     bool
}

// SchemaObjectRepresentation mirrors SchemaObjectRepresentation: the
// entities keyed by type name (in insertion order), _schemaPropertiesMap and
// the linkable keys of _serviceNameModuleConfigMap, plus the GraphQL field
// types of the executable schema (entitiesMap) the query builder casts with.
type SchemaObjectRepresentation struct {
	// Keys lists the entity type names in insertion order.
	Keys     []string
	Entities map[string]*SchemaObjectEntityRepresentation
	// PropertiesMap is _schemaPropertiesMap; PropertiesKeys its keys in
	// insertion order.
	PropertiesMap  map[string]*SchemaPropertiesMapEntry
	PropertiesKeys []string
	// LinkableKeys is _serviceNameModuleConfigMap[service].linkableKeys;
	// ServiceNames the services in insertion order.
	LinkableKeys map[string]map[string]string
	ServiceNames []string
	// Types maps a GraphQL type to its fields and their type (toString()).
	Types map[string]map[string]string
}

// AllFields returns the fields of every entity (allSchemaFields).
func (s *SchemaObjectRepresentation) AllFields() map[string]bool {
	out := map[string]bool{}
	for _, k := range s.Keys {
		for _, f := range s.Entities[k].Fields {
			out[f] = true
		}
	}
	return out
}

// CustomDirectives mirrors CustomDirectives.
var CustomDirectives = struct {
	Listeners struct {
		ConfigurationPropertyName, Name, Directive, Definition string
		IsRequired                                             bool
	}
}{
	Listeners: struct {
		ConfigurationPropertyName, Name, Directive, Definition string
		IsRequired                                             bool
	}{
		ConfigurationPropertyName: "listeners",
		IsRequired:                true,
		Name:                      "Listeners",
		Directive:                 "@Listeners",
		Definition:                "directive @Listeners (values: [String!]) on OBJECT",
	},
}

// executableSchema is the type map of makeExecutableSchema over a cleaned
// schema: the merged type definitions in order (the built-in scalars and
// introspection types have no definition and are left out).
type executableSchema struct {
	*graphql.Schema
}

// MakeSchemaExecutable mirrors makeSchemaExecutable: the schema cleaned
// (cleanGraphQLSchema) and its type definitions merged.
func MakeSchemaExecutable(inputSchema string) (*graphql.Schema, error) {
	doc, err := graphql.Parse(inputSchema)
	if err != nil {
		return nil, err
	}
	cleaned, _ := graphql.CleanGraphQLSchema(doc)
	if len(cleaned.Definitions) == 0 {
		return nil, nil
	}
	return graphql.MergeTypeDefs(cleaned), nil
}

func makeExecutable(inputSchema string) (*executableSchema, error) {
	s, err := MakeSchemaExecutable(inputSchema)
	if err != nil || s == nil {
		return nil, err
	}
	return &executableSchema{s}, nil
}

// typeOf returns the named type of the map (nil when absent).
func (e *executableSchema) typeOf(name string) *graphql.Type {
	if e == nil || e.Schema == nil {
		return nil
	}
	return e.Type(name)
}

// isObjectType reports whether the named type is an object type.
func (e *executableSchema) isObjectType(name string) bool {
	t := e.typeOf(name)
	return t != nil && t.Kind == graphql.KindObject
}

type entityProp struct {
	name    string
	isArray bool
}

// retrieveFieldNode mirrors retrieveEntityPropByType's retrieveFieldNode:
// the first field of the node whose (unwrapped) type is the target.
func retrieveFieldNode(node *graphql.Type, targetEntityName string) *entityProp {
	if node == nil {
		return nil
	}
	for _, field := range node.Fields {
		t := field.Type
		isArray := false
		for t.OfType != nil {
			if t.List {
				isArray = true
			}
			t = t.OfType
		}
		if t.Name == targetEntityName {
			return &entityProp{name: field.Name, isArray: isArray}
		}
	}
	return nil
}

// typeLookup is an entitiesMap / servicesEntityMap argument: a lookup by
// type name (nil when the map is not given).
type typeLookup func(name string) *graphql.Type

func lookupOf(e *executableSchema) typeLookup {
	return func(name string) *graphql.Type { return e.typeOf(name) }
}

// restrictedLookup mirrors the filtered entity maps of
// retrieveLinkModuleAndAlias: only the given names are present.
func restrictedLookup(e *executableSchema, names map[string]bool) typeLookup {
	return func(name string) *graphql.Type {
		if !names[name] {
			return nil
		}
		return e.typeOf(name)
	}
}

// retrieveEntityPropByType mirrors retrieveEntityPropByType (with the
// entity maps; the node form is retrieveFieldNode).
func retrieveEntityPropByType(sourceEntityName, targetEntityName string, entitiesMap, servicesEntityMap typeLookup) *entityProp {
	var prop *entityProp
	if entitiesMap != nil {
		prop = retrieveFieldNode(entitiesMap(sourceEntityName), targetEntityName)
	}
	if servicesEntityMap != nil && prop == nil {
		prop = retrieveFieldNode(servicesEntityMap(sourceEntityName), targetEntityName)
	}
	return prop
}

func propName(p *entityProp) string {
	if p == nil {
		return ""
	}
	return p.name
}

// isListOf returns prop?.isArray as (value, unset).
func isListOf(p *entityProp) (bool, bool) {
	if p == nil {
		return false, true
	}
	return p.isArray, false
}

func aliasNames(a joiner.JoinerServiceConfigAlias) []string {
	return []string(a.Name)
}

// extractNameFromAlias mirrors extractNameFromAlias.
func extractNameFromAlias(aliases []joiner.JoinerServiceConfigAlias) string {
	if len(aliases) == 0 {
		return ""
	}
	names := aliasNames(aliases[0])
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// retrieveAliasForEntity mirrors retrieveAliasForEntity.
func retrieveAliasForEntity(entityName string, aliases []joiner.JoinerServiceConfigAlias) string {
	for _, alias := range aliases {
		names := aliasNames(alias)
		if alias.Entity == entityName {
			if len(names) == 0 {
				return ""
			}
			return names[0]
		}
		for _, name := range names {
			if strings.EqualFold(name, entityName) {
				return name
			}
		}
	}
	return ""
}

func linkableKey(cfg *joiner.ModuleJoinerConfig, key string) string {
	if cfg == nil {
		return ""
	}
	for _, kv := range cfg.LinkableKeys {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// representationBuilder holds the state of buildSchemaObjectRepresentation.
type representationBuilder struct {
	moduleJoinerConfigs []*joiner.ModuleJoinerConfig
	entitiesMap         *executableSchema
	servicesEntityMap   *executableSchema
	logger              *slog.Logger

	keys     []string
	entities map[string]*SchemaObjectEntityRepresentation
	// set marks the entities whose moduleConfig is set
	moduleConfigs map[*SchemaObjectEntityRepresentation]*joiner.ModuleJoinerConfig
	serviceNames  []string
	services      map[string]*joiner.ModuleJoinerConfig

	moduleSchemas map[*joiner.ModuleJoinerConfig]*executableSchema
}

func (b *representationBuilder) moduleSchema(cfg *joiner.ModuleJoinerConfig) (*executableSchema, error) {
	if s, ok := b.moduleSchemas[cfg]; ok {
		return s, nil
	}
	s, err := makeExecutable(cfg.Schema)
	if err != nil {
		return nil, err
	}
	b.moduleSchemas[cfg] = s
	return s, nil
}

// getObjectRepresentationRef mirrors getObjectRepresentationRef.
func (b *representationBuilder) getObjectRepresentationRef(entityName string) *SchemaObjectEntityRepresentation {
	if r, ok := b.entities[entityName]; ok {
		return r
	}
	r := &SchemaObjectEntityRepresentation{Entity: entityName, Parents: []Parent{}, Listeners: []string{}, Fields: []string{}}
	b.entities[entityName] = r
	b.keys = append(b.keys, entityName)
	return r
}

func (b *representationBuilder) setService(name string, cfg *joiner.ModuleJoinerConfig) {
	if _, ok := b.services[name]; !ok {
		b.serviceNames = append(b.serviceNames, name)
	}
	b.services[name] = cfg
}

func (b *representationBuilder) setModuleConfig(rep *SchemaObjectEntityRepresentation, cfg *joiner.ModuleJoinerConfig) {
	b.moduleConfigs[rep] = cfg
	rep.ModuleConfig = moduleConfigOf(cfg)
}

func moduleConfigOf(cfg *joiner.ModuleJoinerConfig) ModuleConfig {
	out := ModuleConfig{ServiceName: cfg.ServiceName, IsLink: cfg.IsLink, PrimaryKeys: cfg.PrimaryKeys, Relationships: []Relationship{}}
	if cfg.IsLink {
		for _, r := range cfg.Relationships {
			out.Relationships = append(out.Relationships, Relationship{ServiceName: r.ServiceName, ForeignKey: r.ForeignKey})
		}
	}
	return out
}

// retrieveModuleAndAlias mirrors retrieveModuleAndAlias.
func (b *representationBuilder) retrieveModuleAndAlias(entityName string) (*joiner.ModuleJoinerConfig, string, error) {
	var relatedModule *joiner.ModuleJoinerConfig
	alias := ""
	for _, cfg := range b.moduleJoinerConfigs {
		if cfg.Schema != "" {
			s, err := b.moduleSchema(cfg)
			if err != nil {
				return nil, "", err
			}
			if s.typeOf(entityName) != nil {
				relatedModule = cfg
			}
		}
		if relatedModule != nil && cfg.Alias != nil {
			alias = retrieveAliasForEntity(entityName, cfg.Alias)
		}
		if relatedModule != nil {
			break
		}
	}
	if relatedModule == nil {
		return nil, "", nil
	}
	if alias == "" {
		return nil, "", fmt.Errorf("Index Module error, the module %s has a schema but does not have any alias for the entity %s. Please add an alias to the module configuration and the entity it correspond to in the args under the entity property.", relatedModule.ServiceName, entityName)
	}
	return relatedModule, alias, nil
}

type linkModuleMetadata struct {
	entityName              string
	alias                   string
	linkModuleConfig        *joiner.ModuleJoinerConfig
	intermediateEntityNames []string
	isInverse               bool
	isList                  bool
	isListUnset             bool
	inverseSideProp         string
}

// retrieveLinkModuleAndAlias mirrors retrieveLinkModuleAndAlias.
func (b *representationBuilder) retrieveLinkModuleAndAlias(primaryEntity string, primaryModuleConfig *joiner.ModuleJoinerConfig, foreignEntity string, foreignModuleConfig *joiner.ModuleJoinerConfig) ([]linkModuleMetadata, error) {
	var out []linkModuleMetadata
	for _, link := range b.moduleJoinerConfigs {
		if !link.IsLink || link.IsReadOnlyLink {
			continue
		}
		if len(link.Relationships) < 2 {
			return nil, fmt.Errorf("Cannot read properties of undefined (reading 'serviceName')")
		}
		linkPrimary := link.Relationships[0]
		linkForeign := link.Relationships[1]

		isDirectMatch := linkPrimary.ServiceName == primaryModuleConfig.ServiceName &&
			linkForeign.ServiceName == foreignModuleConfig.ServiceName &&
			linkPrimary.Entity == primaryEntity
		isInverseMatch := linkPrimary.ServiceName == foreignModuleConfig.ServiceName &&
			linkForeign.ServiceName == primaryModuleConfig.ServiceName &&
			linkPrimary.Entity == foreignEntity
		if !(isDirectMatch || isInverseMatch) {
			continue
		}

		primaryEntityLinkableKey := linkForeign.ForeignKey
		foreignEntityLinkableKey := linkPrimary.ForeignKey
		if isDirectMatch {
			primaryEntityLinkableKey = linkPrimary.ForeignKey
			foreignEntityLinkableKey = linkForeign.ForeignKey
		}
		isTheForeignKeyEntityEqualPrimaryEntity := linkableKey(primaryModuleConfig, primaryEntityLinkableKey) == primaryEntity
		isTheForeignKeyEntityEqualForeignEntity := linkableKey(foreignModuleConfig, foreignEntityLinkableKey) == foreignEntity

		if len(link.Alias) == 0 {
			return nil, fmt.Errorf("Cannot read properties of undefined (reading 'entity')")
		}
		aliasEntity := link.Alias[0].Entity
		filtered := map[string]bool{aliasEntity: true}
		for _, r := range link.Relationships {
			if r.Entity != "" {
				filtered[r.Entity] = true
			}
		}
		filteredEntitiesMap := restrictedLookup(b.entitiesMap, filtered)
		filteredServicesEntityMap := restrictedLookup(b.servicesEntityMap, filtered)

		linkName := ""
		for _, extend := range link.Extends {
			if (extend.ServiceName == primaryModuleConfig.ServiceName || extend.Relationship.ServiceName == foreignModuleConfig.ServiceName) &&
				(extend.Relationship.PrimaryKey == primaryEntityLinkableKey || extend.Relationship.PrimaryKey == foreignEntityLinkableKey) {
				linkName = extend.Relationship.ServiceName
				break
			}
		}
		if linkName == "" {
			return nil, fmt.Errorf("Index Module error, unable to retrieve the link module name for the services %s - %s. Please be sure that the extend relationship service name is set correctly", primaryModuleConfig.ServiceName, foreignModuleConfig.ServiceName)
		}
		if aliasEntity == "" {
			return nil, fmt.Errorf("Index Module error, unable to retrieve the link module entity name for the services %s - %s. Please be sure that the link module alias has an entity property in the args.", primaryModuleConfig.ServiceName, foreignModuleConfig.ServiceName)
		}

		if isTheForeignKeyEntityEqualPrimaryEntity && isTheForeignKeyEntityEqualForeignEntity {
			// The link becomes the parent of the foreign entity.
			inverseSideProp := retrieveEntityPropByType(aliasEntity, primaryEntity, filteredEntitiesMap, filteredServicesEntityMap)
			isList, unset := isListOf(inverseSideProp)
			out = append(out, linkModuleMetadata{
				entityName:              aliasEntity,
				alias:                   extractNameFromAlias(link.Alias),
				linkModuleConfig:        link,
				intermediateEntityNames: []string{},
				isInverse:               isInverseMatch,
				isList:                  isList,
				isListUnset:             unset,
				inverseSideProp:         propName(inverseSideProp),
			})
			continue
		}

		intermediateEntityName := linkableKey(foreignModuleConfig, foreignEntityLinkableKey)
		moduleSchema := primaryModuleConfig.Schema
		if isDirectMatch {
			moduleSchema = foreignModuleConfig.Schema
		}
		if moduleSchema == "" {
			return nil, fmt.Errorf("Index Module error, unable to retrieve the intermediate entity name for the services %s - %s. Please be sure that the foreign module %s has a schema.", primaryModuleConfig.ServiceName, foreignModuleConfig.ServiceName, foreignModuleConfig.ServiceName)
		}

		entitiesMap := b.servicesEntityMap
		var intermediateEntities []string
		foundCount := 0
		foundName := ""
		var isForeignEntityChildOfIntermediateEntity func(entityName string, visited map[string]bool) bool
		isForeignEntityChildOfIntermediateEntity = func(entityName string, visited map[string]bool) bool {
			if visited[entityName] {
				return false
			}
			visited[entityName] = true
			for _, entityType := range entitiesMap.Types {
				inverseSideProp := retrieveFieldNode(entityType, entityName)
				if entityType.Kind == graphql.KindObject && inverseSideProp != nil {
					if entityType.Name == intermediateEntityName {
						foundName = entityType.Name
						foundCount++
						return true
					}
					if isForeignEntityChildOfIntermediateEntity(entityType.Name, visited) {
						intermediateEntities = append(intermediateEntities, entityType.Name)
						return true
					}
				}
			}
			return false
		}
		start := primaryEntity
		if isDirectMatch {
			start = foreignEntity
		}
		isForeignEntityChildOfIntermediateEntity(start, map[string]bool{})
		if foundCount != 1 {
			return nil, fmt.Errorf("Index Module error, unable to retrieve the intermediate entities for the services %s - %s between %s and %s. Multiple paths or no path found. Please check your schema in %s", primaryModuleConfig.ServiceName, foreignModuleConfig.ServiceName, start, intermediateEntityName, foreignModuleConfig.ServiceName)
		}
		intermediateEntities = append(intermediateEntities, foundName)

		target, source := aliasEntity, primaryEntity
		if isDirectMatch {
			target, source = primaryEntity, aliasEntity
		}
		directInverseSideProp := retrieveEntityPropByType(source, target, filteredEntitiesMap, filteredServicesEntityMap)
		isList, unset := isListOf(directInverseSideProp)
		out = append(out, linkModuleMetadata{
			entityName:              aliasEntity,
			alias:                   extractNameFromAlias(link.Alias),
			linkModuleConfig:        link,
			intermediateEntityNames: intermediateEntities,
			inverseSideProp:         propName(directInverseSideProp),
			isList:                  isList,
			isListUnset:             unset,
			isInverse:               isInverseMatch,
		})
	}
	if len(out) == 0 {
		b.logger.Warn(fmt.Sprintf("Index Module warning, unable to retrieve the link module that correspond to the entities %s - %s.", primaryEntity, foreignEntity))
	}
	return out, nil
}

// setCustomDirectives mirrors setCustomDirectives.
func setCustomDirectives(rep *SchemaObjectEntityRepresentation, directives []*graphql.Directive) error {
	var directive *graphql.Directive
	for _, d := range directives {
		if d.Name == CustomDirectives.Listeners.Name {
			directive = d
			break
		}
	}
	if directive == nil {
		return nil
	}
	if len(directive.Arguments) == 0 {
		return fmt.Errorf("Cannot read properties of undefined (reading 'value')")
	}
	values := []string{}
	if v := directive.Arguments[0].Value; v != nil && v.Kind == graphql.ValueList {
		for _, item := range v.Values {
			values = append(values, item.Value)
		}
	}
	rep.Listeners = values
	return nil
}

// gqlGetFieldsAndRelations mirrors gqlGetFieldsAndRelations without
// relations: the fields of an object type whose type is not an object type.
func gqlGetFieldsAndRelations(schemaTypeMap *executableSchema, typeName string) []string {
	t := schemaTypeMap.typeOf(typeName)
	if t == nil || t.Kind != graphql.KindObject {
		return nil
	}
	var out []string
	for _, f := range t.Fields {
		if !schemaTypeMap.isObjectType(f.Type.NamedType()) {
			out = append(out, f.Name)
		}
	}
	return out
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// processEntity mirrors processEntity.
func (b *representationBuilder) processEntity(entityName string) error {
	current := b.getObjectRepresentationRef(entityName)

	if t := b.entitiesMap.typeOf(entityName); t != nil {
		if err := setCustomDirectives(current, t.Directives); err != nil {
			return err
		}
	}

	current.Fields = dedupe(append(current.Fields, gqlGetFieldsAndRelations(b.entitiesMap, entityName)...))

	currentEntityModule, alias, err := b.retrieveModuleAndAlias(entityName)
	if err != nil {
		return err
	}
	if currentEntityModule == nil && len(current.Listeners) > 0 {
		// JSON.stringify({ alias: [{ name: "entity-alias", entity }] })
		example := fmt.Sprintf(`{"alias":[{"name":"entity-alias","entity":%s}]}`, mustJSON(entityName))
		return fmt.Errorf("Index Module error, unable to retrieve the module that corresponds to the entity %s.\nPlease add the entity to the module schema or add an alias to the joiner config like the example below:\n%s", entityName, example)
	}
	if currentEntityModule != nil {
		b.setService(currentEntityModule.ServiceName, currentEntityModule)
		b.setModuleConfig(current, currentEntityModule)
		current.Alias = alias
	}

	// The parent entities: the types with a field of the entity's type.
	var parentEntityNames []string
	for _, value := range b.entitiesMap.Types {
		for _, field := range value.Fields {
			if field.Type.NamedType() == entityName {
				parentEntityNames = append(parentEntityNames, value.Name)
				break
			}
		}
	}
	if len(parentEntityNames) == 0 {
		return nil
	}

	entitiesMap := lookupOf(b.entitiesMap)
	servicesEntityMap := lookupOf(b.servicesEntityMap)

	for _, parent := range parentEntityNames {
		entityFieldInParent := retrieveEntityPropByType(parent, entityName, entitiesMap, servicesEntityMap)
		entityTargetPropertyNameInParent := propName(entityFieldInParent)
		entityTargetPropertyIsListInParent, entityTargetPropertyIsListUnset := isListOf(entityFieldInParent)

		if _, ok := b.entities[parent]; !ok {
			if err := b.processEntity(parent); err != nil {
				return err
			}
		}

		parentRef := b.getObjectRepresentationRef(parent)
		parentModuleConfig := b.moduleConfigs[parentRef]
		currentModuleConfig := b.moduleConfigs[current]

		// The entity is not part of any module: just set the parent.
		if currentModuleConfig == nil {
			parentAlreadyExists := false
			for _, p := range current.Parents {
				if p.Ref.Entity == parentRef.Entity && p.TargetProp == entityTargetPropertyNameInParent {
					parentAlreadyExists = true
				}
			}
			parentPropertyNameWithinCurrentEntity := retrieveEntityPropByType(entityName, parent, entitiesMap, servicesEntityMap)
			if parentAlreadyExists {
				return nil
			}
			current.Parents = append(current.Parents, Parent{
				Ref:             parentRef,
				TargetProp:      entityTargetPropertyNameInParent,
				InverseSideProp: propName(parentPropertyNameWithinCurrentEntity),
				IsList:          entityTargetPropertyIsListInParent,
				IsListUnset:     entityTargetPropertyIsListUnset,
			})
			continue
		}

		if parentModuleConfig == nil {
			return fmt.Errorf("Cannot read properties of null (reading 'serviceName')")
		}

		// Same service (or a link): configure the parent and add the parent
		// id as a field of the entity.
		if currentModuleConfig.ServiceName == parentModuleConfig.ServiceName || parentModuleConfig.IsLink || currentModuleConfig.IsLink {
			parentPropertyNameWithinCurrentEntity := retrieveEntityPropByType(entityName, parent, entitiesMap, servicesEntityMap)
			parentAlreadyExists := false
			for _, p := range current.Parents {
				if p.Ref.Entity == parentRef.Entity && p.TargetProp == entityTargetPropertyNameInParent {
					parentAlreadyExists = true
				}
			}
			if !parentAlreadyExists {
				current.Parents = append(current.Parents, Parent{
					Ref:             parentRef,
					TargetProp:      entityTargetPropertyNameInParent,
					InverseSideProp: propName(parentPropertyNameWithinCurrentEntity),
					IsList:          entityTargetPropertyIsListInParent,
					IsListUnset:     entityTargetPropertyIsListUnset,
				})
			}
			if parentPropertyNameWithinCurrentEntity != nil {
				propertyToAdd := parentPropertyNameWithinCurrentEntity.name + ".id"
				if !containsStr(current.Fields, propertyToAdd) {
					current.Fields = append(current.Fields, propertyToAdd)
				}
			}
			if parentAlreadyExists {
				return nil
			}
			continue
		}

		// Different services: the link module joining them.
		metadatas, err := b.retrieveLinkModuleAndAlias(parentRef.Entity, parentModuleConfig, current.Entity, currentEntityModule)
		if err != nil {
			return err
		}
		for _, meta := range metadatas {
			linkRef := b.getObjectRepresentationRef(meta.entityName)
			serviceKey := meta.linkModuleConfig.ServiceName
			if serviceKey == "" {
				serviceKey = meta.entityName
			}
			b.setService(serviceKey, meta.linkModuleConfig)

			hasParent := false
			for _, p := range linkRef.Parents {
				if p.Ref.Entity == parentRef.Entity {
					hasParent = true
				}
			}
			if !hasParent {
				linkRef.Parents = append(linkRef.Parents, Parent{
					Ref:             parentRef,
					TargetProp:      meta.alias,
					InverseSideProp: meta.inverseSideProp,
					IsList:          meta.isList,
					IsListUnset:     meta.isListUnset,
					IsInverse:       meta.isInverse,
				})
			}
			linkRef.Alias = meta.alias
			linkRef.Listeners = []string{meta.entityName + ".attached", meta.entityName + ".detached"}
			b.setModuleConfig(linkRef, meta.linkModuleConfig)

			fields := []string{"id"}
			for _, r := range meta.linkModuleConfig.Relationships {
				if (r.ServiceName == parentModuleConfig.ServiceName || r.ServiceName == currentEntityModule.ServiceName) && r.ForeignKey != "" {
					fields = append(fields, r.ForeignKey)
				}
			}
			linkRef.Fields = fields

			// The intermediate entities between the link and the entity,
			// from the one next to the link.
			names := meta.intermediateEntityNames
			for i := len(names) - 1; i >= 0; i-- {
				intermediateEntityName := names[i]
				isLastIntermediateEntity := i == len(names)-1
				parentIntermediateRef := linkRef
				if !isLastIntermediateEntity {
					parentIntermediateRef = b.entities[names[i+1]]
				}
				intermediateEntityModule, intermediateEntityAlias, err := b.retrieveModuleAndAlias(intermediateEntityName)
				if err != nil {
					return err
				}
				if intermediateEntityModule == nil {
					return fmt.Errorf("Cannot read properties of null (reading 'serviceName')")
				}
				interRef := b.getObjectRepresentationRef(intermediateEntityName)
				b.setService(intermediateEntityModule.ServiceName, intermediateEntityModule)

				parentPropertyNameWithinIntermediateEntity := retrieveEntityPropByType(intermediateEntityName, parentIntermediateRef.Entity, entitiesMap, servicesEntityMap)
				interIsList, interIsListUnset := isListOf(retrieveEntityPropByType(parentIntermediateRef.Entity, intermediateEntityName, entitiesMap, servicesEntityMap))

				exists := false
				for _, p := range interRef.Parents {
					if p.Ref.Entity == parentIntermediateRef.Entity && p.TargetProp == intermediateEntityAlias {
						exists = true
					}
				}
				if !exists {
					interRef.Parents = append(interRef.Parents, Parent{
						Ref:             parentIntermediateRef,
						TargetProp:      intermediateEntityAlias,
						InverseSideProp: propName(parentPropertyNameWithinIntermediateEntity),
						IsList:          interIsList,
						IsListUnset:     interIsListUnset,
					})
				}
				interRef.Alias = intermediateEntityAlias
				prefix := common.LowerCaseFirst(common.KebabCase(intermediateEntityModule.ServiceName))
				interRef.Listeners = []string{
					eventbus.BuildModuleResourceEventName(prefix, intermediateEntityName, "created"),
					eventbus.BuildModuleResourceEventName(prefix, intermediateEntityName, "updated"),
					eventbus.BuildModuleResourceEventName(prefix, intermediateEntityName, "deleted"),
					eventbus.BuildModuleResourceEventName(prefix, intermediateEntityName, "restored"),
				}
				b.setModuleConfig(interRef, intermediateEntityModule)
				if !containsStr(interRef.Fields, "id") {
					interRef.Fields = append(interRef.Fields, "id")
				}
				// the parent id only between intermediate entities
				if !isLastIntermediateEntity && parentPropertyNameWithinIntermediateEntity != nil {
					propertyToAdd := parentPropertyNameWithinIntermediateEntity.name + ".id"
					if !containsStr(interRef.Fields, propertyToAdd) {
						interRef.Fields = append(interRef.Fields, propertyToAdd)
					}
				}
			}

			// The last intermediate entity (or the link) is the parent of
			// the entity.
			currentParentIntermediateRef := linkRef
			if len(names) > 0 {
				currentParentIntermediateRef = b.entities[names[0]]
				p := retrieveEntityPropByType(current.Entity, currentParentIntermediateRef.Entity, servicesEntityMap, nil)
				if p != nil {
					propertyToAdd := p.name + ".id"
					if !containsStr(current.Fields, propertyToAdd) {
						current.Fields = append(current.Fields, propertyToAdd)
					}
				}
			}
			parentPropertyNameWithinCurrentEntity := retrieveEntityPropByType(current.Entity, currentParentIntermediateRef.Entity, servicesEntityMap, nil)
			isList, isListUnset := isListOf(retrieveEntityPropByType(currentParentIntermediateRef.Entity, current.Entity, entitiesMap, servicesEntityMap))

			found := -1
			for i, p := range current.Parents {
				if p.Ref.Entity == currentParentIntermediateRef.Entity && p.TargetProp == entityTargetPropertyNameInParent {
					found = i
					break
				}
			}
			if found < 0 {
				current.Parents = append(current.Parents, Parent{
					Ref:             currentParentIntermediateRef,
					InSchemaRef:     parentRef,
					TargetProp:      entityTargetPropertyNameInParent,
					InverseSideProp: propName(parentPropertyNameWithinCurrentEntity),
					IsList:          isList,
					IsListUnset:     isListUnset,
				})
			} else if current.Parents[found].InSchemaRef == nil {
				current.Parents[found].InSchemaRef = parentRef
			}
		}
	}
	return nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// getServicesEntityMap mirrors getServicesEntityMap.
func getServicesEntityMap(moduleJoinerConfigs []*joiner.ModuleJoinerConfig, additionalSchema string) (*executableSchema, error) {
	schemas := make([]string, len(moduleJoinerConfigs))
	for i, c := range moduleJoinerConfigs {
		schemas[i] = c.Schema
	}
	return makeExecutable(BaseGraphqlSchema + "\n" + strings.Join(schemas, "\n") + "\n" + additionalSchema)
}

type aliasEntry struct {
	alias      string
	shortCutOf string
	isInverse  bool
	isList     bool
}

// buildAliasMap mirrors buildAliasMap: the representation of every path.
func (b *representationBuilder) buildAliasMap() ([]string, map[string]*SchemaPropertiesMapEntry) {
	var keys []string
	aliasMap := map[string]*SchemaPropertiesMapEntry{}

	var recursivelyBuildAliasPath func(current *SchemaObjectEntityRepresentation, parentPath string, visited map[string]bool, pathStack []string) []aliasEntry
	recursivelyBuildAliasPath = func(current *SchemaObjectEntityRepresentation, parentPath string, visited map[string]bool, pathStack []string) []aliasEntry {
		var aliases []aliasEntry
		pathIdentifier := current.Entity + ":" + parentPath
		if containsStr(pathStack, pathIdentifier) {
			return nil
		}
		pathStack = append(pathStack, pathIdentifier)
		if visited[current.Entity] {
			return nil
		}
		visited[current.Entity] = true

		for _, parentEntity := range current.Parents {
			newParentPath := parentEntity.TargetProp
			if parentPath != "" {
				newParentPath = parentEntity.TargetProp + "." + parentPath
			}
			newVisited := copySet(visited)
			newPathStack := append([]string(nil), pathStack...)

			var parentAliases []aliasEntry
			for _, a := range recursivelyBuildAliasPath(parentEntity.Ref, newParentPath, newVisited, newPathStack) {
				parentAliases = append(parentAliases, aliasEntry{alias: a.alias, isInverse: parentEntity.IsInverse, isList: parentEntity.IsList})
			}
			aliases = append(aliases, parentAliases...)

			// shortcut paths via inSchemaRef
			if parentEntity.InSchemaRef != nil {
				if len(parentAliases) == 0 {
					continue
				}
				shortcutAliases_ := recursivelyBuildAliasPath(parentEntity.InSchemaRef, newParentPath, copySet(visited), append([]string(nil), pathStack...))
				var shortcutAliases []aliasEntry
				for _, shortcutAlias := range shortcutAliases_ {
					for _, aliasObj := range parentAliases {
						if strings.Split(aliasObj.alias, ".")[0] != strings.Split(shortcutAlias.alias, ".")[0] {
							continue
						}
						isList := parentEntity.IsList
						if parentEntity.IsListUnset {
							isList = true
						}
						shortcutAliases = append(shortcutAliases, aliasEntry{alias: shortcutAlias.alias, shortCutOf: aliasObj.alias, isList: isList, isInverse: shortcutAlias.isInverse})
					}
				}
				for _, shortcut := range shortcutAliases {
					dup := false
					for _, a := range aliases {
						if a.alias == shortcut.alias {
							dup = true
							break
						}
					}
					if !dup {
						aliases = append(aliases, shortcut)
					}
				}
			}
		}

		// the entity's own alias
		var pathSegments []string
		if parentPath != "" {
			pathSegments = strings.Split(parentPath, ".")
		}
		baseAlias := current.Alias
		if parentPath != "" {
			baseAlias = current.Alias + "." + parentPath
		}
		if len(pathSegments) > 0 && pathSegments[0] == current.Alias {
			baseAlias = parentPath
		}
		dup := false
		for _, a := range aliases {
			if a.alias == baseAlias {
				dup = true
				break
			}
		}
		if !dup {
			aliases = append(aliases, aliasEntry{alias: baseAlias})
		}
		delete(visited, current.Entity)
		return aliases
	}

	for _, key := range b.keys {
		if omitted(key) {
			continue
		}
		ref := b.entities[key]
		for _, alias := range recursivelyBuildAliasPath(ref, "", map[string]bool{}, nil) {
			if _, ok := aliasMap[alias.alias]; ok && alias.shortCutOf == "" {
				continue
			}
			if _, ok := aliasMap[alias.alias]; !ok {
				keys = append(keys, alias.alias)
			}
			aliasMap[alias.alias] = &SchemaPropertiesMapEntry{Ref: ref, ShortCutOf: alias.shortCutOf, IsList: alias.isList, IsInverse: alias.isInverse}
		}
	}
	return keys, aliasMap
}

func copySet(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

type filterableEntity struct {
	serviceName string
	entity      string
	fields      []string
	schema      string
}

// buildSchemaFromFilterableLinks mirrors buildSchemaFromFilterableLinks:
// the types of the fields links (and aliases) declare filterable. The field
// aliases of an extension are taken in key order (Go maps keep none).
func buildSchemaFromFilterableLinks(moduleJoinerConfigs []*joiner.ModuleJoinerConfig, servicesEntityMap *executableSchema) string {
	var allFilterable []filterableEntity
	for _, config := range moduleJoinerConfigs {
		if !config.IsLink {
			continue
		}
		any := false
		for _, r := range config.Relationships {
			if len(r.Filterable) > 0 {
				any = true
			}
		}
		if !any {
			continue
		}
		for _, relationship := range config.Relationships {
			filterable := append([]string(nil), relationship.Filterable...)
			if len(filterable) == 0 || !containsStr(filterable, "id") {
				filterable = append(filterable, "id")
			}
			fieldAliasMap := map[string][]string{}
			for _, extend := range config.Extends {
				var keys []string
				for k := range extend.FieldAlias {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				fieldAliasMap[extend.ServiceName] = append(keys, extend.Relationship.Alias)
			}
			allFilterable = append(allFilterable, filterableEntity{
				serviceName: relationship.ServiceName,
				entity:      relationship.Entity,
				fields:      dedupe(append(filterable, fieldAliasMap[relationship.ServiceName]...)),
				schema:      config.Schema,
			})
		}
	}

	getGqlType := func(entity, field string) string {
		t := servicesEntityMap.typeOf(entity)
		if t == nil {
			return ""
		}
		f := t.Field(field)
		if f == nil {
			return ""
		}
		fieldType := f.Type.String()
		isArray := strings.HasPrefix(fieldType, "[")
		currentType := strings.NewReplacer("[", "", "]", "", "!", "").Replace(fieldType)
		// type.ofType is an enum: a wrapped enum type
		if f.Type.OfType != nil && f.Type.OfType.OfType == nil {
			if ot := servicesEntityMap.typeOf(f.Type.OfType.Name); ot != nil && ot.Kind == graphql.KindEnum {
				currentType = "String"
			}
		}
		if isArray {
			return "[" + currentType + "]"
		}
		return currentType
	}

	parts := make([]string, len(allFilterable))
	for i, e := range allFilterable {
		if e.schema == "" {
			continue
		}
		normalizedEntity := common.LowerCaseFirst(common.KebabCase(e.entity))
		events := fmt.Sprintf(`@Listeners(values: ["%s.%s.created", "%s.%s.updated", "%s.%s.deleted", "%s.%s.restored"])`,
			e.serviceName, normalizedEntity, e.serviceName, normalizedEntity, e.serviceName, normalizedEntity, e.serviceName, normalizedEntity)
		defs := make([]string, len(e.fields))
		for j, field := range e.fields {
			typ := getGqlType(e.entity, field)
			if typ == "" {
				typ = "String"
			}
			defs[j] = "    " + field + ": " + typ
		}
		parts[i] = "\n      type " + e.entity + " " + events + " {\n        id: ID!\n      }\n        \n      extend type " + e.entity + " {\n" + strings.Join(defs, "\n") + "\n}"
	}
	return strings.Join(parts, "\n\n")
}

// BuildSchemaObjectRepresentation mirrors buildSchemaObjectRepresentation:
// the internal representation of the index schema (baseGraphqlSchema +
// the module option `schema`, or defaultSchema) over the joiner configs of
// the loaded modules and links (MedusaModule.getAllJoinerConfigs()).
func BuildSchemaObjectRepresentation(schema string, moduleJoinerConfigs []*joiner.ModuleJoinerConfig, logger *slog.Logger) (*SchemaObjectRepresentation, error) {
	if logger == nil {
		logger = slog.Default()
	}
	servicesEntityMap, err := getServicesEntityMap(moduleJoinerConfigs, "")
	if err != nil {
		return nil, err
	}
	filterableEntities := buildSchemaFromFilterableLinks(moduleJoinerConfigs, servicesEntityMap)
	augmentedSchema := CustomDirectives.Listeners.Definition + "\n" + schema + "\n" + filterableEntities
	entitiesMap, err := makeExecutable(augmentedSchema)
	if err != nil {
		return nil, err
	}
	if entitiesMap == nil {
		return nil, fmt.Errorf("Cannot read properties of undefined (reading 'getTypeMap')")
	}

	b := &representationBuilder{
		moduleJoinerConfigs: moduleJoinerConfigs,
		entitiesMap:         entitiesMap,
		servicesEntityMap:   servicesEntityMap,
		logger:              logger,
		entities:            map[string]*SchemaObjectEntityRepresentation{},
		moduleConfigs:       map[*SchemaObjectEntityRepresentation]*joiner.ModuleJoinerConfig{},
		services:            map[string]*joiner.ModuleJoinerConfig{},
		moduleSchemas:       map[*joiner.ModuleJoinerConfig]*executableSchema{},
	}
	for _, t := range entitiesMap.Types {
		if t.Kind == graphql.KindScalar {
			continue
		}
		if err := b.processEntity(t.Name); err != nil {
			return nil, err
		}
	}

	out := &SchemaObjectRepresentation{
		Keys:         b.keys,
		Entities:     b.entities,
		LinkableKeys: map[string]map[string]string{},
		ServiceNames: b.serviceNames,
		Types:        map[string]map[string]string{},
	}
	out.PropertiesKeys, out.PropertiesMap = b.buildAliasMap()
	for name, cfg := range b.services {
		keys := map[string]string{}
		for _, kv := range cfg.LinkableKeys {
			keys[kv.Key] = kv.Value
		}
		out.LinkableKeys[name] = keys
	}
	for _, t := range entitiesMap.Types {
		if t.Kind != graphql.KindObject {
			continue
		}
		fields := map[string]string{}
		for _, f := range t.Fields {
			fields[f.Name] = f.Type.String()
		}
		out.Types[t.Name] = fields
	}
	return out, nil
}
