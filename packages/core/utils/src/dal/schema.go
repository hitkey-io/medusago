package dal

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// Schema indexes the models of a module by class name.
type Schema struct {
	entities map[string]*dml.Entity
	// Source is the module service name the entity mutation events are
	// emitted for ("" emits none: link modules, the index module).
	Source string
	// Sink receives the events of the committed transactions.
	Sink EventSink
}

// NewSchema builds the schema of a module from its models.
func NewSchema(models []*dml.Entity) *Schema {
	s := &Schema{entities: map[string]*dml.Entity{}}
	for _, m := range models {
		s.entities[m.Meta.ClassName] = m
	}
	return s
}

// Entity returns a model by class name.
func (s *Schema) Entity(className string) (*dml.Entity, error) {
	e, ok := s.entities[className]
	if !ok {
		return nil, fmt.Errorf("unknown entity %s", className)
	}
	return e, nil
}

// Join describes how a relation is reached from its owner.
type Join struct {
	Relation *dml.MetaProperty
	Target   *dml.Entity
	// Many is true for 1:m and m:n relations.
	Many bool

	// Direct joins: target.TargetColumns = source.SourceColumns.
	SourceColumns []string
	TargetColumns []string

	// Pivot joins (m:n): source.pk = pivot.PivotSourceColumns and
	// pivot.PivotTargetColumns = target.pk.
	PivotTable         string
	PivotSourceColumns []string
	PivotTargetColumns []string
}

// ResolveJoin resolves a relation property of an entity.
func (s *Schema) ResolveJoin(e *dml.Entity, relation string) (*Join, error) {
	prop, ok := e.MetaProperty(relation)
	if !ok || prop.Kind == "scalar" {
		return nil, fmt.Errorf("%s has no relation %s", e.Meta.ClassName, relation)
	}
	target, err := s.Entity(prop.Entity)
	if err != nil {
		return nil, err
	}
	j := &Join{Relation: prop, Target: target}

	switch prop.Kind {
	case "m:1":
		j.SourceColumns = prop.FieldNames
		j.TargetColumns = referenced(prop, target)
	case "1:1":
		if prop.Owner {
			j.SourceColumns = prop.FieldNames
			j.TargetColumns = referenced(prop, target)
		} else {
			inverse, ok := target.MetaProperty(prop.MappedBy)
			if !ok {
				return nil, fmt.Errorf("%s.%s: inverse side %s not found", e.Meta.ClassName, relation, prop.MappedBy)
			}
			j.SourceColumns = referenced(inverse, e)
			j.TargetColumns = inverse.FieldNames
		}
	case "1:m":
		j.Many = true
		inverse, ok := target.MetaProperty(prop.MappedBy)
		if !ok {
			return nil, fmt.Errorf("%s.%s: inverse side %s not found", e.Meta.ClassName, relation, prop.MappedBy)
		}
		j.SourceColumns = referenced(inverse, e)
		j.TargetColumns = inverse.FieldNames
	case "m:n":
		j.Many = true
		owner := prop
		if !prop.Owner {
			inverse, ok := target.MetaProperty(prop.MappedBy)
			if !ok {
				return nil, fmt.Errorf("%s.%s: owner side %s not found", e.Meta.ClassName, relation, prop.MappedBy)
			}
			owner = inverse
		}
		j.PivotTable = owner.PivotTable
		if prop.Owner {
			j.PivotSourceColumns = owner.JoinColumns
			j.PivotTargetColumns = owner.InverseJoinColumns
		} else {
			j.PivotSourceColumns = owner.InverseJoinColumns
			j.PivotTargetColumns = owner.JoinColumns
		}
		j.SourceColumns = e.Meta.PrimaryKey
		j.TargetColumns = target.Meta.PrimaryKey
	default:
		return nil, fmt.Errorf("unsupported relation kind %s", prop.Kind)
	}
	return j, nil
}

func referenced(prop *dml.MetaProperty, target *dml.Entity) []string {
	if len(prop.ReferencedColumnNames) > 0 {
		return prop.ReferencedColumnNames
	}
	return target.Meta.PrimaryKey
}
