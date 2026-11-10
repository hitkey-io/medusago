// Package services mirrors @medusajs/search/src/services.
package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	modulessdktypes "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/search/src/models"
)

// SearchIndexDefinition is a registered search index (name -> entity).
// Index definitions (the `indexes` module option and the plugins' search
// index files) are not ported: the default application registers none.
type SearchIndexDefinition struct {
	Name   string
	Entity string
}

// SearchModuleService mirrors SearchModuleService for an application
// without index definitions: the registry reads (listIndexes, getIndex)
// and the operations that start by resolving an index.
type SearchModuleService struct {
	*modulessdk.MedusaService
	indexes map[string]SearchIndexDefinition
}

// NewSearchModuleService builds the service.
func NewSearchModuleService(opts modulessdktypes.ServiceOptions) (*SearchModuleService, error) {
	return &SearchModuleService{
		MedusaService: modulessdk.NewMedusaService(opts.DB, models.All, nil),
		indexes:       map[string]SearchIndexDefinition{},
	}, nil
}

func (s *SearchModuleService) names() []string {
	out := make([]string, 0, len(s.indexes))
	for name := range s.indexes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ListIndexes mirrors listIndexes: the registered indexes by name.
func (s *SearchModuleService) ListIndexes(ctx context.Context) ([]*dal.Record, error) {
	out := []*dal.Record{}
	for _, name := range s.names() {
		r := dal.NewRecord()
		r.Set("name", name)
		r.Set("entity", s.indexes[name].Entity)
		out = append(out, r)
	}
	return out, nil
}

// GetIndex mirrors getIndex (retrieveIndexDefinition).
func (s *SearchModuleService) GetIndex(name string) (SearchIndexDefinition, error) {
	def, ok := s.indexes[name]
	if !ok {
		registered := strings.Join(s.names(), ", ")
		if registered == "" {
			registered = "(none)"
		}
		return SearchIndexDefinition{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`No search index registered for "%s". Registered indexes: %s`, name, registered))
	}
	return def, nil
}

// ResolveIndex is GetIndex for callers that only need the check.
func (s *SearchModuleService) ResolveIndex(name string) error {
	_, err := s.GetIndex(name)
	return err
}

// DeleteIndex mirrors deleteIndex (deleteIndexEntirely).
func (s *SearchModuleService) DeleteIndex(ctx context.Context, name string) (*dal.Record, error) {
	if _, err := s.GetIndex(name); err != nil {
		return nil, err
	}
	return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Search index providers are not supported yet")
}

// Reindex mirrors reindex(input).
func (s *SearchModuleService) Reindex(ctx context.Context, indexes []string, input *dal.Record) (*dal.Record, error) {
	for _, name := range indexes {
		if _, err := s.GetIndex(name); err != nil {
			return nil, err
		}
	}
	return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Search index providers are not supported yet")
}
