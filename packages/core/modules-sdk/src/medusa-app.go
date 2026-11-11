package modulessdk

import (
	"log/slog"
	"sort"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/graphql"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/initialize"
)

// NonQueryableModules are the modules whose definition sets isQueryable to
// false (definitions.ts); every other module, custom modules included, is
// queryable.
var NonQueryableModules = map[string]bool{
	utilsmodulessdk.EVENT_BUS: true,
	utilsmodulessdk.CACHE:     true,
	utilsmodulessdk.CACHING:   true,
	utilsmodulessdk.INDEX:     true,
	utilsmodulessdk.SEARCH:    true,
	utilsmodulessdk.LOCKING:   true,
}

// moduleLoadOrder is the order in which upstream finishes loading the
// default modules (it loads them concurrently): the order of their joiner
// configs. Other modules follow in configuration order.
var moduleLoadOrder = []string{
	"stock_location", "inventory", "pricing", "promotion", "product", "customer", "sales_channel",
	"cart", "store", "api_key", "order", "settings", "user", "workflows", "auth", "file", "region",
	"currency", "payment", "tax", "notification", "fulfillment",
}

// LoadedModule is a loaded module: its key and the joiner config of its
// service (nil for a module without one).
type LoadedModule struct {
	Key          string
	JoinerConfig *joiner.ModuleJoinerConfig
}

// QueryConfig is what MedusaApp builds Query from.
type QueryConfig struct {
	// JoinerConfigs are the joiner configs of the queryable modules then
	// of the link modules.
	JoinerConfigs []*joiner.ModuleJoinerConfig
	// Links are the initialized link modules.
	Links []*initialize.Link
	// Schema is the merged GraphQL schema.
	Schema *graphql.Schema
	// RelationMap is extractRelationsFromGQL of the merged schema.
	RelationMap map[string]map[string]string
}

var (
	allJoinerConfigsMu sync.Mutex
	allJoinerConfigs   []*joiner.ModuleJoinerConfig
)

// SetAllJoinerConfigs records the joiner configs of the loaded modules and
// link modules (MedusaModule.setJoinerConfig).
func SetAllJoinerConfigs(configs []*joiner.ModuleJoinerConfig) {
	allJoinerConfigsMu.Lock()
	defer allJoinerConfigsMu.Unlock()
	allJoinerConfigs = append([]*joiner.ModuleJoinerConfig(nil), configs...)
}

// GetAllJoinerConfigs mirrors MedusaModule.getAllJoinerConfigs.
func GetAllJoinerConfigs() []*joiner.ModuleJoinerConfig {
	allJoinerConfigsMu.Lock()
	defer allJoinerConfigsMu.Unlock()
	return append([]*joiner.ModuleJoinerConfig(nil), allJoinerConfigs...)
}

// SortModulesByLoadOrder orders loaded modules like upstream registers
// them.
func SortModulesByLoadOrder(modules []LoadedModule) []LoadedModule {
	rank := func(key string) int {
		for i, k := range moduleLoadOrder {
			if k == key {
				return i
			}
		}
		return len(moduleLoadOrder)
	}
	out := append([]LoadedModule(nil), modules...)
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Key) < rank(out[j].Key) })
	return out
}

// BuildQueryConfig mirrors the part of MedusaApp that prepares Query: the
// joiner configs of the queryable modules, the custom links resolved with
// them, the link modules (link-modules initialize) and the merged schema
// with its relation map. linkDefinitions are the core link definitions.
func BuildQueryConfig(modules []LoadedModule, linkDefinitions []*joiner.ModuleJoinerConfig, logger *slog.Logger) (*QueryConfig, error) {
	if logger == nil {
		logger = slog.Default()
	}
	modules = SortModulesByLoadOrder(modules)
	var loadedKeys []string
	byService := map[string]*joiner.ModuleJoinerConfig{}
	out := &QueryConfig{}
	for _, m := range modules {
		loadedKeys = append(loadedKeys, m.Key)
		if NonQueryableModules[m.Key] || m.JoinerConfig == nil {
			continue
		}
		cfg := m.JoinerConfig.Clone()
		if cfg.ServiceName == "" {
			cfg.ServiceName = m.Key
		}
		if len(cfg.PrimaryKeys) == 0 {
			logger.Warn("Primary keys are not defined by the module " + m.Key + ". Setting default primary key to 'id'\n")
			cfg.PrimaryKeys = []string{"id"}
		}
		byService[m.Key] = cfg
		out.JoinerConfigs = append(out.JoinerConfigs, cfg)
	}

	custom, err := utilsmodulessdk.ResolveCustomLinks(out.JoinerConfigs)
	if err != nil {
		return nil, err
	}
	links, err := initialize.Initialize(append(append([]*joiner.ModuleJoinerConfig{}, linkDefinitions...), custom...), initialize.Options{
		LoadedModules: loadedKeys,
		Modules:       func(name string) *joiner.ModuleJoinerConfig { return byService[name] },
		Logger:        logger,
	})
	if err != nil {
		return nil, err
	}
	out.Links = links
	for _, l := range links {
		out.JoinerConfigs = append(out.JoinerConfigs, l.JoinerConfig)
	}

	var schemas []string
	for _, cfg := range out.JoinerConfigs {
		schemas = append(schemas, cfg.Schema)
	}
	schema, err := graphql.BuildSchema(schemas...)
	if err != nil {
		return nil, err
	}
	out.Schema = schema
	out.RelationMap = graphql.ExtractRelationsFromGQL(schema)
	return out, nil
}
