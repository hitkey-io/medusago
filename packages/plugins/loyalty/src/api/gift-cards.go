package api

import (
	"regexp"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	admingiftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/gift-cards"
	admingiftcard "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/gift-cards/id_"
	admingiftcardorders "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/gift-cards/id_/orders"
	storecarts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/carts"
	storecartgiftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/carts/id_/gift-cards"
	storecartstorecredits "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/carts/id_/store-credits"
	storegiftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/gift-cards"
	storegiftcard "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/gift-cards/idOrCode_"
)

// The gift card routes: admin/gift-cards, store/gift-cards and store/carts
// (both cart routes), with their middlewares.
var (
	giftCardRoutes = []medusahttp.RouteFile{
		{Path: "/admin/gift-cards", Route: admingiftcards.Route},
		{Path: "/admin/gift-cards/:id", Route: admingiftcard.Route},
		{Path: "/admin/gift-cards/:id/orders", Route: admingiftcardorders.Route},
		{Path: "/store/carts/:id/gift-cards", Route: storecartgiftcards.Route},
		{Path: "/store/carts/:id/store-credits", Route: storecartstorecredits.Route},
		{Path: "/store/gift-cards/:idOrCode", Route: storegiftcard.Route},
	}
	// allowGiftCardRelation mirrors allowGiftCardRelation: the gift card
	// relation of the store carts and orders routes (matcher
	// /^\/store\/(carts|orders)(?=\/|$)/; the lookahead is a group here).
	allowGiftCardRelation = []medusahttp.MiddlewareRoute{{
		MatcherRe:   regexp.MustCompile(`^/store/(carts|orders)(/|$)`),
		Middlewares: []medusahttp.MiddlewareFunction{middlewares.AllowFields(storegiftcards.GiftCardRelationAllowedFields)},
	}}
	adminGiftCardMiddlewares  = admingiftcards.Middlewares
	storeGiftCardsMiddlewares = storegiftcards.Middlewares
	storeCartMiddlewares      = storecarts.Middlewares
)
