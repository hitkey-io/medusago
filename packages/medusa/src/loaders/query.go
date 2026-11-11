// Package loaders mirrors @medusajs/medusa/src/loaders.
package loaders

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions/readonly"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/services"
)

// NewDefaultQuery builds Query for the default modules: the module services
// and the link module services, over the joiner configs and relation map
// built like upstream with the default configuration.
func NewDefaultQuery(db dal.Querier, moduleServices ...map[string]any) (*query.Query, error) {
	var registered map[string]any
	if len(moduleServices) > 0 {
		registered = moduleServices[0]
	}
	q, _, err := NewDefaultQueryAndLink(db, registered)
	return q, err
}

// NewDefaultQueryAndLink builds Query and Link (remoteLink) for the default
// configuration (with the translation module when it is registered).
func NewDefaultQueryAndLink(db dal.Querier, registered map[string]any) (*query.Query, *modulessdk.Link, error) {
	flags := map[string]any{}
	if _, ok := registered[utilsmodulessdk.TRANSLATION]; ok {
		flags["translation"] = true
	}
	q, link, _, err := NewQueryAndLink(db, config.DefineConfig(config.InputConfig{FeatureFlags: flags}), registered)
	return q, link, err
}

// CoreLinkDefinitions are the link definitions of @medusajs/link-modules in
// upstream export order.
func CoreLinkDefinitions() []*typesjoiner.ModuleJoinerConfig {
	return append(append([]*typesjoiner.ModuleJoinerConfig{}, definitions.All...), readonly.All...)
}

// LoadedModules returns the enabled modules of the configuration with the
// joiner config of their service.
func LoadedModules(cfg config.ConfigModule) []modulessdk.LoadedModule {
	var out []modulessdk.LoadedModule
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		lm := modulessdk.LoadedModule{Key: mc.Key}
		if exp, ok := modules.Registry[mc.Resolve]; ok {
			lm.JoinerConfig = exp.JoinerConfig
		}
		out = append(out, lm)
	}
	return out
}

// BuildQueryConfig builds the joiner configs, link modules and relation map
// of the configured modules and links (MedusaApp).
func BuildQueryConfig(cfg config.ConfigModule) (*modulessdk.QueryConfig, error) {
	return modulessdk.BuildQueryConfig(LoadedModules(cfg), CoreLinkDefinitions(), slog.Default())
}

// NewQueryAndLink builds Query and Link (remoteLink) over the modules and
// links of the configuration: registered are the loaded module services
// by key; the queryable modules missing from it get their generic service.
func NewQueryAndLink(db dal.Querier, cfg config.ConfigModule, registered map[string]any) (*query.Query, *modulessdk.Link, *modulessdk.QueryConfig, error) {
	if registered == nil {
		registered = map[string]any{}
	}
	qc, err := BuildQueryConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	modulessdk.SetAllJoinerConfigs(qc.JoinerConfigs)
	resolveOf := map[string]string{}
	for _, mc := range cfg.Modules {
		resolveOf[mc.Key] = mc.Resolve
	}

	linkServices := map[string]*services.LinkModuleService{}
	svcs := map[string]query.ModuleService{}
	moduleServices := map[string]any{}
	for _, l := range qc.Links {
		if l.Definition.IsReadOnlyLink {
			continue
		}
		// the definition (database config with extra fields) with the
		// extraDataFields of the service joiner config
		def := l.Definition.Clone()
		def.ExtraDataFields = l.JoinerConfig.ExtraDataFields
		link, err := services.NewLinkModuleService(db, def)
		if err != nil {
			return nil, nil, nil, err
		}
		linkServices[l.Definition.ServiceName] = link
		svcs[l.Definition.ServiceName] = query.LinkServiceAdapter{Service: link}
	}
	for _, c := range qc.JoinerConfigs {
		if c.IsLink || c.ServiceName == "" {
			continue
		}
		if s, ok := registered[c.ServiceName].(query.SuffixLister); ok {
			svcs[c.ServiceName] = query.MedusaServiceAdapter{Service: s}
			moduleServices[c.ServiceName] = s
			continue
		}
		exp, ok := modules.Registry[resolveOf[c.ServiceName]]
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
		svcs[c.ServiceName] = query.MedusaServiceAdapter{Service: svc}
		moduleServices[c.ServiceName] = svc
	}
	q, err := query.New(qc.JoinerConfigs, joiner.RelationMap(qc.RelationMap), svcs)
	if err != nil {
		return nil, nil, nil, err
	}
	if _, ok := registered[utilsmodulessdk.TRANSLATION]; ok {
		q.EnableTranslations()
	}
	return q, modulessdk.NewLink(qc.JoinerConfigs, linkServices, moduleServices), qc, nil
}
