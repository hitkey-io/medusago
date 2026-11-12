package modulessdk

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/link"
)

// LinkInput is a side of defineLink: a linkable (InputSource, or a model
// link config through its ToJSON) or DefineLinkInput (InputOptions).
type LinkInput interface {
	linkConfig() (moduleLinkableKeyConfig, error)
}

// DefineLinkInput mirrors InputOptions: a linkable with link options.
type DefineLinkInput struct {
	Linkable      Linkable
	Field         string
	Alias         string
	IsList        bool
	DeleteCascade bool
	Filterable    []string
}

// Shortcut mirrors the shortcut option of read-only links.
type Shortcut struct {
	Property         string
	Path             string
	IsList           bool
	ForwardArguments []string
}

// LinkDatabaseOptions mirrors ExtraOptions.database.
type LinkDatabaseOptions struct {
	Table        string
	IDPrefix     string
	ExtraColumns joiner.ExtraFields
}

// DefineLinkOptions mirrors ExtraOptions and ReadOnlyExtraOptions.
type DefineLinkOptions struct {
	PK       map[string]string
	Database *LinkDatabaseOptions
	ReadOnly bool
	// IsList and Shortcut are read-only link options.
	IsList   bool
	Shortcut []Shortcut
}

type moduleLinkableKeyConfig struct {
	module        string
	entity        string
	key           string
	field         string
	isList        bool
	deleteCascade bool
	primaryKey    string
	alias         string
	hasMany       bool
	filterable    []string
}

func (l Linkable) linkConfig() (moduleLinkableKeyConfig, error) {
	alias := l.Alias
	if alias == "" {
		alias = common.CamelToSnakeCase(l.Field)
	}
	return moduleLinkableKeyConfig{
		key:        l.Linkable,
		alias:      alias,
		field:      l.Field,
		primaryKey: l.PrimaryKey,
		filterable: l.Filterable,
		module:     l.ServiceName,
		entity:     l.Entity,
	}, nil
}

func (c ModelLinkConfig) linkConfig() (moduleLinkableKeyConfig, error) {
	return c.ToJSON().linkConfig()
}

func (in DefineLinkInput) linkConfig() (moduleLinkableKeyConfig, error) {
	src := in.Linkable
	if src.ServiceName == "" {
		b, _ := json.MarshalIndent(in, "", "  ")
		return moduleLinkableKeyConfig{}, fmt.Errorf("Invalid linkable passed for the argument\n%s", b)
	}
	alias := in.Alias
	if alias == "" {
		alias = src.Alias
	}
	if alias == "" {
		alias = common.CamelToSnakeCase(src.Field)
	}
	field := in.Field
	if field == "" {
		field = src.Field
	}
	return moduleLinkableKeyConfig{
		key:           src.Linkable,
		alias:         alias,
		field:         field,
		primaryKey:    src.PrimaryKey,
		isList:        in.IsList,
		hasMany:       in.IsList,
		deleteCascade: in.DeleteCascade,
		filterable:    in.Filterable,
		module:        src.ServiceName,
		entity:        src.Entity,
	}, nil
}

// DefineLinkExport mirrors the object defineLink returns; its fields are
// set once the link is registered with the loaded modules.
type DefineLinkExport struct {
	ServiceName string
	Entity      string
	EntryPoint  string
}

// RegisterLink mirrors a custom link registration function: it builds the
// link definition from the joiner configs of the loaded modules.
type RegisterLink func(modules []*joiner.ModuleJoinerConfig) (*joiner.ModuleJoinerConfig, error)

var (
	customLinksMu sync.Mutex
	customLinks   []RegisterLink
	// pluginLinks are the links of the plugins of the running application
	// (SetPluginLinks); capturing collects the links a plugin defines.
	pluginLinks []RegisterLink
	capturing   *[]RegisterLink
	captureMu   sync.Mutex
)

// SetCustomLink mirrors MedusaModule.setCustomLink.
func SetCustomLink(fn RegisterLink) {
	customLinksMu.Lock()
	defer customLinksMu.Unlock()
	if capturing != nil {
		*capturing = append(*capturing, fn)
		return
	}
	customLinks = append(customLinks, fn)
}

// CaptureCustomLinks runs define (the links of a plugin, defined with
// DefineLink) and returns the links it defines instead of registering them.
func CaptureCustomLinks(define func() error) ([]RegisterLink, error) {
	captureMu.Lock()
	defer captureMu.Unlock()
	var out []RegisterLink
	customLinksMu.Lock()
	capturing = &out
	customLinksMu.Unlock()
	defer func() {
		customLinksMu.Lock()
		capturing = nil
		customLinksMu.Unlock()
	}()
	err := define()
	return out, err
}

// SetPluginLinks sets the links of the plugins of the application (loaded
// before the links of the project, like the plugin links directories).
func SetPluginLinks(links []RegisterLink) {
	customLinksMu.Lock()
	defer customLinksMu.Unlock()
	pluginLinks = append([]RegisterLink(nil), links...)
}

// GetCustomLinks mirrors MedusaModule.getCustomLinks: the links of the
// plugins, then those of the project.
func GetCustomLinks() []RegisterLink {
	customLinksMu.Lock()
	defer customLinksMu.Unlock()
	return append(append([]RegisterLink(nil), pluginLinks...), customLinks...)
}

// ClearCustomLinks forgets the registered custom links.
func ClearCustomLinks() {
	customLinksMu.Lock()
	defer customLinksMu.Unlock()
	customLinks = nil
}

// ResolveCustomLinks calls the registered custom links with the joiner
// configs of the loaded modules, like MedusaApp.
func ResolveCustomLinks(modules []*joiner.ModuleJoinerConfig) ([]*joiner.ModuleJoinerConfig, error) {
	var out []*joiner.ModuleJoinerConfig
	for _, fn := range GetCustomLinks() {
		def, err := fn(modules)
		if err != nil {
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}

func buildFieldAlias(shortcuts []Shortcut) map[string]joiner.FieldAlias {
	if len(shortcuts) == 0 {
		return nil
	}
	out := map[string]joiner.FieldAlias{}
	for _, sc := range shortcuts {
		out[sc.Property] = joiner.FieldAlias{Path: sc.Path, IsList: sc.IsList, ForwardArgumentsOnPath: append([]string{}, sc.ForwardArguments...)}
	}
	return out
}

// shortcutProperties returns the shortcut properties in declaration order
// (the key order of the fieldAlias object built from them).
func shortcutProperties(shortcuts []Shortcut) []string {
	var out []string
	for _, sc := range shortcuts {
		out = append(out, sc.Property)
	}
	return out
}

func findModule(modules []*joiner.ModuleJoinerConfig, name string) *joiner.ModuleJoinerConfig {
	for _, m := range modules {
		if m.ServiceName == name {
			return m
		}
	}
	return nil
}

func moduleNotFound(name string) error {
	return fmt.Errorf(`Service %s was not found. If this is your module, make sure you set isQueryable to true in medusa-config.js:

%s: {
  // ...
  definition: {
    isQueryable: true
  }
}`, name, name)
}

func linkableKeyEntity(cfg *joiner.ModuleJoinerConfig, key string) string {
	for _, kv := range cfg.LinkableKeys {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

func aliasMethodSuffix(aliases []joiner.JoinerServiceConfigAlias, entryPoint string) any {
	for _, a := range aliases {
		for _, n := range a.Name {
			if n == entryPoint {
				return a.Args["methodSuffix"]
			}
		}
	}
	return nil
}

func aliasFor(cfg *joiner.ModuleJoinerConfig, explicit, entity string) string {
	if explicit != "" {
		return explicit
	}
	for _, a := range cfg.Alias {
		if a.Entity == entity && len(a.Name) > 0 {
			return a.Name[0]
		}
	}
	return ""
}

func linkPrimaryKey(side moduleLinkableKeyConfig, opts DefineLinkOptions, cfg *joiner.ModuleJoinerConfig) (string, error) {
	pk := side.primaryKey
	if pk == "" {
		pk = opts.PK[side.module]
	}
	if pk == "" && len(cfg.PrimaryKeys) > 0 {
		pk = cfg.PrimaryKeys[0]
	}
	for _, x := range cfg.PrimaryKeys {
		if x == pk {
			return pk, nil
		}
	}
	return "", fmt.Errorf("Primary key %s is not defined on service %s", pk, side.module)
}

func argsWithSuffix(suffix any) map[string]any {
	if suffix == nil {
		return map[string]any{}
	}
	return map[string]any{"methodSuffix": suffix}
}

// DefineLink mirrors defineLink: it registers a link between two module
// data models (a table link) or, with ReadOnly, a read-only link.
func DefineLink(left, right LinkInput, options ...DefineLinkOptions) (*DefineLinkExport, error) {
	var opts DefineLinkOptions
	if len(options) > 0 {
		opts = options[0]
	}
	a, err := left.linkConfig()
	if err != nil {
		return nil, err
	}
	b, err := right.linkConfig()
	if err != nil {
		return nil, err
	}
	if opts.ReadOnly {
		in, isOptions := left.(DefineLinkInput)
		if !isOptions || in.Field == "" {
			return nil, fmt.Errorf(`ReadOnly link requires "linkable" and "field" to be defined for the left service.`)
		}
		if len(a.filterable) > 0 || len(b.filterable) > 0 {
			return nil, fmt.Errorf(`ReadOnly link does not support filterable fields.`)
		}
		defineReadOnlyLink(a, b, opts)
		return nil, nil
	}

	out := &DefineLinkExport{}
	SetCustomLink(func(modules []*joiner.ModuleJoinerConfig) (*joiner.ModuleJoinerConfig, error) {
		infoA := findModule(modules, a.module)
		if infoA == nil {
			return nil, moduleNotFound(a.module)
		}
		infoB := findModule(modules, b.module)
		if infoB == nil {
			return nil, moduleNotFound(b.module)
		}
		entityA := linkableKeyEntity(infoA, a.key)
		entityB := linkableKeyEntity(infoB, b.key)
		if entityA == "" {
			return nil, fmt.Errorf("Key %s is not linkable on service %s", a.key, a.module)
		}
		if entityB == "" {
			return nil, fmt.Errorf("Key %s is not linkable on service %s", b.key, b.module)
		}
		aliasA := aliasFor(infoA, a.alias, entityA)
		if aliasA == "" {
			return nil, fmt.Errorf("You need to provide an alias for %s.%s", a.module, a.key)
		}
		suffixA := aliasMethodSuffix(infoA.Alias, common.CamelToSnakeCase(a.field))
		aliasB := aliasFor(infoB, b.alias, entityB)
		if aliasB == "" {
			return nil, fmt.Errorf("You need to provide an alias for %s.%s", b.module, b.key)
		}
		suffixB := aliasMethodSuffix(infoB.Alias, common.CamelToSnakeCase(b.field))
		pkA, err := linkPrimaryKey(a, opts, infoA)
		if err != nil {
			return nil, err
		}
		pkB, err := linkPrimaryKey(b, opts, infoB)
		if err != nil {
			return nil, err
		}

		out.ServiceName = link.ComposeLinkName(a.module, aliasA, b.module, aliasB)
		out.EntryPoint = aliasA + "_" + aliasB
		out.Entity = common.ToPascalCase("Link_" + a.module + "_" + aliasA + "_" + b.module + "_" + aliasB)

		propB := aliasB
		if b.isList {
			propB = common.Pluralize(aliasB)
		}
		propA := aliasA
		if a.isList {
			propA = common.Pluralize(aliasA)
		}
		def := &joiner.ModuleJoinerConfig{
			ServiceName: out.ServiceName,
			IsLink:      true,
			Alias: []joiner.JoinerServiceConfigAlias{{
				Name: joiner.StringOrStrings{out.EntryPoint},
				Args: map[string]any{"entity": out.Entity},
			}},
			PrimaryKeys: []string{"id", a.key, b.key},
			Relationships: []joiner.JoinerRelationship{
				{ServiceName: a.module, Entity: a.entity, PrimaryKey: pkA, ForeignKey: a.key, Alias: aliasA, Args: argsWithSuffix(suffixA), DeleteCascade: a.deleteCascade, Filterable: a.filterable, HasMany: a.hasMany},
				{ServiceName: b.module, Entity: b.entity, PrimaryKey: pkB, ForeignKey: b.key, Alias: aliasB, Args: argsWithSuffix(suffixB), DeleteCascade: b.deleteCascade, Filterable: b.filterable, HasMany: b.hasMany},
			},
			Extends: []joiner.JoinerExtension{
				{
					ServiceName: a.module,
					Entity:      a.entity,
					FieldAlias:  buildFieldAlias([]Shortcut{{Property: propB, Path: aliasB + "_link." + aliasB, IsList: b.isList, ForwardArguments: []string{aliasB + "_link." + aliasB}}}),
					Relationship: joiner.JoinerRelationship{
						ServiceName: out.ServiceName, Entity: out.Entity, PrimaryKey: a.key, ForeignKey: pkA, Alias: aliasB + "_link", IsList: b.isList,
					},
				},
				{
					ServiceName: b.module,
					Entity:      b.entity,
					FieldAlias:  buildFieldAlias([]Shortcut{{Property: propA, Path: aliasA + "_link." + aliasA, IsList: a.isList, ForwardArguments: []string{aliasA + "_link." + aliasA}}}),
					Relationship: joiner.JoinerRelationship{
						ServiceName: out.ServiceName, Entity: out.Entity, PrimaryKey: b.key, ForeignKey: pkB, Alias: aliasA + "_link", IsList: a.isList,
					},
				},
			},
		}
		if db := opts.Database; db != nil {
			def.DatabaseConfig = &joiner.DatabaseConfig{TableName: db.Table, IDPrefix: db.IDPrefix, ExtraFields: db.ExtraColumns}
		}
		return def, nil
	})
	return out, nil
}

func defineReadOnlyLink(a, b moduleLinkableKeyConfig, opts DefineLinkOptions) {
	SetCustomLink(func(modules []*joiner.ModuleJoinerConfig) (*joiner.ModuleJoinerConfig, error) {
		if findModule(modules, a.module) == nil {
			return nil, moduleNotFound(a.module)
		}
		if findModule(modules, b.module) == nil {
			return nil, moduleNotFound(b.module)
		}
		isList := opts.IsList || a.isList
		ext := []joiner.JoinerExtension{{
			ServiceName:     a.module,
			Entity:          a.entity,
			FieldAlias:      buildFieldAlias(opts.Shortcut),
			FieldAliasOrder: shortcutProperties(opts.Shortcut),
			Relationship: joiner.JoinerRelationship{
				ServiceName: b.module, Entity: b.entity, PrimaryKey: b.primaryKey, ForeignKey: a.field, Alias: b.alias, IsList: isList,
			},
		}}
		if isList {
			alias := b.alias
			if isList {
				alias = common.Pluralize(b.alias)
			}
			ext = append(ext, joiner.JoinerExtension{
				ServiceName:     a.module,
				Entity:          a.entity,
				FieldAlias:      buildFieldAlias(opts.Shortcut),
				FieldAliasOrder: shortcutProperties(opts.Shortcut),
				Relationship: joiner.JoinerRelationship{
					ServiceName: b.module, Entity: b.entity, PrimaryKey: b.primaryKey, ForeignKey: a.field, Alias: alias, IsList: isList,
				},
			})
		}
		return &joiner.ModuleJoinerConfig{IsLink: true, IsReadOnlyLink: true, Extends: ext}, nil
	})
}
