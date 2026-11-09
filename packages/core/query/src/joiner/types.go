// Package joiner mirrors @medusajs/query/src/joiner: the RemoteJoiner that
// plans a query against the module and link joiner configs, fetches from
// each module service and stitches the results.
package joiner

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// BasePath is the path of the query root.
const BasePath = "_root"

// Arg is a joiner argument (filters, pagination, withDeleted...).
type Arg struct {
	Name  string
	Value any
}

// ExpandSpec is one expand of a RemoteJoinerQuery.
type ExpandSpec struct {
	Property       string
	Fields         []string
	Args           []Arg
	IsAliasMapping bool
}

// RemoteJoinerQuery mirrors RemoteJoinerQuery.
type RemoteJoinerQuery struct {
	Service string
	Alias   string
	Fields  []string
	Expands []ExpandSpec
	Args    []Arg
}

// Relationship mirrors ComputedJoinerRelationship.
type Relationship struct {
	Alias         string
	ForeignKey    string
	PrimaryKey    string
	ServiceName   string
	Entity        string
	Inverse       bool
	IsList        bool
	Args          map[string]any
	PrimaryKeyArr []string
	ForeignKeyArr []string
}

// RelationshipMap maps an alias to one or more relationships, keeping the
// insertion order.
type RelationshipMap struct {
	keys   []string
	values map[string][]*Relationship
}

func newRelationshipMap() *RelationshipMap {
	return &RelationshipMap{values: map[string][]*Relationship{}}
}

func (m *RelationshipMap) add(alias string, rel *Relationship) {
	if _, ok := m.values[alias]; !ok {
		m.keys = append(m.keys, alias)
	}
	m.values[alias] = append(m.values[alias], rel)
}

// Get returns the relationships registered under alias.
func (m *RelationshipMap) Get(alias string) []*Relationship {
	if m == nil {
		return nil
	}
	return m.values[alias]
}

// FieldAliasEntry is a resolved field alias (ResolvedFieldAlias).
type FieldAliasEntry struct {
	Path                   string
	IsList                 bool
	ForwardArgumentsOnPath []string
	Entity                 string
}

// ServiceAlias is a flattened service alias.
type ServiceAlias struct {
	Name   string
	Entity string
	Args   map[string]any
}

// ServiceConfig mirrors InternalJoinerServiceConfig. Relationships and
// FieldAlias are shared between a service config and its alias copies.
type ServiceConfig struct {
	ServiceName    string
	Entity         string
	PrimaryKeys    []string
	Args           map[string]any
	Alias          []ServiceAlias
	IsLink         bool
	IsReadOnlyLink bool
	Relationships  *RelationshipMap
	FieldAlias     map[string][]FieldAliasEntry
}

// ExpandProperty mirrors RemoteExpandProperty (and RemoteNestedExpands for
// the expands merged into the same service).
type ExpandProperty struct {
	Property        string
	Parent          string
	ParentConfig    *ServiceConfig
	ServiceConfig   *ServiceConfig
	Entity          string
	Fields          []string
	Args            []Arg
	Expands         *NestedExpands
	IsAliasMapping  bool
	ExecutionStages [][]ExecutionStage
}

// NestedExpands is an ordered set of nested expands.
type NestedExpands struct {
	Keys   []string
	Values map[string]*ExpandProperty
}

func (n *NestedExpands) getOrCreate(key string) *ExpandProperty {
	if v, ok := n.Values[key]; ok {
		return v
	}
	v := &ExpandProperty{}
	n.Keys = append(n.Keys, key)
	n.Values[key] = v
	return v
}

// ExecutionStage mirrors ExecutionStage.
type ExecutionStage struct {
	Service string
	Entity  string
	Paths   []string
	Depth   int
}

// ShortcutSpec mirrors ShortcutSpec.
type ShortcutSpec struct {
	Location []string
	Property string
	Path     []string
	IsList   bool
}

// ExpandMap is an ordered map path -> expand.
type ExpandMap struct {
	keys   []string
	values map[string]*ExpandProperty
}

func newExpandMap() *ExpandMap { return &ExpandMap{values: map[string]*ExpandProperty{}} }

func (m *ExpandMap) set(path string, e *ExpandProperty) {
	if _, ok := m.values[path]; !ok {
		m.keys = append(m.keys, path)
	}
	m.values[path] = e
}

// Get returns the expand at path.
func (m *ExpandMap) Get(path string) *ExpandProperty { return m.values[path] }

func (m *ExpandMap) has(path string) bool {
	_, ok := m.values[path]
	return ok
}

func (m *ExpandMap) delete(path string) {
	if _, ok := m.values[path]; !ok {
		return
	}
	delete(m.values, path)
	for i, k := range m.keys {
		if k == path {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}

func (m *ExpandMap) clone() *ExpandMap {
	c := newExpandMap()
	for _, k := range m.keys {
		c.set(k, m.values[k])
	}
	return c
}

// QueryPlan mirrors QueryPlan.
type QueryPlan struct {
	Root          *ExpandProperty
	Expands       *ExpandMap
	Shortcuts     []ShortcutSpec
	PkName        string
	PrimaryKeyArg *Arg
	OtherArgs     []Arg
}

// FetchResult is what a data fetcher returns: rows, or rows with pagination
// metadata (path "rows").
type FetchResult struct {
	Rows     []any
	Metadata *dal.Record
	Paged    bool
}

// DataFetcher mirrors IRemoteDataFetcher.
type DataFetcher interface {
	Fetch(ctx context.Context, expand *ExpandProperty, keyField string, ids []any, hasIDs bool, relationship *Relationship) (*FetchResult, error)
}

// Options mirrors RemoteJoinerOptions (the subset supported).
type Options struct {
	ThrowIfKeyNotFound      bool
	ThrowIfRelationNotFound bool
}
