// Package plugins mirrors the plugin resolution of a Medusa application
// (getResolvedPlugins, mergePluginModules and the loaders reading the
// plugin directories): a plugin is a Go package registering its
// Definition, imported by the application (or the stock binary).
package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Resolver resolves a registration of the application container.
type Resolver interface {
	Resolve(key string) any
}

// Module is a module of the plugin (src/modules/<Dir>).
type Module struct {
	Dir     string
	Exports *typesmodulessdk.ModuleExports
}

// Subscriber is a subscriber of the plugin (src/subscribers).
type Subscriber struct {
	Events  []string
	Handler func(ctx context.Context, c Resolver, ev eventbus.Event) error
}

// Definition is what a plugin package registers: the parts of its
// .medusa/server/src directory.
type Definition struct {
	// Name is the package name (e.g. "@medusajs/loyalty-plugin").
	Name    string
	Version string
	// Modules are the modules of src/modules, registered with the plugin
	// options (mergePluginModules).
	Modules []Module
	// Links defines the links of src/links with DefineLink, like importing
	// the files of the directory.
	Links func() error
	// Routes and Middlewares are the API routes of src/api.
	Routes      []medusahttp.RouteFile
	Middlewares []medusahttp.MiddlewareRoute
	// Subscribers are the subscribers of src/subscribers.
	Subscribers []Subscriber
	// Load does what importing src/workflows does (registering workflow
	// hook handlers); it runs once per process.
	Load func()

	loadOnce  sync.Once
	linksOnce sync.Once
	links     []utilsmodulessdk.RegisterLink
	linksErr  error
}

var (
	mu       sync.RWMutex
	registry = map[string]*Definition{}
)

// Register registers a plugin (from the init function of its package).
func Register(d *Definition) {
	mu.Lock()
	defer mu.Unlock()
	registry[d.Name] = d
}

// Lookup returns a registered plugin.
func Lookup(name string) *Definition {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// builtIn are the plugins whose parts are part of the Go core (the default
// draft-order plugin: its routes and workflows live with the core ones).
var builtIn = map[string]bool{"@medusajs/draft-order": true}

// Resolved mirrors getResolvedPlugins: the plugins of the configuration
// (built-in ones excepted), in configuration order. An unknown plugin
// fails like a plugin without a package.json.
func Resolved(cfg config.ConfigModule) ([]*Definition, error) {
	var out []*Definition
	for _, p := range cfg.Plugins {
		if builtIn[p.Resolve] {
			continue
		}
		d := Lookup(p.Resolve)
		if d == nil {
			return nil, fmt.Errorf("Unable to resolve plugin %q. Make sure the plugin directory has a package.json file", p.Resolve)
		}
		out = append(out, d)
	}
	return out, nil
}

// Apply prepares the configuration of an application for its plugins:
// mergePluginModules (the plugin modules, keyed by their service name and
// registered with the plugin options, come first; a module of the same key
// in the configuration replaces it), the plugin links (before the links of
// the project) and the workflow hooks of the plugins. It is idempotent.
func Apply(cfg *config.ConfigModule) error {
	resolved, err := Resolved(*cfg)
	if err != nil {
		return err
	}
	options := map[string]map[string]any{}
	for _, p := range cfg.Plugins {
		options[p.Resolve] = p.Options
	}
	var pluginModules []config.ModuleConfig
	var links []utilsmodulessdk.RegisterLink
	for _, d := range resolved {
		for _, m := range d.Modules {
			key := ""
			if m.Exports.JoinerConfig != nil {
				key = m.Exports.JoinerConfig.ServiceName
			}
			opts := options[d.Name]
			if opts == nil {
				opts = map[string]any{}
			}
			pluginModules = append(pluginModules, config.ModuleConfig{
				Key: key, Resolve: d.Name + "/.medusa/server/src/modules/" + m.Dir, Options: opts, Module: m.Exports,
			})
		}
		if d.Links != nil {
			d.linksOnce.Do(func() { d.links, d.linksErr = utilsmodulessdk.CaptureCustomLinks(d.Links) })
			if d.linksErr != nil {
				return d.linksErr
			}
			links = append(links, d.links...)
		}
		if d.Load != nil {
			d.loadOnce.Do(d.Load)
		}
	}
	utilsmodulessdk.SetPluginLinks(links)
	cfg.Modules = mergeModules(pluginModules, cfg.Modules)
	return nil
}

// mergeModules mirrors { ...pluginModules, ...configModules }: keys in
// first-seen order, values from the configuration.
func mergeModules(pluginModules, configModules []config.ModuleConfig) []config.ModuleConfig {
	configured := map[string]int{}
	for i, mc := range configModules {
		configured[mc.Key] = i
	}
	var out []config.ModuleConfig
	used := map[int]bool{}
	seen := map[string]bool{}
	for _, mc := range pluginModules {
		if seen[mc.Key] {
			continue
		}
		seen[mc.Key] = true
		if i, ok := configured[mc.Key]; ok {
			out = append(out, configModules[i])
			used[i] = true
			continue
		}
		out = append(out, mc)
	}
	for i, mc := range configModules {
		if !used[i] {
			out = append(out, mc)
		}
	}
	return out
}
