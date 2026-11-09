// Package loaders mirrors @medusajs/medusa/src/loaders.
package loaders

import (
	"github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/services"
)

// NewDefaultQuery builds Query for the default modules: the module services
// and the link module services, over the joiner configs and relation map
// upstream loads with the default configuration.
func NewDefaultQuery(db dal.Querier, moduleServices ...map[string]any) (*query.Query, error) {
	registered := map[string]any{}
	if len(moduleServices) > 0 && moduleServices[0] != nil {
		registered = moduleServices[0]
	}
	svcs := map[string]query.ModuleService{}
	for _, cfg := range modulessdk.DefaultJoinerConfigs {
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
				return nil, err
			}
			svcs[cfg.ServiceName] = query.LinkServiceAdapter{Service: link}
			continue
		}
		if s, ok := registered[cfg.ServiceName].(query.SuffixLister); ok {
			svcs[cfg.ServiceName] = query.MedusaServiceAdapter{Service: s}
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
	}
	return query.New(modulessdk.DefaultJoinerConfigs, joiner.RelationMap(modulessdk.DefaultRelationMap), svcs)
}
