// Package loyaltyplugin mirrors @medusajs/loyalty-plugin (gift cards and
// store credit). Importing it registers the plugin: an application enables
// it with { resolve: "@medusajs/loyalty-plugin" } in its plugins.
package loyaltyplugin

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/links"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/subscribers"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/hooks"
)

// Name is the package name of the plugin.
const Name = "@medusajs/loyalty-plugin"

// Plugin is the plugin definition.
var Plugin = &plugins.Definition{
	Name:    Name,
	Version: "2.22.6",
	// src/modules is read in directory order
	Modules: []plugins.Module{
		{Dir: "loyalty", Exports: &loyalty.Module.ModuleExports},
		{Dir: "store-credit", Exports: &storecredit.Module.ModuleExports},
	},
	Links:       links.Define,
	Routes:      api.Routes(),
	Middlewares: api.Middlewares(),
	Subscribers: subscribers.All(),
	Load:        hooks.Register,
}

func init() { plugins.Register(Plugin) }
