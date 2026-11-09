// Package plugins mirrors api/admin/plugins/route.ts.
package plugins

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: the names of the configured plugins.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	plugins := []any{}
	if cfg, ok := req.Scope.Resolve(common.ContainerRegistrationKeys.CONFIG_MODULE).(config.ConfigModule); ok {
		for _, p := range cfg.Plugins {
			r := dal.NewRecord()
			r.Set("name", p.Resolve)
			plugins = append(plugins, r)
		}
	}
	out := dal.NewRecord()
	out.Set("plugins", plugins)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
