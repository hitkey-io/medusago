package modulessdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// MedusaService mirrors MedusaService: the generated methods of a module
// service (listProducts, retrieveProduct, ...) dispatched by name.
type MedusaService struct {
	byName map[string]*InternalService
	plural map[string]*InternalService
}

// NewMedusaService builds the service of a module over its models.
func NewMedusaService(db dal.Querier, models []*dml.Entity) *MedusaService {
	schema := dal.NewSchema(models)
	s := &MedusaService{byName: map[string]*InternalService{}, plural: map[string]*InternalService{}}
	for _, m := range models {
		svc := &InternalService{Repository: &dal.Repository{DB: db, Schema: schema, Entity: m}}
		name := m.Meta.ClassName
		s.byName[name] = svc
		s.plural[Pluralize(name)] = svc
	}
	return s
}

// Pluralize mirrors the pluralization used for the generated method names.
func Pluralize(word string) string {
	lower := strings.ToLower(word)
	switch {
	case strings.HasSuffix(lower, "y") && len(word) > 1 && !strings.ContainsRune("aeiou", rune(lower[len(lower)-2])):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}

// Call invokes a generated method by name with JSON-like arguments
// (decoded with dal.DecodeOrderedJSON).
func (s *MedusaService) Call(ctx context.Context, method string, args []any) (any, error) {
	arg := func(i int) any {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	config, err := ParseFindConfig(arg(1))
	if err != nil {
		return nil, err
	}

	switch {
	case strings.HasPrefix(method, "listAndCount"):
		svc, ok := s.plural[strings.TrimPrefix(method, "listAndCount")]
		if !ok {
			break
		}
		filters, _ := arg(0).(*dal.Record)
		records, count, err := svc.ListAndCount(ctx, filters, config)
		if err != nil {
			return nil, err
		}
		return []any{records, count}, nil
	case strings.HasPrefix(method, "list"):
		svc, ok := s.plural[strings.TrimPrefix(method, "list")]
		if !ok {
			break
		}
		filters, _ := arg(0).(*dal.Record)
		return svc.List(ctx, filters, config)
	case strings.HasPrefix(method, "retrieve"):
		svc, ok := s.byName[strings.TrimPrefix(method, "retrieve")]
		if !ok {
			break
		}
		return svc.Retrieve(ctx, arg(0), config)
	}
	return nil, fmt.Errorf("unknown method %s", method)
}

// ParseFindConfig converts a FindConfig object into FindConfig.
func ParseFindConfig(v any) (FindConfig, error) {
	var c FindConfig
	r, ok := v.(*dal.Record)
	if !ok || r == nil {
		return c, nil
	}
	strs := func(key string) []string {
		raw, _ := r.Get(key)
		list, _ := raw.([]any)
		out := make([]string, 0, len(list))
		for _, x := range list {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	intPtr := func(key string) *int {
		raw, ok := r.Get(key)
		f, isNum := raw.(float64)
		if !ok || !isNum {
			return nil
		}
		n := int(f)
		return &n
	}
	c.Select = strs("select")
	c.Relations = strs("relations")
	c.Take = intPtr("take")
	c.Skip = intPtr("skip")
	if wd, ok := r.Get("withDeleted"); ok {
		c.WithDeleted, _ = wd.(bool)
	}
	if o, ok := r.Get("order"); ok {
		if order, ok := o.(*dal.Record); ok {
			for _, k := range order.Keys() {
				dir, _ := order.Get(k)
				c.Order = append(c.Order, dal.OrderBy{Field: k, Direction: strings.ToUpper(fmt.Sprint(dir))})
			}
		}
	}
	return c, nil
}
