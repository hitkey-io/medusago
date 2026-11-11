package modulessdk

import (
	"fmt"

	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// ModuleDefinition is what Module takes: the module models (the model
// objects of its MedusaService), its migrations and loaders, and optionally
// a service built over the generated MedusaService.
type ModuleDefinition struct {
	Models     []*dml.Entity
	Migrations []migrations.Migration
	Loaders    []typesmodulessdk.Loader
	// Service builds the module service from the generic MedusaService of
	// the models (nil: the MedusaService itself).
	Service func(opts typesmodulessdk.ServiceOptions, base *MedusaService) (any, error)
}

// DefinedModule is the result of Module: the module exports (to register
// in the configuration) and the linkable config of its models (what
// defineLink takes), keyed by lowerCaseFirst(model name).
type DefinedModule struct {
	typesmodulessdk.ModuleExports
	Linkable map[string]ModelLinkConfig
}

// LinkableOf returns the linkable config of a module (the `linkable`
// export of Module(), e.g. ProductModule.linkable.product): the linkable
// keys of its models, keyed by lowerCaseFirst(model name).
func LinkableOf(exp *typesmodulessdk.ModuleExports) map[string]ModelLinkConfig {
	out := map[string]ModelLinkConfig{}
	if exp == nil || exp.JoinerConfig == nil {
		return out
	}
	for _, c := range BuildLinkConfigFromModelObjects(exp.JoinerConfig.ServiceName, exp.Models, exp.JoinerConfig.LinkableKeys) {
		out[c.Field] = *c
	}
	return out
}

// Module mirrors Module(serviceName, { service, loaders }) for a module
// whose service is MedusaService(models): the model metadata is resolved,
// the joiner config is generated (defineJoinerConfig with the models and
// their id prefixes) and the linkable config is built.
func Module(serviceName string, def ModuleDefinition) *DefinedModule {
	models := def.Models
	if err := dml.ResolveMeta(models); err != nil {
		panic(fmt.Sprintf("module %s: %v", serviceName, err))
	}
	joinerConfig := DefineJoinerConfig(serviceName, DefineJoinerConfigOptions{Models: models})
	joinerConfig.IDPrefixToEntityName = BuildIDPrefixToEntityNameFromDmlObjects(models)

	linkable := map[string]ModelLinkConfig{}
	for _, c := range BuildLinkConfigFromModelObjects(serviceName, models, joinerConfig.LinkableKeys) {
		linkable[c.Field] = *c
	}
	build := def.Service
	out := &DefinedModule{
		ModuleExports: typesmodulessdk.ModuleExports{
			Migrations:   def.Migrations,
			Models:       models,
			JoinerConfig: joinerConfig,
			Loaders:      def.Loaders,
			Service: func(opts typesmodulessdk.ServiceOptions) (any, error) {
				base := NewMedusaService(opts.DB, models, joinerConfig)
				if build == nil {
					return base, nil
				}
				return build(opts, base)
			},
		},
		Linkable: linkable,
	}
	return out
}
