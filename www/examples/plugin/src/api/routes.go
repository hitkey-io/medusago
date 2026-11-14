// Package api holds the API routes and middlewares of the plugin.
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist"
)

// region: body
// listWishlists lists the wishlists of the authenticated customer.
func listWishlists(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	var customerID any
	if req.AuthContext != nil {
		customerID, _ = req.AuthContext.Get("actor_id")
	}
	if customerID == nil || customerID == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Customer authentication is required")
	}

	service := req.Scope.Resolve(wishlist.ModuleName).(*wishlist.WishlistModuleService)
	filters := dal.NewRecord()
	filters.Set("customer_id", customerID)

	wishlists, err := service.Call(req.Ctx(), "listWishlists", []any{filters})
	if err != nil {
		return err
	}

	out := dal.NewRecord()
	out.Set("wishlists", wishlists)
	return res.JSON(out)
}

// Routes are the API routes of the plugin, like the route files of src/api.
func Routes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{
		{Path: "/store/wishlists", Route: medusahttp.Route{GET: listWishlists}},
	}
}

// Middlewares are the middlewares of the plugin, like src/api/middlewares.
func Middlewares() []medusahttp.MiddlewareRoute {
	return nil
}

// endregion: body
