// Package api mirrors @medusajs/medusa/src/api: the route files and the
// middlewares of defineMiddlewares, in upstream order.
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth"
)

// Middlewares mirrors the default export of middlewares.ts.
func Middlewares() []medusahttp.MiddlewareRoute {
	var out []medusahttp.MiddlewareRoute
	// upstream order: ...auth, ..., regions, users, ..., stores, ..., sales channels
	out = append(out, auth.AuthRoutesMiddlewares...)
	out = append(out, regions.AdminRegionRoutesMiddlewares...)
	out = append(out, users.AdminUserRoutesMiddlewares...)
	out = append(out, stores.AdminStoreRoutesMiddlewares...)
	out = append(out, saleschannels.AdminSalesChannelRoutesMiddlewares...)
	return out
}
