// Package wishlist is a plugin that adds wishlists to a MedusaGo
// application: a module, its link to customers, an API route, a
// subscriber, a workflow hook and a scheduled job.
package wishlist

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/api"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/jobs"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/links"
	wishlistmodule "github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/subscribers"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/workflows/hooks"
)

// region: definition
// Name is the name of the plugin: the `Resolve` value of its entry in the
// `Plugins` of the application's configuration.
const Name = "github.com/acme/medusa-plugin-wishlist"

// Plugin is the plugin's definition.
var Plugin = &plugins.Definition{
	Name:    Name,
	Version: "1.0.0",
	// the modules of src/modules
	Modules: []plugins.Module{
		{Dir: "wishlist", Exports: &wishlistmodule.Module.ModuleExports},
	},
	// the links of src/links
	Links: links.Define,
	// the API routes and middlewares of src/api
	Routes:      api.Routes(),
	Middlewares: api.Middlewares(),
	// the subscribers of src/subscribers
	Subscribers: subscribers.All(),
	// runs once, when the plugin is loaded: consume workflow hooks and
	// register scheduled jobs here
	Load: func() {
		hooks.Register()
		jobs.Register()
	},
}

// endregion: definition

// region: register
func init() {
	// importing the package registers the plugin
	plugins.Register(Plugin)
}

// endregion: register
