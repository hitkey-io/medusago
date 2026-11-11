// Package utils mirrors @medusajs/caching/src/utils: the cache invalidation
// parser and the default cache strategy.
package utils

import (
	"fmt"
	"math"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
)

// InvalidationEvent mirrors InvalidationEvent.
type InvalidationEvent struct {
	EntityType string
	EntityID   any
	CacheKeys  []string
}

// listAffectingOperations mirrors LIST_AFFECTING_OPERATIONS.
var listAffectingOperations = map[string]bool{
	"created": true, "updated": true, "deleted": true, "restored": true, "attached": true, "detached": true,
}

// CacheInvalidationParser mirrors CacheInvalidationParser: the object types
// of the merged schema (in type map order) and the id prefixes of the
// joiner configs. An object whose id has no registered prefix is matched
// against the types in that order; upstream it follows the order in which
// the modules finished loading (not deterministic), here the order of the
// joiner configs.
type CacheInvalidationParser struct {
	types                []*graphql.Type
	typeMap              map[string]*graphql.Type
	idPrefixToEntityName map[string]string
}

// NewCacheInvalidationParser mirrors the constructor.
func NewCacheInvalidationParser(schema *graphql.Schema, joinerConfigs []*joiner.ModuleJoinerConfig) *CacheInvalidationParser {
	p := &CacheInvalidationParser{typeMap: map[string]*graphql.Type{}, idPrefixToEntityName: map[string]string{}}
	if schema != nil {
		for _, t := range schema.Types {
			if t.Kind == graphql.KindObject && !strings.HasPrefix(t.Name, "__") {
				p.types = append(p.types, t)
				p.typeMap[t.Name] = t
			}
		}
	}
	for _, c := range joinerConfigs {
		if c == nil {
			continue
		}
		for prefix, name := range c.IDPrefixToEntityName {
			p.idPrefixToEntityName[prefix] = name
		}
	}
	return p
}

// keysOf returns the keys of a JSON-like object (nil when not an object).
func keysOf(obj any) ([]string, func(string) any) {
	switch t := obj.(type) {
	case *dal.Record:
		if t == nil {
			return nil, nil
		}
		return t.Keys(), func(k string) any { v, _ := t.Get(k); return v }
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		return keys, func(k string) any { return t[k] }
	}
	return nil, nil
}

// truthyID mirrors `obj.id` being truthy.
func truthyID(id any) bool { return caching.Truthy(id) }

// ParseObjectForEntities mirrors parseObjectForEntities.
func (p *CacheInvalidationParser) ParseObjectForEntities(obj any, parentType string, isInArray bool) []caching.EntityReference {
	var entities []caching.EntityReference
	keys, get := keysOf(obj)
	if get == nil {
		if list, ok := obj.([]any); ok {
			// an array is an object too: its items are parsed
			for _, item := range list {
				if l, ok := item.([]any); ok {
					for _, x := range l {
						entities = append(entities, p.ParseObjectForEntities(x, "", true)...)
					}
				} else if _, g := keysOf(item); g != nil {
					entities = append(entities, p.ParseObjectForEntities(item, "", false)...)
				}
			}
		}
		return entities
	}
	detected := p.detectEntityType(keys, get, parentType)
	if detected != "" && truthyID(get("id")) {
		entities = append(entities, caching.EntityReference{Type: detected, ID: get("id"), IsInArray: isInArray})
	}
	for _, k := range keys {
		value := get(k)
		if list, ok := value.([]any); ok {
			for _, item := range list {
				entities = append(entities, p.ParseObjectForEntities(item, p.getRelationshipType(detected, k), true)...)
			}
		} else if records, ok := value.([]*dal.Record); ok {
			for _, item := range records {
				entities = append(entities, p.ParseObjectForEntities(item, p.getRelationshipType(detected, k), true)...)
			}
		} else if _, g := keysOf(value); g != nil {
			entities = append(entities, p.ParseObjectForEntities(value, p.getRelationshipType(detected, k), false)...)
		}
	}
	return entities
}

// detectEntityType mirrors detectEntityType.
func (p *CacheInvalidationParser) detectEntityType(keys []string, get func(string) any, suggested string) string {
	if id, ok := get("id").(string); ok {
		parts := strings.Split(id, "_")
		if len(parts) > 1 {
			if name, ok := p.idPrefixToEntityName[parts[0]]; ok && name != "" {
				return name
			}
		}
	}
	if suggested != "" {
		if t, ok := p.typeMap[suggested]; ok && objectMatchesType(keys, get, t) {
			return suggested
		}
	}
	for _, t := range p.types {
		if objectMatchesType(keys, get, t) {
			return t.Name
		}
	}
	return ""
}

// objectMatchesType mirrors objectMatchesType: an id and at least half of
// the object's keys being fields of the type.
func objectMatchesType(keys []string, get func(string) any, t *graphql.Type) bool {
	if !truthyID(get("id")) || t.Field("id") == nil {
		return false
	}
	matching := 0
	for _, k := range keys {
		if t.Field(k) != nil {
			matching++
		}
	}
	return float64(matching) >= math.Max(1, float64(len(keys))*0.5)
}

// getRelationshipType mirrors getRelationshipType.
func (p *CacheInvalidationParser) getRelationshipType(parentType, field string) string {
	if parentType == "" {
		return ""
	}
	t, ok := p.typeMap[parentType]
	if !ok {
		return ""
	}
	f := t.Field(field)
	if f == nil || f.Type == nil {
		return ""
	}
	name := f.Type.NamedType()
	if _, ok := p.typeMap[name]; ok {
		return name
	}
	return ""
}

// BuildInvalidationEvents mirrors buildInvalidationEvents (operation
// "updated" by default).
func (p *CacheInvalidationParser) BuildInvalidationEvents(entities []caching.EntityReference, operation string) []InvalidationEvent {
	if operation == "" {
		operation = "updated"
	}
	var events []InvalidationEvent
	processed := map[string]bool{}
	for _, e := range entities {
		key := fmt.Sprintf("%s:%s", e.Type, idString(e.ID))
		if processed[key] {
			continue
		}
		processed[key] = true
		events = append(events, InvalidationEvent{EntityType: e.Type, EntityID: e.ID, CacheKeys: buildAffectedCacheKeys(e, operation)})
	}
	return events
}

// idString mirrors the template literal of an id.
func idString(id any) string {
	switch t := id.(type) {
	case string:
		return t
	case float64:
		if t == math.Trunc(t) && math.Abs(t) < 1e21 {
			return fmt.Sprintf("%d", int64(t))
		}
	case nil:
		return "undefined"
	}
	return fmt.Sprint(id)
}

// buildAffectedCacheKeys mirrors buildAffectedCacheKeys.
func buildAffectedCacheKeys(e caching.EntityReference, operation string) []string {
	keys := []string{fmt.Sprintf("%s:%s", e.Type, idString(e.ID))}
	if e.IsInArray || listAffectingOperations[operation] {
		keys = append(keys, e.Type+":list:*")
	}
	return keys
}
