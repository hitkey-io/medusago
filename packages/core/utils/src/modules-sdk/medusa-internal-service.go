package modulessdk

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// FindConfig mirrors FindConfig.
type FindConfig struct {
	Select      []string
	Relations   []string
	Take        *int
	Skip        *int
	Order       []dal.OrderBy
	WithDeleted bool
}

// InternalService mirrors MedusaInternalService for one model.
type InternalService struct {
	Repository *dal.Repository
}

func (s *InternalService) model() *dml.Entity { return s.Repository.Entity }

// applyDefaultOrdering mirrors applyDefaultOrdering: order by the primary
// keys when no order is given.
func (s *InternalService) applyDefaultOrdering(config *FindConfig) {
	if len(config.Order) > 0 {
		return
	}
	for _, pk := range s.model().Meta.PrimaryKey {
		config.Order = append(config.Order, dal.OrderBy{Field: pk, Direction: "ASC"})
	}
}

func buildQuery(filters *dal.Record, config FindConfig) dal.FindOptions {
	return dal.FindOptions{
		Where:       filters,
		Select:      config.Select,
		Relations:   config.Relations,
		Order:       config.Order,
		Take:        config.Take,
		Skip:        config.Skip,
		WithDeleted: config.WithDeleted,
	}
}

// List mirrors list.
func (s *InternalService) List(ctx context.Context, filters *dal.Record, config FindConfig) ([]*dal.Record, error) {
	s.applyDefaultOrdering(&config)
	return s.Repository.Find(ctx, buildQuery(filters, config))
}

// ListAndCount mirrors listAndCount.
func (s *InternalService) ListAndCount(ctx context.Context, filters *dal.Record, config FindConfig) ([]*dal.Record, int, error) {
	s.applyDefaultOrdering(&config)
	return s.Repository.FindAndCount(ctx, buildQuery(filters, config))
}

func lowerCaseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// Retrieve mirrors retrieve for single primary key models.
func (s *InternalService) Retrieve(ctx context.Context, id any, config FindConfig) (*dal.Record, error) {
	e := s.model()
	pks := e.Meta.PrimaryKey
	idStr, isString := id.(string)
	if id == nil || (isString && len(pks) > 1) || (!isString && len(pks) == 1) || (isString && idStr == "" && len(pks) == 1) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("%s - %s must be defined", lowerCaseFirst(e.Meta.ClassName), strings.Join(pks, ", ")))
	}

	where := dal.NewRecord()
	where.Set(pks[0], id)
	records, err := s.Repository.Find(ctx, buildQuery(where, config))
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("%s with %s: %v was not found", e.Meta.ClassName, strings.Join(pks, ", "), id))
	}
	return records[0], nil
}
