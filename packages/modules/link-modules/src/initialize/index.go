// Package initialize mirrors @medusajs/link-modules/src/initialize: the
// link modules are built from the link definitions and the loaded modules.
package initialize

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/link"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/utils"
)

// Options are what initialize needs from the module runtime.
type Options struct {
	// LoadedModules are the keys of the loaded modules
	// (MedusaModule.getLoadedModules()).
	LoadedModules []string
	// Modules resolves the joiner config of a loaded module.
	Modules utils.ModuleConfigs
	Logger  *slog.Logger
}

// Link is an initialized link module: its key, its definition (with its
// schema and alias entities) and the joiner config its service exposes.
type Link struct {
	Key          string
	Definition   *joiner.ModuleJoinerConfig
	JoinerConfig *joiner.ModuleJoinerConfig
	Primary      *joiner.JoinerRelationship
	Foreign      *joiner.JoinerRelationship
}

// serviceJoinerConfig mirrors getModuleService (dynamic-service-class.ts):
// the database config is reduced to the table name and extraDataFields
// defaults to the extra field names.
func serviceJoinerConfig(definition *joiner.ModuleJoinerConfig) *joiner.ModuleJoinerConfig {
	if definition.IsReadOnlyLink {
		return definition
	}
	cfg := definition.Clone()
	db := cfg.DatabaseConfig
	cfg.DatabaseConfig = nil
	if db != nil && db.TableName != "" {
		cfg.DatabaseConfig = &joiner.DatabaseConfig{TableName: db.TableName}
	}
	if cfg.ExtraDataFields == nil {
		cfg.ExtraDataFields = []string{}
		if db != nil {
			for _, f := range db.ExtraFields {
				cfg.ExtraDataFields = append(cfg.ExtraDataFields, f.Name)
			}
		}
	}
	return cfg
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Initialize mirrors initialize: the link definitions (the core ones then
// the application ones) whose modules are loaded become link modules. The
// read-only links come first, like the order in which upstream finishes
// loading them.
func Initialize(definitions []*joiner.ModuleJoinerConfig, opts Options) ([]*Link, error) {
	var readOnly, tables []*Link
	keys := map[string]bool{}
	for _, def := range definitions {
		definition := def.Clone()
		var primary, foreign *joiner.JoinerRelationship
		if len(definition.Relationships) > 0 {
			primary = &definition.Relationships[0]
		}
		if len(definition.Relationships) > 1 {
			foreign = &definition.Relationships[1]
		}
		if len(definition.Relationships) != 2 && !definition.IsReadOnlyLink {
			return nil, fmt.Errorf("Link module %s can only link 2 modules.", definition.ServiceName)
		} else if foreign != nil && len(strings.Split(foreign.ForeignKey, ",")) > 1 && !definition.IsReadOnlyLink {
			return nil, fmt.Errorf("Foreign key cannot be a composed key.")
		}
		if definition.ExtraDataFields != nil {
			var defined []string
			if definition.DatabaseConfig != nil {
				for _, f := range definition.DatabaseConfig.ExtraFields {
					defined = append(defined, f.Name)
				}
			}
			var missing []string
			for _, f := range definition.ExtraDataFields {
				if !contains(defined, f) {
					missing = append(missing, f)
				}
			}
			if len(missing) > 0 {
				return nil, fmt.Errorf("extraDataFields (fieldNames: %s) need to be configured under databaseConfig (serviceName: %s).", strings.Join(missing, ","), definition.ServiceName)
			}
		}

		var serviceKey string
		if !definition.IsReadOnlyLink {
			serviceKey = definition.ServiceName
			if serviceKey == "" {
				serviceKey = link.ComposeLinkName(primary.ServiceName, primary.ForeignKey, foreign.ServiceName, foreign.ForeignKey)
			}
		} else {
			b, _ := json.Marshal(definition.Extends)
			serviceKey = common.SimpleHash(string(b))
		}
		if contains(opts.LoadedModules, serviceKey) {
			continue
		} else if keys[serviceKey] {
			return nil, fmt.Errorf("Link module %s already defined.", serviceKey)
		}

		if definition.IsReadOnlyLink {
			var extended []joiner.JoinerExtension
			for _, ext := range definition.Extends {
				if contains(opts.LoadedModules, ext.ServiceName) && contains(opts.LoadedModules, ext.Relationship.ServiceName) {
					extended = append(extended, ext)
				}
			}
			definition.Extends = extended
			if len(extended) == 0 {
				continue
			}
		} else if !contains(opts.LoadedModules, primary.ServiceName) || !contains(opts.LoadedModules, foreign.ServiceName) {
			continue
		}

		schema, err := utils.GenerateGraphQLSchema(definition, primary, foreign, opts.Modules, opts.Logger)
		if err != nil {
			return nil, err
		}
		definition.Schema = schema
		if definition.Alias == nil {
			definition.Alias = []joiner.JoinerServiceConfigAlias{}
		}
		for i := range definition.Alias {
			if definition.Alias[i].Args == nil {
				definition.Alias[i].Args = map[string]any{}
			}
			definition.Alias[i].Entity = utils.LinkEntityName(definition, *primary, *foreign)
		}
		keys[serviceKey] = true
		l := &Link{Key: serviceKey, Definition: definition, JoinerConfig: serviceJoinerConfig(definition), Primary: primary, Foreign: foreign}
		if definition.IsReadOnlyLink {
			readOnly = append(readOnly, l)
		} else {
			tables = append(tables, l)
		}
	}
	return append(readOnly, tables...), nil
}
