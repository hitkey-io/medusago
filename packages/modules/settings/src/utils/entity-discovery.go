// Package utils mirrors @medusajs/settings/src/utils.
package utils

import (
	"regexp"
	"strings"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// ToKebabCasePlural mirrors toKebabCasePlural.
func ToKebabCasePlural(name string) string {
	return common.Pluralize(common.KebabCase(name))
}

// normalizeEntityName mirrors normalizeEntityName.
func normalizeEntityName(name string) string {
	capitalize := func(parts []string) string {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(common.UpperCaseFirst(p))
		}
		return b.String()
	}
	switch {
	case strings.Contains(name, "-"):
		return common.Singularize(capitalize(strings.Split(name, "-")))
	case strings.Contains(name, "_"):
		return common.Singularize(capitalize(strings.Split(name, "_")))
	}
	return common.Singularize(common.UpperCaseFirst(name))
}

// DiscoveredEntity mirrors DiscoveredEntity.
type DiscoveredEntity struct {
	Name         string
	PluralName   string
	Module       string
	GraphqlType  string
	Schema       string
	PrimaryKeys  []string
	LinkableKeys joiner.OrderedStringMap
}

// EntityInfo mirrors EntityInfo (the API response item).
type EntityInfo struct {
	Name          string
	PluralName    string
	Module        string
	PropertyCount int
	HasOverrides  bool
}

// EntityDiscoveryService mirrors EntityDiscoveryService: the entities of
// the joiner config schemas of the loaded modules.
type EntityDiscoveryService struct {
	mu          sync.Mutex
	configs     []*joiner.ModuleJoinerConfig
	initialized bool
	schema      *graphql.Schema
	entities    []DiscoveredEntity
}

// Initialize mirrors initialize(joinerConfigs).
func (s *EntityDiscoveryService) Initialize(configs []*joiner.ModuleJoinerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs = configs
	s.initialized = true
	s.schema = nil
	s.entities = nil
}

// IsInitialized mirrors isInitialized.
func (s *EntityDiscoveryService) IsInitialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// schemaTypeMap mirrors getSchemaTypeMap: the merged and cleaned schema of
// the joiner config schemas.
func (s *EntityDiscoveryService) schemaTypeMap() *graphql.Schema {
	if s.schema != nil {
		return s.schema
	}
	var fragments []string
	for _, c := range s.configs {
		if c.Schema != "" {
			fragments = append(fragments, c.Schema)
		}
	}
	if len(fragments) == 0 {
		s.schema = graphql.MergeTypeDefs()
		return s.schema
	}
	schema, err := graphql.BuildSchema(fragments...)
	if err != nil {
		s.schema = graphql.MergeTypeDefs()
		return s.schema
	}
	s.schema = schema
	return schema
}

var typeDefinitionRe = regexp.MustCompile(`type\s+(\w+)\s*\{`)

// DiscoverEntities mirrors discoverEntities.
func (s *EntityDiscoveryService) DiscoverEntities() []DiscoveredEntity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.discoverEntities()
}

func (s *EntityDiscoveryService) discoverEntities() []DiscoveredEntity {
	if !s.initialized {
		return nil
	}
	if s.entities != nil {
		return s.entities
	}
	entities := []DiscoveredEntity{}
	seen := map[string]bool{}
	for _, c := range s.configs {
		if c.IsLink || c.Schema == "" {
			continue
		}
		for _, m := range typeDefinitionRe.FindAllStringSubmatch(c.Schema, -1) {
			name := m[1]
			if seen[name] {
				continue
			}
			if strings.HasSuffix(name, "Link") || strings.HasPrefix(name, "_") || name == "Query" || name == "Mutation" {
				continue
			}
			seen[name] = true
			pks := c.PrimaryKeys
			if pks == nil {
				pks = []string{"id"}
			}
			entities = append(entities, DiscoveredEntity{
				Name:         name,
				PluralName:   ToKebabCasePlural(name),
				Module:       c.ServiceName,
				GraphqlType:  name,
				Schema:       c.Schema,
				PrimaryKeys:  pks,
				LinkableKeys: c.LinkableKeys,
			})
		}
	}
	s.entities = entities
	return entities
}

// GetEntity mirrors getEntity: by name, plural name or normalized name.
func (s *EntityDiscoveryService) GetEntity(name string) *DiscoveredEntity {
	entities := s.DiscoverEntities()
	lower := strings.ToLower(name)
	for i := range entities {
		if strings.ToLower(entities[i].Name) == lower {
			return &entities[i]
		}
	}
	for i := range entities {
		if strings.ToLower(entities[i].PluralName) == lower {
			return &entities[i]
		}
	}
	normalized := strings.ToLower(normalizeEntityName(name))
	for i := range entities {
		if strings.ToLower(entities[i].Name) == normalized {
			return &entities[i]
		}
	}
	return nil
}

// HasEntity mirrors hasEntity.
func (s *EntityDiscoveryService) HasEntity(name string) bool {
	return s.GetEntity(name) != nil
}

// GetEntityInfo mirrors getEntityInfo: the field count of the entity type
// in the merged schema.
func (s *EntityDiscoveryService) GetEntityInfo(entity DiscoveredEntity, hasOverrides bool) EntityInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	if t := s.schemaTypeMap().Type(entity.GraphqlType); t != nil && (t.Kind == graphql.KindObject || t.Kind == graphql.KindInterface || t.Kind == graphql.KindInput) {
		count = len(t.Fields)
	}
	return EntityInfo{
		Name:          entity.Name,
		PluralName:    entity.PluralName,
		Module:        entity.Module,
		PropertyCount: count,
		HasOverrides:  hasOverrides,
	}
}

// LocaleCompare approximates String.prototype.localeCompare (ICU root
// collation) for identifiers: letters compare case-insensitively first,
// then lowercase sorts before uppercase.
func LocaleCompare(a, b string) int {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		if la < lb {
			return -1
		}
		return 1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		// lowercase first
		if a[i] > b[i] {
			return -1
		}
		return 1
	}
	return 0
}
