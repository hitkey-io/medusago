// Package modules mirrors api/admin/modules/route.ts.
package modules

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// upstreamRegistrationOrder is the order MedusaModule.getModuleKeys()
// returns for the default modules. Upstream registers a module when its
// (asynchronous) loading completes; this is the order observed with the
// default configuration. Other modules follow in configuration order.
var upstreamRegistrationOrder = []string{
	"cache", "event_bus", "locking", "search", "stock_location", "inventory", "product", "pricing",
	"promotion", "customer", "sales_channel", "cart", "api_key", "store", "settings", "order", "user",
	"workflows", "auth", "file", "currency", "region", "notification", "fulfillment", "tax", "payment",
	"translation", "index",
}

// ModuleKeys mirrors MedusaModule.getModuleKeys for the enabled modules.
func ModuleKeys(cfg config.ConfigModule) []string {
	enabled := map[string]bool{}
	var configOrder []string
	for _, m := range cfg.Modules {
		if m.Disable {
			continue
		}
		enabled[m.Key] = true
		configOrder = append(configOrder, m.Key)
	}
	var out []string
	seen := map[string]bool{}
	for _, k := range upstreamRegistrationOrder {
		if enabled[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	for _, k := range configOrder {
		if !seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cfg, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.CONFIG_MODULE).(config.ConfigModule)
	var mods []any
	for _, k := range ModuleKeys(cfg) {
		r := dal.NewRecord()
		r.Set("key", k)
		mods = append(mods, r)
	}
	if mods == nil {
		mods = []any{}
	}
	out := dal.NewRecord()
	out.Set("modules", mods)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
