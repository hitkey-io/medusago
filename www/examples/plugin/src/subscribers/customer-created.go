// Package subscribers holds the subscribers of the plugin.
package subscribers

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist"
)

// region: body
// All are the subscribers of the plugin.
func All() []plugins.Subscriber {
	return []plugins.Subscriber{customerCreated}
}

// customerCreated creates a wishlist for every new customer.
var customerCreated = plugins.Subscriber{
	Events: []string{"customer.created"},
	Handler: func(ctx context.Context, c plugins.Resolver, event eventbus.Event) error {
		data, _ := event.Data.(*dal.Record)
		if data == nil {
			return nil
		}
		customerID, _ := data.Get("id")

		service := c.Resolve(wishlist.ModuleName).(*wishlist.WishlistModuleService)
		wishlistData := dal.NewRecord()
		wishlistData.Set("customer_id", customerID)

		_, err := service.Call(ctx, "createWishlists", []any{wishlistData})
		return err
	},
}

// endregion: body
