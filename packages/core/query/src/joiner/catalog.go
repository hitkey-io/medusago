package joiner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
)

// RelationMap maps entity -> property -> related entity.
type RelationMap map[string]map[string]string

// GraphCatalog mirrors GraphCatalog: the index of the service configs.
type GraphCatalog struct {
	cache       map[string]*ServiceConfig
	relationMap RelationMap
}

type extendBucket struct {
	fieldAlias    map[string][]FieldAliasEntry
	fieldAliasKey []string
	relationships *RelationshipMap
}

func newRelationship(r joiner.JoinerRelationship) *Relationship {
	rel := &Relationship{
		Alias:       r.Alias,
		ForeignKey:  r.ForeignKey,
		PrimaryKey:  r.PrimaryKey,
		ServiceName: r.ServiceName,
		Entity:      r.Entity,
		Inverse:     r.Inverse,
		IsList:      r.IsList,
		Args:        r.Args,
	}
	ensureKeyArrays(rel)
	return rel
}

func ensureKeyArrays(rel *Relationship) {
	rel.PrimaryKeyArr = strings.Split(rel.PrimaryKey, ",")
	fks := strings.Split(rel.ForeignKey, ",")
	for i, fk := range fks {
		parts := strings.Split(fk, ".")
		fks[i] = parts[len(parts)-1]
	}
	rel.ForeignKeyArr = fks
}

// NewGraphCatalog indexes the service configs, like the GraphCatalog
// constructor (with autoCreateServiceNameAlias disabled, as Query does).
func NewGraphCatalog(configs []*joiner.ModuleJoinerConfig, relationMap RelationMap) (*GraphCatalog, error) {
	c := &GraphCatalog{cache: map[string]*ServiceConfig{}, relationMap: relationMap}

	type entry struct {
		src *joiner.ModuleJoinerConfig
		svc *ServiceConfig
	}
	var entries []entry
	for _, cfg := range configs {
		svc := &ServiceConfig{
			ServiceName:    cfg.ServiceName,
			PrimaryKeys:    cfg.PrimaryKeys,
			IsLink:         cfg.IsLink,
			IsReadOnlyLink: cfg.IsReadOnlyLink,
			Relationships:  newRelationshipMap(),
			FieldAlias:     map[string][]FieldAliasEntry{},
		}
		for _, r := range cfg.Relationships {
			svc.Relationships.add(r.Alias, newRelationship(r))
		}
		entries = append(entries, entry{cfg, svc})
	}

	buckets := map[string]*extendBucket{}
	var bucketOrder []string

	for _, e := range entries {
		if e.svc.ServiceName != "" {
			if _, exists := c.cache[e.svc.ServiceName]; exists {
				return nil, fmt.Errorf(`Service "%s" is already defined.`, e.svc.ServiceName)
			}
		}
		isReadOnly := e.svc.ServiceName == "" || e.svc.IsReadOnlyLink
		if !isReadOnly {
			// indexAliases: flatten alias names.
			for _, a := range e.src.Alias {
				for _, name := range a.Name {
					e.svc.Alias = append(e.svc.Alias, ServiceAlias{Name: name, Entity: a.Entity, Args: a.Args})
				}
			}
			for _, alias := range e.svc.Alias {
				if defined, ok := c.cache["alias_"+alias.Name]; ok {
					if defined.ServiceName == e.svc.ServiceName {
						continue
					}
					return nil, fmt.Errorf(`Cannot add alias "%s" for "%s". It is already defined for Service "%s".`, alias.Name, e.svc.ServiceName, defined.ServiceName)
				}
				var args map[string]any
				if e.svc.Args != nil || alias.Args != nil {
					args = map[string]any{}
					for k, v := range e.svc.Args {
						args[k] = v
					}
					for k, v := range alias.Args {
						args[k] = v
					}
				}
				rel := &Relationship{
					Alias:       alias.Name,
					Entity:      alias.Entity,
					ForeignKey:  alias.Name + "_id",
					PrimaryKey:  "id",
					ServiceName: e.svc.ServiceName,
					Args:        args,
				}
				ensureKeyArrays(rel)
				e.svc.Relationships.add(alias.Name, rel)
				c.cacheByAlias(e.svc, alias)
			}
			c.cache[e.svc.ServiceName] = e.svc
		}

		// collectExtends
		for _, ext := range e.src.Extends {
			b, ok := buckets[ext.ServiceName]
			if !ok {
				b = &extendBucket{fieldAlias: map[string][]FieldAliasEntry{}, relationships: newRelationshipMap()}
				buckets[ext.ServiceName] = b
				bucketOrder = append(bucketOrder, ext.ServiceName)
			}
			rel := newRelationship(ext.Relationship)
			b.relationships.add(rel.Alias, rel)
			for name, fa := range ext.FieldAlias {
				entry := FieldAliasEntry{Path: fa.Path, IsList: fa.IsList, ForwardArgumentsOnPath: fa.ForwardArgumentsOnPath, Entity: ext.Entity}
				for _, existing := range b.fieldAlias[name] {
					if existing.Entity == ext.Entity {
						return nil, fmt.Errorf(`Cannot add alias "%s" for "%s". It is already defined for Entity "%s".`, name, ext.ServiceName, ext.Entity)
					}
				}
				if _, seen := b.fieldAlias[name]; !seen {
					b.fieldAliasKey = append(b.fieldAliasKey, name)
				}
				b.fieldAlias[name] = append(b.fieldAlias[name], entry)
			}
		}
	}

	// mergeExtends
	for _, serviceName := range bucketOrder {
		b := buckets[serviceName]
		svc, ok := c.cache[serviceName]
		if !ok {
			return nil, fmt.Errorf(`Service "%s" was not found`, serviceName)
		}
		for _, alias := range b.relationships.keys {
			for _, rel := range b.relationships.values[alias] {
				svc.Relationships.add(alias, rel)
			}
		}
		for _, name := range b.fieldAliasKey {
			svc.FieldAlias[name] = b.fieldAlias[name]
		}
		for _, alias := range svc.Relationships.keys {
			if _, conflict := b.fieldAlias[alias]; conflict {
				return nil, fmt.Errorf(`Conflict configuration for service "%s". The following aliases are already defined as relationships: %s`, serviceName, alias)
			}
		}
	}
	return c, nil
}

func (c *GraphCatalog) cacheByAlias(svc *ServiceConfig, alias ServiceAlias) {
	name := "alias_" + alias.Name
	if _, ok := c.cache[name]; ok {
		return
	}
	copyCfg := *svc
	copyCfg.Entity = alias.Entity
	if svc.Args != nil || alias.Args != nil {
		copyCfg.Args = map[string]any{}
		for k, v := range svc.Args {
			copyCfg.Args[k] = v
		}
		for k, v := range alias.Args {
			copyCfg.Args[k] = v
		}
	}
	c.cache[name] = &copyCfg
	if alias.Entity != "" {
		c.cache["entity_"+alias.Entity] = &copyCfg
	}
}

// GetServiceConfig mirrors getServiceConfig.
func (c *GraphCatalog) GetServiceConfig(serviceName, serviceAlias, entity string) *ServiceConfig {
	if entity != "" {
		if s, ok := c.cache["entity_"+entity]; ok {
			return s
		}
	}
	if serviceAlias != "" {
		return c.cache["alias_"+serviceAlias]
	}
	return c.cache[serviceName]
}

// GetEntity mirrors getEntity.
func (c *GraphCatalog) GetEntity(entity, prop string) string {
	return c.relationMap[entity][prop]
}

// GetEntityRelationship mirrors getEntityRelationship.
func (c *GraphCatalog) GetEntityRelationship(parent *ServiceConfig, property, entity string) *Relationship {
	if parent == nil {
		return nil
	}
	rels := parent.Relationships.Get(property)
	if len(rels) == 0 {
		return nil
	}
	if len(rels) == 1 {
		return rels[0]
	}
	propEntity := entity
	if propEntity == "" {
		propEntity = parent.Entity
	}
	if propEntity == "" {
		return rels[0]
	}
	for _, r := range rels {
		if r.Entity == propEntity {
			return r
		}
	}
	if svc := c.GetServiceConfig("", "", propEntity); svc != nil {
		for _, r := range rels {
			if slices.Contains(svc.PrimaryKeys, r.PrimaryKey) {
				return r
			}
		}
	}
	return nil
}
