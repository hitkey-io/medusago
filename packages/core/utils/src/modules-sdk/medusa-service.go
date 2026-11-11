package modulessdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
)

// MedusaService mirrors MedusaService: the generated methods of a module
// service (listProducts, retrieveProduct, createProducts, ...) dispatched by
// name.
type MedusaService struct {
	byName       map[string]*InternalService
	bySuffix     map[string]*InternalService
	linkableKeys map[string][]linkableKey
	schema       *dal.Schema
	events       *ModuleEventSink
}

type linkableKey struct {
	mapTo     string
	valueFrom string
}

// NewMedusaService builds the service of a module over its models. The
// method suffixes and linkable keys come from the module joiner config.
func NewMedusaService(db dal.Querier, models []*dml.Entity, joinerConfig *joiner.ModuleJoinerConfig) *MedusaService {
	schema := dal.NewSchema(models)
	events := &ModuleEventSink{}
	// MedusaService subscribes every internal service to the entity
	// mutations, except for the index module.
	if joinerConfig != nil && joinerConfig.ServiceName != "index" {
		schema.Source = joinerConfig.ServiceName
		schema.Sink = events
	}
	s := &MedusaService{byName: map[string]*InternalService{}, bySuffix: map[string]*InternalService{}, linkableKeys: map[string][]linkableKey{}, schema: schema, events: events}
	for _, m := range models {
		svc := &InternalService{Repository: &dal.Repository{DB: db, Schema: schema, Entity: m}}
		s.byName[m.Meta.ClassName] = svc
		s.bySuffix[Pluralize(m.Meta.ClassName)] = svc
	}
	if joinerConfig != nil {
		for _, alias := range joinerConfig.Alias {
			suffix, _ := alias.Args["methodSuffix"].(string)
			if svc, ok := s.byName[alias.Entity]; ok && suffix != "" {
				s.bySuffix[suffix] = svc
			}
		}
		// buildModelsNameToLinkableKeysMap
		for _, kv := range joinerConfig.LinkableKeys {
			parts := strings.Split(kv.Key, "_")
			s.linkableKeys[kv.Value] = append(s.linkableKeys[kv.Value], linkableKey{mapTo: kv.Key, valueFrom: parts[len(parts)-1]})
		}
	}
	return s
}

// Pluralize is the fallback for models without a method suffix.
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

// Internal returns the internal service of a model, like the
// <model>Service_ properties of upstream module services.
func (s *MedusaService) Internal(className string) *InternalService {
	return s.byName[className]
}

// ListBySuffix calls list<suffix>, the method Query uses to fetch data.
func (s *MedusaService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config FindConfig) ([]*dal.Record, error) {
	svc, ok := s.bySuffix[suffix]
	if !ok {
		return nil, fmt.Errorf("method list%s does not exist", suffix)
	}
	return svc.List(ctx, filters, config)
}

// ListAndCountBySuffix calls listAndCount<suffix>.
func (s *MedusaService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config FindConfig) ([]*dal.Record, int, error) {
	svc, ok := s.bySuffix[suffix]
	if !ok {
		return nil, 0, fmt.Errorf("method listAndCount%s does not exist", suffix)
	}
	return svc.ListAndCount(ctx, filters, config)
}

// mapCascade mirrors mapObjectTo over the soft deleted / restored entities.
func (s *MedusaService) mapCascade(res *dal.CascadeResult) *dal.Record {
	out := dal.NewRecord()
	if res == nil {
		return out
	}
	for _, class := range res.Classes {
		configs, ok := s.linkableKeys[class]
		if !ok {
			continue
		}
		for _, c := range configs {
			values := make([]any, 0, len(res.Entities[class]))
			for _, row := range res.Entities[class] {
				values = append(values, row[c.valueFrom])
			}
			out.Set(c.mapTo, values)
		}
	}
	return out
}

var methodPrefixes = []string{"listAndCount", "list", "retrieve", "create", "update", "upsert", "delete", "softDelete", "restore"}

// Call invokes a generated method by name with JSON-like arguments
// (decoded with dal.DecodeOrderedJSON).
func (s *MedusaService) Call(ctx context.Context, method string, args []any) (any, error) {
	arg := func(i int) any {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	for _, prefix := range methodPrefixes {
		if !strings.HasPrefix(method, prefix) {
			continue
		}
		name := strings.TrimPrefix(method, prefix)
		if prefix == "retrieve" {
			svc, ok := s.byName[name]
			if !ok {
				break
			}
			config, err := ParseFindConfig(arg(1))
			if err != nil {
				return nil, err
			}
			return svc.Retrieve(ctx, arg(0), config)
		}
		svc, ok := s.bySuffix[name]
		if !ok {
			continue
		}
		switch prefix {
		case "listAndCount":
			config, err := ParseFindConfig(arg(1))
			if err != nil {
				return nil, err
			}
			filters, _ := arg(0).(*dal.Record)
			records, count, err := svc.ListAndCount(ctx, filters, config)
			if err != nil {
				return nil, err
			}
			return []any{records, count}, nil
		case "list":
			config, err := ParseFindConfig(arg(1))
			if err != nil {
				return nil, err
			}
			filters, _ := arg(0).(*dal.Record)
			return svc.List(ctx, filters, config)
		case "create", "upsert":
			input := arg(0)
			var items []*dal.Record
			list, isList := input.([]any)
			if !isList {
				list = []any{input}
			}
			for _, item := range list {
				if rec, ok := item.(*dal.Record); ok {
					items = append(items, rec)
				}
			}
			create := svc.Create
			if prefix == "upsert" {
				create = svc.Upsert
			}
			created, err := create(ctx, items)
			if err != nil {
				return nil, err
			}
			if isList {
				return created, nil
			}
			if len(created) == 0 {
				return nil, nil
			}
			return created[0], nil
		case "update":
			return svc.Update(ctx, arg(0))
		case "delete":
			_, err := svc.Delete(ctx, arg(0))
			return nil, err
		case "softDelete":
			res, err := svc.SoftDelete(ctx, arg(0))
			if err != nil {
				return nil, err
			}
			return s.mapCascade(res), nil
		case "restore":
			res, err := svc.Restore(ctx, arg(0))
			if err != nil {
				return nil, err
			}
			return s.mapCascade(res), nil
		}
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

// withDB returns a copy of the service whose internal services use db.
func (s *MedusaService) withDB(db dal.Querier) *MedusaService {
	c := &MedusaService{byName: map[string]*InternalService{}, bySuffix: map[string]*InternalService{}, linkableKeys: s.linkableKeys, schema: s.schema, events: s.events}
	copies := map[*InternalService]*InternalService{}
	for name, svc := range s.byName {
		repo := *svc.Repository
		repo.DB = db
		n := &InternalService{Repository: &repo}
		copies[svc] = n
		c.byName[name] = n
	}
	for suffix, svc := range s.bySuffix {
		c.bySuffix[suffix] = copies[svc]
	}
	return c
}

// DB returns the database handle of the service.
func (s *MedusaService) DB() dal.Querier {
	for _, svc := range s.byName {
		return svc.Repository.DB
	}
	return nil
}

// InTransaction mirrors @InjectTransactionManager: fn runs with a copy of
// the service bound to one transaction (a savepoint when already in one).
//
// The outermost transaction emits the entity mutation events of the module
// once committed (see dal.Transaction).
func (s *MedusaService) InTransaction(ctx context.Context, fn func(tx *MedusaService) error) error {
	return dal.Transaction(ctx, s.DB(), s.schema, func(tx dal.Querier) error {
		if tx == s.DB() {
			return fn(s)
		}
		return fn(s.withDB(tx))
	})
}

// SetEventBus sets the event bus the module emits its events to.
func (s *MedusaService) SetEventBus(bus eventbus.Emitter) {
	s.events.SetEventBus(bus)
}

// EventSink returns the emitter of the module events.
func (s *MedusaService) EventSink() dal.EventSink {
	if s.schema.Sink == nil {
		return nil
	}
	return s.events
}

// EventSource returns the module service name of the events ("" when the
// module emits none).
func (s *MedusaService) EventSource() string { return s.schema.Source }

// SaveEvent saves an explicitly built event of this module (see the
// package-level SaveEvent).
func (s *MedusaService) SaveEvent(ctx context.Context, ev *dal.ReadyEvent) {
	SaveEvent(ctx, s.DB(), s.EventSink(), ev)
}
