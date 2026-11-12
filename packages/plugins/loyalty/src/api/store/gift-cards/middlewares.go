package giftcards

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// Middlewares mirrors storeGiftCardsMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/gift-cards/:code", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(StoreGetGiftCardParams, RetrieveGiftCardTransformQueryConfig),
	}},
}
