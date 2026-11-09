package modulessdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/services"
)

// Link mirrors Link (remoteLink / ContainerRegistrationKeys.LINK): it
// creates and dismisses links between modules through the link modules.
type Link struct {
	pairs         map[string]*services.LinkModuleService
	services      map[string]*services.LinkModuleService
	modules       map[string]any
	moduleConfigs map[string]*joiner.ModuleJoinerConfig
	relations     map[string]map[string][]cascadeRelationship
	relationOrder []string
}

// NewLink builds Link over the joiner configs (in load order), the link
// module services (keyed by service name) and the module services (keyed
// by module key, used for cascades).
func NewLink(configs []*joiner.ModuleJoinerConfig, linkServices map[string]*services.LinkModuleService, moduleServices map[string]any) *Link {
	l := &Link{pairs: map[string]*services.LinkModuleService{}, services: linkServices, modules: moduleServices, moduleConfigs: map[string]*joiner.ModuleJoinerConfig{}}
	for _, cfg := range configs {
		if cfg.IsReadOnlyLink || cfg.ServiceName == "" {
			continue
		}
		if cfg.IsLink {
			svc := linkServices[cfg.ServiceName]
			if svc == nil {
				continue
			}
			if len(svc.Config.Relationships) == 2 {
				p, f := svc.Config.Relationships[0], svc.Config.Relationships[1]
				l.pairs[strings.Join([]string{p.ServiceName, p.ForeignKey, f.ServiceName, f.ForeignKey}, "-")] = svc
			}
		} else {
			l.moduleConfigs[cfg.ServiceName] = cfg
		}
		l.addModule(cfg, cfg.ServiceName)
	}
	return l
}

// LinkDefinition is { [moduleA]: { key: value }, [moduleB]: { key: value },
// data? } in insertion order.
type LinkDefinition = *dal.Record

type linkDataConfig struct {
	moduleA, moduleB       string
	primaryKeys            []string
	moduleAKey, moduleBKey string
}

func modulesOf(link LinkDefinition) []string {
	var out []string
	for _, k := range link.Keys() {
		if k != "data" {
			out = append(out, k)
		}
	}
	return out
}

func sub(link LinkDefinition, module string) *dal.Record {
	v, _ := link.Get(module)
	r, _ := v.(*dal.Record)
	if r == nil {
		r = dal.NewRecord()
	}
	return r
}

func getLinkDataConfig(link LinkDefinition) linkDataConfig {
	mods := modulesOf(link)
	var a, b string
	if len(mods) > 0 {
		a = mods[0]
	}
	if len(mods) > 1 {
		b = mods[1]
	}
	pks := sub(link, a).Keys()
	return linkDataConfig{moduleA: a, moduleB: b, primaryKeys: pks, moduleAKey: strings.Join(pks, ","), moduleBKey: strings.Join(sub(link, b).Keys(), ",")}
}

func linkingErrorMessage(c linkDataConfig, method string) string {
	return fmt.Sprintf("Module to type %s and %s by keys %s and %s was not found. Ensure the link exists, keys are correct, and link is passed in the correct order to method 'remoteLink.%s'.\n", c.moduleA, c.moduleB, c.moduleAKey, c.moduleBKey, method)
}

func (l *Link) getLinkModuleOrThrow(link LinkDefinition, method string) (*services.LinkModuleService, linkDataConfig, error) {
	if len(modulesOf(link)) > 2 {
		return nil, linkDataConfig{}, fmt.Errorf("Only two modules can be linked.")
	}
	c := getLinkDataConfig(link)
	svc := l.pairs[strings.Join([]string{c.moduleA, c.moduleAKey, c.moduleB, c.moduleBKey}, "-")]
	if svc == nil {
		return nil, c, fmt.Errorf("%s", linkingErrorMessage(c, method))
	}
	return svc, c, nil
}

// Create mirrors create.
func (l *Link) Create(ctx context.Context, links []LinkDefinition) ([]*dal.Record, error) {
	type batch struct {
		svc     *services.LinkModuleService
		inputs  []services.LinkInput
		filters []any
		unique  map[string]any
	}
	var order []string
	batches := map[string]*batch{}
	for _, link := range links {
		svc, c, err := l.getLinkModuleOrThrow(link, "create")
		if err != nil {
			return nil, err
		}
		name := svc.Config.ServiceName
		b := batches[name]
		if b == nil {
			b = &batch{svc: svc, unique: map[string]any{}}
			batches[name] = b
			order = append(order, name)
		}
		rels := svc.Config.Relationships
		modA, modB := rels[0], rels[1]
		valA, _ := sub(link, c.moduleA).Get(modA.ForeignKey)
		valB, _ := sub(link, c.moduleB).Get(modB.ForeignKey)
		dupErr := common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot create multiple links between '%s' and '%s'", modA.ServiceName, modB.ServiceName))
		assertUnique := func(key string, value, pair any) error {
			mk := fmt.Sprintf("%s:%v", key, value)
			if existing, ok := b.unique[mk]; ok && existing != pair {
				return dupErr
			}
			b.unique[mk] = pair
			return nil
		}
		eq := func(k string, v any) *dal.Record {
			r := dal.NewRecord()
			r.Set(k, v)
			return r
		}
		switch {
		case !modA.HasMany && !modB.HasMany:
			f := dal.NewRecord()
			f.Set("$or", []any{eq(modA.ForeignKey, valA), eq(modB.ForeignKey, valB)})
			b.filters = append(b.filters, f)
			if err := assertUnique(modA.ForeignKey, valA, valB); err != nil {
				return nil, err
			}
			if err := assertUnique(modB.ForeignKey, valB, valA); err != nil {
				return nil, err
			}
		case !modA.HasMany:
			f := dal.NewRecord()
			ne := dal.NewRecord()
			ne.Set("$ne", valA)
			f.Set(modA.ForeignKey, ne)
			f.Set(modB.ForeignKey, valB)
			b.filters = append(b.filters, f)
			if err := assertUnique(modB.ForeignKey, valB, valA); err != nil {
				return nil, err
			}
		case !modB.HasMany:
			f := dal.NewRecord()
			ne := dal.NewRecord()
			ne.Set("$ne", valB)
			f.Set(modB.ForeignKey, ne)
			f.Set(modA.ForeignKey, valA)
			b.filters = append(b.filters, f)
			if err := assertUnique(modA.ForeignKey, valA, valB); err != nil {
				return nil, err
			}
		}
		pk, _ := sub(link, c.moduleA).Get(c.primaryKeys[0])
		fk, _ := sub(link, c.moduleB).Get(c.moduleBKey)
		in := services.LinkInput{Primary: pk, Foreign: fk}
		if d, ok := link.Get("data"); ok {
			if dr, ok := d.(*dal.Record); ok {
				in.Extra = dr
			}
		}
		b.inputs = append(b.inputs, in)
	}
	for _, name := range order {
		b := batches[name]
		if len(b.filters) == 0 {
			continue
		}
		f := dal.NewRecord()
		f.Set("$or", b.filters)
		take := 1
		existing, err := b.svc.List(ctx, f, utilsmodulessdk.FindConfig{Take: &take})
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			rels := b.svc.Config.Relationships
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot create multiple links between '%s' and '%s'", rels[0].ServiceName, rels[1].ServiceName))
		}
	}
	var out []*dal.Record
	for _, name := range order {
		rows, err := batches[name].svc.Create(ctx, batches[name].inputs)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}

// Dismiss mirrors dismiss.
func (l *Link) Dismiss(ctx context.Context, links []LinkDefinition) ([]*dal.Record, error) {
	var order []string
	byService := map[string][][2]any{}
	svcs := map[string]*services.LinkModuleService{}
	for _, link := range links {
		svc, c, err := l.getLinkModuleOrThrow(link, "dismiss")
		if err != nil {
			return nil, err
		}
		name := svc.Config.ServiceName
		if _, ok := byService[name]; !ok {
			order = append(order, name)
			svcs[name] = svc
		}
		pk, _ := sub(link, c.moduleA).Get(c.primaryKeys[0])
		fk, _ := sub(link, c.moduleB).Get(c.moduleBKey)
		byService[name] = append(byService[name], [2]any{pk, fk})
	}
	var out []*dal.Record
	for _, name := range order {
		rows, err := svcs[name].Dismiss(ctx, byService[name])
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}

// NewLinkDefinition builds a link definition { moduleA: { keyA: valueA },
// moduleB: { keyB: valueB } }.
func NewLinkDefinition(moduleA, keyA string, valueA any, moduleB, keyB string, valueB any) LinkDefinition {
	a := dal.NewRecord()
	a.Set(keyA, valueA)
	b := dal.NewRecord()
	b.Set(keyB, valueB)
	l := dal.NewRecord()
	l.Set(moduleA, a)
	l.Set(moduleB, b)
	return l
}

// GetLinkModule mirrors getLinkModule(moduleA, keyA, moduleB, keyB).
func (l *Link) GetLinkModule(moduleA, keyA, moduleB, keyB string) *services.LinkModuleService {
	return l.pairs[strings.Join([]string{moduleA, keyA, moduleB, keyB}, "-")]
}
