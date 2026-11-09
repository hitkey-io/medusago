// Package customers mirrors api/store/customers.
package customers

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

var defaultStoreCustomersFields = []string{"id", "email", "company_name", "first_name", "last_name", "phone", "metadata", "has_account", "deleted_at", "created_at", "updated_at", "*addresses"}

// DefaultStoreCustomerAddressFields mirrors defaultStoreCustomerAddressFields.
var DefaultStoreCustomerAddressFields = []string{"id", "address_name", "company", "customer_id", "first_name", "last_name", "address_1", "address_2", "city", "province", "postal_code", "country_code", "phone", "metadata", "is_default_shipping", "is_default_billing", "created_at", "updated_at"}

// Query configs.
var (
	RetrieveConfig        = httputils.QueryConfig{Defaults: defaultStoreCustomersFields, Allowed: storeutils.BuildAllowedFields(defaultStoreCustomersFields, []string{"orders"}, storeutils.PrefixAllowedFields("addresses", DefaultStoreCustomerAddressFields))}
	RetrieveAddressConfig = httputils.QueryConfig{Defaults: DefaultStoreCustomerAddressFields, Allowed: storeutils.BuildAllowedFields(DefaultStoreCustomerAddressFields)}
	ListAddressesConfig   = httputils.QueryConfig{Defaults: DefaultStoreCustomerAddressFields, Allowed: storeutils.BuildAllowedFields(DefaultStoreCustomerAddressFields), IsList: true}
)

func nullishString() *zod.Schema { return zod.String().Nullish() }

// Validators.
var (
	StoreGetCustomerParams = utils.CreateSelectParams()
	StoreCreateCustomer    = zod.Object(
		zod.F("email", zod.String().Email().Nullish()),
		zod.F("company_name", nullishString()),
		zod.F("first_name", nullishString()),
		zod.F("last_name", nullishString()),
		zod.F("phone", nullishString()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	)
	StoreUpdateCustomer = zod.Object(
		zod.F("company_name", nullishString()),
		zod.F("first_name", nullishString()),
		zod.F("last_name", nullishString()),
		zod.F("phone", nullishString()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	)
	StoreCreateCustomerAddress = commonvalidators.AddressPayload().Merge(zod.Object(
		zod.F("address_name", nullishString()),
		zod.F("is_default_shipping", zod.Boolean().Optional()),
		zod.F("is_default_billing", zod.Boolean().Optional()),
	))
	StoreGetCustomerAddressesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).Merge(zod.Object(
		zod.F("q", zod.String().Optional()),
		zod.F("city", utils.StringOrStrings().Optional()),
		zod.F("country_code", utils.StringOrStrings().Optional()),
		zod.F("postal_code", utils.StringOrStrings().Optional()),
	))
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

func body(s *zod.Schema) medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformBody(httputils.Schema(s))
}

func query(s *zod.Schema, c httputils.QueryConfig) medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(s, c)
}

const addressPath = "/store/customers/me/addresses/:address_id"

// Middlewares mirrors storeCustomerRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/store/customers", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}),
		body(StoreCreateCustomer), query(StoreGetCustomerParams, RetrieveConfig))},
	{Methods: []string{"ALL"}, Matcher: "/store/customers/me*", Middlewares: mw(middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}))},
	{Methods: []string{"GET"}, Matcher: "/store/customers/me", Middlewares: mw(query(StoreGetCustomerParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/store/customers/me", Middlewares: mw(body(StoreUpdateCustomer), query(StoreGetCustomerParams, RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/store/customers/me/addresses", Middlewares: mw(query(StoreGetCustomerAddressesParams, ListAddressesConfig))},
	{Methods: []string{"POST"}, Matcher: "/store/customers/me/addresses", Middlewares: mw(body(StoreCreateCustomerAddress), query(StoreGetCustomerParams, RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: addressPath, Middlewares: mw(query(utils.CreateSelectParams(), RetrieveAddressConfig))},
	{Methods: []string{"POST"}, Matcher: addressPath, Middlewares: mw(body(StoreCreateCustomerAddress), query(StoreGetCustomerParams, RetrieveConfig))},
	{Methods: []string{"DELETE"}, Matcher: addressPath, Middlewares: mw(query(utils.CreateSelectParams(), RetrieveConfig))},
}

// RefetchCustomer mirrors refetchCustomer.
func RefetchCustomer(req *medusahttp.MedusaRequest, id any) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	v := dal.NewRecord()
	v.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "customer", v, req.QueryConfig.Fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// ActorID returns req.auth_context.actor_id.
func ActorID(req *medusahttp.MedusaRequest) any {
	if req.AuthContext == nil {
		return nil
	}
	v, _ := req.AuthContext.Get("actor_id")
	return v
}

// RespondCustomer writes { customer } with status 200.
func RespondCustomer(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	c, err := RefetchCustomer(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("customer", c)
	return res.Status(200).JSON(out)
}

// POST mirrors POST /store/customers.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if a := ActorID(req); a != nil && a != "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Request already authenticated as a customer.")
	}
	data, _ := req.ValidatedBody.(*dal.Record)
	identity, _ := req.AuthContext.Get("auth_identity_id")
	customer, err := workflows.CreateCustomerAccountWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateCustomerAccountInput{AuthIdentityID: identity, CustomerData: data})
	if err != nil {
		return err
	}
	id, _ := customer.Get("id")
	return RespondCustomer(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
