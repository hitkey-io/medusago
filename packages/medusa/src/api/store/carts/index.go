package carts

import (
	"fmt"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()).Nullish() }

var itemSchema = zod.Object(
	zod.F("variant_id", zod.String()),
	zod.F("quantity", zod.Number().Gt(0)),
	zod.F("metadata", metadata()),
)

// Validators.
var (
	StoreGetCartsCart = utils.CreateSelectParams()
	CreateCart        = zod.Object(
		zod.F("region_id", zod.String().Nullish()),
		zod.F("shipping_address", zod.Union(commonvalidators.AddressPayload(), zod.String()).Optional()),
		zod.F("billing_address", zod.Union(commonvalidators.AddressPayload(), zod.String()).Optional()),
		zod.F("email", zod.String().Email().Nullish()),
		zod.F("currency_code", zod.String().Nullish()),
		zod.F("items", zod.Array(itemSchema).Optional()),
		zod.F("sales_channel_id", zod.String().Nullish()),
		zod.F("promo_codes", zod.Array(zod.String()).Optional()),
		zod.F("metadata", metadata()),
		zod.F("locale", zod.String().Nullish()),
	).Strict()
	StoreCreateCart        = utils.WithAdditionalData(CreateCart)
	storeCartUpsertAddress = commonvalidators.AddressPayload().Merge(zod.Object(zod.F("id", zod.String().Optional())))
	// UpdateCart mirrors UpdateCart.
	UpdateCart = zod.Object(
		zod.F("region_id", zod.String().Optional()),
		zod.F("email", zod.String().Email().Nullish()),
		zod.F("billing_address", storeCartUpsertAddress.Optional()),
		zod.F("shipping_address", storeCartUpsertAddress.Optional()),
		zod.F("sales_channel_id", zod.String().Nullish()),
		zod.F("metadata", metadata()),
		zod.F("promo_codes", zod.Array(zod.String()).Optional()),
		zod.F("locale", zod.String().Nullish()),
	).Strict()
	// StoreUpdateCart mirrors StoreUpdateCart.
	StoreUpdateCart = utils.WithAdditionalData(UpdateCart)
	// StoreCartPromotions mirrors StoreAddCartPromotions / StoreRemoveCartPromotions.
	StoreCartPromotions  = zod.Object(zod.F("promo_codes", zod.Array(zod.String()))).Strict()
	shippingMethodSchema = zod.Object(
		zod.F("option_id", zod.String()),
		zod.F("data", zod.Record(zod.String(), zod.Unknown()).Optional()),
	)
	// StoreAddCartShippingMethods mirrors StoreAddCartShippingMethods.
	StoreAddCartShippingMethods = zod.Union(shippingMethodSchema, zod.Array(shippingMethodSchema))
	// StoreAddCartLineItem mirrors StoreAddCartLineItem.
	StoreAddCartLineItem = zod.Object(
		zod.F("variant_id", zod.String()),
		zod.F("quantity", zod.Number().Gt(0)),
		zod.F("metadata", metadata()),
	)
	// StoreUpdateCartLineItem mirrors StoreUpdateCartLineItem.
	StoreUpdateCartLineItem = zod.Object(
		zod.F("quantity", zod.Number().Gte(0)),
		zod.F("metadata", metadata()),
	)
)

// MaybeAttachPublishableKeyScopes mirrors maybeAttachPublishableKeyScopes.
func MaybeAttachPublishableKeyScopes(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	if key := req.Header.Get("x-publishable-api-key"); key != "" {
		f := dal.NewRecord()
		f.Set("token", key)
		v := dal.NewRecord()
		v.Set("filters", f)
		r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "api_key", v, []string{"sales_channels.id"})
		if err != nil {
			return err
		}
		ids := []any{}
		if len(r.Data) > 0 {
			if k, ok := r.Data[0].(*dal.Record); ok {
				scs, _ := k.Get("sales_channels")
				for _, sc := range records(scs) {
					id, _ := sc.Get("id")
					ids = append(ids, id)
				}
			}
		}
		req.Locals["publishableApiKeyScopes"] = ids
	}
	next(nil)
	return nil
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// EnsurePublishableKeyAndSalesChannelMatch mirrors ensurePublishableKeyAndSalesChannelMatch.
func EnsurePublishableKeyAndSalesChannelMatch(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	if req.Header.Get("x-publishable-api-key") != "" {
		ids, _ := req.Locals["publishableApiKeyScopes"].([]any)
		body, _ := req.ValidatedBody.(*dal.Record)
		var channel any
		if body != nil {
			channel, _ = body.Get("sales_channel_id")
		}
		if req.Errors == nil {
			req.Errors = []any{}
		}
		if len(ids) > 0 {
			if s, ok := channel.(string); ok && s != "" {
				found := false
				for _, id := range ids {
					if id == s {
						found = true
					}
				}
				if !found {
					req.Errors = append(req.Errors, fmt.Sprintf("Sales channel ID in payload %s is not associated with the Publishable API Key in the header.", s))
				}
			} else {
				if len(ids) > 1 {
					req.Errors = append(req.Errors, "Cannot assign sales channel to cart. The Publishable API Key in the header has multiple associated sales channels. Please provide a sales channel ID in the request body.")
				} else if body != nil {
					body.Set("sales_channel_id", ids[0])
				}
			}
		}
	}
	next(nil)
	return nil
}

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

func query() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(StoreGetCartsCart, RetrieveConfig)
}

// Middlewares mirrors storeCartRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/carts/:id", Middlewares: mw(query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreCreateCart)), query(),
		MaybeAttachPublishableKeyScopes, EnsurePublishableKeyAndSalesChannelMatch)},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreUpdateCart)), query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/line-items", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreAddCartLineItem)), query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/line-items/:line_id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreUpdateCartLineItem)), query())},
	{Methods: []string{"DELETE"}, Matcher: "/store/carts/:id/line-items/:line_id", Middlewares: mw(query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/shipping-methods", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreAddCartShippingMethods)), query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/promotions", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreCartPromotions)), query())},
	{Methods: []string{"DELETE"}, Matcher: "/store/carts/:id/promotions", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(StoreCartPromotions)), query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/complete", Middlewares: mw(httputils.ValidateAndTransformQuery(orders.StoreGetOrderParams, orders.RetrieveConfig))},
}

// RefetchCart mirrors refetchCart.
func RefetchCart(req *medusahttp.MedusaRequest, id any) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	v := dal.NewRecord()
	v.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "cart", v, req.QueryConfig.Fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 || r.Data[0] == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Cart with id '%v' not found", id))
	}
	return r.Data[0], nil
}

// RespondCart writes { cart } with status 200.
func RespondCart(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	cart, err := RefetchCart(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("cart", cart)
	return res.Status(200).JSON(out)
}

// POST mirrors POST /store/carts.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	if body != nil {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			in.Set(k, v)
		}
	}
	var actor any
	if req.AuthContext != nil {
		actor, _ = req.AuthContext.Get("actor_id")
	}
	if actor != nil {
		in.Set("customer_id", actor)
	}
	cart, err := cartworkflows.CreateCartWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	id, _ := cart.Get("id")
	return RespondCart(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}

// Get returns r[k] (nil when absent or r is nil).
func Get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}
