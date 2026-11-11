// Package loaders mirrors @medusajs/medusa/src/loaders.
package loaders

import (
	"github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/services"
)

// NewDefaultQuery builds Query for the default modules: the module services
// and the link module services, over the joiner configs and relation map
// upstream loads with the default configuration.
func NewDefaultQuery(db dal.Querier, moduleServices ...map[string]any) (*query.Query, error) {
	var registered map[string]any
	if len(moduleServices) > 0 {
		registered = moduleServices[0]
	}
	q, _, err := NewDefaultQueryAndLink(db, registered)
	return q, err
}

// NewDefaultQueryAndLink builds Query and Link (remoteLink) over the same
// link module services.
func NewDefaultQueryAndLink(db dal.Querier, registered map[string]any) (*query.Query, *modulessdk.Link, error) {
	if registered == nil {
		registered = map[string]any{}
	}
	linkServices := map[string]*services.LinkModuleService{}
	svcs := map[string]query.ModuleService{}
	moduleServices := map[string]any{}
	configs, relationMap := queryConfigs(registered)
	for _, cfg := range configs {
		if cfg.IsReadOnlyLink || cfg.ServiceName == "" {
			continue
		}
		if cfg.IsLink {
			// The runtime joiner config keeps only the table name of the
			// database config; the link definition has the extra columns.
			linkCfg := cfg
			for _, def := range definitions.All {
				if def.ServiceName == cfg.ServiceName {
					c := *cfg
					c.DatabaseConfig = def.DatabaseConfig
					linkCfg = &c
					break
				}
			}
			link, err := services.NewLinkModuleService(db, linkCfg)
			if err != nil {
				return nil, nil, err
			}
			if bus, ok := registered[utilsmodulessdk.EVENT_BUS].(eventbus.Emitter); ok {
				link.SetEventBus(bus)
			}
			linkServices[cfg.ServiceName] = link
			svcs[cfg.ServiceName] = query.LinkServiceAdapter{Service: link}
			continue
		}
		if s, ok := registered[cfg.ServiceName].(query.SuffixLister); ok {
			svcs[cfg.ServiceName] = query.MedusaServiceAdapter{Service: s}
			moduleServices[cfg.ServiceName] = s
			continue
		}
		exp, ok := modules.Registry[utilsmodulessdk.MODULE_PACKAGE_NAMES[cfg.ServiceName]]
		if !ok || len(exp.Models) == 0 {
			continue
		}
		var svc query.SuffixLister = utilsmodulessdk.NewMedusaService(db, exp.Models, exp.JoinerConfig)
		if exp.Service != nil {
			if built, err := exp.Service(typesmodulessdk.ServiceOptions{DB: db}); err == nil {
				if s, ok := built.(query.SuffixLister); ok {
					svc = s
				}
			}
		}
		svcs[cfg.ServiceName] = query.MedusaServiceAdapter{Service: svc}
		moduleServices[cfg.ServiceName] = svc
	}
	q, err := query.New(configs, relationMap, svcs)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := registered[utilsmodulessdk.TRANSLATION]; ok {
		q.EnableTranslations()
	}
	return q, modulessdk.NewLink(configs, linkServices, moduleServices), nil
}

// queryConfigs returns the joiner configs and relation map of the loaded
// modules: the default configuration plus what the feature flagged modules
// add when they are registered (translation).
func queryConfigs(registered map[string]any) ([]*typesjoiner.ModuleJoinerConfig, joiner.RelationMap) {
	if _, ok := registered[utilsmodulessdk.TRANSLATION]; !ok {
		return modulessdk.DefaultJoinerConfigs, joiner.RelationMap(modulessdk.DefaultRelationMap)
	}
	configs := append(append([]*typesjoiner.ModuleJoinerConfig{}, modulessdk.DefaultJoinerConfigs...), modulessdk.TranslationJoinerConfigs...)
	relations := joiner.RelationMap{}
	for entity, props := range modulessdk.DefaultRelationMap {
		relations[entity] = map[string]string{}
		for k, v := range props {
			relations[entity][k] = v
		}
	}
	for entity, props := range modulessdk.TranslationRelationMap {
		if relations[entity] == nil {
			relations[entity] = map[string]string{}
		}
		for k, v := range props {
			relations[entity][k] = v
		}
	}
	return configs, relations
}
